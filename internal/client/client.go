package client

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"mime/multipart"
	"net/http"
	"net/url"
	"strconv"
	"strings"
	"time"

	"github.com/symbolsecurity/cli/internal/auth"
	"github.com/symbolsecurity/cli/internal/output"
	"github.com/symbolsecurity/cli/internal/version"
)

const (
	defaultTimeout = 30 * time.Second
	maxBodyBytes   = 32 << 20
	maxPerPage     = 500
	maxAllItems    = 10000
	maxAllPages    = 200
)

type Token struct {
	AccessToken  string `json:"accessToken"`
	RefreshToken string `json:"refreshToken"`
	TokenType    string `json:"tokenType"`
}

type APIError struct {
	Message   string `json:"message"`
	Reason    string `json:"reason"`
	RequestID string `json:"request_id"`
}

type PageOpts struct {
	Page    int
	PerPage int
	All     bool
	Limit   int
}

type Result struct {
	Status     int
	Body       json.RawMessage
	Pagination *output.Pagination
}

type Client struct {
	BaseURL   string
	HTTP      *http.Client
	Store     *auth.Store
	CompanyID string
	DryRun    bool
	Log       io.Writer
	creds     auth.Credentials
	scoped    *Token
}

func DefaultHTTP() *http.Client {
	return &http.Client{Timeout: defaultTimeout}
}

func New(baseURL string, store *auth.Store, companyID string, httpClient *http.Client) *Client {
	if httpClient == nil {
		httpClient = &http.Client{Timeout: defaultTimeout}
	}
	if httpClient.Timeout == 0 {
		httpClient.Timeout = defaultTimeout
	}
	return &Client{
		BaseURL:   strings.TrimRight(baseURL, "/"),
		HTTP:      httpClient,
		Store:     store,
		CompanyID: companyID,
	}
}

func (c *Client) load() error {
	if c.creds.AccessToken != "" || c.creds.APIKey != "" {
		return nil
	}
	if c.Store == nil {
		return auth.ErrNotLoggedIn
	}
	creds, err := c.Store.Load()
	if err != nil {
		return err
	}
	c.creds = creds
	return nil
}

func (c *Client) Login(ctx context.Context, apiKey string) (Token, error) {
	tok, err := c.exchange(ctx, apiKey, "")
	if err != nil {
		return Token{}, err
	}
	c.creds = auth.Credentials{
		APIKey:       apiKey,
		AccessToken:  tok.AccessToken,
		RefreshToken: tok.RefreshToken,
		TokenType:    tok.TokenType,
	}
	if c.Store != nil {
		if err := c.Store.Save(c.creds); err != nil {
			return Token{}, err
		}
	}
	return tok, nil
}

func (c *Client) Refresh(ctx context.Context) (Token, error) {
	if err := c.load(); err != nil {
		return Token{}, err
	}
	tok, err := c.refresh(ctx)
	if err != nil {
		return Token{}, err
	}
	return tok, nil
}

func (c *Client) exchange(ctx context.Context, apiKey, companyID string) (Token, error) {
	q := url.Values{}
	if companyID != "" {
		q.Set("company_id", companyID)
	}
	res, err := c.do(ctx, http.MethodPost, "/auth/access/", q, nil, "Bearer "+apiKey, false)
	if err != nil {
		return Token{}, err
	}
	if res.Status >= 400 {
		return Token{}, statusError(res.Status, res.Body, false)
	}
	var tok Token
	if err := json.Unmarshal(res.Body, &tok); err != nil {
		return Token{}, err
	}
	return tok, nil
}

