package log

import (
	"encoding/json"
	"regexp"
	"strings"
)

var (
	bearerRe   = regexp.MustCompile(`(?i)bearer\s+[A-Za-z0-9._\-]+`)
	apiKeyRe   = regexp.MustCompile(`(?i)(api[_-]?key|x-api-key)\s*[:=]\s*['"]?[A-Za-z0-9._\-]{8,}`)
	queryKeyRe = regexp.MustCompile(`(?i)([?&](?:api_key|key|token|access_token)=)[^&\s"']+`)
	skRe       = regexp.MustCompile(`sk-[A-Za-z0-9._\-]{8,}`)
)

type Redactor struct {
	canaries []string
}

func NewRedactor(canaries ...string) *Redactor {
	return &Redactor{canaries: canaries}
}

func (r *Redactor) RedactString(s string) string {
	out := s
	out = bearerRe.ReplaceAllString(out, "Bearer [REDACTED]")
	out = apiKeyRe.ReplaceAllString(out, "$1 [REDACTED]")
	out = queryKeyRe.ReplaceAllString(out, "$1[REDACTED]")
	out = skRe.ReplaceAllString(out, "sk-[REDACTED]")
	for _, c := range r.canaries {
		if c != "" {
			out = strings.ReplaceAll(out, c, "[REDACTED]")
		}
	}
	return out
}

func (r *Redactor) RedactMap(in map[string]any) map[string]any {
	if in == nil {
		return nil
	}
	allow := map[string]bool{
		"id": true, "run_id": true, "time": true, "mode": true, "agent": true,
		"kind": true, "status": true, "message": true, "artifact_ref": true,
		"duration": true, "usage": true, "type": true, "provider": true,
	}
	out := make(map[string]any, len(in))
	for k, v := range in {
		if !allow[k] {
			continue
		}
		out[k] = r.redactValue(v)
	}
	return out
}

func (r *Redactor) redactValue(v any) any {
	switch t := v.(type) {
	case string:
		return r.RedactString(t)
	case map[string]any:
		return r.RedactMap(t)
	default:
		return v
	}
}

func (r *Redactor) RedactJSON(data []byte) []byte {
	var v any
	if err := json.Unmarshal(data, &v); err != nil {
		return []byte(r.RedactString(string(data)))
	}
	redacted := r.redactAny(v)
	out, err := json.Marshal(redacted)
	if err != nil {
		return []byte(r.RedactString(string(data)))
	}
	return out
}

func (r *Redactor) redactAny(v any) any {
	switch t := v.(type) {
	case string:
		return r.RedactString(t)
	case map[string]any:
		out := make(map[string]any, len(t))
		for k, val := range t {
			out[k] = r.redactAny(val)
		}
		return out
	case []any:
		out := make([]any, len(t))
		for i, val := range t {
			out[i] = r.redactAny(val)
		}
		return out
	default:
		return v
	}
}

func ContainsSecret(s string, canaries ...string) bool {
	r := NewRedactor(canaries...)
	return r.RedactString(s) != s
}