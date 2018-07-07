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
	Name    string
	Version string
	Pause   bool
	Health  bool
	Stop    bool
}

func generate() {
	if !(healthable || stopable || pausable) {
		kingpin.Fatalf("Please specify what interfaces to generate, see --help")
	}

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
		Name:    name,
		Version: backplane.Version,
		Pause:   pausable,
		Health:  healthable,
		Stop:    stopable,
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
			Timeout:     10,
		},
		Actions: []*agent.Action{},
		Schema:  "https://choria.io/schemas/mcorpc/ddl/v1/agent.json",
	}

	act := &agent.Action{
		Name:        "ping",
		Description: "Backplane communications test",
		Display:     "failed",
		Input:       json.RawMessage("{}"),
		Output:      make(map[string]*agent.ActionOutputItem),
	}

	act.Output["version"] = &agent.ActionOutputItem{
		Default:     "0.0.0",
		Description: "The version of the Choria Backplane system in use",
		DisplayAs:   "Choria Backplane",
	}

	ddl.Actions = append(ddl.Actions, act)

	if stopable {
		act = &agent.Action{
			Name:        "stop",
			Description: "Stops the managed service",
			Display:     "failed",
			Input:       json.RawMessage("{}"),
			Output:      make(map[string]*agent.ActionOutputItem),
		}

		act.Output["delay"] = &agent.ActionOutputItem{
			Default:     "",
			Description: "How long after running the action the shutdown will be initiated",
			DisplayAs:   "Delay",
		}

		ddl.Actions = append(ddl.Actions, act)
	}

	if healthable {
		act = &agent.Action{
			Name:        "health",
			Description: "Checks the health of the managed service",
			Display:     "failed",
			Input:       json.RawMessage("{}"),
			Output:      make(map[string]*agent.ActionOutputItem),
		}

		act.Output["result"] = &agent.ActionOutputItem{
			Default:     "",
			Description: "The result from the check method",
			DisplayAs:   "Result",
		}

		act.Output["healthy"] = &agent.ActionOutputItem{
			Default:     false,
			Description: "Status indicator for the checked service",
			DisplayAs:   "Healthy",
		}

		ddl.Actions = append(ddl.Actions, act)
	}

	if pausable {
		for _, action := range []string{"info", "pause", "resume", "flip"} {
			act = &agent.Action{
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

	j, err := json.MarshalIndent(ddl, "", "   ")
	if err != nil {
		return err
	}

	err = ioutil.WriteFile(out, j, 0666)
	if err != nil {
		return err
	}

	return nil
}
