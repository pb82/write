package main

import (
	"bytes"
	"context"
	"flag"
	"fmt"
	"github.com/ghodss/yaml"
	"github.com/golang/protobuf/proto"
	"github.com/golang/snappy"
	"go.buf.build/protocolbuffers/go/prometheus/prometheus"
	"net/http"
	"net/url"
	"os"
	"path"
	"time"
	"write/ingest"
	"write/progression"
)

const (
	RemoteWriteUrl    = "http://localhost:9090/api/v1/write"
	DefaultConfigFile = "./config.yml"
)

type ConfigTimeseries struct {
	Series      string `json:"series"`
	Progression string `json:"progression"`
}

type ConfigRoot struct {
	Interval string             `json:"interval"`
	Series   []ConfigTimeseries `json:"time_series"`
}

func sendRequest(wr *prometheus.WriteRequest, url *url.URL) {
	data, _ := proto.Marshal(wr)
	encoded := snappy.Encode(nil, data)

	body := bytes.NewReader(encoded)
	req, err := http.NewRequest("POST", url.String(), body)
	if err != nil {
		panic(err)
	}

	req.Header.Set("Content-Type", "application/x-protobuf")
	req.Header.Set("Content-Encoding", "snappy")
	req.Header.Set("X-Prometheus-Remote-Write-Version", "0.1.0")

	httpClient := http.Client{
		Timeout: 30 * time.Second,
	}

	resp, err := httpClient.Do(req.WithContext(context.TODO()))
	if err != nil {
		panic(err)
	}

	fmt.Println(fmt.Sprintf("remote write status code: %v", resp.StatusCode))
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

	for _, wr := range writeRequests {
		sendRequest(&wr, parsedUrl)
	}
}
