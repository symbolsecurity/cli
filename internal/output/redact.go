package output

import (
	"encoding/json"
	"regexp"
	"strings"
)

var (
	ssnRe  = regexp.MustCompile(`^\d{3}-\d{2}-\d{4}$`)
	ccnRe  = regexp.MustCompile(`^\d{13,19}$`)
	keyRe  = regexp.MustCompile(`(?i)(password|passwd|secret|ssn|social.?security|credit.?card|card.?number|ccn|cvv|api[_-]?key|private[_-]?key|authorization)`)
	skipRe = regexp.MustCompile(`(?i)(id|user_id|company_id|status|email|name)$`)
)

func Redact(v any) any {
	b, err := json.Marshal(v)
	if err != nil {
		return v
	}
	var n any
	if err := json.Unmarshal(b, &n); err != nil {
		return v
	}
	return redactValue(n, "")
}

func redactValue(v any, key string) any {
	switch t := v.(type) {
	case map[string]any:
		out := make(map[string]any, len(t))
		for k, val := range t {
			out[k] = redactValue(val, k)
		}
		return out
	case []any:
		out := make([]any, len(t))
		for i, val := range t {
			out[i] = redactValue(val, key)
		}
		return out
	case string:
		if shouldRedactKey(key) || looksSecret(t) {
			return "[redacted]"
		}
		return t
	default:
		return v
	}
}

func shouldRedactKey(key string) bool {
	if key == "" || skipRe.MatchString(key) {
		return false
	}
	if strings.EqualFold(key, "id") {
		return false
	}
	return keyRe.MatchString(key)
}

func looksSecret(s string) bool {
	s = strings.ReplaceAll(s, " ", "")
	if ssnRe.MatchString(s) {
		return true
	}
	if ccnRe.MatchString(s) && luhn(s) {
		return true
	}
	return false
}

func luhn(s string) bool {
	sum := 0
	alt := false
	for i := len(s) - 1; i >= 0; i-- {
		n := int(s[i] - '0')
		if n < 0 || n > 9 {
			return false
		}
		if alt {
			n *= 2
			if n > 9 {
				n -= 9
			}
		}
		sum += n
		alt = !alt
	}
	return sum%10 == 0
}
