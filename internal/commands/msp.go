package commands

import (
	"net/http"
	"net/url"

	"github.com/spf13/cobra"

	"github.com/symbolsecurity/cli/internal/client"
	"github.com/symbolsecurity/cli/internal/ident"
	"github.com/symbolsecurity/cli/internal/output"
)

func (rt *Runtime) companiesCmd() *cobra.Command {
	cmd := &cobra.Command{Use: "companies", Short: "MSP companies"}
	cmd.AddCommand(
		rt.companiesListCmd(),
		rt.companiesShowCmd(),
		rt.companiesCreateCmd(),
		rt.companiesUpdateCmd(),
		rt.companiesDeleteCmd(),
		rt.companiesFeaturesCmd(),
	)
	return cmd
}

func (rt *Runtime) companiesListCmd() *cobra.Command {
	return &cobra.Command{
		Use:   "list",
		Short: "List MSP child companies",
		RunE: func(cmd *cobra.Command, args []string) error {
			data, pag, err := rt.Client.ListMSP(rt.ctx(cmd), "/msp/companies", nil, rt.page())
			if err != nil {
				return rt.Out.Fail(err)
			}
			return rt.Out.Success(data, listSummary(data, "company"), []output.Breadcrumb{
				crumb("show", "symbol companies show <id>"),
				crumb("create", "symbol companies create --name <name>"),
			}, pag)
		},
	}
}

func (rt *Runtime) companiesShowCmd() *cobra.Command {
	return &cobra.Command{
		Use:   "show <id>",
		Short: "Show a company",
		Args:  uuidArg(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			res, err := rt.Client.Call(rt.ctx(cmd), http.MethodGet, "/msp/companies/"+args[0]+"/", nil, nil, true)
			if err != nil {
				return rt.Out.Fail(err)
			}
			return rt.Out.Success(client.AsData(res.Body), "company", []output.Breadcrumb{
				crumb("users", "symbol users list --company "+args[0]),
				crumb("features", "symbol companies features list "+args[0]),
			}, nil)
		},
	}
}

func companyBody(cmd *cobra.Command, name, line1, line2, city, country, state, tz, zip string) map[string]any {
	body := map[string]any{}
	set := func(flag, key, val string) {
		if cmd.Flags().Changed(flag) && val != "" {
			body[key] = val
		}
	}
	if name != "" {
		body["name"] = name
	}
	set("address-line1", "addressLine1", line1)
	set("address-line2", "addressLine2", line2)
	set("city", "city", city)
	set("country", "country", country)
	set("state", "state", state)
	set("timezone", "timezone", tz)
	set("zip-code", "zipCode", zip)
	return body
}

func companyFlags(cmd *cobra.Command, name, line1, line2, city, country, state, tz, zip *string) {
	cmd.Flags().StringVar(name, "name", "", "Company name")
	cmd.Flags().StringVar(line1, "address-line1", "", "Address line 1")
	cmd.Flags().StringVar(line2, "address-line2", "", "Address line 2")
	cmd.Flags().StringVar(city, "city", "", "City")
	cmd.Flags().StringVar(country, "country", "", "Country")
	cmd.Flags().StringVar(state, "state", "", "State")
	cmd.Flags().StringVar(tz, "timezone", "", "Timezone")
	cmd.Flags().StringVar(zip, "zip-code", "", "Zip code")
}

func (rt *Runtime) companiesCreateCmd() *cobra.Command {
	var name, line1, line2, city, country, state, tz, zip string
	cmd := &cobra.Command{
		Use:   "create",
		Short: "Create a child company",
		RunE: func(cmd *cobra.Command, args []string) error {
			if name == "" {
				return rt.Out.Fail(output.Usage("name is required", "Pass --name"))
			}
			body := companyBody(cmd, name, line1, line2, city, country, state, tz, zip)
			res, err := rt.Client.Call(rt.ctx(cmd), http.MethodPost, "/msp/companies/", nil, body, true)
			if err != nil {
				return rt.Out.Fail(err)
			}
			return rt.Out.Success(client.AsData(res.Body), "company created", []output.Breadcrumb{crumb("list", "symbol companies list")}, nil)
		},
	}
	companyFlags(cmd, &name, &line1, &line2, &city, &country, &state, &tz, &zip)
	_ = cmd.MarkFlagRequired("name")
	return cmd
}

func (rt *Runtime) companiesUpdateCmd() *cobra.Command {
	var name, line1, line2, city, country, state, tz, zip string
	cmd := &cobra.Command{
		Use:   "update <id>",
		Short: "Update a child company",
		Args:  uuidArg(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			body := companyBody(cmd, "", line1, line2, city, country, state, tz, zip)
			if cmd.Flags().Changed("name") {
				body["name"] = name
			}
			if len(body) == 0 {
				return rt.Out.Fail(output.Usage("no fields to update", "Pass at least one field flag"))
			}
			res, err := rt.Client.Call(rt.ctx(cmd), http.MethodPut, "/msp/companies/"+args[0], nil, body, true)
			if err != nil {
				return rt.Out.Fail(err)
			}
			return rt.Out.Success(client.AsData(res.Body), "company updated", []output.Breadcrumb{crumb("show", "symbol companies show "+args[0])}, nil)
		},
	}
	companyFlags(cmd, &name, &line1, &line2, &city, &country, &state, &tz, &zip)
	return cmd
}

