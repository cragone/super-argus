package level

import "testing"

func TestNormalize(t *testing.T) {
	cases := map[string]string{
		"FATAL": Fatal, "Panic": Fatal, "critical": Fatal,
		"error": Error, "ERR": Error,
		"warn": Warning, "WARNING": Warning,
		"info": Info, "Notice": Info,
		"debug": Unknown, "trace": Unknown, "": Unknown, "banana": Unknown,
	}
	for in, want := range cases {
		if got := Normalize(in); got != want {
			t.Errorf("Normalize(%q) = %q, want %q", in, got, want)
		}
	}
}

func TestDetect(t *testing.T) {
	cases := []struct{ line, want string }{
		{`{"level":"error","msg":"x"}`, Error},
		{`{"severity":"WARNING","msg":"x"}`, Warning},
		{`{"Level":"fatal"}`, Fatal},
		{`{"level":"debug","msg":"an error occurred"}`, Error}, // unrecognized field falls through to token scan
		{`ts=1 level=warn msg="hi"`, Warning},
		{`ts=1 level="info" msg=hi`, Info},
		{`2026-01-01 ERROR something broke`, Error},
		{`[WARN] disk low`, Warning},
		{`just some text`, Unknown},
		{`information_schema query`, Unknown},
		{``, Unknown},
	}
	for _, c := range cases {
		if got := Detect(c.line); got != c.want {
			t.Errorf("Detect(%q) = %q, want %q", c.line, got, c.want)
		}
	}
}

func TestResolve(t *testing.T) {
	if got := Resolve(map[string]string{"level": "WARN"}, "ERROR boom"); got != Warning {
		t.Errorf("explicit label should win, got %q", got)
	}
	if got := Resolve(map[string]string{"severity": "err"}, "hello"); got != Error {
		t.Errorf("severity fallback, got %q", got)
	}
	if got := Resolve(map[string]string{"level": "verbose"}, "ERROR boom"); got != Error {
		t.Errorf("unrecognized label should fall back to detection, got %q", got)
	}
	if got := Resolve(map[string]string{"service": "api"}, "plain"); got != Unknown {
		t.Errorf("got %q", got)
	}
}
