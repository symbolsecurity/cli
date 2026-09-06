package output

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"os"
	"sort"
	"strings"

	"github.com/itchyny/gojq"
	"golang.org/x/term"
)

type Breadcrumb struct {
	Action string `json:"action"`
	Cmd    string `json:"cmd"`
}

type Pagination struct {
	Page               int `json:"page"`
	PerPage            int `json:"per_page"`
	Offset             int `json:"offset,omitempty"`
	TotalEntriesSize   int `json:"total_entries_size"`
	CurrentEntriesSize int `json:"current_entries_size"`
	TotalPages         int `json:"total_pages"`
}

type Envelope struct {
	OK          bool         `json:"ok"`
	Data        any          `json:"data,omitempty"`
	Summary     string       `json:"summary,omitempty"`
	Breadcrumbs []Breadcrumb `json:"breadcrumbs,omitempty"`
	Pagination  *Pagination  `json:"pagination,omitempty"`
	Error       string       `json:"error,omitempty"`
	Code        string       `json:"code,omitempty"`
	Retryable   bool         `json:"retryable,omitempty"`
	Hint        string       `json:"hint,omitempty"`
}

type Error struct {
	Message   string
	Code      string
	Retryable bool
	Hint      string
	Status    int
}

func (e *Error) Error() string {
	if e == nil {
		return ""
	}
	return e.Message
}

func AuthError(hint string) *Error {
	if hint == "" {
		hint = "Run: symbol auth login"
	}
	return &Error{Message: "Unauthorized", Code: "auth_error", Hint: hint}
}

func Usage(msg, hint string) *Error {
	return &Error{Message: msg, Code: "usage_error", Hint: hint}
}

type Options struct {
	JSON       bool
	Agent      bool
	MD         bool
	JQ         string
	Full       bool
	ForceJSON  bool
	IsTerminal bool
}

type Printer struct {
	Out  io.Writer
	Err  io.Writer
	Opts Options
}

func NewPrinter(out, err io.Writer, opts Options) *Printer {
	if out == nil {
		out = os.Stdout
	}
	if err == nil {
		err = os.Stderr
	}
	return &Printer{Out: out, Err: err, Opts: opts}
}

func IsTerminal(w io.Writer) bool {
	f, ok := w.(*os.File)
	if !ok {
		return false
	}
	return term.IsTerminal(int(f.Fd()))
}

func (p *Printer) jsonMode() bool {
	if p.Opts.JSON || p.Opts.Agent || p.Opts.ForceJSON {
		return true
	}
	if p.Opts.MD {
		return false
	}
	return !p.Opts.IsTerminal
}

func (p *Printer) Success(data any, summary string, crumbs []Breadcrumb, pag *Pagination) error {
	if !p.Opts.Full {
		data = Redact(data)
	}
	if p.Opts.JQ != "" {
		var err error
		data, err = applyJQ(p.Opts.JQ, data)
		if err != nil {
			return p.Fail(&Error{Message: err.Error(), Code: "usage_error", Hint: "Check --jq expression"})
		}
	}
	if p.Opts.Agent {
		return p.writeJSON(data)
	}
	env := Envelope{OK: true, Data: data, Summary: summary, Breadcrumbs: crumbs, Pagination: pag}
	if p.jsonMode() {
		return p.writeJSON(env)
	}
	return p.writeHuman(env)
}

func (p *Printer) Fail(err error) error {
	e := &Error{Message: err.Error(), Code: "api_error"}
	if ae, ok := err.(*Error); ok {
		e = ae
	}
	env := Envelope{
		OK:        false,
		Error:     e.Message,
		Code:      e.Code,
		Retryable: e.Retryable,
		Hint:      e.Hint,
	}
	if p.Opts.Agent || p.jsonMode() {
		if werr := p.writeJSON(env); werr != nil {
			return werr
		}
		return e
	}
	msg := e.Message
	if e.Hint != "" {
		msg += "\n" + e.Hint
	}
	fmt.Fprintln(p.Out, msg)
	return e
}

func (p *Printer) writeJSON(v any) error {
	enc := json.NewEncoder(p.Out)
	enc.SetEscapeHTML(false)
	return enc.Encode(v)
}

func (p *Printer) writeHuman(env Envelope) error {
	if env.Summary != "" {
		fmt.Fprintln(p.Out, env.Summary)
	}
	if err := writeTable(p.Out, env.Data); err != nil {
		return err
	}
	if env.Pagination != nil && env.Pagination.TotalPages > 1 {
		fmt.Fprintf(p.Out, "page %d/%d (%d total)\n", env.Pagination.Page, env.Pagination.TotalPages, env.Pagination.TotalEntriesSize)
	}
	if len(env.Breadcrumbs) > 0 {
		fmt.Fprintln(p.Out, "next:")
		for _, b := range env.Breadcrumbs {
			fmt.Fprintf(p.Out, "  %s: %s\n", b.Action, b.Cmd)
		}
	}
	return nil
}

