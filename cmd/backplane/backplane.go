package main

import (
	"os"

	"github.com/choria-io/go-backplane"
	kingpin "gopkg.in/alecthomas/kingpin.v2"
)

var (
	pausable   bool
	healthable bool
	stopable   bool
	name       string
)

func main() {
	app := kingpin.New("backplane", "Choria Backplane")
	app.Version(backplane.Version)
	app.Author("R.I.Pienaar <rip@devco.net>")

	g := app.Command("generate", "Generates DDL files for the backplane generated agents")
	g.Flag("pause", "Generate actions for the Pausable interface").BoolVar(&pausable)
	g.Flag("stop", "Generate actions for the Stopable interface").BoolVar(&stopable)
	g.Flag("health", "Generate actions for the HealthCheckable interface").BoolVar(&healthable)

	cmd := kingpin.MustParse(app.Parse(os.Args[1:]))

	switch cmd {
	case g.FullCommand():
		generate()
	default:
		kingpin.Fatalf("%s has not been implemented", cmd)
	}
}
