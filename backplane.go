// Package backplane allows you to create a Choria based management backplane for your application
//
// Your application will live on the Choria network where it can be discovered and managed remotely
// on a very large scale with built in authentication, auditing and authorization.
//
// You will be able to interact with your application from the Choria CLI, Ruby API or Go API and
// perform some or all of the below
//
//  * Circuit Breaker that can pause and resume your application
//  * Healthchecks to query the internal health of your application
//  * Stop the application
//
// Additionally data about your application like it's configuration will be exposed to the Choria
// discovery subsystem
package backplane

import (
	"context"
	"fmt"
	"sync"

	"github.com/choria-io/go-choria/mcorpc"
	"github.com/choria-io/go-choria/server"
	"github.com/sirupsen/logrus"
)

// Version is the version of the management backplane
const Version = "0.0.1"

// Management is a embeddable Choria based backplane for your Go application
type Management struct {
	cfg     *Config
	cserver *server.Instance
	mu      *sync.Mutex
	factsMu *sync.Mutex
	log     *logrus.Entry
	agent   *mcorpc.Agent
}

// Run creates a new instance of the backplane
func Run(ctx context.Context, wg *sync.WaitGroup, name string, conf ConfigProvider, opts ...Option) (m *Management, err error) {
	m = &Management{
		mu: &sync.Mutex{},
	}

	m.cfg, err = newConfig(name, conf, opts...)
	if err != nil {
		return nil, fmt.Errorf("could not initialize Choria backplane: %s", err)
	}

	m.log = m.cfg.fw.Logger("backplane")

	if m.cfg.factsource != nil {
		m.exposeFacts(ctx, wg)
	}

	err = m.startServer(ctx, wg)
	if err != nil {
		return nil, fmt.Errorf("could not start Choria server: %s", err)
	}

	err = m.startAgents(ctx)
	if err != nil {
		return nil, fmt.Errorf("could not start backplane agents: %s", err)
	}

	return
}
