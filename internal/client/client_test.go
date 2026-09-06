package client

import (
	"context"
	"encoding/json"
	"io"
	"net/http"
	"net/http/httptest"
	"sync/atomic"
	"testing"

	"github.com/symbolsecurity/cli/internal/auth"
	"github.com/symbolsecurity/cli/internal/output"
)

func TestListPagination(t *testing.T) {
	var pages atomic.Int32
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		n := pages.Add(1)
		pag := output.Pagination{Page: int(n), PerPage: 1, TotalPages: 2, TotalEntriesSize: 2, CurrentEntriesSize: 1}
		b, _ := json.Marshal(pag)
		w.Header().Set("X-Pagination", string(b))
		if n == 1 {
			_ = json.NewEncoder(w).Encode([]map[string]string{{"id": "1"}})
			return
		}
		_ = json.NewEncoder(w).Encode([]map[string]string{{"id": "2"}})
	}))
	t.Cleanup(srv.Close)
	c := newTestClient(t, srv.URL)
	data, pag, err := c.List(context.Background(), "/users/", nil, PageOpts{All: true, PerPage: 1})
	if err != nil {
		t.Fatal(err)
	}
	arr := data.([]any)
	if len(arr) != 2 || pag.TotalPages != 2 {
		t.Fatalf("%v %+v", data, pag)
	}
}

func TestRefreshOn401(t *testing.T) {
	var n atomic.Int32
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		switch r.URL.Path {
		case "/auth/refresh/":
			_ = json.NewEncoder(w).Encode(Token{AccessToken: "new", RefreshToken: "r2", TokenType: "bearer"})
		case "/users/":
			if n.Add(1) == 1 {
				w.WriteHeader(http.StatusUnauthorized)
				_, _ = io.WriteString(w, `{"message":"Unauthorized"}`)
				return
			}
			if got := r.Header.Get("Authorization"); got != "Bearer new" {
				t.Errorf("auth %s", got)
			}
			_ = json.NewEncoder(w).Encode([]any{})
		default:
			w.WriteHeader(404)
		}
	}))
	t.Cleanup(srv.Close)
	c := newTestClient(t, srv.URL)
	_, err := c.Get(context.Background(), "/users/", nil)
	if err != nil {
		t.Fatal(err)
	}
	creds, _ := c.Store.Load()
	if creds.AccessToken != "new" || creds.RefreshToken != "r2" {
		t.Fatalf("not persisted: %+v", creds)
	}
}

func TestNoRetryOnPost(t *testing.T) {
	var n atomic.Int32
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		n.Add(1)
		w.WriteHeader(500)
		_, _ = io.WriteString(w, `{"message":"boom"}`)
	}))
	t.Cleanup(srv.Close)
	c := newTestClient(t, srv.URL)
	_, err := c.Send(context.Background(), http.MethodPost, "/users/", map[string]string{"email": "a@b.c"})
	if err == nil {
		t.Fatal("expected error")
	}
	if n.Load() != 1 {
		t.Fatalf("retried POST: %d", n.Load())
	}
}

func TestRetryOn500(t *testing.T) {
	var n atomic.Int32
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if n.Add(1) == 1 {
			w.WriteHeader(500)
			_, _ = io.WriteString(w, `{"message":"boom"}`)
			return
		}
		_ = json.NewEncoder(w).Encode([]any{})
	}))
	t.Cleanup(srv.Close)
	c := newTestClient(t, srv.URL)
	_, err := c.Get(context.Background(), "/users/", nil)
	if err != nil {
		t.Fatal(err)
	}
}

func TestScopedAuthFailsClosed(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path == "/auth/access/" {
			w.WriteHeader(http.StatusNotFound)
			_, _ = io.WriteString(w, `{"message":"company not found"}`)
			return
		}
		t.Fatalf("unexpected %s", r.URL.Path)
	}))
	t.Cleanup(srv.Close)
	c := newTestClient(t, srv.URL)
	c.CompanyID = "11111111-1111-1111-1111-111111111111"
	_, err := c.Get(context.Background(), "/users/", nil)
	if err == nil {
		t.Fatal("expected error")
	}
}

func TestLoginExchange(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/auth/access/" {
			t.Fatalf("path %s", r.URL.Path)
		}
		if r.Header.Get("Authorization") != "Bearer secret" {
			t.Fatalf("auth %s", r.Header.Get("Authorization"))
		}
		_ = json.NewEncoder(w).Encode(Token{AccessToken: "a", RefreshToken: "r", TokenType: "bearer"})
	}))
	t.Cleanup(srv.Close)
	c := newTestClient(t, srv.URL)
	c.creds = auth.Credentials{}
	tok, err := c.Login(context.Background(), "secret")
	if err != nil || tok.AccessToken != "a" {
		t.Fatalf("%+v %v", tok, err)
	}
}

func newTestClient(t *testing.T, base string) *Client {
	t.Helper()
	store := auth.NewStore(t.TempDir(), "default", auth.NewMemoryKeyring())
	_ = store.Save(auth.Credentials{APIKey: "k", AccessToken: "old", RefreshToken: "r1"})
	return New(base, store, "", &http.Client{})
}