func (rt *Runtime) companiesDeleteCmd() *cobra.Command {
	return &cobra.Command{
		Use:   "delete <id>",
		Short: "Delete a child company",
		Args:  uuidArg(1),
		Annotations: map[string]string{
			"gotchas": "Destructive; requires --yes",
		},
		RunE: func(cmd *cobra.Command, args []string) error {
			if err := rt.requireYes("delete this company"); err != nil {
				return rt.Out.Fail(err)
			}
			res, err := rt.Client.Call(rt.ctx(cmd), http.MethodDelete, "/msp/companies/"+args[0]+"/", nil, nil, true)
			if err != nil {
				return rt.Out.Fail(err)
			}
			data := client.AsData(res.Body)
			if len(res.Body) == 0 {
				data = map[string]any{"ok": true}
			}
			return rt.Out.Success(data, "company deleted", []output.Breadcrumb{crumb("list", "symbol companies list")}, nil)
		},
	}
}

func (rt *Runtime) companiesFeaturesCmd() *cobra.Command {
	cmd := &cobra.Command{Use: "features", Short: "Company feature flags"}
	cmd.AddCommand(rt.featuresListCmd(), rt.featuresEnableCmd(), rt.featuresDisableCmd())
	return cmd
}

func (rt *Runtime) companyArg(args []string) (string, error) {
	id := rt.Company
	if len(args) > 0 && args[0] != "" {
		id = args[0]
	}
	if id == "" {
		return "", output.Usage("company id is required", "Pass <company_id> or --company")
	}
	if err := ident.UUID(id, "company id"); err != nil {
		return "", err
	}
	return id, nil
}

func (rt *Runtime) featuresListCmd() *cobra.Command {
	return &cobra.Command{
		Use:   "list [company_id]",
		Short: "List features for a child company",
		RunE: func(cmd *cobra.Command, args []string) error {
			id, err := rt.companyArg(args)
			if err != nil {
				return rt.Out.Fail(err)
			}
			res, err := rt.Client.Call(rt.ctx(cmd), http.MethodGet, "/msp/companies/"+id+"/features", nil, nil, true)
			if err != nil {
				return rt.Out.Fail(err)
			}
			data := client.AsData(res.Body)
			return rt.Out.Success(data, listSummary(data, "feature"), []output.Breadcrumb{
				crumb("enable", "symbol companies features enable "+id+" <feature>"),
				crumb("disable", "symbol companies features disable "+id+" <feature> --yes"),
			}, nil)
		},
	}
}

func (rt *Runtime) featuresEnableCmd() *cobra.Command {
	return &cobra.Command{
		Use:   "enable [company_id] <feature>",
		Short: "Enable a feature",
		Args:  cobra.RangeArgs(1, 2),
		RunE: func(cmd *cobra.Command, args []string) error {
			id, feature, err := rt.featureArgs(args)
			if err != nil {
				return rt.Out.Fail(err)
			}
			res, err := rt.Client.Call(rt.ctx(cmd), http.MethodPut, "/msp/companies/"+id+"/features/"+feature+"/enable/", nil, nil, true)
			if err != nil {
				return rt.Out.Fail(err)
			}
			return rt.Out.Success(client.AsData(res.Body), "feature enabled", []output.Breadcrumb{crumb("list", "symbol companies features list "+id)}, nil)
		},
	}
}

func (rt *Runtime) featuresDisableCmd() *cobra.Command {
	return &cobra.Command{
		Use:   "disable [company_id] <feature>",
		Short: "Disable a feature",
		Args:  cobra.RangeArgs(1, 2),
		Annotations: map[string]string{
			"gotchas": "Requires --yes",
		},
		RunE: func(cmd *cobra.Command, args []string) error {
			if err := rt.requireYes("disable this feature"); err != nil {
				return rt.Out.Fail(err)
			}
			id, feature, err := rt.featureArgs(args)
			if err != nil {
				return rt.Out.Fail(err)
			}
			res, err := rt.Client.Call(rt.ctx(cmd), http.MethodPut, "/msp/companies/"+id+"/features/"+feature+"/disable/", nil, nil, true)
			if err != nil {
				return rt.Out.Fail(err)
			}
			return rt.Out.Success(client.AsData(res.Body), "feature disabled", []output.Breadcrumb{crumb("list", "symbol companies features list "+id)}, nil)
		},
	}
}

func (rt *Runtime) featureArgs(args []string) (string, string, error) {
	if len(args) == 2 {
		if err := ident.UUID(args[0], "company id"); err != nil {
			return "", "", err
		}
		seg, err := ident.Seg(args[1], "feature")
		if err != nil {
			return "", "", err
		}
		return args[0], seg, nil
	}
	if rt.Company == "" {
		return "", "", output.Usage("company id is required", "Pass <company_id> <feature> or --company with <feature>")
	}
	if err := ident.UUID(rt.Company, "company id"); err != nil {
		return "", "", err
	}
	seg, err := ident.Seg(args[0], "feature")
	if err != nil {
		return "", "", err
	}
	return rt.Company, seg, nil
}

