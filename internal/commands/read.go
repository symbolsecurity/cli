package commands

import (
	"net/url"

	"github.com/spf13/cobra"

	"github.com/symbolsecurity/cli/internal/output"
)

func (rt *Runtime) assessmentsCmd() *cobra.Command {
	cmd := &cobra.Command{Use: "assessments", Short: "Assessments"}
	cmd.AddCommand(&cobra.Command{
		Use:   "list",
		Short: "List assessments",
		RunE: func(cmd *cobra.Command, args []string) error {
			var status string
			status, _ = cmd.Flags().GetString("status")
			q := url.Values{}
			qset(q, "status", status)
			return rt.list(cmd, rt.companyPath("/assessments"), q, "assessment", []output.Breadcrumb{
				crumb("users", "symbol users list"),
			})
		},
	})
	cmd.Commands()[0].Flags().String("status", "", "Assessment status")
	return cmd
}

func (rt *Runtime) phishingCmd() *cobra.Command {
	cmd := &cobra.Command{Use: "phishing", Short: "Reported phishing"}
	var from, until, email, category string
	list := &cobra.Command{
		Use:   "list",
		Short: "List reported phishing",
		RunE: func(cmd *cobra.Command, args []string) error {
			q := url.Values{}
			qset(q, "from", from)
			qset(q, "until", until)
			qset(q, "email", email)
			qset(q, "category", category)
			return rt.list(cmd, rt.companyPath("/reported-phishing"), q, "report", []output.Breadcrumb{
				crumb("simulations", "symbol simulations list"),
			})
		},
	}
	list.Flags().StringVar(&from, "from", "", "From date")
	list.Flags().StringVar(&until, "until", "", "Until date")
	list.Flags().StringVar(&email, "email", "", "Email filter")
	list.Flags().StringVar(&category, "category", "", "Category")
	cmd.AddCommand(list)
	return cmd
}

func (rt *Runtime) simulationsCmd() *cobra.Command {
	cmd := &cobra.Command{Use: "simulations", Short: "Threat simulations"}
	var from, until, status, users string
	list := &cobra.Command{
		Use:   "list",
		Short: "List threat simulations",
		RunE: func(cmd *cobra.Command, args []string) error {
			q := url.Values{}
			qset(q, "from", from)
			qset(q, "until", until)
			qset(q, "status", status)
			qset(q, "users", users)
			return rt.list(cmd, rt.companyPath("/threat-simulations/list"), q, "simulation", []output.Breadcrumb{
				crumb("phishing", "symbol phishing list"),
			})
		},
	}
	list.Flags().StringVar(&from, "from", "", "From date")
	list.Flags().StringVar(&until, "until", "", "Until date")
	list.Flags().StringVar(&status, "status", "", "Status")
	list.Flags().StringVar(&users, "users", "", "User filter")
	cmd.AddCommand(list)
	return cmd
}
