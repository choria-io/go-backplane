package backplane

import (
	"fmt"
	"time"

	"github.com/choria-io/go-choria/choria"
	chconf "github.com/choria-io/go-choria/config"
)

// Config configures the backplane
type Config struct {
	brokers      []string
	collectives  []string
	logfile      string
	loglevel     string
	tls          *TLSConf
	name         string
	provider     ConfigProvider
	opts         []Option
	pausable     Pausable
	factsource   FactSource
	fw           *choria.Framework
	ccfg         *chconf.Config
	factinterval time.Duration
}

// TLSConf describes the TLS config for a NATS connection
type TLSConf struct {
	Identity string `json:"identity" yaml:"identity"`
	SSLDir   string `json:"ssl_dir" yaml:"ssl_dir"`
	Scheme   string `json:"scheme" yaml:"scheme"`
	CA       string `json:"ca" yaml:"ca"`
	Cert     string `json:"cert" yaml:"cert"`
	Key      string `json:"key" yaml:"key"`
	Cache    string `json:"cache" yaml:"cache"`
}

// ConfigProvider provides management backplane configuration
type ConfigProvider interface {
	MiddlewareHosts() []string
	Collectives() []string
	LogFile() string
	LogLevel() string
	TLS() *TLSConf
}

// Option is a func that can configure the backplane
type Option func(*Config)

func newConfig(name string, cfg ConfigProvider, opts ...Option) (c *Config, err error) {
	c = &Config{
		name:         name,
		provider:     cfg,
		factinterval: 600 * time.Second,
		opts:         opts,
	}

	c.brokers = cfg.MiddlewareHosts()
	c.collectives = cfg.Collectives()
	c.logfile = cfg.LogFile()
	c.loglevel = cfg.LogLevel()
	c.tls = cfg.TLS()

	for _, opt := range opts {
		opt(c)
	}

	if len(c.brokers) == 0 {
		return nil, fmt.Errorf("please specify backplane brokers")
	}

	if len(c.collectives) == 0 {
		return nil, fmt.Errorf("please specify a list of collectives")
	}

	if c.loglevel == "" {
		c.loglevel = "warn"
	}

	c.ccfg, err = chconf.NewDefaultConfig()
	if err != nil {
		return
	}

	c.ccfg.Collectives = c.collectives
	c.ccfg.MainCollective = c.collectives[0]
	c.ccfg.LogFile = c.logfile
	c.ccfg.LogLevel = c.loglevel
	c.ccfg.Choria.UseSRVRecords = false
	c.ccfg.Choria.MiddlewareHosts = c.brokers
	c.ccfg.DisableTLS = true

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

// ManageFactSource configures a fact source for discovery data
// without supplying a factsource only basic discoverable data will be provided
func ManageFactSource(f FactSource) Option {
	return func(c *Config) {
		c.factsource = f
	}
}

// FactWriteInterval is the frequency that facts will be written to disk, 600 seconds is default
func FactWriteInterval(i time.Duration) Option {
	return func(c *Config) {
		c.factinterval = i
	}
}