func (c *Client) refresh(ctx context.Context) (Token, error) {
	run := func() (Token, error) {
		if c.Store != nil {
			creds, err := c.Store.Load()
			if err == nil {
				c.creds = creds
			}
		}
		if c.creds.RefreshToken == "" {
			return Token{}, output.AuthError("Run: symbol auth login")
		}
		body := Token{RefreshToken: c.creds.RefreshToken}
		res, err := c.do(ctx, http.MethodPost, "/auth/refresh/", nil, body, "", false)
		if err != nil {
			return Token{}, err
		}
		if res.Status >= 400 {
			return Token{}, statusError(res.Status, res.Body, false)
		}
		var tok Token
		if err := json.Unmarshal(res.Body, &tok); err != nil {
			return Token{}, err
		}
		c.creds.AccessToken = tok.AccessToken
		c.creds.RefreshToken = tok.RefreshToken
		if tok.TokenType != "" {
			c.creds.TokenType = tok.TokenType
		}
		if c.Store != nil {
			if err := c.Store.Save(c.creds); err != nil {
				return Token{}, err
			}
		}
		return tok, nil
	}
	if c.Store == nil {
		return run()
	}
	var tok Token
	err := c.Store.WithLock(func() error {
		var e error
		tok, e = run()
		return e
	})
	return tok, err
}

func (c *Client) ensureScoped(ctx context.Context) error {
	if c.CompanyID == "" || c.scoped != nil {
		return nil
	}
	if err := c.load(); err != nil {
		return err
	}
	if c.creds.APIKey == "" {
		return nil
	}
	tok, err := c.exchange(ctx, c.creds.APIKey, c.CompanyID)
	if err != nil {
		return err
	}
	c.scoped = &tok
	return nil
}

func (c *Client) Path(companyPath string) string {
	return companyPath
}

func (c *Client) resolve(path string, msp bool) string {
	if msp || c.CompanyID == "" || c.scoped != nil {
		return path
	}
	if strings.HasPrefix(path, "/msp/") {
		return path
	}
	return "/msp/companies/" + url.PathEscape(c.CompanyID) + path
}

func (c *Client) Get(ctx context.Context, path string, query url.Values) (Result, error) {
	return c.Call(ctx, http.MethodGet, path, query, nil, false)
}

func (c *Client) Send(ctx context.Context, method, path string, body any) (Result, error) {
	return c.Call(ctx, method, path, nil, body, false)
}

func (c *Client) Call(ctx context.Context, method, path string, query url.Values, body any, msp bool) (Result, error) {
	if err := c.load(); err != nil {
		if errors.Is(err, auth.ErrNotLoggedIn) {
			return Result{}, output.AuthError("Run: symbol auth login")
		}
		return Result{}, err
	}
	if err := checkPath(path); err != nil {
		return Result{}, err
	}
	if !msp {
		if err := c.ensureScoped(ctx); err != nil {
			return Result{}, err
		}
		path = c.resolve(path, msp)
	}
	if c.DryRun && !idempotent(method) {
		payload := map[string]any{"dry_run": true, "method": method, "path": path}
		if body != nil {
			payload["body"] = body
		}
		b, err := json.Marshal(payload)
		if err != nil {
			return Result{}, err
		}
		return Result{Status: http.StatusOK, Body: b}, nil
	}
	token := c.creds.AccessToken
	if !msp && c.scoped != nil {
		token = c.scoped.AccessToken
	}
	res, err := c.doRetry(ctx, method, path, query, body, "Bearer "+token)
	if err != nil {
		return Result{}, err
	}
	if res.Status == http.StatusUnauthorized {
		if !msp && c.scoped != nil && c.creds.APIKey != "" {
			c.scoped = nil
			if err := c.ensureScoped(ctx); err == nil && c.scoped != nil {
				res, err = c.doRetry(ctx, method, path, query, body, "Bearer "+c.scoped.AccessToken)
				if err != nil {
					return Result{}, err
				}
			}
		}
		if res.Status == http.StatusUnauthorized {
			if _, rerr := c.refresh(ctx); rerr != nil {
				return Result{}, output.AuthError("Run: symbol auth login")
			}
			token = c.creds.AccessToken
			if !msp && c.scoped != nil {
				c.scoped = nil
				_ = c.ensureScoped(ctx)
				if c.scoped != nil {
					token = c.scoped.AccessToken
				}
			}
			res, err = c.doRetry(ctx, method, path, query, body, "Bearer "+token)
			if err != nil {
				return Result{}, err
			}
		}
	}
	if res.Status == http.StatusUnauthorized {
		return Result{}, output.AuthError("Run: symbol auth login")
	}
	if res.Status >= 400 {
		retryable := res.Status >= 500
		return Result{}, statusError(res.Status, res.Body, retryable)
	}
	return res, nil
}

