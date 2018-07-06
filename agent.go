package backplane

import (
	"context"

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
		Timeout:     2,
	}

	agent := mcorpc.New(md.Name, md, m.cfg.fw, m.log.WithField("agent", md.Name))

	if m.cfg.pausable != nil {
		agent.MustRegisterAction("info", m.infoAction)
		agent.MustRegisterAction("pause", m.pauseAction)
		agent.MustRegisterAction("resume", m.resumeAction)
		agent.MustRegisterAction("flip", m.flipAction)
	}

	return m.cserver.RegisterAgent(ctx, md.Name, agent)
}

func (m *Management) pauseAction(ctx context.Context, req *mcorpc.Request, reply *mcorpc.Reply, agent *mcorpc.Agent, conn choria.ConnectorInfo) {
	m.mu.Lock()
	defer m.mu.Unlock()

	m.cfg.pausable.Pause()

	m.sinfo(reply)
}

func (m *Management) resumeAction(ctx context.Context, req *mcorpc.Request, reply *mcorpc.Reply, agent *mcorpc.Agent, conn choria.ConnectorInfo) {
	m.mu.Lock()
	defer m.mu.Unlock()

	m.cfg.pausable.Resume()

	m.sinfo(reply)
}

func (m *Management) flipAction(ctx context.Context, req *mcorpc.Request, reply *mcorpc.Reply, agent *mcorpc.Agent, conn choria.ConnectorInfo) {
	m.mu.Lock()
	defer m.mu.Unlock()

	m.cfg.pausable.Flip()

	m.sinfo(reply)
}

func (m *Management) infoAction(ctx context.Context, req *mcorpc.Request, reply *mcorpc.Reply, agent *mcorpc.Agent, conn choria.ConnectorInfo) {
	m.mu.Lock()
	defer m.mu.Unlock()

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
