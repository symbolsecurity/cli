package ident

import (
	"net/url"
	"regexp"
	"strings"

	"github.com/symbolsecurity/cli/internal/output"
)

var (
	profileRe = regexp.MustCompile(`^[A-Za-z0-9][A-Za-z0-9._-]{0,63}$`)
	uuidRe    = regexp.MustCompile(`(?i)^[0-9a-f]{8}-[0-9a-f]{4}-[0-9a-f]{4}-[0-9a-f]{4}-[0-9a-f]{12}$`)
)

func Profile(s string) error {
	if s == "" {
		return output.Usage("profile is required", "Use a name like default or msp")
	}
	if !profileRe.MatchString(s) {
		return output.Usage("invalid profile name", "Use letters, digits, dot, underscore, or dash")
	}
	return nil
}

func UUID(s, name string) error {
	if s == "" {
		return output.Usage(name+" is required", "Pass a UUID "+name)
	}
	if !uuidRe.MatchString(s) {
		return output.Usage("invalid "+name, "Pass a UUID")
	}
	return nil
}

func Seg(s, name string) (string, error) {
	s = strings.TrimSpace(s)
	if s == "" {
		return "", output.Usage(name+" is required", "Pass a "+name)
	}
	if strings.Contains(s, "..") || strings.ContainsAny(s, "/\\") {
		return "", output.Usage("invalid "+name, "Value cannot contain slashes or ..")
	}
	return url.PathEscape(s), nil
}