func (c *Client) List(ctx context.Context, path string, query url.Values, opts PageOpts) (any, *output.Pagination, error) {
	return c.list(ctx, path, query, opts, false)
}

func (c *Client) ListMSP(ctx context.Context, path string, query url.Values, opts PageOpts) (any, *output.Pagination, error) {
	return c.list(ctx, path, query, opts, true)
}

func (c *Client) list(ctx context.Context, path string, query url.Values, opts PageOpts, msp bool) (any, *output.Pagination, error) {
	if opts.All && opts.Page > 0 {
		return nil, nil, output.Usage("--all cannot be combined with --page", "Use --all or --page, not both")
	}
	if opts.Limit > 0 && opts.Page > 0 {
		return nil, nil, output.Usage("--limit cannot be combined with --page", "Use --limit or --page, not both")
	}
	if query == nil {
		query = url.Values{}
	}
	page := opts.Page
	if page < 1 {
		page = 1
	}
	per := opts.PerPage
	if per < 1 {
		per = 50
	}
	if per > maxPerPage {
		per = maxPerPage
	}
	walk := opts.All || opts.Limit > 0
	limit := opts.Limit
	if opts.All && limit == 0 {
		limit = maxAllItems
	}
	var items []any
	var last *output.Pagination
	pages := 0
	for {
		q := cloneValues(query)
		q.Set("page", strconv.Itoa(page))
		q.Set("per_page", strconv.Itoa(per))
		res, err := c.Call(ctx, http.MethodGet, path, q, nil, msp)
		if err != nil {
			return nil, nil, err
		}
		last = res.Pagination
		chunk, obj := decodeList(res.Body)
		if !walk {
			n := len(chunk)
			if obj != nil {
				n = 1
			}
			if last == nil {
				last = &output.Pagination{Page: page, PerPage: per, CurrentEntriesSize: n}
			}
			if obj != nil {
				return obj, last, nil
			}
			return chunk, last, nil
		}
		if obj != nil {
			return obj, last, nil
		}
		items = append(items, chunk...)
		pages++
		if limit > 0 && len(items) >= limit {
			items = items[:limit]
			if last != nil {
				last.CurrentEntriesSize = len(items)
			}
			break
		}
		if last == nil || page >= last.TotalPages || len(chunk) == 0 {
			break
		}
		if pages >= maxAllPages {
			return nil, last, output.Usage("--all exceeded page cap", "Narrow filters or use --limit")
		}
		page++
	}
	if last != nil {
		last.CurrentEntriesSize = len(items)
	}
	return items, last, nil
}

func (c *Client) Multipart(ctx context.Context, method, path string, fields map[string][]string) (Result, error) {
	if err := c.load(); err != nil {
		if errors.Is(err, auth.ErrNotLoggedIn) {
			return Result{}, output.AuthError("Run: symbol auth login")
		}
		return Result{}, err
	}
	if err := checkPath(path); err != nil {
		return Result{}, err
	}
	if c.DryRun {
		b, err := json.Marshal(map[string]any{"dry_run": true, "method": method, "path": path, "fields": fields})
		if err != nil {
			return Result{}, err
		}
		return Result{Status: http.StatusOK, Body: b}, nil
	}
	res, err := c.multipartOnce(ctx, method, path, fields, c.creds.AccessToken)
	if err != nil {
		return Result{}, err
	}
	if res.Status == http.StatusUnauthorized {
		if _, rerr := c.refresh(ctx); rerr != nil {
			return Result{}, output.AuthError("Run: symbol auth login")
		}
		res, err = c.multipartOnce(ctx, method, path, fields, c.creds.AccessToken)
		if err != nil {
			return Result{}, err
		}
	}
	if res.Status == http.StatusUnauthorized {
		return Result{}, output.AuthError("Run: symbol auth login")
	}
	if res.Status >= 400 {
		return Result{}, statusError(res.Status, res.Body, res.Status >= 500)
	}
	return res, nil
}

