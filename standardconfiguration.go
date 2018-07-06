package backplane

// StandardConfiguration implements ConfigProvider
// you can use this as a helper in your own code
// to give users the ability to configure the backplane
type StandardConfiguration struct {
	Brokers       []string      `json:"brokers" yaml:"brokers"`
	Collective    string        `json:"collective" yaml:"collective"`
	LogFilePath   string        `json:"logfile" yaml:"logfile"`
	Loglevel      string        `json:"loglevel" yaml:"loglevel"`
	TLSConf       *TLSConf      `json:"tls" yaml:"tls"`
	Authorization Authorization `json:"auth" yaml:"auth"`
}

// MiddlewareHosts is the hosts that runs Choria Brokers in host:port format
func (s *StandardConfiguration) MiddlewareHosts() []string {
	return s.Brokers
}

// Collectives is a list of collectives to join, the first will be the main collective
func (s *StandardConfiguration) Collectives() []string {
	return []string{s.Collective}
}

// LogFile is the file to log to
func (s *StandardConfiguration) LogFile() string {
	return s.LogFilePath
}

// LogLevel is the level to log at
func (s *StandardConfiguration) LogLevel() string {
	return s.Loglevel
}

// TLS is the TLS configuration
func (s *StandardConfiguration) TLS() *TLSConf {
	return s.TLSConf
}

// Auth is the authorized certificates for the backplane
func (s *StandardConfiguration) Auth() Authorization {
	return s.Authorization
}
