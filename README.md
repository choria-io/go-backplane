# Choria Management Backplane

This is a embedable Choria Server that you can use to provide a backplane for your Golang application.

You can use it to provide a secure, scalable and flexibile managment interface right inside your application with no dependencies other than a Choria broker infrastructure.

At present it is focussed on creating circuit breakers, health checks, managing logging level and shutdown actions that allow you to affect change of a large fleet of (micro)services rapidly and securely via CLI, Ruby API, Go API or Playbooks.

Using the Choria discovery abilities you can target subsets of services across your entire fleet using metadata of your choice.

Once embedded you can manage your fleet with commands like this:

Pause, resume and request info for all services in `DC1`

```
$ backplane yourapp pause -W dc=DC1
```

```
$ backplane yourapp resume -W dc=DC1
```

```
$ backplane yourapp health -W dc=DC1
```

```
$ backplane yourapp debuglvl -W dc=DC1
```

[![GoDoc](https://godoc.org/github.com/choria-io/go-backplane?status.svg)](https://godoc.org/github.com/choria-io/go-backplane)

## Motivation

It's typical for applications to expose REST interfaces that let one do things like circuit breaking on their internals.  This works perfectly fine in the general case of a small setup with one or two locations.

At scale though where one have 30+ DCs with machines behind various layers of bastion nodes and so forth this model rapidly breaks down as you'll have machine generated hostnames and ports and many services.  It becomes impossible to know what services are where and just gaining access to a HTTP port in all your data centers is a problem.

Instead one would set up a central management Choria infrastructure where these managed microservices will connect to.  The services connect out to the management network which is much easier to manage from a security perspective. Using the Choria discovery features you can target all or subsets of Microservices across your entire multi DC fleet in a fast way from either the CLI, Ruby API, Golang APIs or Choria Playbooks.

While a similar outcome can be achieved with a side car model - simply write a Ruby based agent for your app and deploy it into the running Choria Server on that node - this model is convenient for isolation and for true microservices in containers etc.

## Features

  * Circuit breaker interface
  * Health Check interface
  * Shutdown interface
  * Ability to switch running log levels of your applications
  * Ability to publish data from your app to the Choria Data Adapters that can convert the data to streaming data systems
  * Standardised configuration
  * TLS using PuppetCA or manual configuration
  * Expose Facts to the Choria discovery system
  * Authorization of Read Only and Full access based on certificate of the client

## Roadmap

  * Thread dump endpoint

## Exposed Actions

The following actions are exposed to the Choria network:

|Action    |Description|Interface|
|----------|-----------|---------|
|info      |Information such as pause state and facts|always present|
|ping      |Test connectivity to the backplane|always present|
|pause     |Pauses your application|Pausable|
|resume    |Resumes your application|Pausable|
|flip      |If paused, resume.  If not paused, pause.|Pausable|
|shutdown  |Shuts down your service after short delay|Stopable|
|health    |Checks the internal health of your service|HealthCheckable|
|debuglvl  |Sets the app to debug level logging|LogLevelSetable|
|infolvl  |Sets the app to info level logging|LogLevelSetable|
|warnlvl  |Sets the app to warning level logging|LogLevelSetable|
|critlvl  |Sets the app to critical level logging|LogLevelSetable|

## Infrastructure Requirements

The backplane agents use a Middleware server to connect to the management CLI. If you already have [Choria](https://choria.io) installed you have everything you need.  If you do not have Choria you can install if you wish, alternatively you just need a [NATS](https://github.com/nats-io/gnatsd) Server.

Installing Choria has a number of advantages:

 * You get a richer [RPC CLI](https://choria.io/docs/concepts/cli/), Ruby API and [Playbooks](https://choria.io/docs/playbooks/) that can all interact with your backplane
 * You can scale to massive scale and [federate multiple sites, regions, networks](https://choria.io/docs/federation/) together

If you chose not to install Choria the `backplane` CLI can be downloaded from the releases page, it needs a small config file like:

```
loglevel = warning
plugin.choria.middleware_hosts = nats.example.net:4222
```

Place this in `/etc/choria/client.cfg`, if you configured Choria the `backplane` CLI will just work.

## Setting up the Choria Client for Ruby API, Go API, CLI and Playbooks

If you wish to use the Ruby API or Choria CLI you should install the [mcollective_agent_backplane](https://forge.puppet.com/choria/mcollective_agent_backplane/readme) module from the Puppet Forge

```
mcollective::plugin_classes:
  - mcollective_agent_backplane
```

From there the usual `mco rpc`, Ruby API's and Playbooks will function.  For more information about Choria see it's [website](https://choria.io).

## Embeding in your application

To embed this backplane in your own Go code you need to implement a few interfaces, not all are required you can selectively enable just what you need.

**NOTE:** A working example is in the [example](example) directory

### Health Checks

To allow your application to be health checked you need to implement the `HealthCheckAble` interface, a simple version is here:

```go
import (
    backplane "github.com/choria-io/go-backplane/backplane"
)

type App struct {
    config *Config
    paused bool
    configured bool // setting this is not shown
}

type health struct {
    Configured bool
}

func (a *App) HealthCheck() (result interface{}, ok bool) {
    r := &health{
        Configured: a.configured,
    }

    return r, a.configured
}
```

The example is obviously over simplified and achieves very little - you can do any internal health checks you desired, I suggest keeping it fast and not testing remote APIs if you run many managed services.

Your result should be a structure - or something that satisfies the json interfaces.

Once enabled using the `backplane.ManageHealthCheck()` option (see below under embedding) this will be accessible via the `health` action.

### Circuit Breaker

To allow your application to be paused and resumed you need to implement the `Pausable` interface, a simple version that builds on the example above can be seen here:

```go
func (a *App) Pause() {
    a.paused = true
}

func (a *App) Resume() {
    a.paused = false
}

func (a *App) Flip() {
    a.paused = !a.paused
}

func (a *App) Paused() bool {
    return a.paused
}

func (a *App) Work(ctx context.Context) {
    ticker := time.NewTicker(time.Duration(a.config.Interval) * time.Millisecond)

    for {
        select {
            case <-ticker.C:
                if !a.Paused() {
                    fmt.Println("doing work")
                }
            case <-ctx.Done():
                return
        }
    }
}
```

Here the `Work()` method will do some work every configured interval unless the system is paused.

Once enabled using the `backplane.ManagePausable()` option (see below under embedding) this will be accessible via the `info`, `pause`, `resume` and `flip` actions.

### Shutdown

You can allow remote shutdowns of your application, to achieve this implement the `Stopable` interface:

```go
func (a *App) Shutdown() {
    os.Exit(0)
}
```

When you invoke the `shutdown` action via the Choria API it will schedule a shutdown after a random sleep duration rather than call it immediately.

You can combine this with a `Pausable` to drain connections first, but we don't support doing that automatically at present but might in the future.

Once enabled via the `backplane.ManageStopable()` option (see below under embedding) this will be accessible via the `shutdown` action.

### Log Level

You can allow the running log level of your application to be manipulated, to achieve this implement the `LogLevelSetable` interface:

Backplane has just 4 levels that it knows about - Debug, Info, Warn and Critical - you have to create a bit of a translation between what your application understand, here is a example using logrus:

```go
func (a *App) SetLogLevel(level backplane.LogLevel) {
	switch level {
	case backplane.InfoLevel:
		a.SetLogLevel(logrus.InfoLevel)
    case backplane.WarnLevel:
        a.SetLogLevel(logrus.WarnLevel)
    case backplane.CriticalLevel:
        a.SetLogLevel(logrus.FatalLevel)
    default:
        a.SetLogLevel(logrus.DebugLevel)
	}
}

func (a *App) GetLogLevel() backplane.LogLevel {
    switch a.LogLevel() {
    case logrus.InfoLevel:
        return backplane.InfoLevel
    case logrus.WarnLevel:
        return backplane.WarnLevel
    case logrus.ErrorLevel, logrus.FatalLevel, logrus.PanicLevel:
        return backplane.CriticalLevel
    default:
        return backplane.DebugLevel
    }
}
```

Once enabled via the `backplane.ManageLogLevel()` option (see below under embedding) this will be accessible via the `debuglvl`, `infolvl`, `warnlvl` and `critlvl` actions - `info` will show the active log level.

### Information Source

The `InfoSource` interface is required to expose some internals of your application to Choria, you should mark the structure fields up with `json` tags as this will be serialized to JSON.

Here we simply expose our running config as facts, you can return any structure here and that'll become facts.

```go
func (a *App) FactData() interface{} {
    return a.config
}

func (a *App) Version() string {
    return "0.0.1"
}
```

### Publishing Data 

You can publish data from your application to the [Choria Data Adapter](https://choria.io/docs/adapters/) system which can receive the data in a scalable manner and transform it to Streaming Data system.

This is useful for publishing IoT environmental data or other similar data to a network, your data will traverse the network maintained by the backplane and will be secured using PKI and TLS.  

When you start the backplane (full details below) pass the `backplane.StartDataPublisher()` option to `backplane.Run()`.

From your code you can publish any data:

```go
    // initialize the backplane, full example in "Starting the Server" section
    opts := []backplane.Option{
        // other options
        backplane.StartDataPublisher(),
    }

    pb, err := backplane.Run(ctx, wg, a.config.Management, opts...)
    if err != nil {
        panic(err)
    }

    // dat is a []byte with any information you wish to publish
    dat := gatherEnvironmentData()

    // publishes data on a NATS topica called acme.iot
    pb.DataOutbox() <- &backplane.DataItem{
        Data: dat,
        Destination: "acme.iot",
    }
```

You can configure the Choria Broker to receive this data and publish it to NATS Streaming:

```ini
plugin.choria.adapters = iot
plugin.choria.adapter.iot.type = nats_stream
plugin.choria.adapter.iot.stream.servers = stan1:4222,stan2:4222
plugin.choria.adapter.iot.stream.clusterid = prod
plugin.choria.adapter.iot.ingest.topic = acme.iot
plugin.choria.adapter.iot.ingest.protocol = reply
```

This ingest the `acme.iot` topic, rewrite the data and publish it to NATS Streaming `prod` cluster on `stan1:4222` and `stan2:4222`.

### Configure Choria

You have to supply some basic configuration to the Choria framework, you need to implement the `ConfigProvider` interface, you're welcome to do this yourself but we provide one you can use.  We recommend you use this one so that all backplane managed interface have the same configuration format:

```go
type Config struct {
	Interval   int                              `yaml:"interval"`
	Management *backplane.StandardConfiguration `yaml:"management"`
}
```

This implements the full `ConfigProvider` interface and supports TLS etc, it's got tags for YAML and JSON already.

You config file might look like this:

```yaml
# your own config here
interval: 600

# Standard Backplane specific configuration here
management:
    name: app
    logfile: "/var/log/app/backplane.log"
    loglevel: warn
    tls:
        scheme: file
        ca: /path/to/ca.pem
        cert: /path/to/cert.pem
        key: /path/to/key.pem
        cache: /path/to/ssl_cache

    auth:
        full:
            - sre.choria

        read_only:
            - 1stline.choria

    brokers:
        - choria1.example.net:4222
        - choria2.example.net:4222
```

#### Authorization

Authorization is supported by a simple allow all, allow readonly or insecure flags. The configuration above allows the user `sre.choria` to pause, resume, flip and shutdown plus all the read only actions while the `1stline.choria` user can get info and health checks. The strings supplied are treated as Regular Expressions.

Authorization can be disabled with the following, any user will be able to do anything now:

```yaml
auth:
    insecure: true
```

#### TLS

As you can see TLS is supported in the configuration, it's optional but recommended - and required unless you build custom brokers or use NATS.io `gnatsd` without security.

If you use Puppet you can simplify the TLS like this:

```yaml
tls:
    scheme: puppet
```

It will then use the nodes certificates etc if you run it as root, if not root use the [pki-enroll](https://github.com/choria-io/go-security) command to enroll the system into the PuppetCA

If you have your own CA or already enrolled you can configure it manually as above.  The `cache` is simply a directory on the node where Choria will write some cached public certificates.

### Starting the server

Above we built a simple pausable, shutdownable and health checkable application that does some work unless paused, it exposes it's configuration as facts, now we can just embed our server and start it:

```go
func (a *App) startBackPlane(ctx context.Context, wg *sync.Waitgroup) error {
    if a.config.Management != nil {
        opts := []backplane.Option{
            backplane.ManageInfoSource(a.config),
            backplane.ManagePausable(a),
            backplane.ManageHealthCheck(a),
            backplane.ManageStopable(a),
            backplane.ManageLogLevel(a),
            backplane.StartDataPublisher(),
        }

        _, err := backplane.Run(ctx, wg, a.config.Management, opts...)
        if err != nil {
            return err
        }
    }

    return nil
}
```

Once you call `startBackPlane()` in your startup cycle it will start a Choria instance with the `discovery`, `choria_util` and `backplane` agents, the `backplane` agent will have all the actions listed in the earlier table, your config will be shown in the `info` action and you can discovery it using any of the facts.

If you only supply some of `ManageInfoSource`, `ManagePausable`, `ManageHealthCheck`, `ManageLogLevel`, `StartDataPublisher` and `ManageStopable` the features of the agent will be selectively disabled as per the table earlier.

All backplane managed services will use the `backplane` agent name, to differentiate the `name` will be used to construct a sub collective name so each app is effectively contained. The upcoming CLI will be built around this design.

## Docker Demo

A Docker based demo is included, you need `docker-compose` setup and working, this demo sets up 2 backplane services and the CLI ready to use, no Choria infrastructure is needed when security is not configured, just a NATS server.  This demo uses the official NATS image for this.

```
$ docker-compose up --scale demo1=2 --scale demo2=2
....
demo2_1   | 2018/07/11 08:48:52 demo2: doing work
demo1_1   | 2018/07/11 08:48:52 demo1: doing work
demo2_2   | 2018/07/11 08:48:53 demo2: doing work
demo1_2   | 2018/07/11 08:48:54 demo1: doing work
demo2_1   | 2018/07/11 08:48:54 demo2: doing work
demo1_1   | 2018/07/11 08:48:54 demo1: doing work
demo2_2   | 2018/07/11 08:48:55 demo2: doing work
```

This starts 2 instances of the 2 demo services up along with a broker, it will open port 4222 on your host.

From another shell you can now use the backplane CLI to manage this network, replace the IP address with the address on your network card.

**NOTE:** We pass `--insecure` else the system will initiate the Choria security system requiring certificates etc which the compose demo does not have.

Lets look at the information available about the services, their features and more:

```
$ docker run -e BROKER=192.168.1.78:4222 choria/backplane --insecure demo2 info
Starting discovery process for demo2 backplan managed services: 2

Performing info... ✓ 2 / 2

  ceb29c26146b:
           App Version: 0.0.1
     Backplane Version: 0.0.2
                Paused: false
               Healthy: true
         Pause Feature: ✓
         Facts Feature: ✓
        Health Feature: ✓
      Shutdown Feature: ✓

  b93660b8feb6:
           App Version: 0.0.1
     Backplane Version: 0.0.2
                Paused: false
               Healthy: true
         Pause Feature: ✓
         Facts Feature: ✓
        Health Feature: ✓
      Shutdown Feature: ✓


Managed 2 service(s) in 22ms
```

You can circuit break the services which should stop any work from happening in the ones you target:

```
% docker run -e BROKER=192.168.1.78:4222 choria/backplane --insecure demo2 pause
Starting discovery process for demo2 backplan managed services: 2

Performing pause... ✓ 2 / 2


Managed 2 service(s) in 36ms
```

Your logs will now show lines like `demo2: skipping work while paused`.

A note about the Choria display model, generally we assume you will manage large numbers of nodes or services and will try to show what is relevant.  In the above command it showed no output about individual services because it was able to successfully pause each.  It would only show you failures.  However when we requested `info` obviously you'd like to see the data so it shows it.

You can resume the work again:

```
% docker run -e BROKER=192.168.1.78:4222 choria/backplane --insecure demo2 resume
...
```

All instances should resume logging `doing work` lines.

You can use this setup to test your own embedded services, just point them at this broker without configuring TLS in their configuration and they should be visible via the CLI.

The demo applications do not support much by way of facts as they are very simple at the moment, you can see from `backplane help exec` though that complex filtering is supported:

```
usage: backplane exec [<flags>] <service> <action>

Executes a action against a set of backplane managed services

Flags:
      --help             Show context-sensitive help (also try --help-long and --help-man).
      --version          Show application version.
  -v, --verbose          Enable verbose output
  -d, --debug            Enable debug logging
  -F, --wf=FACTS ...     Match services with a certain fact
  -I, --wi=IDENTITY ...  Match services with a certain Choria identity
      --timeout=TIMEOUT  How long to wait for services to respond
      --config=CONFIG    Configuration file to use
      --insecure         Disable TLS security

Args:
  <service>  The services name to manage
  <action>   Action to perform against the managed service
```

The facts that would be exposed for filtering are those from the InfoSource and you could for example do `-F Interval=600` in this case - directly exposed from the example app configuration.

Using this we could for example pause just one specific service:

```
% docker run -e BROKER=192.168.1.78:4222 choria/backplane --insecure demo2 pause -I 1786991ad26d
Starting discovery process for demo2 backplan managed services: 1

Performing pause... ✓ 1 / 1


Managed 1 service(s) in 19ms
```

Only 1 node were managed, info confirms:

```
Performing info... ✓ 2 / 2

  117a7530ac36:
           App Version: 0.0.1
     Backplane Version: 0.0.2
                Paused: false
               Healthy: true
         Pause Feature: ✓
         Facts Feature: ✓
        Health Feature: ✓
      Shutdown Feature: ✓

  1786991ad26d:
           App Version: 0.0.1
     Backplane Version: 0.0.2
                Paused: true
               Healthy: true
         Pause Feature: ✓
         Facts Feature: ✓
        Health Feature: ✓
      Shutdown Feature: ✓
```

1 is paused and 1 is not, your logs should also confirm.

Additionally on every `doing work` line data gets published to the NATS network topic `myapp.data` in the Choria format.  You can view these using the a [nats client](https://github.com/nats-io/go-nats/tree/master/examples/nats-sub) or had this been a Choria Broker you could adapt these messages to a NATS Stream using the Choria Adapter Framework.