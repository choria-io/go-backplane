package cmd

import (
	"context"
	"encoding/json"
	"fmt"
	"os"
	"os/signal"
	"strconv"
	"strings"
	"sync"
	"syscall"
	"text/tabwriter"
	"time"

	"github.com/hokaccha/go-prettyjson"

	"github.com/choria-io/go-backplane/backplane"
	"github.com/choria-io/mcorpc-agent-provider/mcorpc"
	"github.com/fatih/color"

	"github.com/sirupsen/logrus"

	"github.com/choria-io/go-choria/choria"
	"github.com/choria-io/go-choria/config"
	"github.com/choria-io/go-client/client"
	"github.com/choria-io/go-client/discovery/broadcast"
	"github.com/choria-io/go-protocol/protocol"
	rpcc "github.com/choria-io/mcorpc-agent-provider/mcorpc/client"

	kingpin "gopkg.in/alecthomas/kingpin.v2"
)

var (
	action  string
	service string

	wf       []string
	wi       []string
	timeout  int
	cfile    string
	insecure bool

	fw      *choria.Framework
	err     error
	rpc     *rpcc.RPC
	ctx     context.Context
	cancel  func()
	log     *logrus.Entry
	mu      = &sync.Mutex{}
	debug   bool
	verbose bool
)

// Run runs the backplane command line
func Run() {
	app := kingpin.New("backplane", "Choria Backplane")
	app.Version(backplane.Version)
	app.Author("R.I.Pienaar <rip@devco.net>")

	app.Flag("verbose", "Enable verbose output").Short('v').BoolVar(&verbose)
	app.Flag("debug", "Enable debug logging").Short('d').BoolVar(&debug)

	e := app.Command("exec", "Executes a action against a set of backplane managed services").Default()
	e.Arg("service", "The services name to manage").Required().StringVar(&service)
	e.Arg("action", "Action to perform against the managed service").Required().EnumVar(&action, "pause", "resume", "flip", "health", "shutdown", "ping", "info")

	e.Flag("wf", "Match services with a certain fact").Short('F').PlaceHolder("FACTS").StringsVar(&wf)
	e.Flag("wi", "Match services with a certain Choria identity").Short('I').PlaceHolder("IDENTITY").StringsVar(&wi)
	e.Flag("timeout", "How long to wait for services to respond").IntVar(&timeout)
	e.Flag("config", "Configuration file to use").StringVar(&cfile)
	e.Flag("insecure", "Disable TLS security").BoolVar(&insecure)

	cmd := kingpin.MustParse(app.Parse(os.Args[1:]))

	ctx, cancel = context.WithCancel(context.Background())
	defer cancel()

	go interruptWatcher()

	switch cmd {
	case e.FullCommand():
		err := execute()
		kingpin.FatalIfError(err, "Failed to manage services")
	default:
		kingpin.Fatalf("%s has not been implemented", cmd)
	}
}

func execute() error {
	fw, err = configure()
	if err != nil {
		return err
	}

	log = fw.Logger("backplane")

	rpc, err = rpcc.New(fw, "backplane", rpcc.DDL(backplane.AgentDDL()))
	if err != nil {
		return fmt.Errorf("could not create agent: %s", err)
	}

	switch action {
	case "pause", "resume", "flip":
		wf = append(wf, "backplane_pausable=true")
		err = genericRequest(action, false)
	case "shutdown":
		wf = append(wf, "backplane_stopable=true")
		err = genericRequest(action, false)
	case "health":
		err = healthRequest()
	case "info":
		err = infoRequest()
	case "ping":
		err = genericRequest(action, true)
	}

	if err != nil {
		return fmt.Errorf("could not perform action: %s", err)
	}

	return nil
}

func infoRequest() error {
	err := performAction(action, func(svcs string, reply *rpcc.RPCReply, last bool) {
		if reply.Statuscode == mcorpc.OK {
			info := &backplane.InfoReply{}
			err := json.Unmarshal(reply.Data, info)
			if err != nil {
				log.Errorf("Could not decode reply from %s: %s", svcs, err)
			}

			fmt.Printf("  %s:\n\n", svcs)
			fmt.Printf("           App Version: %s\n", info.Version)
			fmt.Printf("     Backplane Version: %s\n", info.BackplaneVersion)
			if info.PauseFeature {
				fmt.Printf("                Paused: %v\n", info.Paused)
			}
			if info.HealthFeature {
				fmt.Printf("               Healthy: %v\n", info.Healthy)
			}
			fmt.Printf("         Pause Feature: %s\n", boolTick(info.PauseFeature))
			fmt.Printf("         Facts Feature: %s\n", boolTick(info.FactsFeature))
			fmt.Printf("        Health Feature: %s\n", boolTick(info.HealthFeature))
			fmt.Printf("      Shutdown Feature: %s\n", boolTick(info.ShutdownFeature))

			if verbose {
				formatter := prettyjson.NewFormatter()
				formatter.DisabledColor = true
				p, err := formatter.Marshal(info.Facts)
				if err == nil {
					fmt.Println("                 Facts:")
					fmt.Println(indentString(string(p), "                        "))
				}
			}

			fmt.Println()
		} else {
			fmt.Printf("%40s:               %s\n", svcs, color.RedString(reply.Statusmsg))
		}
	})

	if !verbose {
		fmt.Println("\nPass -v to show contents of individual instance facts")
	}

	return err
}

func genericRequest(action string, showAll bool) error {
	tw := tabwriter.NewWriter(os.Stdout, 0, 0, 2, ' ', tabwriter.AlignRight)

	err := performAction(action, func(svcs string, reply *rpcc.RPCReply, last bool) {
		if reply.Statuscode == mcorpc.OK {
			if showAll || verbose {
				fmt.Fprintf(tw, "%s\t\t%s\n", svcs, string(reply.Data))
			}
		} else {
			fmt.Fprintf(tw, "%s\t\t%s\n", svcs, reply.Statusmsg)
		}

		if last {
			tw.Flush()
		}
	})

	return err
}