func (c *Client) multipartOnce(ctx context.Context, method, path string, fields map[string][]string, token string) (Result, error) {
	var buf bytes.Buffer
	w := multipart.NewWriter(&buf)
	for k, vs := range fields {
		for _, v := range vs {
			if err := w.WriteField(k, v); err != nil {
				return Result{}, err
			}
		}
	}
	if err := w.Close(); err != nil {
		return Result{}, err
	}
	u := c.BaseURL + path
	req, err := http.NewRequestWithContext(ctx, method, u, &buf)
	if err != nil {
		return Result{}, err
	}
	req.Header.Set("Content-Type", w.FormDataContentType())
	req.Header.Set("Accept", "application/json")
	req.Header.Set("User-Agent", userAgent())
	req.Header.Set("Authorization", "Bearer "+token)
	res, err := c.HTTP.Do(req)
	if err != nil {
		return Result{}, &output.Error{Message: err.Error(), Code: "network_error", Retryable: true}
	}
	defer res.Body.Close()
	b, err := readBody(res.Body)
	if err != nil {
		return Result{}, err
	}
	c.logf("%s %s %d", method, path, res.StatusCode)
	return Result{Status: res.StatusCode, Body: json.RawMessage(bytes.TrimSpace(b))}, nil
}

func (c *Client) Ping(ctx context.Context) error {
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, c.BaseURL+"/", nil)
	if err != nil {
		return err
	}
	res, err := c.HTTP.Do(req)
	if err != nil {
		return err
	}
	res.Body.Close()
	return nil
}

func (c *Client) doRetry(ctx context.Context, method, path string, query url.Values, body any, authz string) (Result, error) {
	res, err := c.do(ctx, method, path, query, body, authz, true)
	if !idempotent(method) {
		return res, err
	}
	if err != nil {
		res, err = c.do(ctx, method, path, query, body, authz, true)
		if err != nil {
			return Result{}, err
		}
	}
	if res.Status >= 500 {
		res2, err2 := c.do(ctx, method, path, query, body, authz, true)
		if err2 == nil {
			return res2, nil
		}
	}
	return res, nil
}

func (c *Client) do(ctx context.Context, method, path string, query url.Values, body any, authz string, retryable bool) (Result, error) {
	u, err := url.Parse(c.BaseURL + path)
	if err != nil {
		return Result{}, err
	}
	if query != nil {
		q := u.Query()
		for k, vs := range query {
			for _, v := range vs {
				q.Add(k, v)
			}
		}
		u.RawQuery = q.Encode()
	}
	var rdr io.Reader
	if body != nil {
		b, err := json.Marshal(body)
		if err != nil {
			return Result{}, err
		}
		rdr = bytes.NewReader(b)
	}
	req, err := http.NewRequestWithContext(ctx, method, u.String(), rdr)
	if err != nil {
		return Result{}, err
	}
	req.Header.Set("Accept", "application/json")
	req.Header.Set("User-Agent", userAgent())
	if body != nil {
		req.Header.Set("Content-Type", "application/json")
	}
	if authz != "" {
		req.Header.Set("Authorization", authz)
	}
	res, err := c.HTTP.Do(req)
	if err != nil {
		return Result{}, &output.Error{Message: err.Error(), Code: "network_error", Retryable: retryable}
	}
	defer res.Body.Close()
	b, err := readBody(res.Body)
	if err != nil {
		return Result{}, err
	}
	c.logf("%s %s %d", method, u.Path, res.StatusCode)
	out := Result{Status: res.StatusCode, Body: json.RawMessage(bytes.TrimSpace(b))}
	if ph := res.Header.Get("X-Pagination"); ph != "" {
		var p output.Pagination
		if json.Unmarshal([]byte(ph), &p) == nil {
			out.Pagination = &p
		}
	}
	return out, nil
}

