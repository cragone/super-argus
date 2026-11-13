// Package level maps the many ways clients spell a log severity onto five
// canonical buckets: fatal, error, warning, info and unknown.
package level

import (
	"encoding/json"
	"slices"
	"strings"
)

// Canonical level values, used as the value of the "level" stream label.
const (
	Fatal   = "fatal"
	Error   = "error"
	Warning = "warning"
	Info    = "info"
	Unknown = "unknown"
)

// Label is the stream label that carries the canonical level.
const Label = "level"

// All lists the canonical levels, most severe first.
var All = []string{Fatal, Error, Warning, Info, Unknown}

var aliases = map[string]string{
	"fatal": Fatal, "panic": Fatal, "emerg": Fatal, "emergency": Fatal,
	"alert": Fatal, "crit": Fatal, "critical": Fatal, "severe": Fatal,
	"error": Error, "err": Error, "errors": Error,
	"warn": Warning, "warning": Warning,
	"info": Info, "information": Info, "notice": Info,
}

// Normalize maps a raw level spelling to a canonical bucket (Unknown if unrecognized).
func Normalize(raw string) string {
	if l, ok := aliases[strings.ToLower(strings.TrimSpace(raw))]; ok {
		return l
	}
	return Unknown
}

func known(raw string) bool {
	_, ok := aliases[strings.ToLower(strings.TrimSpace(raw))]
	return ok
}

var fieldKeys = []string{"level", "severity", "lvl", "levelname", "log_level", "loglevel"}

const scanWindow = 200

// Detect finds a level in a log line: a JSON level field, a logfmt level=
// pair, or the first level-looking token near the start of the line.
func Detect(line string) string {
	t := strings.TrimSpace(line)
	if strings.HasPrefix(t, "{") {
		var m map[string]any
		if json.Unmarshal([]byte(t), &m) == nil {
			for _, k := range fieldKeys {
				if s, ok := m[k].(string); ok && known(s) {
					return Normalize(s)
				}
			}
			for k, v := range m { // case-variant keys such as "Level"
				if s, ok := v.(string); ok && isFieldKey(k) && known(s) {
					return Normalize(s)
				}
			}
		}
	}
	head := line
	if len(head) > scanWindow {
		head = head[:scanWindow]
	}
	if l, ok := logfmt(head); ok {
		return l
	}
	return token(head)
}

func isFieldKey(k string) bool {
	return slices.Contains(fieldKeys, strings.ToLower(k))
}

// logfmt looks for key=value pairs whose key is a level key.
func logfmt(s string) (string, bool) {
	for f := range strings.FieldsSeq(s) {
		k, v, ok := strings.Cut(f, "=")
		if !ok || !isFieldKey(k) {
			continue
		}
		v = strings.Trim(v, `"',;`)
		if known(v) {
			return Normalize(v), true
		}
	}
	return "", false
}

// token returns the level of the first standalone level word in s.
func token(s string) string {
	isWord := func(r rune) bool {
		return r >= 'a' && r <= 'z' || r >= 'A' && r <= 'Z' || r >= '0' && r <= '9' || r == '_'
	}
	words := strings.FieldsFunc(s, func(r rune) bool { return !isWord(r) })
	for _, w := range words {
		if known(w) {
			return Normalize(w)
		}
	}
	return Unknown
}

// Resolve picks the level for a pushed line: an explicit label (or severity
// fallback) wins when recognized; otherwise the line is inspected.
func Resolve(labels map[string]string, line string) string {
	for _, k := range []string{Label, "severity"} {
		if v, ok := labels[k]; ok && known(v) {
			return Normalize(v)
		}
	}
	return Detect(line)
}
