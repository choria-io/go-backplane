# Choria Management Backplane

This is a embedable Choria Server that you can use to provide a backplane for your application.

At present it is focussed on creating a circuit breaker that allow you to pause/resume/query the circuit breaker status of a large fleet of (micro)services.

Using the Choria discovery abilities you can target subsets of services for circuit breaking.  In time other management capabilities will be added via other interfaces.

Once embedded you can manage your fleet with commands like this:

Pause, resume and request info for all services in `DC1`

```
$ mco rpc circuit pause -W dc=DC1
```

```
$ mco rpc circuit resume -W dc=DC1
```

```
$ mco rpc circuit info -W dc=DC1
```

Your system can also expose it's configuration and other items as facts that can be used for fine tuned targeting of actions

## Motivation

It's typical for applications to expose REST interfaces that let one do things like circuit breaking on their internals.  This works perfectly fine in the general case of a small setup with one or two locations.

At scale though where one have 30+ DCs with machines behind various layers of bastion node and so forth this model rapidly breaks down as you'll have machine generated hostnames and ports.  It becomes impossible to know what services are where and just gaining access to a HTTP port in all your data centers is a problem.

Instead by relying on the Choria discovery system one would set up a central management Choria infrastructure where these managed microservices will connect to.  The services connect out to the management network which is much easier to manage from a security perspective.

There one can use commands like the above `mco rpc` commands to target the entire fleet of microservices at the same time giving you rapid access to these essential circuit breaking facilities.

## Features

  * Circuit breaker interface
  * Standardised configuration
  * TLS using PuppetCA or manual configuration
  * Expose Facts to the Choria discovery system
  * Authorization of Read Only and Full access based on certificate of the client

## Roadmap

  * Ability to add your own actions to the agent
  * Ability to pass in entire agents into the running instance
  * Generation of DDL files

## Embeding

To embed this backplane in your own Go code you need to implement a few interfaces.

### Circuit Breaker

To allow your application to be paused and resumed you need to implement the `Pausable` interface:

```go
type Pausable interface {
	Pause()
	Resume()
	Flip()
	Paused() bool
	Version() string
}
```

A simple version of this can be seen here:

```go
import (
    backplane "github.com/choria-io/go-backplane"
)

type App struct {
    config *Config
    paused bool
}

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

func (a *App) Version() string {
    return "0.0.1"
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

Here the `Work()` method will do some work every 500 milliseconds unless the system is paused.

### Configure Choria

You have to supply some basic configuration to the Choria framework, you need to implement the `ConfigProvider` interface, you're welcome to do this yourself but we provide one you can use.  We recommend we use this one so that all backplane managed interface have the same configuration format:

```go
type Config struct {
    Interval int
    Management *backplane.StandardConfiguration `json:"management" yaml:"management"`
}
```

This implements the full `ConfigProvider` interface and supports TLS etc, it's got tags for YAML and JSON already.

You config file might look like this:

```yaml
# your own config here
interval: 600

# Backplane specific configuration here
management:
    collective: app
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

#### Authorization

Authorization is supported by a simple allow all, allow readonly or insecure flags. The configuration above allows the user `sre.choria` to pause, resume and flip the service while the `1stline.choria` user can get info.

Authorization can be disabled with the following:

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

### Fact Source

The `FactSource` interface is required to expose some internals of your application to Choria, you should mark the structure fields up with `json` tags as this will be serialized to JSON.

Here we simply expose our running config as facts, you can return any structure here and that'll become facts.

```go
func (a *App) FactData() interface{} {
    return a.config
}
```

### Starting the server

Above we built a simple pausable application that does some work unless paused, it exposes it's configuration as facts, now we can just embed our server and start it:

```go
func (a *App) startBackPlane(ctx context.Context, wg *sync.Waitgroup) error {
    if a.config.Management != nil {
        _, err := backplane.Run(ctx, wg, "app", a.config.Management, backplane.ManageFactSource(a.config), backplane.ManagePausable(a))
		if err != nil {
			return err
		}
    }

    return nil
}
```

Once you call `startBackPlane()` in your startup cycle it will start a Choria instance with the `discovery`, `choria_util` and `app` agents, the app agent will have `info`, `pause`, `resume` and `flip` actions, your config will be shown in the `info` action and you can discovery it using any of the facts.