func (rt *Runtime) programsCmd() *cobra.Command {
	cmd := &cobra.Command{Use: "programs", Short: "MSP program templates"}
	cmd.AddCommand(rt.programsListCmd(), rt.programsAssignCmd())
	return cmd
}

func (rt *Runtime) programsListCmd() *cobra.Command {
	return &cobra.Command{
		Use:   "list",
		Short: "List program templates",
		RunE: func(cmd *cobra.Command, args []string) error {
			data, pag, err := rt.Client.ListMSP(rt.ctx(cmd), "/msp/program_templates", nil, rt.page())
			if err != nil {
				return rt.Out.Fail(err)
			}
			return rt.Out.Success(data, listSummary(data, "program"), []output.Breadcrumb{
				crumb("assign", "symbol programs assign <template_id> --company <id>"),
			}, pag)
		},
	}
}

func (rt *Runtime) programsAssignCmd() *cobra.Command {
	return &cobra.Command{
		Use:   "assign <template_id>",
		Short: "Assign a program template to a company",
		Args:  uuidArg(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			if rt.Company == "" {
				return rt.Out.Fail(output.Usage("company is required", "Pass --company <id>"))
			}
			res, err := rt.Client.Call(rt.ctx(cmd), http.MethodPost, "/msp/companies/"+rt.Company+"/programs/"+args[0]+"/assign", nil, nil, true)
			if err != nil {
				return rt.Out.Fail(err)
			}
			return rt.Out.Success(client.AsData(res.Body), "program assigned", []output.Breadcrumb{crumb("list", "symbol programs list")}, nil)
		},
	}
}

func (rt *Runtime) mspCmd() *cobra.Command {
	cmd := &cobra.Command{Use: "msp", Short: "MSP-only activity, schedules, and users"}
	cmd.AddCommand(rt.mspActivityCmd(), rt.mspSchedulesCmd(), rt.mspUsersCmd())
	return cmd
}

func (rt *Runtime) mspActivityCmd() *cobra.Command {
	var month, year, companyID string
	cmd := &cobra.Command{
		Use:   "activity",
		Short: "MSP monthly activity",
		RunE: func(cmd *cobra.Command, args []string) error {
			q := url.Values{}
			qset(q, "month", month)
			qset(q, "year", year)
			cid := companyID
			if cid == "" {
				cid = rt.Company
			}
			qset(q, "company_id", cid)
			data, pag, err := rt.Client.ListMSP(rt.ctx(cmd), "/msp/activity/", q, rt.page())
			if err != nil {
				return rt.Out.Fail(err)
			}
			return rt.Out.Success(data, listSummary(data, "item"), []output.Breadcrumb{crumb("schedules", "symbol msp schedules")}, pag)
		},
	}
	cmd.Flags().StringVar(&month, "month", "", "Month")
	cmd.Flags().StringVar(&year, "year", "", "Year")
	cmd.Flags().StringVar(&companyID, "company-id", "", "Filter by company")
	return cmd
}

func (rt *Runtime) mspSchedulesCmd() *cobra.Command {
	var month, year, companyID string
	var includeInactive bool
	cmd := &cobra.Command{
		Use:   "schedules",
		Short: "MSP monthly schedules",
		RunE: func(cmd *cobra.Command, args []string) error {
			q := url.Values{}
			qset(q, "month", month)
			qset(q, "year", year)
			cid := companyID
			if cid == "" {
				cid = rt.Company
			}
			qset(q, "company_id", cid)
			if includeInactive {
				q.Set("include_inactive", "true")
			}
			data, pag, err := rt.Client.ListMSP(rt.ctx(cmd), "/msp/schedules/", q, rt.page())
			if err != nil {
				return rt.Out.Fail(err)
			}
			return rt.Out.Success(data, listSummary(data, "schedule"), []output.Breadcrumb{crumb("activity", "symbol msp activity")}, pag)
		},
	}
	cmd.Flags().StringVar(&month, "month", "", "Month")
	cmd.Flags().StringVar(&year, "year", "", "Year")
	cmd.Flags().StringVar(&companyID, "company-id", "", "Filter by company")
	cmd.Flags().BoolVar(&includeInactive, "include-inactive", false, "Include inactive")
	return cmd
}

func (rt *Runtime) mspUsersCmd() *cobra.Command {
	var typ string
	cmd := &cobra.Command{
		Use:   "users",
		Short: "List users across the MSP",
		RunE: func(cmd *cobra.Command, args []string) error {
			q := url.Values{}
			qset(q, "type", typ)
			data, pag, err := rt.Client.ListMSP(rt.ctx(cmd), "/msp/users/", q, rt.page())
			if err != nil {
				return rt.Out.Fail(err)
			}
			return rt.Out.Success(data, listSummary(data, "user"), []output.Breadcrumb{crumb("companies", "symbol companies list")}, pag)
		},
	}
	cmd.Flags().StringVar(&typ, "type", "", "User type")
	return cmd
}
