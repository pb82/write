A tool for generating Prometheus metrics
========================================

This is a small tool that allows you to start a Prometheus instance and send metrics data to it.
It reuses the `promtool` nontaion for defining series and values.

Why?
====

It's useful for testing queries and Alerts against a real Prometheus instance.

How it works
============

This tool uses the Prometheus remote write API to send either precalculated or realtime samples.

Getting started
===============

Build: `make build`

1. Start Prometheus using the included Makefile: `make prom`
2. Define your series in a file, e.g. `config.yaml`
3. Run the tool: `./write --prometheus.url=http://localhost:9090 --config.file=config.yml`
4. Open the Prometheus UI in a browser under `localhost:9090`
5. You should see the data and you can interact with it

Config file syntax
==================

The layout of the config file is similar to `promtool` with some additions:

```yaml
---
interval: <time.Duration, how often to send samples>
time_series:
  - series: example_series{example_label="example_value"}
    progression: <precalculated series>
    realtime: <realtime series>
```

You can have any number of time series.

Precalculated series
--------------------

Progression: <Series>+
Series: <Initial><Op><Increment>x<Times>
Op: + | -
Initial: Number
Increment: Number
Times: Number

Realtime series
---------------

Realtime: <Series>+
Series: <Initial><Op><Increment>
Op: + | -
Initial: Number
Increment: Number

Scripting
=========

Lua scripts can be defined in a file and referenced with `--scripting.file=<lua file>`.
Functions in the file can be invoked in the `<Increment>` part of either precalculated or realtime series.
Script functions are passed the initial value and the number of repetitions, e.g.

```lua
math.randomseed(unixtimemillis())

function rnd(initialValue, times)
    return math.random()
end
```

defines a function that returns random values. Invoke it in a realtime series using parantheses:

```yaml
  - series: example_series{example_label="example_value"}
    realtime: "0+(rnd)"
```