package backplane

import (
	"context"
	"encoding/json"
	"math/rand"
	"strings"
	"time"

	"github.com/choria-io/go-choria/inter"
	"github.com/choria-io/go-choria/providers/agent/mcorpc"
	"github.com/choria-io/go-choria/providers/agent/mcorpc/ddl/agent"
	"github.com/choria-io/go-choria/providers/agent/mcorpc/ddl/common"
	"github.com/choria-io/go-choria/server/agents"
)

// Pausable is a service that can be paused
type Pausable interface {
	// Pause should pause operations within your app immediately
	Pause()

	// Resume should resume operations within your app immediately
	Resume()

	// Flip should invert the pause state in an atomic manner
	Flip()

	// Paused should report the pause state
	Paused() bool
}

// HealthCheckable describes a application that can be checked using the backplane
type HealthCheckable interface {
	// HealthCheck should return as its result a struct that can be JSON converted
	HealthCheck() (result interface{}, ok bool)
}

// Stopable describes an application that can be stopped using the backplane
type Stopable interface {
	// Shutdown will be called after some delay and should exit the application
	Shutdown()
}

// LogLevel is a valid log level
type LogLevel int8

const (
	_ = iota

	// DebugLevel is developer level debugging information
	DebugLevel LogLevel = iota

	// InfoLevel is end user informative information
	InfoLevel

	// WarnLevel is end user warning information
	WarnLevel

	// CriticalLevel is end user critical information
	CriticalLevel
)

// LogLevelSetable describes an application that can have its log levels adjusted at runtime
type LogLevelSetable interface {
	SetLogLevel(LogLevel)
	GetLogLevel() LogLevel
}

// HealthReply is the reply from the health action
type HealthReply struct {
	Result  json.RawMessage `json:"result"`
	Healthy bool            `json:"healthy"`
}

// ShutdownReply is the reply from the shutdown action
type ShutdownReply struct {
	Delay string `json:"delay"`
}

// InfoReply is the reply from the info action
type InfoReply struct {
	BackplaneVersion string      `json:"backplane_version"`
	Version          string      `json:"version"`
	Paused           bool        `json:"paused"`
	Facts            interface{} `json:"facts"`
	Healthy          bool        `json:"healthy"`
	LogLevel         string      `json:"loglevel"`
	HealthFeature    bool        `json:"healthcheck_feature"`
	PauseFeature     bool        `json:"pause_feature"`
	ShutdownFeature  bool        `json:"shutdown_feature"`
	FactsFeature     bool        `json:"facts_feature"`
	LogLevelFeature  bool        `json:"loglevel_feature"`
}

// PausableReply is the reply format expected from Pausable actions
type PausableReply struct {
	Paused bool `json:"paused"`
}

// PingReply is the reply format from the ping action
type PingReply struct {
	Version string `json:"version"`
}

// LogLevelReply is the reply format from the log level actions
type LogLevelReply struct {
	Level string `json:"level"`
}

func (m *Management) startAgents(ctx context.Context) (err error) {
	md := AgentMetadata()

	agent := mcorpc.New(md.Name, md, m.cfg.fw, m.log.WithField("agent", md.Name))

	if m.cfg.pausable != nil {
		agent.MustRegisterAction("pause", m.fullAction(m.pauseAction))
		agent.MustRegisterAction("resume", m.fullAction(m.resumeAction))
		agent.MustRegisterAction("flip", m.fullAction(m.flipAction))
	}

	if m.cfg.stopable != nil {
		agent.MustRegisterAction("shutdown", m.fullAction(m.shutdownAction))
	}

	if m.cfg.healthcheckable != nil {
		agent.MustRegisterAction("health", m.roAction(m.healthAction))
	}

	if m.cfg.logsetable != nil {
		agent.MustRegisterAction("debuglvl", m.fullAction(m.debugLevelAction))
		agent.MustRegisterAction("infolvl", m.fullAction(m.infoLevelAction))
		agent.MustRegisterAction("warnlvl", m.fullAction(m.warnLevelAction))
		agent.MustRegisterAction("critlvl", m.fullAction(m.critLevelAction))
	}

	agent.MustRegisterAction("info", m.roAction(m.infoAction))
	agent.MustRegisterAction("ping", m.roAction(m.pingAction))

	return m.cserver.RegisterAgent(ctx, md.Name, agent)
}

