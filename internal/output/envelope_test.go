package output

import (
	"bytes"
	"encoding/json"
	"strings"
	"testing"
)

func TestSuccessJSONEnvelope(t *testing.T) {
	var buf bytes.Buffer
	p := NewPrinter(&buf, &buf, Options{JSON: true})
	err := p.Success([]any{map[string]any{"id": "1", "email": "a@b.c"}}, "1 user", []Breadcrumb{{Action: "create", Cmd: "symbol users create"}}, &Pagination{Page: 1, PerPage: 50, TotalPages: 1, TotalEntriesSize: 1})
	if err != nil {
		t.Fatal(err)
	}
	var env Envelope
	if err := json.Unmarshal(buf.Bytes(), &env); err != nil {
		t.Fatal(err)
	}
	if !env.OK || env.Summary != "1 user" || len(env.Breadcrumbs) != 1 {
		t.Fatalf("unexpected envelope: %+v", env)
	}
}

func TestAgentRawData(t *testing.T) {
	var buf bytes.Buffer
	p := NewPrinter(&buf, &buf, Options{Agent: true})
	if err := p.Success(map[string]any{"id": "1"}, "1 user", nil, nil); err != nil {
		t.Fatal(err)
	}
	if strings.Contains(buf.String(), `"ok"`) {
		t.Fatalf("agent mode should not wrap: %s", buf.String())
	}
}

func TestFailEnvelope(t *testing.T) {
	var buf bytes.Buffer
	p := NewPrinter(&buf, &buf, Options{JSON: true})
	err := p.Fail(AuthError("Run: symbol auth login"))
	if err == nil {
		t.Fatal("expected error")
	}
	var env Envelope
	if json.Unmarshal(buf.Bytes(), &env) != nil || env.OK || env.Code != "auth_error" || !strings.Contains(env.Hint, "auth login") {
		t.Fatalf("unexpected error envelope: %s", buf.String())
	}
}

func TestJQ(t *testing.T) {
	var buf bytes.Buffer
	p := NewPrinter(&buf, &buf, Options{Agent: true, JQ: ".[0].email"})
	if err := p.Success([]any{map[string]any{"email": "a@b.c"}}, "", nil, nil); err != nil {
		t.Fatal(err)
	}
	if strings.TrimSpace(buf.String()) != `"a@b.c"` {
		t.Fatalf("got %s", buf.String())
	}
}

func TestRedact(t *testing.T) {
	jwt := "eyJhbGciOiJIUzI1NiIsInR5cCI6IkpXVCJ9.e30.dBjftJeZ4CVP-mB92K27uhbUJU1p1r_wW1gFWFOEjXk"
	got := Redact(map[string]any{
		"email":        "a@b.c",
		"password":     "hunter2",
		"ssn":          "123-45-6789",
		"note":         "4111111111111111",
		"accessToken":  jwt,
		"refreshToken": jwt,
		"token":        "secret-token",
		"blob":         jwt,
		"tax":          "123456789",
	}).(map[string]any)
	if got["email"] != "a@b.c" {
		t.Fatalf("email redacted: %v", got["email"])
	}
	if got["password"] != "[redacted]" || got["ssn"] != "[redacted]" || got["note"] != "[redacted]" {
		t.Fatalf("expected redaction: %+v", got)
	}
	if got["accessToken"] != "[redacted]" || got["refreshToken"] != "[redacted]" || got["token"] != "[redacted]" {
		t.Fatalf("tokens not redacted: %+v", got)
	}
	if got["blob"] != "[redacted]" {
		t.Fatalf("JWT value not redacted: %+v", got["blob"])
	}
	if got["tax"] != "[redacted]" {
		t.Fatalf("undashed SSN not redacted: %+v", got["tax"])
	}
}
