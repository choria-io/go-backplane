package backplane

import (
	"context"
	"encoding/json"
	"fmt"
	"io/ioutil"
	"os"
	"sync"
	"time"

	"github.com/choria-io/go-choria/build"
)

// InfoSource supplies fact data
type InfoSource interface {
	// FactData should return any data you wish to expose to the fact system, it should
	// be a JSON serializable struct and it's best if its a flat one with k=v pairs
	FactData() interface{}

	// Version is any version string of your application
	Version() string
}

func (m *Management) exposeFacts(ctx context.Context, wg *sync.WaitGroup) (f string, err error) {
	tf, err := ioutil.TempFile("", m.cfg.name+"-choria-backplane-facts")
	if err != nil {
		return "", fmt.Errorf("could not create temporary facts file: %s", err)
	}
	tf.Close()

	wg.Add(1)
	go m.fsWriter(ctx, wg, m.cfg.infosource, tf.Name())

	return tf.Name(), nil
}

func (m *Management) fsWriter(ctx context.Context, wg *sync.WaitGroup, fs InfoSource, target string) {
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
		case <-time.Tick(m.cfg.factInterval):
			writer()
		case <-ctx.Done():
			return
		}
	}
}

func (m *Management) convertFacts(fs InfoSource) (out map[string]interface{}, err error) {
	in, err := json.Marshal(fs.FactData())
	if err != nil {
		return
	}

	err = json.Unmarshal(in, &out)
	if err != nil {
		return
	}

	out["backplane_version"] = build.Version
	out["backplane_name"] = m.cfg.name
	out["backplane_pausable"] = m.cfg.pausable != nil
	out["backplane_stopable"] = m.cfg.stopable != nil
	out["backplane_healthcheckable"] = m.cfg.healthcheckable != nil

	return
}

func (m *Management) write(fs InfoSource, target string) error {
	m.factsMu.Lock()
	defer m.factsMu.Unlock()

	m.log.Debugf("Starting fact source dump to %s", target)

	facts, err := m.convertFacts(fs)
	if err != nil {
		return err
	}

	j, err := json.Marshal(facts)
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
