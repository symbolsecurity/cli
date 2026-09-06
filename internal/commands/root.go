package commands

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"os"
	"os/signal"
	"strings"
	"syscall"

	"github.com/spf13/cobra"
	"github.com/spf13/pflag"

	"github.com/symbolsecurity/cli/internal/auth"
	"github.com/symbolsecurity/cli/internal/client"
	"github.com/symbolsecurity/cli/internal/config"
	"github.com/symbolsecurity/cli/internal/ident"
	"github.com/symbolsecurity/cli/internal/output"
	"github.com/symbolsecurity/cli/internal/version"
)

type Env struct {
	Args       []string
	Home       string
	Cwd        string
	Stdin      io.Reader
	Stdout     io.Writer
	Stderr     io.Writer
	HTTPClient *http.Client
	Keyring    auth.Keyring
	Getenv     func(string) string
	IsTerminal func(io.Writer) bool
}

type Runtime struct {
	Env     Env
	Config  *config.Config
	Store   *auth.Store
	Client  *client.Client
	Out     *output.Printer
	JSON    bool
	Agent   bool
	MD      bool
	JQ      string
	Company string
	Profile string
	Yes     bool
	Full    bool
	Page    int
	PerPage int
	All     bool
	Limit   int
	Verbose bool
	DryRun  bool
}

func Execute(args []string) error {
	home, err := os.UserHomeDir()
	if err != nil {
		return err
	}
	cwd, _ := os.Getwd()
	return ExecuteEnv(Env{
		Args:       args,
		Home:       home,
		Cwd:        cwd,
		Stdin:      os.Stdin,
		Stdout:     os.Stdout,
		Stderr:     os.Stderr,
		Keyring:    auth.OSKeyring{},
		Getenv:     os.Getenv,
		IsTerminal: output.IsTerminal,
	})
}

func ExecuteEnv(env Env) error {
	if env.Getenv == nil {
		env.Getenv = os.Getenv
	}
	if env.Stdin == nil {
		env.Stdin = os.Stdin
	}
	if env.Stdout == nil {
		env.Stdout = os.Stdout
	}
	if env.Stderr == nil {
		env.Stderr = os.Stderr
	}
	if env.IsTerminal == nil {
		env.IsTerminal = output.IsTerminal
	}
	rt := &Runtime{Env: env}
	cmd := rt.root()
	cmd.SetArgs(env.Args)
	cmd.SetIn(env.Stdin)
	cmd.SetOut(env.Stdout)
	cmd.SetErr(env.Stderr)
	ctx, stop := signal.NotifyContext(context.Background(), os.Interrupt, syscall.SIGTERM)
	defer stop()
	err := cmd.ExecuteContext(ctx)
	if err != nil {
		if _, ok := err.(*output.Error); ok {
			return err
		}
		if rt.Out != nil {
			return rt.Out.Fail(err)
		}
		p := output.NewPrinter(env.Stdout, env.Stderr, output.Options{
			JSON:       true,
			ForceJSON:  !env.IsTerminal(env.Stdout),
			IsTerminal: env.IsTerminal(env.Stdout),
		})
		return p.Fail(err)
	}
	return nil
}

