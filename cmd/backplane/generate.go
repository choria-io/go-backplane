package main

import (
	"encoding/json"
	"fmt"
	"io/ioutil"
	"os"

	"github.com/alecthomas/template"
	backplane "github.com/choria-io/go-backplane"
	"github.com/choria-io/go-choria/mcorpc/ddl/agent"
	"github.com/choria-io/go-choria/server/agents"
	kingpin "gopkg.in/alecthomas/kingpin.v2"
)

type ddl struct {
	Name     string
	Version  string
	Pausable bool
}

func generate() {
	kingpin.FatalIfError(generateJSON(), "Could not generate JSON DDL")
	kingpin.FatalIfError(generateDDL(), "Could not generate Ruby DDL")

	fmt.Printf("Wrote %s.json and %s.ddl\n", name, name)
}

func generateDDL() error {
	out := name + ".ddl"

	if _, err := os.Stat(out); err == nil {
		return fmt.Errorf("%s already exist", out)
	}

	d := ddl{
		Name:     name,
		Pausable: pausable,
		Version:  backplane.Version,
	}

	tmpl := template.New("ddl")
	tmpl, err := tmpl.Parse(ddlTempl)
	if err != nil {
		return err
	}

	of, err := os.Create(out)
	if err != nil {
		return err
	}
	defer of.Close()

	err = tmpl.Execute(of, d)
	if err != nil {
		return err
	}

	return nil
}

func generateJSON() error {
	out := name + ".json"

	if _, err := os.Stat(out); err == nil {
		return fmt.Errorf("%s already exist", out)
	}

	ddl := agent.DDL{
		Metadata: &agents.Metadata{
			Name:        name,
			Description: "Choria Management Backplane",
			Author:      "R.I.Pienaar <rip@devco.net>",
			Version:     backplane.Version,
			License:     "Apache-2.0",
			URL:         "https://choria.io",
			Timeout:     2,
		},
		Actions: []*agent.Action{},
		Schema:  "https://choria.io/schemas/mcorpc/ddl/v1/agent.json",
	}

	if pausable {
		for _, action := range []string{"info", "pause", "resume", "flip"} {
			act := &agent.Action{
				Name:        action,
				Description: action,
				Display:     "always",
				Input:       json.RawMessage("{}"),
				Output:      make(map[string]*agent.ActionOutputItem),
			}

			act.Output["paused"] = &agent.ActionOutputItem{
				Default:     false,
				Description: "Paused State",
				DisplayAs:   "Paused",
			}

			if action == "info" {
				act.Output["version"] = &agent.ActionOutputItem{
					Default:     "",
					Description: "Application Version",
					DisplayAs:   "Version",
				}

				act.Output["facts"] = &agent.ActionOutputItem{
					Default:     json.RawMessage("{}"),
					Description: "Application Facts",
					DisplayAs:   "Facts",
				}
			}

			ddl.Actions = append(ddl.Actions, act)
		}
	}

	j, err := json.Marshal(ddl)
	if err != nil {
		return err
	}

	err = ioutil.WriteFile(out, j, 0666)
	if err != nil {
		return err
	}

	return nil
}