func (m *Management) roAction(a mcorpc.Action) mcorpc.Action {
	return func(ctx context.Context, req *mcorpc.Request, reply *mcorpc.Reply, agent *mcorpc.Agent, conn inter.ConnectorInfo) {
		if !m.cfg.auth.ROAllowed(req.CallerID) {
			reply.Statuscode = mcorpc.Aborted
			reply.Statusmsg = "You are not authorized to call this agent or action."

			return
		}

		m.mu.Lock()
		defer m.mu.Unlock()

		a(ctx, req, reply, agent, conn)
	}
}

func (m *Management) fullAction(a mcorpc.Action) mcorpc.Action {
	return func(ctx context.Context, req *mcorpc.Request, reply *mcorpc.Reply, agent *mcorpc.Agent, conn inter.ConnectorInfo) {
		if !m.cfg.auth.FullAllowed(req.CallerID) {
			reply.Statuscode = mcorpc.Aborted
			reply.Statusmsg = "You are not authorized to call this agent or action."

			return
		}

		m.mu.Lock()
		defer m.mu.Unlock()

		a(ctx, req, reply, agent, conn)
	}
}

func (m *Management) debugLevelAction(ctx context.Context, req *mcorpc.Request, reply *mcorpc.Reply, agent *mcorpc.Agent, conn inter.ConnectorInfo) {
	m.cfg.logsetable.SetLogLevel(DebugLevel)
	reply.Data = &LogLevelReply{
		Level: "debug",
	}
}

func (m *Management) infoLevelAction(ctx context.Context, req *mcorpc.Request, reply *mcorpc.Reply, agent *mcorpc.Agent, conn inter.ConnectorInfo) {
	m.cfg.logsetable.SetLogLevel(InfoLevel)
	reply.Data = &LogLevelReply{
		Level: "info",
	}
}

func (m *Management) warnLevelAction(ctx context.Context, req *mcorpc.Request, reply *mcorpc.Reply, agent *mcorpc.Agent, conn inter.ConnectorInfo) {
	m.cfg.logsetable.SetLogLevel(WarnLevel)
	reply.Data = &LogLevelReply{
		Level: "warning",
	}
}

func (m *Management) critLevelAction(ctx context.Context, req *mcorpc.Request, reply *mcorpc.Reply, agent *mcorpc.Agent, conn inter.ConnectorInfo) {
	m.cfg.logsetable.SetLogLevel(CriticalLevel)
	reply.Data = &LogLevelReply{
		Level: "critical",
	}
}

func (m *Management) pingAction(ctx context.Context, req *mcorpc.Request, reply *mcorpc.Reply, agent *mcorpc.Agent, conn inter.ConnectorInfo) {
	reply.Data = &PingReply{
		Version: agent.Metadata().Version,
	}
}

func (m *Management) healthAction(ctx context.Context, req *mcorpc.Request, reply *mcorpc.Reply, agent *mcorpc.Agent, conn inter.ConnectorInfo) {
	r, ok := m.cfg.healthcheckable.HealthCheck()

	j, err := json.Marshal(r)
	if err != nil {
		j = []byte(`{"error":"could not JSON encode result"}`)
	}

	reply.Data = &HealthReply{
		Healthy: ok,
		Result:  json.RawMessage(j),
	}
}

func (m *Management) shutdownAction(ctx context.Context, req *mcorpc.Request, reply *mcorpc.Reply, agent *mcorpc.Agent, conn inter.ConnectorInfo) {
	delay := time.Duration(rand.Intn(int(m.cfg.maxStopDelay))) + time.Second

	r := func(d time.Duration) {
		time.Sleep(d)
		agent.Log.Warnf("Shutting down after shutdown action invoked by the backplane")
		m.cfg.stopable.Shutdown()
	}

	agent.Log.Warnf("Scheduling shutdown after %s delay", delay)

	go r(delay)

	reply.Data = ShutdownReply{
		Delay: delay.String(),
	}
}

