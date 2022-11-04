package main

import (
	"bytes"
	"context"
	"errors"
	"flag"
	"fmt"
	"github.com/ghodss/yaml"
	"github.com/golang/protobuf/proto"
	"github.com/golang/snappy"
	"go.buf.build/protocolbuffers/go/prometheus/prometheus"
	"log"
	"net/http"
	"net/url"
	"os"
	"os/signal"
	"path"
	"sync"
	"syscall"
	"time"
	"write/ingest"
	"write/progression"
)

const (
	DefaultConfigFile = "./config.yml"
)

type ConfigTimeseries struct {
	Series      string `json:"series"`
	Progression string `json:"progression"`
	Realtime    string `json:"realtime"`
}

type ConfigRoot struct {
	Interval string             `json:"interval"`
	Series   []ConfigTimeseries `json:"time_series"`
}

type RealtimeContext struct {
	rt progression.ProgressionProvider
	ts *prometheus.TimeSeries
}

func sendRequest(wr *prometheus.WriteRequest, url *url.URL) error {
	data, _ := proto.Marshal(wr)
	encoded := snappy.Encode(nil, data)

	body := bytes.NewReader(encoded)
	req, err := http.NewRequest("POST", url.String(), body)
	if err != nil {
		return err
	}

	req.Header.Set("Content-Type", "application/x-protobuf")
	req.Header.Set("Content-Encoding", "snappy")
	req.Header.Set("X-Prometheus-Remote-Write-Version", "0.1.0")

	httpClient := http.Client{
		Timeout: 30 * time.Second,
	}

	resp, err := httpClient.Do(req.WithContext(context.TODO()))
	if err != nil {
		return err
	}

	defer resp.Body.Close()
	if resp.StatusCode < 200 || resp.StatusCode >= 300 {
		if resp.StatusCode == 400 {
			// possibly duplicate data? ignore it.
			log.Println("invalid data detected, ignoring it")
			return nil
		}

		return errors.New(fmt.Sprintf("unexpected remote write status code: %v", resp.StatusCode))
	}

	return nil
}

var (
	prometheusUrl *string
	configFile    *string
)

func init() {
	prometheusUrl = flag.String("prometheus.url", "", "prometheus http url")
	configFile = flag.String("config.file", DefaultConfigFile, "config file location")
}

func main() {
	flag.Parse()

	if prometheusUrl == nil || *prometheusUrl == "" {
		fmt.Println("missing value: prometheus.url")
		os.Exit(1)
	}

	if configFile == nil || *configFile == "" {
		fmt.Println("missing value: config.file")
		os.Exit(1)
	}

	parsedUrl, err := url.Parse(*prometheusUrl)
	if err != nil {
		panic(err)
	}

	parsedUrl.Path = path.Join(parsedUrl.Path, "/api/v1/write")

	raw, _ := os.ReadFile(*configFile)
	root := ConfigRoot{}
	yaml.Unmarshal(raw, &root)

	interval, err := time.ParseDuration(root.Interval)
	if err != nil {
		panic(err)
	}

	scanner := ingest.NewTimeseriesScanner()
	progScanner := progression.NewProgressionScanner()

	var writeRequests []prometheus.WriteRequest
	var realtimeProgressions []RealtimeContext

	for _, ts := range root.Series {
		tokens, err := scanner.Scan(ts.Series)
		if err != nil {
			panic(err)
		}

		parser := ingest.NewTimeseriesParser(tokens)
		parsedTimeseries, err := parser.Parse()
		if err != nil {
			panic(err)
		}

		if ts.Progression != "" {
			progTokens, err := progScanner.Scan(ts.Progression)
			progParser := progression.NewProgressionParser(progTokens)
			progressions, err := progParser.Parse(interval)
			if err != nil {
				panic(err)
			}

			writeRequest := prometheus.WriteRequest{}
			writeRequest.Timeseries = append(writeRequest.Timeseries, parsedTimeseries)
			writeRequest.Metadata = append(writeRequest.Metadata, &prometheus.MetricMetadata{
				Type: prometheus.MetricMetadata_GAUGE,
			})

			for true {
				valid, value, timestamp := progressions.Next()
				if !valid {
					break
				}

				if value != nil {
					parsedTimeseries.Samples = append(parsedTimeseries.Samples, &prometheus.Sample{
						Value:     *value,
						Timestamp: timestamp,
					})
				}
			}
			writeRequests = append(writeRequests, writeRequest)
		}

		if ts.Realtime != "" {
			rtTokens, err := progScanner.Scan(ts.Realtime)
			progParser := progression.NewProgressionParser(rtTokens)
			rt, err := progParser.ParseRealtime()
			if err != nil {
				panic(err)
			}

			realtimeProgressions = append(realtimeProgressions, RealtimeContext{
				rt: rt,
				ts: parsedTimeseries,
			})
		}

	}

	for _, wr := range writeRequests {
		err = sendRequest(&wr, parsedUrl)
		if err != nil {
			log.Fatalf("error writing series %v: %v", wr.String(), err)
		}
	}

	log.Println("done writing precalculated series")

	if len(realtimeProgressions) > 0 {
		log.Println("entering realtime mode")
		wg := sync.WaitGroup{}
		stop := make(chan bool)
		sigs := make(chan os.Signal)
		signal.Notify(sigs, syscall.SIGTERM, syscall.SIGABRT, syscall.SIGINT)
		go func() {
			sig := <-sigs
			switch sig {
			case syscall.SIGTERM, syscall.SIGABRT, syscall.SIGINT:
				stop <- true
			}
		}()

		for _, rt := range realtimeProgressions {
			wg.Add(1)
			go func() {
				for {
					select {
					case _ = <-stop:
						log.Println("stop signal received")
						wg.Done()
						return
					case <-time.After(interval):
						valid, value, timestamp := rt.rt.Next()
						if valid && value != nil {
							timeseries := prometheus.TimeSeries{}
							timeseries.Labels = rt.ts.Labels
							timeseries.Samples = append(timeseries.Samples, &prometheus.Sample{
								Value:     *value,
								Timestamp: timestamp,
							})
							wr := &prometheus.WriteRequest{}
							wr.Timeseries = append(wr.Timeseries, &timeseries)
							err = sendRequest(wr, parsedUrl)
							if err != nil {
								log.Fatalf("error writing series %v: %v", wr.String(), err)
							} else {
								log.Println(fmt.Sprintf("next value: %v", wr.String()))
							}
						}
					}
				}
			}()
		}

		wg.Wait()
	}
}
