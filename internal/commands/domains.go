package commands

import (
	"net/http"
	"net/url"

	"github.com/spf13/cobra"

	"github.com/symbolsecurity/cli/internal/output"
)

func (rt *Runtime) domainsCmd() *cobra.Command {
	cmd := &cobra.Command{Use: "domains", Short: "Company domains"}
	cmd.AddCommand(rt.domainsListCmd(), rt.domainsCreateCmd(), rt.domainsThreatsCmd())
	return cmd
}

func (rt *Runtime) domainsListCmd() *cobra.Command {
	var domain string
	cmd := &cobra.Command{
		Use:   "list",
		Short: "List domains",
		RunE: func(cmd *cobra.Command, args []string) error {
			q := url.Values{}
			qset(q, "domain", domain)
			return rt.list(cmd, rt.companyPath("/domains"), q, "domain", []output.Breadcrumb{
				crumb("create", "symbol domains create --domain example.com"),
				crumb("threats", "symbol domains threats"),
			})
		},
	}
	cmd.Flags().StringVar(&domain, "domain", "", "Domain filter")
	return cmd
}

func (rt *Runtime) domainsCreateCmd() *cobra.Command {
	var domain string
	cmd := &cobra.Command{
		Use:   "create",
		Short: "Create a domain",
		RunE: func(cmd *cobra.Command, args []string) error {
			if domain == "" {
				return rt.Out.Fail(output.Usage("domain is required", "Pass --domain"))
			}
			q := url.Values{}
			q.Set("domain", domain)
			res, err := rt.Client.Call(rt.ctx(cmd), http.MethodPost, rt.companyPath("/domains"), q, map[string]string{"domain": domain}, false)
			if err != nil {
				return rt.Out.Fail(err)
			}
			return rt.Out.Success(clientData(res.Body), "domain created", []output.Breadcrumb{crumb("list", "symbol domains list")}, nil)
		},
	}
	cmd.Flags().StringVar(&domain, "domain", "", "Domain name")
	_ = cmd.MarkFlagRequired("domain")
	return cmd
}

func (rt *Runtime) domainsThreatsCmd() *cobra.Command {
	var domain string
	cmd := &cobra.Command{
		Use:   "threats",
		Short: "List domain threats",
		RunE: func(cmd *cobra.Command, args []string) error {
			q := url.Values{}
			qset(q, "domain", domain)
			return rt.list(cmd, rt.companyPath("/domains/threats"), q, "alert", []output.Breadcrumb{
				crumb("list", "symbol domains list"),
			})
		},
	}
	cmd.Flags().StringVar(&domain, "domain", "", "Domain filter")
	return cmd
}