func (rt *Runtime) root() *cobra.Command {
	cmd := &cobra.Command{
		Use:           "symbol",
		Short:         "Symbol Security CLI",
		SilenceUsage:  true,
		SilenceErrors: true,
		PersistentPreRunE: func(cmd *cobra.Command, args []string) error {
			return rt.setup(cmd)
		},
	}
	cmd.PersistentFlags().BoolVar(&rt.JSON, "json", false, "JSON envelope output")
	cmd.PersistentFlags().BoolVar(&rt.Agent, "agent", false, "Raw data on success; enveloped errors")
	cmd.PersistentFlags().BoolVar(&rt.MD, "md", false, "Human tables")
	cmd.PersistentFlags().StringVar(&rt.JQ, "jq", "", "jq expression applied to the payload")
	cmd.PersistentFlags().StringVar(&rt.Company, "company", "", "MSP child company id")
	cmd.PersistentFlags().StringVar(&rt.Profile, "profile", "", "Named identity")
	cmd.PersistentFlags().BoolVar(&rt.Yes, "yes", false, "Confirm destructive actions")
	cmd.PersistentFlags().BoolVar(&rt.Full, "full", false, "Disable PII redaction")
	cmd.PersistentFlags().IntVar(&rt.Page, "page", 0, "Page number")
	cmd.PersistentFlags().IntVar(&rt.PerPage, "per-page", 50, "Items per page")
	cmd.PersistentFlags().BoolVar(&rt.All, "all", false, "Fetch all pages")
	cmd.PersistentFlags().IntVar(&rt.Limit, "limit", 0, "Stop after N items")
	cmd.PersistentFlags().BoolVar(&rt.Verbose, "verbose", false, "Log HTTP method, path, and status to stderr")
	cmd.PersistentFlags().BoolVar(&rt.DryRun, "dry-run", false, "Print mutating requests without sending them")

	cmd.AddCommand(
		rt.authCmd(),
		rt.versionCmd(),
		rt.doctorCmd(),
		rt.usersCmd(),
		rt.trainingCmd(),
		rt.policiesCmd(),
		rt.threatsCmd(),
		rt.assessmentsCmd(),
		rt.phishingCmd(),
		rt.simulationsCmd(),
		rt.domainsCmd(),
		rt.emailThreatsCmd(),
		rt.webhooksCmd(),
		rt.invoicesCmd(),
		rt.ticketsCmd(),
		rt.companiesCmd(),
		rt.programsCmd(),
		rt.mspCmd(),
		rt.setupCmd(),
		rt.commandsCmd(),
		rt.completionCmd(),
	)

	orig := cmd.HelpFunc()
	cmd.SetHelpFunc(func(c *cobra.Command, args []string) {
		agent, _ := c.Flags().GetBool("agent")
		js, _ := c.Flags().GetBool("json")
		if agent || js {
			_ = writeHelpJSON(c, rt.Env.Stdout)
			return
		}
		orig(c, args)
	})
	return cmd
}

func (rt *Runtime) setup(cmd *cobra.Command) error {
	cfg, err := config.Load(rt.Env.Home, rt.Env.Cwd, rt.Env.Getenv)
	if err != nil {
		return err
	}
	if rt.Profile != "" {
		cfg.Profile = rt.Profile
	} else {
		rt.Profile = cfg.Profile
	}
	if rt.Company != "" {
		cfg.Company = rt.Company
	} else {
		rt.Company = cfg.Company
	}
	term := rt.Env.IsTerminal(rt.Env.Stdout)
	rt.Out = output.NewPrinter(rt.Env.Stdout, rt.Env.Stderr, output.Options{
		JSON:       rt.JSON,
		Agent:      rt.Agent,
		MD:         rt.MD,
		JQ:         rt.JQ,
		Full:       rt.Full,
		IsTerminal: term,
	})
	if err := ident.Profile(cfg.Profile); err != nil {
		return rt.Out.Fail(err)
	}
	if cfg.Company != "" {
		if err := ident.UUID(cfg.Company, "company id"); err != nil {
			return rt.Out.Fail(err)
		}
	}
	rt.Config = cfg
	rt.Store = auth.NewStore(rt.Env.Home, cfg.Profile, rt.Env.Keyring)
	httpClient := rt.Env.HTTPClient
	if httpClient == nil {
		httpClient = client.DefaultHTTP()
	}
	rt.Client = client.New(cfg.BaseURL, rt.Store, cfg.Company, httpClient)
	rt.Client.DryRun = rt.DryRun
	if rt.Verbose {
		rt.Client.Log = rt.Env.Stderr
	}
	if rt.Full && !rt.Yes {
		return rt.Out.Fail(output.Usage("--full requires --yes", "Pass --yes to disable PII redaction"))
	}
	if rt.Full {
		fmt.Fprintln(rt.Env.Stderr, "warning: PII redaction disabled (--full)")
	}
	if u, err := url.Parse(cfg.BaseURL); err == nil && u.Scheme != "https" && !isLoopback(u.Hostname()) {
		fmt.Fprintf(rt.Env.Stderr, "warning: SYMBOL_BASE_URL is not HTTPS (%s)\n", cfg.BaseURL)
	}
	return nil
}

func isLoopback(host string) bool {
	switch strings.ToLower(host) {
	case "localhost", "127.0.0.1", "::1":
		return true
	default:
		return false
	}
}