func (m *Management) pauseAction(ctx context.Context, req *mcorpc.Request, reply *mcorpc.Reply, agent *mcorpc.Agent, conn inter.ConnectorInfo) {
	m.cfg.pausable.Pause()

	m.pinfo(reply)
}

func (m *Management) resumeAction(ctx context.Context, req *mcorpc.Request, reply *mcorpc.Reply, agent *mcorpc.Agent, conn inter.ConnectorInfo) {
	m.cfg.pausable.Resume()

	m.pinfo(reply)
}

func (m *Management) flipAction(ctx context.Context, req *mcorpc.Request, reply *mcorpc.Reply, agent *mcorpc.Agent, conn inter.ConnectorInfo) {
	m.cfg.pausable.Flip()

	m.pinfo(reply)
}

func (m *Management) infoAction(ctx context.Context, req *mcorpc.Request, reply *mcorpc.Reply, agent *mcorpc.Agent, conn inter.ConnectorInfo) {
	info := &InfoReply{
		BackplaneVersion: agent.Metadata().Version,
		Version:          "unknown",
		LogLevel:         "unknown",
	}

	if m.cfg.infosource != nil {
		info.Version = m.cfg.infosource.Version()
		info.Facts = m.cfg.infosource.FactData()
		info.FactsFeature = true
	}

	if m.cfg.healthcheckable != nil {
		_, info.Healthy = m.cfg.healthcheckable.HealthCheck()
		info.HealthFeature = true
	}

	if m.cfg.pausable != nil {
		info.Paused = m.cfg.pausable.Paused()
		info.PauseFeature = true
	}

	if m.cfg.stopable != nil {
		info.ShutdownFeature = true
	}

	if m.cfg.logsetable != nil {
		switch m.cfg.logsetable.GetLogLevel() {
		case DebugLevel:
			info.LogLevel = "debug"
		case InfoLevel:
			info.LogLevel = "info"
		case WarnLevel:
			info.LogLevel = "warning"
		case CriticalLevel:
			info.LogLevel = "critical"
		default:
			info.LogLevel = "unknown"
		}
		info.LogLevelFeature = true
	}

	reply.Data = info
}

func (m *Management) pinfo(r *mcorpc.Reply) {
	r.Data = &PausableReply{
		Paused: m.cfg.pausable.Paused(),
	}
}

// AgentMetadata returns the agent metadata
func AgentMetadata() *agents.Metadata {
	return &agents.Metadata{
		Name:        "backplane",
		Description: "Choria Management Backplane",
		Author:      "R.I.Pienaar <rip@devco.net>",
		Version:     Version,
		License:     "Apache-2.0",
		URL:         "https://choria.io",
		Timeout:     10,
	}
}

