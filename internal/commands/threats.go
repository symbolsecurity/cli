package commands

import (
	"net/http"
	"net/url"
	"strings"

	"github.com/spf13/cobra"

	"github.com/symbolsecurity/cli/internal/output"
)

func (rt *Runtime) threatsCmd() *cobra.Command {
	cmd := &cobra.Command{Use: "threats", Short: "Cyber threat results and keywords"}
	cmd.AddCommand(
		rt.threatsListCmd(),
		rt.threatsStatusCmd(),
		rt.threatsIgnoreCmd(),
		rt.threatsDeleteCmd(),
		rt.threatsKeywordsCmd(),
	)
	return cmd
}

func (rt *Runtime) threatsListCmd() *cobra.Command {
	var status, from, until string
	var showIgnored bool
	cmd := &cobra.Command{
		Use:   "list",
		Short: "List cyber threat results",
		Annotations: map[string]string{
			"gotchas": "Paginate; do not --all huge lists unless asked",
		},
		RunE: func(cmd *cobra.Command, args []string) error {
			q := url.Values{}
			qset(q, "status", status)
			qset(q, "from", from)
			qset(q, "until", until)
			if showIgnored {
				q.Set("show_ignored", "true")
			}
			return rt.list(cmd, rt.companyPath("/cyber-threats/results/"), q, "threat", []output.Breadcrumb{
				crumb("status", "symbol threats status <id> --to PENDING|URGENT|RESOLVED"),
				crumb("ignore", "symbol threats ignore <id> --yes"),
			})
		},
	}
	cmd.Flags().StringVar(&status, "status", "", "PENDING|URGENT|RESOLVED")
	cmd.Flags().StringVar(&from, "from", "", "From date")
	cmd.Flags().StringVar(&until, "until", "", "Until date")
	cmd.Flags().BoolVar(&showIgnored, "show-ignored", false, "Include ignored results")
	return cmd
}

func (rt *Runtime) threatsStatusCmd() *cobra.Command {
	var to string
	cmd := &cobra.Command{
		Use:   "status <id>",
		Short: "Update a threat result status",
		Args:  uuidArg(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			to = strings.ToUpper(to)
			switch to {
			case "PENDING", "URGENT", "RESOLVED":
			default:
				return rt.Out.Fail(output.Usage("invalid status", "Use --to PENDING|URGENT|RESOLVED"))
			}
			body := map[string]any{"status": to}
			return rt.mutate(cmd, http.MethodPut, rt.companyPath("/cyber-threats/results/"+args[0]+"/change-status/"), body, "threat status updated", []output.Breadcrumb{
				crumb("list", "symbol threats list"),
			})
		},
	}
	cmd.Flags().StringVar(&to, "to", "", "PENDING|URGENT|RESOLVED")
	_ = cmd.MarkFlagRequired("to")
	return cmd
}

func (rt *Runtime) threatsIgnoreCmd() *cobra.Command {
	return &cobra.Command{
		Use:   "ignore <id>",
		Short: "Ignore a threat result",
		Args:  uuidArg(1),
		Annotations: map[string]string{
			"gotchas": "Requires --yes",
		},
		RunE: func(cmd *cobra.Command, args []string) error {
			if err := rt.requireYes("ignore this threat"); err != nil {
				return rt.Out.Fail(err)
			}
			return rt.mutate(cmd, http.MethodPut, rt.companyPath("/cyber-threats/results/"+args[0]+"/ignore/"), nil, "threat ignored", []output.Breadcrumb{
				crumb("list", "symbol threats list"),
			})
		},
	}
}

func (rt *Runtime) threatsDeleteCmd() *cobra.Command {
	return &cobra.Command{
		Use:   "delete <id>",
		Short: "Delete a threat result",
		Args:  uuidArg(1),
		Annotations: map[string]string{
			"gotchas": "Destructive; requires --yes",
		},
		RunE: func(cmd *cobra.Command, args []string) error {
			if err := rt.requireYes("delete this threat"); err != nil {
				return rt.Out.Fail(err)
			}
			return rt.mutate(cmd, http.MethodDelete, rt.companyPath("/cyber-threats/results/"+args[0]), nil, "threat deleted", []output.Breadcrumb{
				crumb("list", "symbol threats list"),
			})
		},
	}
}

func (rt *Runtime) threatsKeywordsCmd() *cobra.Command {
	cmd := &cobra.Command{Use: "keywords", Short: "Cyber threat keywords"}
	cmd.AddCommand(rt.threatsKeywordsListCmd(), rt.threatsKeywordsCreateCmd(), rt.threatsKeywordsDeleteCmd())
	return cmd
}

func (rt *Runtime) threatsKeywordsListCmd() *cobra.Command {
	return &cobra.Command{
		Use:   "list",
		Short: "List keywords",
		RunE: func(cmd *cobra.Command, args []string) error {
			return rt.list(cmd, rt.companyPath("/cyber-threats/keywords/"), nil, "keyword", []output.Breadcrumb{
				crumb("create", "symbol threats keywords create --type <type> --value <value>"),
			})
		},
	}
}

func (rt *Runtime) threatsKeywordsCreateCmd() *cobra.Command {
	var typ, value string
	cmd := &cobra.Command{
		Use:   "create",
		Short: "Create a keyword",
		RunE: func(cmd *cobra.Command, args []string) error {
			if value == "" {
				return rt.Out.Fail(output.Usage("value is required", "Pass --value"))
			}
			body := []map[string]string{{"type": typ, "value": value}}
			return rt.mutate(cmd, http.MethodPost, rt.companyPath("/cyber-threats/keywords/"), body, "keyword created", []output.Breadcrumb{
				crumb("list", "symbol threats keywords list"),
			})
		},
	}
	cmd.Flags().StringVar(&typ, "type", "", "Keyword type")
	cmd.Flags().StringVar(&value, "value", "", "Keyword value")
	_ = cmd.MarkFlagRequired("value")
	return cmd
}

func (rt *Runtime) threatsKeywordsDeleteCmd() *cobra.Command {
	return &cobra.Command{
		Use:   "delete <keyword>",
		Short: "Delete a keyword",
		Args:  cobra.ExactArgs(1),
		Annotations: map[string]string{
			"gotchas": "Destructive; requires --yes",
		},
		RunE: func(cmd *cobra.Command, args []string) error {
			if err := rt.requireYes("delete this keyword"); err != nil {
				return rt.Out.Fail(err)
			}
			seg, err := pathSeg(args[0], "keyword")
			if err != nil {
				return rt.Out.Fail(err)
			}
			return rt.mutate(cmd, http.MethodDelete, rt.companyPath("/cyber-threats/keywords/"+seg+"/"), nil, "keyword deleted", []output.Breadcrumb{
				crumb("list", "symbol threats keywords list"),
			})
		},
	}
}
