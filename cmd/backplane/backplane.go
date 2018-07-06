package main

import (
	"os"

	"github.com/choria-io/go-backplane"
	kingpin "gopkg.in/alecthomas/kingpin.v2"
)

var (
	pausable bool
	name     string
)

func main() {
	app := kingpin.New("backplane", "Choria Backplane")
	app.Version(backplane.Version)
	app.Author("R.I.Pienaar <rip@devco.net>")

	g := app.Command("generate", "Generates DDL files for the backplane generated agents")
	g.Flag("name", "Agent name to generate").Required().StringVar(&name)
	g.Flag("pausable", "Generate actions for the Pausable interface").Default("true").BoolVar(&pausable)

	cmd := kingpin.MustParse(app.Parse(os.Args[1:]))

	switch cmd {
	case g.FullCommand():
		generate()
	default:
		kingpin.Fatalf("%s has not been implemented", cmd)
	}
}
