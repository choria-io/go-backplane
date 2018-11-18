package main

import (
	"context"
	"encoding/json"
	"io/ioutil"
	"log"
	"os"
	"os/signal"
	"strconv"
	"sync"
	"syscall"
	"time"

	backplane "github.com/choria-io/go-backplane/backplane"
	"gopkg.in/yaml.v2"
)

// Config configures myapp
type Config struct {
	Interval   int                              `yaml:"interval"`
	Name       string                           `yaml:"name"`
	LogLevel   string                           `yaml:"loglevel"`
	Management *backplane.StandardConfiguration `yaml:"management"`
}

// App is a application using the backplane to provide
// a circuit breaker, health check and shutdown backplane
type App struct {
	config     *Config
	bp         *backplane.Management
	paused     bool
	configured bool
}

func (a *App) work(ctx context.Context, wg *sync.WaitGroup) {
	defer wg.Done()

	ticker := time.NewTicker(time.Duration(a.config.Interval) * time.Second)

	for {
		select {
		case <-ticker.C:
			if a.Paused() {
				log.Printf(a.config.Name + ": skipping work while paused")
				continue
			}

			dat, err := a.data()
			if err != nil {
				log.Printf("Could not generate data: %s", err)
				continue
			}

			a.bp.DataOutbox() <- &backplane.DataItem{Data: dat, Destination: "myapp.data"}

			log.Println(a.config.Name + ": doing work - published " + string(dat))
		case <-ctx.Done():
			return
		}
	}
}

func (a *App) data() ([]byte, error) {
	d := make(map[string]string)
	d["name"] = a.config.Name
	d["timestamp"] = strconv.Itoa(int(time.Now().Unix()))
	d["work"] = "sample work"

	return json.Marshal(&d)
}

func main() {
	if _, err := os.Stat("myapp.yaml"); err != nil {
		log.Fatal("Cannot find myapp.yaml")
	}

	c, err := ioutil.ReadFile("myapp.yaml")
	if err != nil {
		log.Fatalf("Could not read myapp.yaml: %s", err)
	}

	config := &Config{}
	err = yaml.Unmarshal(c, config)
	if err != nil {
		log.Fatalf("Could not parse myapp.yaml: %s", err)
	}

	if config.Interval == 0 {
		config.Interval = 10
	}

	if config.Management == nil {
		log.Fatal("Management configuration is not provided")
	}

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	wg := &sync.WaitGroup{}

	go interruptWatcher(ctx, cancel)

	app := &App{
		config:     config,
		paused:     false,
		configured: true,
	}

	opts := []backplane.Option{
		backplane.ManageInfoSource(app),
		backplane.ManagePausable(app),
		backplane.ManageHealthCheck(app),
		backplane.ManageStopable(app),
		backplane.ManageLogLevel(app),
		backplane.StartDataPublisher(),
	}

	app.bp, err = backplane.Run(ctx, wg, app.config.Management, opts...)
	if err != nil {
		log.Fatalf("Could not start backplane: %s", err)
	}

	log.Println("Starting work")

	wg.Add(1)
	go app.work(ctx, wg)

	wg.Wait()
}

func interruptWatcher(ctx context.Context, cancel func()) {
	sigs := make(chan os.Signal, 1)
	signal.Notify(sigs, syscall.SIGINT, syscall.SIGTERM)

	select {
	case sig := <-sigs:
		log.Printf("Shutting down on %s", sig)
		cancel()
	case <-ctx.Done():
		return
	}
}
