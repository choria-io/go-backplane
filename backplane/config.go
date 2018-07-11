package backplane

import (
	"fmt"
	"regexp"
	"time"

	"github.com/choria-io/go-protocol/protocol"

	"github.com/choria-io/go-choria/choria"
	chconf "github.com/choria-io/go-choria/config"
)

// Config configures the backplane
type Config struct {
	name         string
	auth         Authorization
	brokers      []string
	appname      string
	logfile      string
	loglevel     string
	tls          *TLSConf
	provider     ConfigProvider
	opts         []Option
	fw           *choria.Framework
	ccfg         *chconf.Config
	factInterval time.Duration
	maxStopDelay time.Duration

	pausable        Pausable
	infosource      InfoSource
	healthcheckable HealthCheckable
	stopable        Stopable
}

// TLSConf describes the TLS config for a NATS connection
type TLSConf struct {
	// Identity will be the certificate name to use when attempting to find certs and validate connections
	Identity string `json:"identity" yaml:"identity"`

	// SSLDir is where the puppet scheme will look for the standard Puppet format directories
	SSLDir string `json:"ssl_dir" yaml:"ssl_dir"`

	// Scheme is either puppet or manual
	Scheme string `json:"scheme" yaml:"scheme"`

	// CA sets the path to the ca file in the manual scheme
	CA string `json:"ca" yaml:"ca"`

	// Cert sets the path to the certificate file in the manual scheme
	Cert string `json:"cert" yaml:"cert"`

	// Key sets the path to the key file in the manual scheme
	Key string `json:"key" yaml:"key"`

	// Cache sets the path to the directory the manual scheme will use to store certificates received over the wire
	Cache string `json:"cache" yaml:"cache"`
}

// ConfigProvider provides management backplane configuration
type ConfigProvider interface {
	// MiddlewareHosts are hosts in host:port format to connect to
	MiddlewareHosts() []string

	// Name is a unique name for this backplane, similarly named backplanes are grouped together
	// and isolated from others.  Valid names are ^[a-z0-9]+.
	Name() string

	// LogFile is the file to use for logging the backplane related logs, "" means stdout
	LogFile() string

	// LogLevel is the logging level, one of debug, info, warn, error
	LogLevel() string

	// TLS is a TLS configuration, nil meaning disable security
	TLS() *TLSConf

	// Auth configured allow/deny lists of who may access the service
	Auth() Authorization
}

// Option is a func that can configure the backplane
type Option func(*Config)

func newConfig(name string, cfg ConfigProvider, opts ...Option) (c *Config, err error) {
	c = &Config{
		name:         name,
		provider:     cfg,
		factInterval: 600 * time.Second,
		maxStopDelay: 10 * time.Second,
		opts:         opts,
	}

	if cfg.Name() == "" {
		return nil, fmt.Errorf("please specify an application name")
	}

	c.brokers = cfg.MiddlewareHosts()
	c.appname = fmt.Sprintf("%s_backplane", cfg.Name())
	c.logfile = cfg.LogFile()
	c.loglevel = cfg.LogLevel()
	c.tls = cfg.TLS()
	c.auth = cfg.Auth()

	for _, opt := range opts {
		opt(c)
	}

	if len(c.brokers) == 0 {
		return nil, fmt.Errorf("please specify backplane brokers")
	}

	if c.loglevel == "" {
		c.loglevel = "warn"
	}

	ok, err := regexp.MatchString("^[a-z0-9]+_backplane$", c.appname)
	if !ok || err != nil {
		return nil, fmt.Errorf("the application name must match ^[a-z0-9]+$")
	}

	c.ccfg, err = chconf.NewDefaultConfig()
	if err != nil {
		return
	}

	c.ccfg.Collectives = []string{c.appname}
	c.ccfg.MainCollective = c.appname
	c.ccfg.LogFile = c.logfile
	c.ccfg.LogLevel = c.loglevel
	c.ccfg.Choria.UseSRVRecords = false
	c.ccfg.Choria.MiddlewareHosts = c.brokers

	if c.tls != nil {
		c.ccfg.DisableTLS = false
		if c.tls.Identity != "" {
			c.ccfg.Identity = c.tls.Identity
		}

		switch c.tls.Scheme {
		case "puppet":
			c.ccfg.Choria.SecurityProvider = "puppet"
			c.ccfg.Choria.SSLDir = c.tls.SSLDir

		case "file", "manual":
			c.ccfg.Choria.SecurityProvider = "file"
			c.ccfg.Choria.FileSecurityCA = c.tls.CA
			c.ccfg.Choria.FileSecurityCertificate = c.tls.Cert
			c.ccfg.Choria.FileSecurityKey = c.tls.Key
			c.ccfg.Choria.FileSecurityCache = c.tls.Cache

		default:
			return nil, fmt.Errorf("security provider '%s' is not supported", c.tls.Scheme)
		}
	} else {
		c.ccfg.DisableTLS = true
		protocol.Secure = "false"
		c.ccfg.Choria.SecurityProvider = "file"
	}

	c.fw, err = choria.NewWithConfig(c.ccfg)
	if err != nil {
		return
	}

	return
}

// ManagePausable supplies a class that can be paused using the management agent
// without supplying a pausable the circuit breaker features are not enabled
func ManagePausable(p Pausable) Option {
	return func(c *Config) {
		c.pausable = p
	}
}

// ManageHealthCheck supplies a class that can be health checked using the
// management agent, without supplying this the health action will not be available
func ManageHealthCheck(h HealthCheckable) Option {
	return func(c *Config) {
		c.healthcheckable = h
	}
}

// ManageStopable supplies a class that can be stopped using the management
// agent, without supplying this the stop action will not be available
func ManageStopable(s Stopable) Option {
	return func(c *Config) {
		c.stopable = s
	}
}

// ManageInfoSource configures a fact source for discovery data
// without supplying a info source only basic discoverable data will be provided
func ManageInfoSource(f InfoSource) Option {
	return func(c *Config) {
		c.infosource = f
	}
}

// FactWriteInterval is the frequency that facts will be written to disk, 600 seconds is default
func FactWriteInterval(i time.Duration) Option {
	return func(c *Config) {
		c.factInterval = i
	}
}

// MaxStopDelay is the maximum time to wait before calling stop
func MaxStopDelay(i time.Duration) Option {
	return func(c *Config) {
		c.maxStopDelay = i
	}
}
