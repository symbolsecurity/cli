package commands

import (
	"net/http"
	"net/url"

	"github.com/spf13/cobra"

	"github.com/symbolsecurity/cli/internal/output"
)

func (rt *Runtime) usersCmd() *cobra.Command {
	cmd := &cobra.Command{Use: "users", Short: "Manage users"}
	cmd.AddCommand(rt.usersListCmd(), rt.usersCreateCmd(), rt.usersUpdateCmd(), rt.usersDeleteCmd())
	return cmd
}

func (rt *Runtime) usersListCmd() *cobra.Command {
	var keyword, trainingStatus string
	cmd := &cobra.Command{
		Use:   "list",
		Short: "List users",
		Annotations: map[string]string{
			"gotchas": "Paginate; do not --all huge lists unless asked",
		},
		RunE: func(cmd *cobra.Command, args []string) error {
			q := url.Values{}
			qset(q, "keyword", keyword)
			qset(q, "training_status", trainingStatus)
			return rt.list(cmd, rt.companyPath("/users/"), q, "user", []output.Breadcrumb{
				crumb("create", "symbol users create --email <email> --first-name <name> --last-name <name>"),
				crumb("update", "symbol users update <id>"),
				crumb("delete", "symbol users delete <id> --yes"),
			})
		},
	}
	cmd.Flags().StringVar(&keyword, "keyword", "", "Search keyword")
	cmd.Flags().StringVar(&trainingStatus, "training-status", "", "OVERDUE|COMPLETED|PENDING")
	return cmd
}

func (rt *Runtime) usersCreateCmd() *cobra.Command {
	var email, first, last, title, categories string
	cmd := &cobra.Command{
		Use:   "create",
		Short: "Create a user",
		RunE: func(cmd *cobra.Command, args []string) error {
			if email == "" {
				return rt.Out.Fail(output.Usage("email is required", "Pass --email"))
			}
			body := map[string]any{"email": email, "firstName": first, "lastName": last, "title": title}
			if cats := csv(categories); len(cats) > 0 {
				body["categories"] = cats
			}
			return rt.mutate(cmd, http.MethodPost, rt.companyPath("/users/"), body, "user created", []output.Breadcrumb{
				crumb("list", "symbol users list"),
			})
		},
	}
	cmd.Flags().StringVar(&email, "email", "", "Email")
	cmd.Flags().StringVar(&first, "first-name", "", "First name")
	cmd.Flags().StringVar(&last, "last-name", "", "Last name")
	cmd.Flags().StringVar(&title, "title", "", "Title")
	cmd.Flags().StringVar(&categories, "categories", "", "Comma-separated categories")
	_ = cmd.MarkFlagRequired("email")
	return cmd
}

func (rt *Runtime) usersUpdateCmd() *cobra.Command {
	var email, first, last, title, categories string
	cmd := &cobra.Command{
		Use:   "update <id>",
		Short: "Update a user",
		Args:  uuidArg(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			body := map[string]any{}
			if cmd.Flags().Changed("email") {
				body["email"] = email
			}
			if cmd.Flags().Changed("first-name") {
				body["firstName"] = first
			}
			if cmd.Flags().Changed("last-name") {
				body["lastName"] = last
			}
			if cmd.Flags().Changed("title") {
				body["title"] = title
			}
			if cmd.Flags().Changed("categories") {
				body["categories"] = csv(categories)
			}
			if len(body) == 0 {
				return rt.Out.Fail(output.Usage("no fields to update", "Pass --email, --first-name, --last-name, --title, or --categories"))
			}
			return rt.mutate(cmd, http.MethodPut, rt.companyPath("/users/"+args[0]+"/"), body, "user updated", []output.Breadcrumb{
				crumb("list", "symbol users list"),
			})
		},
	}
	cmd.Flags().StringVar(&email, "email", "", "Email")
	cmd.Flags().StringVar(&first, "first-name", "", "First name")
	cmd.Flags().StringVar(&last, "last-name", "", "Last name")
	cmd.Flags().StringVar(&title, "title", "", "Title")
	cmd.Flags().StringVar(&categories, "categories", "", "Comma-separated categories")
	return cmd
}

func (rt *Runtime) usersDeleteCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "delete <id>",
		Short: "Delete a user",
		Args:  uuidArg(1),
		Annotations: map[string]string{
			"gotchas": "Destructive; requires --yes",
		},
		RunE: func(cmd *cobra.Command, args []string) error {
			if err := rt.requireYes("delete this user"); err != nil {
				return rt.Out.Fail(err)
			}
			return rt.mutate(cmd, http.MethodDelete, rt.companyPath("/users/"+args[0]+"/"), nil, "user deleted", []output.Breadcrumb{
				crumb("list", "symbol users list"),
			})
		},
	}
	return cmd
}
