package commands

import (
	"encoding/json"
	"net/http"
	"net/url"

	"github.com/spf13/cobra"

	"github.com/symbolsecurity/cli/internal/client"
	"github.com/symbolsecurity/cli/internal/output"
)

func clientData(body json.RawMessage) any {
	return client.AsData(body)
}

func (rt *Runtime) emailThreatsCmd() *cobra.Command {
	cmd := &cobra.Command{Use: "email-threats", Short: "Email threat alerts"}
	cmd.AddCommand(
		&cobra.Command{
			Use:   "alerts",
			Short: "List email threat alerts",
			RunE: func(cmd *cobra.Command, args []string) error {
				return rt.list(cmd, rt.companyPath("/email-threats/alerts"), nil, "alert", []output.Breadcrumb{
					crumb("activate", "symbol email-threats activate"),
					crumb("deactivate", "symbol email-threats deactivate --yes"),
				})
			},
		},
		&cobra.Command{
			Use:   "activate",
			Short: "Activate email threat alerts",
			RunE: func(cmd *cobra.Command, args []string) error {
				return rt.mutate(cmd, http.MethodPut, rt.companyPath("/email-threats/activate"), nil, "email threats activated", []output.Breadcrumb{
					crumb("alerts", "symbol email-threats alerts"),
				})
			},
		},
		&cobra.Command{
			Use:   "deactivate",
			Short: "Deactivate email threat alerts",
			Annotations: map[string]string{
				"gotchas": "Requires --yes",
			},
			RunE: func(cmd *cobra.Command, args []string) error {
				if err := rt.requireYes("deactivate email threats"); err != nil {
					return rt.Out.Fail(err)
				}
				return rt.mutate(cmd, http.MethodPut, rt.companyPath("/email-threats/deactivate"), nil, "email threats deactivated", []output.Breadcrumb{
					crumb("alerts", "symbol email-threats alerts"),
				})
			},
		},
	)
	return cmd
}

func (rt *Runtime) webhooksCmd() *cobra.Command {
	cmd := &cobra.Command{Use: "webhooks", Short: "Event subscriptions"}
	cmd.AddCommand(rt.webhooksListCmd(), rt.webhooksCreateCmd(), rt.webhooksUpdateCmd(), rt.webhooksDeleteCmd(), rt.webhooksTestCmd())
	return cmd
}

func (rt *Runtime) webhooksListCmd() *cobra.Command {
	return &cobra.Command{
		Use:   "list",
		Short: "List webhooks",
		RunE: func(cmd *cobra.Command, args []string) error {
			return rt.list(cmd, rt.companyPath("/event_subscriptions/"), nil, "webhook", []output.Breadcrumb{
				crumb("create", "symbol webhooks create --endpoint <url> --event-types <types>"),
			})
		},
	}
}

func (rt *Runtime) webhooksCreateCmd() *cobra.Command {
	var endpoint, types string
	cmd := &cobra.Command{
		Use:   "create",
		Short: "Create a webhook",
		RunE: func(cmd *cobra.Command, args []string) error {
			if endpoint == "" {
				return rt.Out.Fail(output.Usage("endpoint is required", "Pass --endpoint"))
			}
			body := map[string]any{"endpoint": endpoint, "eventTypes": csv(types)}
			return rt.mutate(cmd, http.MethodPost, rt.companyPath("/event_subscriptions/"), body, "webhook created", []output.Breadcrumb{
				crumb("list", "symbol webhooks list"),
				crumb("test", "symbol webhooks test <id> --event <type>"),
			})
		},
	}
	cmd.Flags().StringVar(&endpoint, "endpoint", "", "Callback URL")
	cmd.Flags().StringVar(&types, "event-types", "", "Comma-separated event types")
	_ = cmd.MarkFlagRequired("endpoint")
	return cmd
}