func healthRequest() error {
	wf = append(wf, "backplane_healthcheckable=true")

	tw := tabwriter.NewWriter(os.Stdout, 0, 0, 2, ' ', tabwriter.AlignRight)

	err := performAction(action, func(svcs string, reply *rpcc.RPCReply, last bool) {
		if reply.Statuscode == mcorpc.OK {
			info := &backplane.HealthReply{}
			err := json.Unmarshal(reply.Data, info)
			if err != nil {
				log.Errorf("Could not decode reply from %s: %s", svcs, err)
			}

			if info.Healthy {
				if verbose {
					fmt.Fprintf(tw, "%s\t\t%s\n", svcs, color.GreenString("Healthy"))
					fmt.Fprintf(tw, "\t\t%s\n", string(info.Result))
				}
			} else {
				fmt.Fprintf(tw, "%s\t\t%s\n", svcs, color.RedString("Unhealthy"))
				fmt.Fprintf(tw, "\t\t%s\n", string(info.Result))
			}
		} else {
			fmt.Fprintf(tw, svcs+"\t\t"+reply.Statusmsg)
		}

		if last {
			tw.Flush()
		}
	})

	return err
}

func performAction(action string, cb func(s string, r *rpcc.RPCReply, last bool)) error {
	nodes, err := discover()
	if err != nil {
		return err
	}

	if len(nodes) == 0 {
		return fmt.Errorf("did not discover any nodes")
	}

	replies, stats, err := request(action, json.RawMessage("{}"), nodes)
	if err != nil {
		return fmt.Errorf("request failed: %s", err)
	}

	if len(replies) == 0 {
		return fmt.Errorf("no responses received")
	}

	count := len(replies)
	seen := 0

	for svcs, reply := range replies {
		seen++

		cb(svcs, reply, seen == count)
	}

	d, _ := stats.RequestDuration()
	fmt.Printf("\nManaged %d service(s) in %dms\n", stats.OKCount(), int64(d/time.Millisecond))

	return nil
}

func configure() (*choria.Framework, error) {
	if cfile == "" {
		cfile = choria.UserConfig()
	}

	cfg, err := config.NewConfig(cfile)
	if err != nil {
		return nil, fmt.Errorf("could not initialize choria: %s", err)
	}

	cfg.Collectives = []string{fmt.Sprintf("%s_backplane", service)}
	cfg.MainCollective = cfg.Collectives[0]

	if debug {
		cfg.LogLevel = "debug"
	}

	if insecure {
		cfg.DisableTLS = true
		cfg.Choria.SecurityProvider = "file"
		protocol.Secure = "false"
	}

	fw, err := choria.NewWithConfig(cfg)
	if err != nil {
		return nil, fmt.Errorf("could not initialize choria: %s", err)
	}

	return fw, nil
}

func discover() (n []string, err error) {
	filter, err := client.NewFilter(
		client.FactFilter(wf...),
		client.IdentityFilter(wi...),
	)
	if err != nil {
		return n, fmt.Errorf("could not parse filters: %s", err)
	}

	fmt.Printf("Starting discovery process for %s backplane managed services: ", service)

	b := broadcast.New(fw)

	n, err = b.Discover(ctx, broadcast.Filter(filter))
	if err != nil {
		fmt.Println("")
		return n, fmt.Errorf("could not perform discovery: %s", err)
	}

	fmt.Printf("%d\n\n", len(n))

	return
}

func interruptWatcher() {
	sigs := make(chan os.Signal, 1)
	signal.Notify(sigs, syscall.SIGINT, syscall.SIGTERM)

	for {
		select {
		case sig := <-sigs:
			switch sig {
			case syscall.SIGINT, syscall.SIGTERM:
				log.Infof("Shutting down on %s", sig)
				cancel()
			}
		case <-ctx.Done():
			return
		}
	}
}

func twirl(msg string, max int, current int) string {
	charset := []string{"▖", "▘", "▝", "▗"}
	index := current % len(charset)
	char := charset[index]

	if current == max {
		char = color.GreenString("✓")
	}

	format := fmt.Sprintf("\r%%s %%s %%%dd / %%d", len(strconv.Itoa(max)))

	return fmt.Sprintf(format, msg, char, current, max)
}

func request(action string, input json.RawMessage, nodes []string) (replies map[string]*rpcc.RPCReply, stats *rpcc.Stats, err error) {
	replies = make(map[string]*rpcc.RPCReply)
	cnt := 0

	result, err := rpc.Do(ctx, action, input, rpcc.Targets(nodes), rpcc.ReplyHandler(func(r protocol.Reply, rep *rpcc.RPCReply) {
		mu.Lock()
		defer mu.Unlock()

		cnt++

		replies[r.SenderID()] = rep

		fmt.Print(twirl(fmt.Sprintf("Performing %s...", action), len(nodes), cnt))
	}))

	if err != nil {
		return
	}

	fmt.Printf("\n\n")

	stats = result.Stats()

	return
}

func boolTick(b bool) string {
	if b {
		return "✓"
	}

	return "✘"
}

func indentString(text string, indent string) string {
	if text[len(text)-1:] == "\n" {
		result := ""
		for _, j := range strings.Split(text[:len(text)-1], "\n") {
			result += indent + j + "\n"
		}
		return result
	}
	result := ""
	for _, j := range strings.Split(strings.TrimRight(text, "\n"), "\n") {
		result += indent + j + "\n"
	}
	return result[:len(result)-1]
}