func statusError(status int, body []byte, retryable bool) error {
	var ae APIError
	_ = json.Unmarshal(body, &ae)
	msg := ae.Message
	if ae.Reason != "" && (msg == "" || msg != ae.Reason) {
		if msg == "" {
			msg = ae.Reason
		} else {
			msg = msg + ": " + ae.Reason
		}
	}
	if msg == "" {
		msg = strings.TrimSpace(string(body))
	}
	if msg == "" {
		msg = http.StatusText(status)
	}
	code := "api_error"
	hint := ""
	switch status {
	case http.StatusUnauthorized:
		code = "auth_error"
		hint = "Run: symbol auth login"
	case http.StatusNotFound:
		code = "not_found"
	case http.StatusUnprocessableEntity, http.StatusBadRequest:
		code = "validation_error"
	}
	if status >= 500 {
		code = "api_error"
		retryable = true
	}
	return &output.Error{Message: msg, Code: code, Retryable: retryable, Hint: hint, Status: status, RequestID: ae.RequestID}
}

func decodeList(body json.RawMessage) ([]any, any) {
	trim := bytes.TrimSpace(body)
	if len(trim) == 0 || bytes.Equal(trim, []byte("null")) {
		return []any{}, nil
	}
	if trim[0] == '[' {
		var arr []any
		if err := json.Unmarshal(trim, &arr); err != nil {
			return []any{}, json.RawMessage(trim)
		}
		return arr, nil
	}
	var obj map[string]any
	if err := json.Unmarshal(trim, &obj); err != nil {
		return []any{}, json.RawMessage(trim)
	}
	if inner, ok := obj["items"]; ok {
		if arr, ok := inner.([]any); ok {
			return arr, nil
		}
	}
	return nil, obj
}

func cloneValues(v url.Values) url.Values {
	out := url.Values{}
	for k, vs := range v {
		out[k] = append([]string{}, vs...)
	}
	return out
}

func AsData(body json.RawMessage) any {
	trim := bytes.TrimSpace(body)
	if len(trim) == 0 || bytes.Equal(trim, []byte("null")) {
		return map[string]any{}
	}
	var v any
	if err := json.Unmarshal(trim, &v); err != nil {
		return string(trim)
	}
	return v
}

func Count(data any) (int, bool) {
	switch v := data.(type) {
	case []any:
		return len(v), true
	case []map[string]any:
		return len(v), true
	default:
		return 0, false
	}
}

func userAgent() string {
	return "symbol-cli/" + version.Version
}

func idempotent(method string) bool {
	switch method {
	case http.MethodGet, http.MethodHead, http.MethodOptions:
		return true
	default:
		return false
	}
}

func checkPath(path string) error {
	if strings.Contains(path, "..") {
		return output.Usage("invalid path", "Path cannot contain ..")
	}
	return nil
}

func readBody(r io.Reader) ([]byte, error) {
	b, err := io.ReadAll(io.LimitReader(r, maxBodyBytes+1))
	if err != nil {
		return nil, err
	}
	if len(b) > maxBodyBytes {
		return nil, &output.Error{Message: "response too large", Code: "api_error", Hint: "Narrow filters or raise paging limits"}
	}
	return b, nil
}

func (c *Client) logf(format string, args ...any) {
	if c.Log == nil {
		return
	}
	fmt.Fprintf(c.Log, format+"\n", args...)
}
