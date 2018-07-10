# Choria Management Backplane

This is a embedable Choria Server that you can use to provide a backplane for your Golang application.

You can use it to provide a secure, scalable and flexibile managment interface right inside your application with no dependencies other than a Choria broker infrastructure.

At present it is focussed on creating circuit breakers, health checks and shutdown ations that allow you to affect change of a large fleet of (micro)services rapidly and securely via CLI, Ruby API, Go API or Playbooks.

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

## Embeding

To embed this backplane in your own Go code you need to implement a few interfaces, not all are required you can selectively enable just what you need.

**NOTE:** A working example is in the [example](example) directory

### Health Checks

To allow your application to be health checked you need to implement the `HealthCheckAble` interface, a simple version is here:

```go
import (
    backplane "github.com/choria-io/go-backplane"
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

If you only supply some of `ManageInfoSource`, `ManagePausable`, `ManageHealthCheck` and `ManageStopable` the features of the agent will be selectively disabled as per the table earlier.

All backplane managed services will use the `backplane` agent name, to differentiate the `name` will be used to construct a sub collective name so each app is effectively contained. The upcoming CLI will be built around this design.