package backplane

import (
	"context"
	"encoding/json"
	"math/rand"
	"time"

	"github.com/choria-io/go-choria/choria"
	"github.com/choria-io/go-choria/mcorpc"
	"github.com/choria-io/go-choria/server/agents"
)

// Pausable is a processor that can be paused
type Pausable interface {
	Pause()
	Resume()
	Flip()
	Paused() bool
	Version() string
}

// HealthCheckable describes a application that can be checked using the backplane
type HealthCheckable interface {
	HealthCheck() (result interface{}, ok bool)
}

// Stopable describes an application that can be stopped using the backplane
type Stopable interface {
	Shutdown()
}

type healthReply struct {
	Result  json.RawMessage `json:"result"`
	Healthy bool            `json:"healthy"`
}

type stopReply struct {
	Delay string `json:"delay"`
}

type infoReply struct {
	Version string      `json:"version"`
	Paused  bool        `json:"paused"`
	Facts   interface{} `json:"facts"`
}

type simpleReply struct {
	Paused bool `json:"paused"`
}

func (m *Management) startAgents(ctx context.Context) (err error) {
	md := &agents.Metadata{
		Name:        m.cfg.name,
		Description: "Choria Management Backplane",
		Author:      "R.I.Pienaar <rip@devco.net>",
		Version:     Version,
		License:     "Apache-2.0",
		URL:         "https://choria.io",
		Timeout:     10,
	}

	agent := mcorpc.New(md.Name, md, m.cfg.fw, m.log.WithField("agent", md.Name))

	if m.cfg.pausable != nil {
		agent.MustRegisterAction("info", m.roAction(m.infoAction))
		agent.MustRegisterAction("pause", m.fullAction(m.pauseAction))
		agent.MustRegisterAction("resume", m.fullAction(m.resumeAction))
		agent.MustRegisterAction("flip", m.fullAction(m.flipAction))
	}

	if m.cfg.stopable != nil {
		agent.MustRegisterAction("stop", m.fullAction(m.stopAction))
	}

	if m.cfg.healthcheckable != nil {
		agent.MustRegisterAction("health", m.roAction(m.healthAction))
	}

	return m.cserver.RegisterAgent(ctx, md.Name, agent)
}

func (m *Management) roAction(a mcorpc.Action) mcorpc.Action {
	return func(ctx context.Context, req *mcorpc.Request, reply *mcorpc.Reply, agent *mcorpc.Agent, conn choria.ConnectorInfo) {
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
	return func(ctx context.Context, req *mcorpc.Request, reply *mcorpc.Reply, agent *mcorpc.Agent, conn choria.ConnectorInfo) {
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

func (m *Management) healthAction(ctx context.Context, req *mcorpc.Request, reply *mcorpc.Reply, agent *mcorpc.Agent, conn choria.ConnectorInfo) {
	r, ok := m.cfg.healthcheckable.HealthCheck()
	if !ok {
		reply.Statuscode = mcorpc.Aborted
		reply.Statusmsg = "Service is not healthy"
	}

	j, err := json.Marshal(r)
	if err != nil {
		j = []byte(`{"error":"could not JSON encode result"}`)
	}

	reply.Data = &healthReply{
		Healthy: ok,
		Result:  json.RawMessage(j),
	}
}

func (m *Management) stopAction(ctx context.Context, req *mcorpc.Request, reply *mcorpc.Reply, agent *mcorpc.Agent, conn choria.ConnectorInfo) {
	delay := time.Duration(rand.Intn(int(m.cfg.maxStopDelay)))*time.Second + time.Second

	r := func(d time.Duration) {
		time.Sleep(d)
		agent.Log.Warnf("Shutting down after stop action invoked by the backplane")
		m.cfg.stopable.Shutdown()
	}

	agent.Log.Warnf("Scheduling shutdown after %s delay", delay)

	go r(delay)

	reply.Data = stopReply{
		Delay: delay.String(),
	}
}

func (m *Management) pauseAction(ctx context.Context, req *mcorpc.Request, reply *mcorpc.Reply, agent *mcorpc.Agent, conn choria.ConnectorInfo) {
	m.cfg.pausable.Pause()

	m.sinfo(reply)
}

func (m *Management) resumeAction(ctx context.Context, req *mcorpc.Request, reply *mcorpc.Reply, agent *mcorpc.Agent, conn choria.ConnectorInfo) {
	m.cfg.pausable.Resume()

	m.sinfo(reply)
}

func (m *Management) flipAction(ctx context.Context, req *mcorpc.Request, reply *mcorpc.Reply, agent *mcorpc.Agent, conn choria.ConnectorInfo) {
	m.cfg.pausable.Flip()

	m.sinfo(reply)
}

func (m *Management) infoAction(ctx context.Context, req *mcorpc.Request, reply *mcorpc.Reply, agent *mcorpc.Agent, conn choria.ConnectorInfo) {
	m.info(reply)
}

func (m *Management) sinfo(r *mcorpc.Reply) {
	r.Data = &simpleReply{
		Paused: m.cfg.pausable.Paused(),
	}
}

func (m *Management) info(r *mcorpc.Reply) {
	r.Data = &infoReply{
		Paused:  m.cfg.pausable.Paused(),
		Version: m.cfg.pausable.Version(),
		Facts:   m.cfg.factsource.FactData(),
	}
}