func (rt *Runtime) webhooksUpdateCmd() *cobra.Command {
	var endpoint, types string
	cmd := &cobra.Command{
		Use:   "update <id>",
		Short: "Update a webhook",
		Args:  cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			body := map[string]any{}
			if cmd.Flags().Changed("endpoint") {
				body["endpoint"] = endpoint
			}
			if cmd.Flags().Changed("event-types") {
				body["eventTypes"] = csv(types)
			}
			if len(body) == 0 {
				return rt.Out.Fail(output.Usage("no fields to update", "Pass --endpoint or --event-types"))
			}
			return rt.mutate(cmd, http.MethodPut, rt.companyPath("/event_subscriptions/"+args[0]+"/"), body, "webhook updated", []output.Breadcrumb{
				crumb("list", "symbol webhooks list"),
			})
		},
	}
	cmd.Flags().StringVar(&endpoint, "endpoint", "", "Callback URL")
	cmd.Flags().StringVar(&types, "event-types", "", "Comma-separated event types")
	return cmd
}

func (rt *Runtime) webhooksDeleteCmd() *cobra.Command {
	return &cobra.Command{
		Use:   "delete <id>",
		Short: "Delete a webhook",
		Args:  cobra.ExactArgs(1),
		Annotations: map[string]string{
			"gotchas": "Destructive; requires --yes",
		},
		RunE: func(cmd *cobra.Command, args []string) error {
			if err := rt.requireYes("delete this webhook"); err != nil {
				return rt.Out.Fail(err)
			}
			return rt.mutate(cmd, http.MethodDelete, rt.companyPath("/event_subscriptions/"+args[0]+"/"), nil, "webhook deleted", []output.Breadcrumb{
				crumb("list", "symbol webhooks list"),
			})
		},
	}
}

func (rt *Runtime) webhooksTestCmd() *cobra.Command {
	var event string
	cmd := &cobra.Command{
		Use:   "test <id>",
		Short: "Send a test event to a webhook",
		Args:  cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			body := map[string]any{}
			if event != "" {
				body["event"] = event
			}
			return rt.mutate(cmd, http.MethodPost, rt.companyPath("/event_subscriptions/"+args[0]+"/test/"), body, "webhook tested", []output.Breadcrumb{
				crumb("list", "symbol webhooks list"),
			})
		},
	}
	cmd.Flags().StringVar(&event, "event", "", "Event type to send")
	return cmd
}

func (rt *Runtime) invoicesCmd() *cobra.Command {
	cmd := &cobra.Command{Use: "invoices", Short: "Invoices"}
	cmd.AddCommand(&cobra.Command{
		Use:   "list",
		Short: "List invoices",
		RunE: func(cmd *cobra.Command, args []string) error {
			return rt.list(cmd, "/invoices/", nil, "invoice", nil)
		},
	})
	return cmd
}

func (rt *Runtime) ticketsCmd() *cobra.Command {
	cmd := &cobra.Command{Use: "tickets", Short: "Support tickets (admin)"}
	cmd.AddCommand(rt.ticketsListCmd(), rt.ticketsConversationCmd(), rt.ticketsNotesCmd(), rt.ticketsReplyCmd(), rt.ticketsStatusCmd())
	return cmd
}

func (rt *Runtime) ticketsListCmd() *cobra.Command {
	var filter, keyword, from, until, status, priority, companies, categories string
	cmd := &cobra.Command{
		Use:   "list",
		Short: "List tickets",
		RunE: func(cmd *cobra.Command, args []string) error {
			q := url.Values{}
			qset(q, "filter", filter)
			qset(q, "Keyword", keyword)
			qset(q, "From", from)
			qset(q, "Until", until)
			qset(q, "Status", status)
			qset(q, "priority", priority)
			qset(q, "Companies", companies)
			qset(q, "Categories", categories)
			return rt.list(cmd, "/tickets/", q, "ticket", []output.Breadcrumb{
				crumb("conversation", "symbol tickets conversation <id>"),
				crumb("status", "symbol tickets status <id> --to <status-id>"),
			})
		},
	}
	cmd.Flags().StringVar(&filter, "filter", "", "Filter")
	cmd.Flags().StringVar(&keyword, "keyword", "", "Keyword")
	cmd.Flags().StringVar(&from, "from", "", "From date")
	cmd.Flags().StringVar(&until, "until", "", "Until date")
	cmd.Flags().StringVar(&status, "status", "", "Status")
	cmd.Flags().StringVar(&priority, "priority", "", "Priority")
	cmd.Flags().StringVar(&companies, "companies", "", "Companies")
	cmd.Flags().StringVar(&categories, "categories", "", "Categories")
	return cmd
}

