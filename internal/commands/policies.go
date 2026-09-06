package commands

import (
	"net/http"
	"net/url"

	"github.com/spf13/cobra"

	"github.com/symbolsecurity/cli/internal/output"
)

func (rt *Runtime) policiesCmd() *cobra.Command {
	cmd := &cobra.Command{Use: "policies", Short: "Policies and assignments"}
	cmd.AddCommand(rt.policiesListCmd(), rt.policiesAssignmentsCmd(), rt.policiesAssignCmd())
	return cmd
}

func (rt *Runtime) policiesListCmd() *cobra.Command {
	return &cobra.Command{
		Use:   "list",
		Short: "List policies",
		RunE: func(cmd *cobra.Command, args []string) error {
			return rt.list(cmd, rt.companyPath("/policies"), nil, "policy", []output.Breadcrumb{
				crumb("assign", "symbol policies assign --policy-id <id> --user-ids <id>"),
				crumb("assignments", "symbol policies assignments"),
			})
		},
	}
}

func (rt *Runtime) policiesAssignmentsCmd() *cobra.Command {
	var policyID, status, userID, from, until string
	cmd := &cobra.Command{
		Use:   "assignments",
		Short: "List policy assignments",
		RunE: func(cmd *cobra.Command, args []string) error {
			q := url.Values{}
			qset(q, "policy_id", policyID)
			qset(q, "status", status)
			qset(q, "user_id", userID)
			qset(q, "from", from)
			qset(q, "until", until)
			return rt.list(cmd, rt.companyPath("/policies/assignments"), q, "assignment", []output.Breadcrumb{
				crumb("assign", "symbol policies assign --policy-id <id> --user-ids <id>"),
			})
		},
	}
	cmd.Flags().StringVar(&policyID, "policy-id", "", "Policy id")
	cmd.Flags().StringVar(&status, "status", "", "Assignment status")
	cmd.Flags().StringVar(&userID, "user-id", "", "User id")
	cmd.Flags().StringVar(&from, "from", "", "From date")
	cmd.Flags().StringVar(&until, "until", "", "Until date")
	return cmd
}

func (rt *Runtime) policiesAssignCmd() *cobra.Command {
	var policyID, userIDs, target string
	var notify, all bool
	cmd := &cobra.Command{
		Use:   "assign",
		Short: "Assign a policy to users",
		RunE: func(cmd *cobra.Command, args []string) error {
			if policyID == "" {
				return rt.Out.Fail(output.Usage("policy-id is required", "Pass --policy-id"))
			}
			if target == "" {
				if all {
					target = "all"
				} else {
					target = "custom"
				}
			}
			body := map[string]any{
				"policy_id":        policyID,
				"notify":           notify,
				"target_selection": target,
			}
			if ids := csv(userIDs); len(ids) > 0 {
				body["user_ids"] = ids
			}
			if target == "custom" && body["user_ids"] == nil {
				return rt.Out.Fail(output.Usage("user-ids required for custom target", "Pass --user-ids or --all"))
			}
			return rt.mutate(cmd, http.MethodPost, rt.companyPath("/policies/assign"), body, "policy assigned", []output.Breadcrumb{
				crumb("assignments", "symbol policies assignments"),
			})
		},
	}
	cmd.Flags().StringVar(&policyID, "policy-id", "", "Policy id")
	cmd.Flags().StringVar(&userIDs, "user-ids", "", "Comma-separated user ids")
	cmd.Flags().StringVar(&target, "target", "", "all or custom")
	cmd.Flags().BoolVar(&notify, "notify", false, "Notify users")
	cmd.Flags().BoolVar(&all, "all-users", false, "Assign to all users")
	return cmd
}
