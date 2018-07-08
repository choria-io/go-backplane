package main

import (
	"context"
	"io/ioutil"
	"log"
	"os"
	"os/signal"
	"sync"
	"syscall"
	"time"

	backplane "github.com/choria-io/go-backplane"
	"gopkg.in/yaml.v2"
)

// Config configures myapp
type Config struct {
	Interval   int                              `yaml:"interval"`
	Management *backplane.StandardConfiguration `yaml:"management"`
}

// App is a application using the backplane to provide
// a circuit breaker, health check and shutdown backplane
type App struct {
	config     *Config
	paused     bool
	configured bool // setting this is not shown
}

type health struct {
	Configured bool
}

func (a *App) work(ctx context.Context, wg *sync.WaitGroup) {
	defer wg.Done()

	ticker := time.NewTicker(time.Duration(a.config.Interval) * time.Millisecond)

	for {
		select {
		case <-ticker.C:
			if a.Paused() {
				log.Printf("Skipping work while paused")
				continue
			}

			log.Println("doing work")
		case <-ctx.Done():
			return
		}
	}
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
	}

	_, err = backplane.Run(ctx, wg, "app", app.config.Management, opts...)
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
