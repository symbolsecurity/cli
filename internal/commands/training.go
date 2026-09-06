package commands

import (
	"net/http"
	"net/url"

	"github.com/spf13/cobra"

	"github.com/symbolsecurity/cli/internal/output"
)

func (rt *Runtime) trainingCmd() *cobra.Command {
	cmd := &cobra.Command{Use: "training", Short: "Training assignments and assets"}
	cmd.AddCommand(rt.trainingListCmd(), rt.trainingAssetsCmd(), rt.trainingLeaderboardCmd(), rt.trainingAssignCmd())
	return cmd
}

func (rt *Runtime) trainingListCmd() *cobra.Command {
	var from, until, status, users, courses string
	cmd := &cobra.Command{
		Use:   "list",
		Short: "List training assignments",
		RunE: func(cmd *cobra.Command, args []string) error {
			q := url.Values{}
			qset(q, "from", from)
			qset(q, "until", until)
			qset(q, "status", status)
			qset(q, "users", users)
			qset(q, "courses", courses)
			return rt.list(cmd, rt.companyPath("/training/list"), q, "training", []output.Breadcrumb{
				crumb("assign", "symbol training assign --assets <id> --users <id>"),
				crumb("assets", "symbol training assets"),
			})
		},
	}
	cmd.Flags().StringVar(&from, "from", "", "From date")
	cmd.Flags().StringVar(&until, "until", "", "Until date")
	cmd.Flags().StringVar(&status, "status", "", "Assignment status")
	cmd.Flags().StringVar(&users, "users", "", "User filter")
	cmd.Flags().StringVar(&courses, "courses", "", "Course filter")
	return cmd
}

func (rt *Runtime) trainingAssetsCmd() *cobra.Command {
	return &cobra.Command{
		Use:   "assets",
		Short: "List training assets",
		RunE: func(cmd *cobra.Command, args []string) error {
			return rt.list(cmd, rt.companyPath("/training/assets"), nil, "asset", []output.Breadcrumb{
				crumb("assign", "symbol training assign --assets <id> --users <id>"),
			})
		},
	}
}

func (rt *Runtime) trainingLeaderboardCmd() *cobra.Command {
	return &cobra.Command{
		Use:   "leaderboard",
		Short: "Show training leaderboard",
		RunE: func(cmd *cobra.Command, args []string) error {
			return rt.get(cmd, rt.companyPath("/training/leaderboard"), nil, "leaderboard", []output.Breadcrumb{
				crumb("list", "symbol training list"),
			})
		},
	}
}

func (rt *Runtime) trainingAssignCmd() *cobra.Command {
	var assets, users, due string
	var allUsers bool
	cmd := &cobra.Command{
		Use:   "assign",
		Short: "Assign training to users",
		RunE: func(cmd *cobra.Command, args []string) error {
			as := csv(assets)
			if len(as) == 0 {
				return rt.Out.Fail(output.Usage("assets are required", "Pass --assets <id>[,<id>]"))
			}
			body := map[string]any{"assets": as, "all_users": allUsers}
			if us := csv(users); len(us) > 0 {
				body["users"] = us
			}
			if due != "" {
				body["due_date"] = due
			}
			if !allUsers && body["users"] == nil {
				return rt.Out.Fail(output.Usage("users or --all-users is required", "Pass --users <id>[,<id>] or --all-users"))
			}
			return rt.mutate(cmd, http.MethodPost, rt.companyPath("/training/assign"), body, "training assigned", []output.Breadcrumb{
				crumb("list", "symbol training list"),
			})
		},
	}
	cmd.Flags().StringVar(&assets, "assets", "", "Comma-separated asset ids")
	cmd.Flags().StringVar(&users, "users", "", "Comma-separated user ids")
	cmd.Flags().StringVar(&due, "due-date", "", "Due date")
	cmd.Flags().BoolVar(&allUsers, "all-users", false, "Assign to all users")
	return cmd
}
