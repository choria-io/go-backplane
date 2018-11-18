package backplane

import (
	"context"
	"sync"

	"github.com/choria-io/go-choria/server/data"
)

// DataItem contains a single data message
type DataItem struct {
	// Data is the raw data to publish
	Data []byte

	// Destination let you set custom NATS targets, when this is not set
	// the TargetAgent will be used to create a normal agent target
	Destination string

	// TargetAgent lets you pick where to send the data as a request
	TargetAgent string
}

// DataOutbox returns the channel to use for publishing data to the network from the backplane
func (m *Management) DataOutbox() chan *DataItem {
	return m.outbox
}

// StartRegistration implements registration.RegistrationDataProvider
func (m *Management) StartRegistration(ctx context.Context, wg *sync.WaitGroup, interval int, output chan *data.RegistrationItem) {
	for {
		select {
		case msg := <-m.outbox:
			output <- &data.RegistrationItem{
				Data:        &msg.Data,
				Destination: msg.Destination,
				TargetAgent: msg.TargetAgent,
			}
		case <-ctx.Done():
			return
		}
	}
}

func (m *Management) startDataPublisher(ctx context.Context, wg *sync.WaitGroup) error {
	return m.cserver.RegisterRegistrationProvider(ctx, wg, m)
}