func (rt *Runtime) ctx(cmd *cobra.Command) context.Context {
	if cmd != nil && cmd.Context() != nil {
		return cmd.Context()
	}
	return context.Background()
}

func (rt *Runtime) page() client.PageOpts {
	return client.PageOpts{Page: rt.Page, PerPage: rt.PerPage, All: rt.All, Limit: rt.Limit}
}

func (rt *Runtime) requireYes(action string) error {
	if rt.Yes {
		return nil
	}
	return output.Usage("destructive action requires --yes", "Pass --yes to "+action)
}

func (rt *Runtime) nonInteractive() bool {
	v := rt.Env.Getenv("SYMBOL_NONINTERACTIVE")
	return v == "1" || strings.EqualFold(v, "true")
}

func (rt *Runtime) versionCmd() *cobra.Command {
	return &cobra.Command{
		Use:   "version",
		Short: "Print CLI version",
		RunE: func(cmd *cobra.Command, args []string) error {
			data := map[string]string{"version": version.Version, "commit": version.Commit}
			return rt.Out.Success(data, "symbol "+version.Version, nil, nil)
		},
	}
}

func writeHelpJSON(cmd *cobra.Command, w io.Writer) error {
	type flag struct {
		Name       string `json:"name"`
		Shorthand  string `json:"shorthand,omitempty"`
		Type       string `json:"type"`
		Default    string `json:"default,omitempty"`
		Usage      string `json:"usage"`
		Persistent bool   `json:"persistent,omitempty"`
	}
	type sub struct {
		Name  string `json:"name"`
		Use   string `json:"use"`
		Short string `json:"short"`
	}
	var flags []flag
	cmd.NonInheritedFlags().VisitAll(func(f *pflag.Flag) {
		if f.Hidden {
			return
		}
		flags = append(flags, flag{Name: f.Name, Shorthand: f.Shorthand, Type: f.Value.Type(), Default: f.DefValue, Usage: f.Usage})
	})
	cmd.InheritedFlags().VisitAll(func(f *pflag.Flag) {
		if f.Hidden {
			return
		}
		flags = append(flags, flag{Name: f.Name, Shorthand: f.Shorthand, Type: f.Value.Type(), Default: f.DefValue, Usage: f.Usage, Persistent: true})
	})
	var subs []sub
	for _, c := range cmd.Commands() {
		if c.Hidden || c.Name() == "help" {
			continue
		}
		subs = append(subs, sub{Name: c.Name(), Use: c.UseLine(), Short: c.Short})
	}
	gotchas := []string{}
	if cmd.Annotations != nil && cmd.Annotations["gotchas"] != "" {
		gotchas = strings.Split(cmd.Annotations["gotchas"], "\n")
	}
	env := output.Envelope{
		OK: true,
		Data: map[string]any{
			"use":         cmd.UseLine(),
			"short":       cmd.Short,
			"long":        cmd.Long,
			"flags":       flags,
			"subcommands": subs,
			"gotchas":     gotchas,
		},
		Summary: "help for " + cmd.CommandPath(),
	}
	enc := json.NewEncoder(w)
	enc.SetEscapeHTML(false)
	return enc.Encode(env)
}

func (rt *Runtime) completionCmd() *cobra.Command {
	return &cobra.Command{
		Use:       "completion [bash|zsh|fish|powershell]",
		Short:     "Generate shell completion",
		Args:      cobra.ExactArgs(1),
		ValidArgs: []string{"bash", "zsh", "fish", "powershell"},
		RunE: func(cmd *cobra.Command, args []string) error {
			root := cmd.Root()
			switch args[0] {
			case "bash":
				return root.GenBashCompletion(rt.Env.Stdout)
			case "zsh":
				return root.GenZshCompletion(rt.Env.Stdout)
			case "fish":
				return root.GenFishCompletion(rt.Env.Stdout, true)
			case "powershell":
				return root.GenPowerShellCompletionWithDesc(rt.Env.Stdout)
			default:
				return rt.Out.Fail(output.Usage("unknown shell", "Use bash, zsh, fish, or powershell"))
			}
		},
	}
}

func crumb(action, cmd string) output.Breadcrumb {
	return output.Breadcrumb{Action: action, Cmd: cmd}
}

func plural(n int, word string) string {
	if n == 1 {
		return fmt.Sprintf("1 %s", word)
	}
	return fmt.Sprintf("%d %ss", n, word)
}