func (rt *Runtime) ticketsConversationCmd() *cobra.Command {
	return &cobra.Command{
		Use:   "conversation <id>",
		Short: "List ticket conversation",
		Args:  cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			return rt.list(cmd, "/tickets/"+args[0]+"/conversations", nil, "message", []output.Breadcrumb{
				crumb("reply", "symbol tickets reply <id> --body <text> --to <email>"),
				crumb("notes", "symbol tickets notes <id>"),
			})
		},
	}
}

func (rt *Runtime) ticketsNotesCmd() *cobra.Command {
	var body, parentID string
	cmd := &cobra.Command{
		Use:   "notes <id>",
		Short: "List or add internal notes",
		Args:  cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			if body != "" {
				fields := map[string][]string{"body": {body}}
				if parentID != "" {
					fields["parent_id"] = []string{parentID}
				}
				res, err := rt.Client.Multipart(rt.ctx(cmd), http.MethodPost, "/tickets/"+args[0]+"/internal-notes", fields)
				if err != nil {
					return rt.Out.Fail(err)
				}
				return rt.Out.Success(client.AsData(res.Body), "note added", []output.Breadcrumb{crumb("notes", "symbol tickets notes "+args[0])}, nil)
			}
			return rt.list(cmd, "/tickets/"+args[0]+"/internal-notes", nil, "note", []output.Breadcrumb{
				crumb("add", "symbol tickets notes <id> --body <text>"),
			})
		},
	}
	cmd.Flags().StringVar(&body, "body", "", "Add a note with this body")
	cmd.Flags().StringVar(&parentID, "parent-id", "", "Parent note id")
	return cmd
}

func (rt *Runtime) ticketsReplyCmd() *cobra.Command {
	var body, to, cc string
	cmd := &cobra.Command{
		Use:   "reply <id>",
		Short: "Reply to a ticket",
		Args:  cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			if body == "" || to == "" {
				return rt.Out.Fail(output.Usage("body and to are required", "Pass --body and --to"))
			}
			fields := map[string][]string{"body": {body}, "to_emails": csv(to)}
			if cc != "" {
				fields["cc_emails"] = csv(cc)
			}
			res, err := rt.Client.Multipart(rt.ctx(cmd), http.MethodPost, "/tickets/"+args[0]+"/reply", fields)
			if err != nil {
				return rt.Out.Fail(err)
			}
			return rt.Out.Success(client.AsData(res.Body), "reply sent", []output.Breadcrumb{crumb("conversation", "symbol tickets conversation "+args[0])}, nil)
		},
	}
	cmd.Flags().StringVar(&body, "body", "", "Reply body")
	cmd.Flags().StringVar(&to, "to", "", "Comma-separated recipient emails")
	cmd.Flags().StringVar(&cc, "cc", "", "Comma-separated cc emails")
	_ = cmd.MarkFlagRequired("body")
	_ = cmd.MarkFlagRequired("to")
	return cmd
}

func (rt *Runtime) ticketsStatusCmd() *cobra.Command {
	var to string
	cmd := &cobra.Command{
		Use:   "status <id>",
		Short: "Update ticket status, or list statuses without an id",
		RunE: func(cmd *cobra.Command, args []string) error {
			if len(args) == 0 {
				return rt.list(cmd, "/tickets/statuses", nil, "status", []output.Breadcrumb{
					crumb("update", "symbol tickets status <id> --to <status-id>"),
				})
			}
			if to == "" {
				return rt.Out.Fail(output.Usage("--to is required", "Pass --to <status-id> or omit the id to list statuses"))
			}
			return rt.mutate(cmd, http.MethodPut, "/tickets/"+args[0]+"/update-status", map[string]string{"status": to}, "ticket status updated", []output.Breadcrumb{
				crumb("list", "symbol tickets list"),
			})
		},
	}
	cmd.Flags().StringVar(&to, "to", "", "Status id")
	return cmd
}
