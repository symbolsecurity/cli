package commands

import (
	"net/http"
	"net/url"
	"strings"

	"github.com/spf13/cobra"

	"github.com/symbolsecurity/cli/internal/client"
	"github.com/symbolsecurity/cli/internal/output"
)

func (rt *Runtime) list(cmd *cobra.Command, path string, query url.Values, word string, crumbs []output.Breadcrumb) error {
	data, pag, err := rt.Client.List(rt.ctx(cmd), path, query, rt.page())
	if err != nil {
		return rt.Out.Fail(err)
	}
	return rt.Out.Success(data, plural(client.Count(data), word), crumbs, pag)
}

func (rt *Runtime) get(cmd *cobra.Command, path string, query url.Values, summary string, crumbs []output.Breadcrumb) error {
	res, err := rt.Client.Get(rt.ctx(cmd), path, query)
	if err != nil {
		return rt.Out.Fail(err)
	}
	return rt.Out.Success(client.AsData(res.Body), summary, crumbs, res.Pagination)
}

func (rt *Runtime) mutate(cmd *cobra.Command, method, path string, body any, summary string, crumbs []output.Breadcrumb) error {
	res, err := rt.Client.Send(rt.ctx(cmd), method, path, body)
	if err != nil {
		return rt.Out.Fail(err)
	}
	data := client.AsData(res.Body)
	if (method == http.MethodDelete || method == http.MethodPut) && len(strings.TrimSpace(string(res.Body))) == 0 {
		data = map[string]any{"ok": true}
	}
	return rt.Out.Success(data, summary, crumbs, nil)
}

func (rt *Runtime) companyPath(companyPath string) string {
	return rt.Client.Path(companyPath)
}

func csv(v string) []string {
	if v == "" {
		return nil
	}
	parts := strings.Split(v, ",")
	out := make([]string, 0, len(parts))
	for _, p := range parts {
		p = strings.TrimSpace(p)
		if p != "" {
			out = append(out, p)
		}
	}
	return out
}

func qset(q url.Values, k, v string) {
	if v != "" {
		q.Set(k, v)
	}
}
