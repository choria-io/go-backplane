package backplane

import (
	"context"
	"encoding/json"
	"fmt"
	"io/ioutil"
	"os"
	"sync"
	"time"
)

// FactSource supplies fact data
type FactSource interface {
	FactData() interface{}
}

func (m *Management) exposeFacts(ctx context.Context, wg *sync.WaitGroup) (f string, err error) {
	tf, err := ioutil.TempFile("", m.cfg.name+"-choria-backplane-facts")
	if err != nil {
		return "", fmt.Errorf("could not create temporary facts file: %s", err)
	}
	tf.Close()

	wg.Add(1)
	go m.fsWriter(ctx, wg, m.cfg.factsource, tf.Name())

	return tf.Name(), nil
}

func (m *Management) fsWriter(ctx context.Context, wg *sync.WaitGroup, fs FactSource, target string) {
	defer wg.Done()
	defer os.Remove(target)

	writer := func() {
		err := m.write(fs, target)
		if err != nil {
			m.log.Errorf("Could not write fact data to %s: %s", target, err)
		}
	}

	m.log.Infof("Writing management interface fact data to %s", target)

	m.factsMu = &sync.Mutex{}

	writer()

	for {
		select {
		case <-time.Tick(m.cfg.factinterval):
			writer()
		case <-ctx.Done():
			return
		}
	}
}

func (m *Management) write(fs FactSource, target string) error {
	m.factsMu.Lock()
	defer m.factsMu.Unlock()

	m.log.Debugf("Starting fact source dump to %s", target)

	j, err := json.Marshal(fs.FactData())
	if err != nil {
		return err
	}

	tf, err := ioutil.TempFile("", "")
	if err != nil {
		return err
	}
	defer os.Remove(tf.Name())

	tf.Close()

	err = ioutil.WriteFile(tf.Name(), j, 0644)
	if err != nil {
		return err
	}

	err = os.Rename(tf.Name(), target)
	if err != nil {
		return err
	}

	m.log.Debugf("Completed %d bytes fact source dump to %s", len(j), target)

	return nil
}
