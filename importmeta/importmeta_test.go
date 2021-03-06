package importmeta

import (
	"bytes"
	"fmt"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
)

type MockCodec struct {
	meta ImportMeta
	err  error
}

func (c MockCodec) ImportMeta(req *http.Request) (ImportMeta, error) {
	return c.meta, c.err
}

func TestIsGoGet(t *testing.T) {
	for i, test := range []struct {
		Method string
		URL    string
		Expect bool
	}{
		{"GET", "/foo/bar?baz=qux", false},
		{"GET", "/foo/bar?baz=qux&go-get=1&quux", true},
		{"GET", "/foo/bar?go-get&quux", true},
		{"PUT", "/foo/bar?go-get&quux", false},
	} {
		req, err := http.NewRequest(test.Method, test.URL, nil)
		if err != nil {
			t.Fatalf("test %d: error constructing request: %v", i, err)
		}
		if IsGoGet(req) != test.Expect {
			t.Errorf("test %d: IsGoGet(%v) returned %v, not %v", i, req, !test.Expect, test.Expect)
		}
	}
}

func TestMiddlewareBehavior(t *testing.T) {
	for i, test := range []struct {
		isgoget  bool
		codec    MockCodec
		responds bool
		check    func(*httptest.ResponseRecorder) error
	}{
		{true, MockCodec{}, true, nil},
		{true, MockCodec{err: ErrNotFound}, true, nil}, // TODO check error
		{true, MockCodec{err: fmt.Errorf("boom")}, true, nil},
		{false, MockCodec{}, false, nil},
		{false, MockCodec{err: ErrNotFound}, false, nil},
		{false, MockCodec{err: fmt.Errorf("boom")}, false, nil},
	} {
		var req *http.Request
		resp := httptest.NewRecorder()
		resp.Body = new(bytes.Buffer)
		if test.isgoget {
			req, _ = http.NewRequest("GET", "/foo?go-get=1", nil)
		} else {
			req, _ = http.NewRequest("GET", "/foo", nil)
		}
		m := Middleware(test.codec)
		m.ServeHTTP(resp, req)
		body := resp.Body.Bytes()
		if test.responds && len(body) == 0 {
			t.Errorf("test %d: middleware failed to respond", i)
		} else if !test.responds && len(body) > 0 {
			t.Errorf("test %d: middleware erroneously responded (%d)", i, resp.Code)
		}
		if test.check != nil {
			err := test.check(resp)
			if err != nil {
				fmt.Errorf("test %d: %v", i, err)
			}
		}
	}
}

type LogRecorder struct {
	logs []string
}

func (rec *LogRecorder) Log(msg string) {
	rec.logs = append(rec.logs, msg)
}

// for coverage. just make sure logf doesn't blow up for some crazy reason.
func TestLog(t *testing.T) {
	rec := new(LogRecorder)
	Logger = rec
	defer func() { Logger = nil }()
	logf("hello, %v", "world")
	logf("goodbye, %v", "friend")
	if len(rec.logs) != 2 {
		t.Fatalf("unexpected number of log entries (%d): %v", len(rec.logs), rec.logs)
	}
	if rec.logs[0] != "hello, world" {
		t.Fatalf("unexpected content of log entry 0: %v", rec.logs[0])
	}
	if rec.logs[1] != "goodbye, friend" {
		t.Fatalf("unexpected content of log entry 1: %v", rec.logs[1])
	}
}

// make sure that pkgTemplate successfully renders and generates good meta tags.
func TestTemplate(t *testing.T) {
	meta := ImportMeta{
		Pkg:     "foo.io/bar/baz",
		RootPkg: "foo.io/bar",
		VCS:     "git",
		Repo:    "https://github.com/mcfoo/bar",
	}
	buf := new(bytes.Buffer)
	err := pkgTemplate.Execute(buf, meta)
	if err != nil {
		t.Fatalf("error rendering template: %v", err)
	}
	html := buf.String()
	metas := []string{
		`<meta http-equiv="refresh" content="0; URL='http://godoc.org/foo.io/bar/baz'">`,
		`<meta name="go-import" content="foo.io/bar git https://github.com/mcfoo/bar">`,
	}
	t.Logf("template output: %q", html)
	for i := range metas {
		ok := strings.Contains(html, metas[i])
		if !ok {
			t.Errorf("template output missing meta tag: %v", metas[i])
		}
	}
}