// AgentDDL creates a DDL for the agent
func AgentDDL() *agent.DDL {
	ddl := &agent.DDL{
		Metadata: AgentMetadata(),
		Actions:  []*agent.Action{},
		Schema:   "https://choria.io/schemas/mcorpc/ddl/v1/agent.json",
	}

	act := &agent.Action{
		Name:        "ping",
		Description: "Backplane communications test",
		Display:     "failed",
		Input:       make(map[string]*common.InputItem),
		Output: map[string]*common.OutputItem{
			"version": {
				Description: "The version of the Choria Backplane system in use",
				DisplayAs:   "Choria Backplane",
				Type:        "string",
			},
		},
		Aggregation: []agent.ActionAggregateItem{
			{
				Function:  "summary",
				Arguments: json.RawMessage(`["version"]`),
			},
		},
	}

	ddl.Actions = append(ddl.Actions, act)

	act = &agent.Action{
		Name:        "info",
		Description: "Information about the managed service",
		Display:     "always",
		Input:       make(map[string]*common.InputItem),
		Output: map[string]*common.OutputItem{
			"backplane_version": {
				Description: "The version of the Choria Backplane system in use",
				DisplayAs:   "Choria Backplane",
				Type:        "string",
			},

			"version": {
				Description: "Service Version",
				DisplayAs:   "Version",
				Type:        "string",
			},

			"healthy": {
				Description: "Health Status",
				DisplayAs:   "Healthy",
				Type:        "boolean",
			},

			"loglevel": {
				Description: "Active log level",
				DisplayAs:   "Log Level",
				Type:        "string",
			},

			"healthcheck_feature": {
				Description: "If the HealthCheckable interface is used",
				DisplayAs:   "Health Feature",
				Type:        "boolean",
			},

			"pause_feature": {
				Description: "If the Pausable interface is used",
				DisplayAs:   "Circuit Breaker Feature",
				Type:        "boolean",
			},

			"shutdown_feature": {
				Description: "If the Stopable interface is used",
				DisplayAs:   "Shutdown Feature",
				Type:        "boolean",
			},

			"facts_feature": {
				Description: "If the InfoSource interface is used",
				DisplayAs:   "Facts Feature",
				Type:        "boolean",
			},

			"loglevel_feature": {
				Description: "If the LogLevelSetable interface is used",
				DisplayAs:   "Log Level Feature",
				Type:        "boolean",
			},
		},
		Aggregation: []agent.ActionAggregateItem{
			{
				Function:  "summary",
				Arguments: json.RawMessage(`["version"]`),
			},
			{
				Function:  "summary",
				Arguments: json.RawMessage(`["paused"]`),
			},
			{
				Function:  "summary",
				Arguments: json.RawMessage(`["healthy"]`),
			},
		},
	}

	ddl.Actions = append(ddl.Actions, act)

	act = &agent.Action{
		Name:        "shutdown",
		Description: "Terminates the managed service",
		Display:     "failed",
		Input:       make(map[string]*common.InputItem),
		Output: map[string]*common.OutputItem{
			"delay": {
				Description: "How long after running the action the shutdown will be initiated",
				DisplayAs:   "Delay",
				Type:        "string",
			},
		},
		Aggregation: []agent.ActionAggregateItem{
			{
				Function:  "summary",
				Arguments: json.RawMessage(`["delay"]`),
			},
		},
	}

	ddl.Actions = append(ddl.Actions, act)

	act = &agent.Action{
		Name:        "health",
		Description: "Checks the health of the managed service",
		Display:     "failed",
		Input:       make(map[string]*common.InputItem),
		Output: map[string]*common.OutputItem{
			"result": {
				Description: "The result from the check method",
				DisplayAs:   "Result",
				Type:        "string",
			},
			"healthy": {
				Description: "Status indicator for the checked service",
				DisplayAs:   "Healthy",
				Type:        "boolean",
			},
		},
		Aggregation: []agent.ActionAggregateItem{
			{
				Function:  "summary",
				Arguments: json.RawMessage(`["healthy"]`),
			},
		},
	}

	ddl.Actions = append(ddl.Actions, act)

	for _, action := range strings.Fields("pause resume flip") {
		act = &agent.Action{
			Name:        action,
			Description: action,
			Display:     "always",
			Input:       make(map[string]*common.InputItem),
			Output: map[string]*common.OutputItem{
				"paused": {
					Description: "Circuit Breaker pause state",
					DisplayAs:   "Paused",
					Type:        "boolean",
				},
			},
			Aggregation: []agent.ActionAggregateItem{
				{
					Function:  "summary",
					Arguments: json.RawMessage(`["paused"]`),
				},
			},
		}

		ddl.Actions = append(ddl.Actions, act)
	}

	for _, action := range strings.Fields("debuglvl infolvl warnlvl critlvl") {
		act = &agent.Action{
			Name:        action,
			Description: action,
			Display:     "always",
			Input:       make(map[string]*common.InputItem),
			Output: map[string]*common.OutputItem{
				"level": {
					Description: "Log level that was activated",
					DisplayAs:   "Log Level",
					Type:        "string",
				},
			},
			Aggregation: []agent.ActionAggregateItem{
				{
					Function:  "summary",
					Arguments: json.RawMessage(`["level"]`),
				},
			},
		}

		ddl.Actions = append(ddl.Actions, act)
	}

	return ddl
}