func writeTable(w io.Writer, data any) error {
	items := asObjects(data)
	if len(items) == 0 {
		if data == nil {
			return nil
		}
		b, err := json.MarshalIndent(data, "", "  ")
		if err != nil {
			return err
		}
		fmt.Fprintln(w, string(b))
		return nil
	}
	keys := tableKeys(items)
	if len(keys) == 0 {
		b, err := json.MarshalIndent(data, "", "  ")
		if err != nil {
			return err
		}
		fmt.Fprintln(w, string(b))
		return nil
	}
	widths := make([]int, len(keys))
	for i, k := range keys {
		widths[i] = len(k)
	}
	rows := make([][]string, 0, len(items))
	for _, item := range items {
		row := make([]string, len(keys))
		for i, k := range keys {
			row[i] = cell(item[k])
			if len(row[i]) > widths[i] {
				widths[i] = len(row[i])
			}
		}
		rows = append(rows, row)
	}
	printRow(w, keys, widths)
	sep := make([]string, len(keys))
	for i, n := range widths {
		sep[i] = strings.Repeat("-", n)
	}
	printRow(w, sep, widths)
	for _, row := range rows {
		printRow(w, row, widths)
	}
	return nil
}

func printRow(w io.Writer, cols []string, widths []int) {
	parts := make([]string, len(cols))
	for i, c := range cols {
		parts[i] = fmt.Sprintf("%-*s", widths[i], c)
	}
	fmt.Fprintln(w, strings.Join(parts, "  "))
}

func asObjects(data any) []map[string]any {
	switch v := data.(type) {
	case []any:
		out := make([]map[string]any, 0, len(v))
		for _, item := range v {
			m, ok := item.(map[string]any)
			if !ok {
				return nil
			}
			out = append(out, m)
		}
		return out
	case []map[string]any:
		return v
	case map[string]any:
		if inner, ok := v["items"]; ok {
			return asObjects(inner)
		}
		return []map[string]any{v}
	default:
		b, err := json.Marshal(data)
		if err != nil {
			return nil
		}
		var arr []map[string]any
		if json.Unmarshal(b, &arr) == nil && len(arr) > 0 {
			return arr
		}
		var obj map[string]any
		if json.Unmarshal(b, &obj) == nil {
			if inner, ok := obj["items"]; ok {
				return asObjects(inner)
			}
			return []map[string]any{obj}
		}
		return nil
	}
}

func tableKeys(items []map[string]any) []string {
	pref := []string{"id", "email", "name", "firstName", "lastName", "status", "title", "feature"}
	seen := map[string]bool{}
	var keys []string
	for _, k := range pref {
		for _, item := range items {
			if _, ok := item[k]; ok && !seen[k] {
				seen[k] = true
				keys = append(keys, k)
			}
		}
	}
	var extra []string
	for _, item := range items {
		for k, v := range item {
			if seen[k] {
				continue
			}
			switch v.(type) {
			case map[string]any, []any:
				continue
			}
			seen[k] = true
			extra = append(extra, k)
		}
	}
	sort.Strings(extra)
	if len(extra) > 6 {
		extra = extra[:6]
	}
	return append(keys, extra...)
}

func cell(v any) string {
	if v == nil {
		return ""
	}
	switch t := v.(type) {
	case string:
		if len(t) > 48 {
			return t[:45] + "..."
		}
		return t
	case bool:
		if t {
			return "true"
		}
		return "false"
	default:
		b, err := json.Marshal(t)
		if err != nil {
			return fmt.Sprint(t)
		}
		s := string(b)
		if len(s) > 48 {
			return s[:45] + "..."
		}
		return s
	}
}

func applyJQ(expr string, data any) (any, error) {
	q, err := gojq.Parse(expr)
	if err != nil {
		return nil, err
	}
	code, err := gojq.Compile(q)
	if err != nil {
		return nil, err
	}
	var in any
	b, err := json.Marshal(data)
	if err != nil {
		return nil, err
	}
	dec := json.NewDecoder(bytes.NewReader(b))
	dec.UseNumber()
	if err := dec.Decode(&in); err != nil {
		return nil, err
	}
	iter := code.Run(in)
	var out []any
	for {
		v, ok := iter.Next()
		if !ok {
			break
		}
		if err, ok := v.(error); ok {
			return nil, err
		}
		out = append(out, v)
	}
	if len(out) == 1 {
		return out[0], nil
	}
	return out, nil
}
