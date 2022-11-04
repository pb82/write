package progression

import "time"

type ProgressionProvider interface {
	Next() (bool, *float64, int64)
}

type Realtime struct {
	timesAlready float64
	Initial      float64
	Increment    float64
}

type Progression struct {
	NoData       bool
	timesAlready float64
	Initial      float64
	Increment    float64
	Times        float64
}

type ProgressionList struct {
	index          int
	iterations     int64
	interval       time.Duration
	startTimestamp int64
	progressions   []*Progression
}

func (p *Realtime) Next() (bool, *float64, int64) {
	nextVal := p.Initial + (p.timesAlready * p.Increment)
	p.timesAlready++
	return true, &nextVal, time.Now().UnixMilli()
}

func (p *Progression) Next() (bool, *float64) {
	if p.timesAlready >= p.Times {
		return false, nil
	}
	p.timesAlready += 1
	var val float64
	if p.NoData {
		return true, nil
	} else {
		val = p.Initial + (float64(p.timesAlready) * p.Increment)
		return true, &val
	}
}

func (p *ProgressionList) Next() (bool, *float64, int64) {
	valid, val := p.progressions[p.index].Next()
	if !valid {
		if p.index >= len(p.progressions)-1 {
			return false, nil, 0
		}
		p.index += 1
		return p.Next()
	}
	ts := p.startTimestamp + (p.iterations * p.interval.Milliseconds())
	p.iterations++
	return true, val, ts
}

func (p *ProgressionList) count() int64 {
	var sum int64 = 0
	for _, progression := range p.progressions {
		sum += int64(progression.Times)
	}
	return sum
}
