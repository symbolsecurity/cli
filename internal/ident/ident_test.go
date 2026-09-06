package ident

import "testing"

func TestProfile(t *testing.T) {
	if err := Profile("default"); err != nil {
		t.Fatal(err)
	}
	if err := Profile("../etc"); err == nil {
		t.Fatal("expected error")
	}
	if err := Profile("msp-prod"); err != nil {
		t.Fatal(err)
	}
}

func TestUUID(t *testing.T) {
	if err := UUID("11111111-1111-1111-1111-111111111111", "id"); err != nil {
		t.Fatal(err)
	}
	if err := UUID("u1", "id"); err == nil {
		t.Fatal("expected error")
	}
}

func TestSeg(t *testing.T) {
	got, err := Seg("acme.com", "keyword")
	if err != nil || got != "acme.com" {
		t.Fatalf("%s %v", got, err)
	}
	if _, err := Seg("../x", "keyword"); err == nil {
		t.Fatal("expected error")
	}
	if _, err := Seg("a/b", "keyword"); err == nil {
		t.Fatal("expected error")
	}
}
