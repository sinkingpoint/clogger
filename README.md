# Clogger

Clogger is a WIP version of my idealized logging system. In many ways, it is similar to [syslog-ng](https://github.com/syslog-ng/syslog-ng), [FluentBit](https://fluentbit.io/), or [LogStash](https://www.elastic.co/logstash/), but tailored with features that I want in a logging system.

## Features

Some of the notable features that already exist:

- Arbitrary filters, written in [Tengo](http://github.com/d5/tengo)
- Deep Observability - Clogger comes with Metrics, Logs, and Tracing out of the box to help debug and monitor pipelines
- Buffer locations - In the case that an output destination is down, buffer outputs can be configured as an alternative location to send messages to

## Configuration

Any valid clogger configuration is also a valid [GraphViz DOT](https://graphviz.org/doc/info/lang.html) file, meaning that you can directly render out your configurations into diagrams of your pipeline.

Inputs, Outputs, and Filters all form nodes in the graph, with edges being the pipes between them. Properties of a node are defined in the DOT attributes, e.g. to construct a Unix Socket Input (an input that receives data over a UNIX socket), you can use the following (note that attributes with special chars have to be quoted):

```
MyInput [type=unix listen="/run/clogger.sock"]
```

The one mandatory attribute on each node is the `type` attribute - this defines what kind of input, output, or filter it is. Beyond that, each node type can define its own attributes (such as the above Unix input defining a `listen` attribute to determine where to place the socket).

You could also construct a StdOutput (An output that sends messages to the local stdout) similarly:

```
MyOutput [type=stdout format=console color=true]
```

Note that we've specified that we want to output a Colored Console output instead of the default JSON.

We can string them together into a complete config like so:

```
digraph pipeline {
    MyInput [type=unix listen="/run/clogger/clogger.sock"]
    MyOutput [type=stdout format=console color=true]

    MyInput -> MyOutput
}
```

Which creates a Clogger instance that reads data from a Unix socket and writes it to the console

