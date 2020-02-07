package backplane

import (
	"context"
	"fmt"
	"sync"

	gorpc "github.com/choria-io/go-choria/providers/agent/mcorpc/golang"
	"github.com/choria-io/go-choria/server"
)

func (m *Management) startServer(ctx context.Context, wg *sync.WaitGroup) (err error) {
	m.cserver, err = server.NewInstance(m.cfg.fw)
	if err != nil {
		return fmt.Errorf("could not initialize the backplane Choria Server: %s", err)
	}

	m.cserver.DenyAgent("rpcutil")
	m.cserver.DenyAgent("choria_util")
	m.cserver.SetComponent("backplane")

	server.RegisterAdditionalAgentProvider(&gorpc.Provider{})

	wg.Add(1)
	err = m.cserver.Run(ctx, wg)
	if err != nil {
		return
	}

	return
}
