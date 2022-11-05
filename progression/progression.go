package progression

import (
	"github.com/Shopify/go-lua"
	"time"
)

type ProgressionProvider interface {
	Next() (bool, *float64, int64)
	WithLuaState(state *lua.State)
}

type Realtime struct {
	timesAlready float64
	Initial      float64
	Increment    float64
	Fn           string
	luaState     *lua.State
}

type Progression struct {
	NoData       bool
	timesAlready float64
	Initial      float64
	Increment    float64
	Times        float64
	Fn           string
}

type ProgressionList struct {
	index          int
	iterations     int64
	interval       time.Duration
	startTimestamp int64
	progressions   []*Progression
	luaState       *lua.State
}

func (p *Realtime) Next() (bool, *float64, int64) {
	var nextVal float64
	if p.Fn != "" && p.luaState != nil {
		p.luaState.Global(p.Fn)
		p.luaState.PushNumber(p.Initial)
		p.luaState.PushNumber(p.timesAlready)
		p.luaState.Call(2, 1)
		lua.CheckNumber(p.luaState, p.luaState.Top())
		nextVal, _ = p.luaState.ToNumber(p.luaState.Top())
		// empty stack
		p.luaState.Pop(p.luaState.Top())
	} else {
		nextVal = p.Initial + (p.timesAlready * p.Increment)
	}
	p.timesAlready++
	return true, &nextVal, time.Now().UnixMilli()
}

func (p *Realtime) WithLuaState(state *lua.State) {
	p.luaState = state
}

func (p *Progression) Next(luaState *lua.State) (bool, *float64) {
	if p.timesAlready >= p.Times {
		return false, nil
	}
	p.timesAlready += 1
	var val float64
	if p.NoData {
		return true, nil
	}

	if p.Fn != "" && luaState != nil {
		luaState.Global(p.Fn)
		luaState.PushNumber(p.Initial)
		luaState.PushNumber(p.timesAlready)
		luaState.Call(2, 1)
		lua.CheckNumber(luaState, luaState.Top())
		val, _ = luaState.ToNumber(luaState.Top())
		// empty stack
		luaState.Pop(luaState.Top())
	} else {
		val = p.Initial + (p.timesAlready * p.Increment)
	}

	return true, &val

}

func (p *ProgressionList) Next() (bool, *float64, int64) {
	valid, val := p.progressions[p.index].Next(p.luaState)
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

func (p *ProgressionList) WithLuaState(state *lua.State) {
	p.luaState = state
}
