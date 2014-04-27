package importmeta

import (
	"bytes"
	"fmt"
	"net/http"
	"net/http/httptest"
	"net/url"
	"testing"
)

type MockCodec struct {
	meta ImportMeta
	err  error
}

func (c MockCodec) ImportMeta(req *url.URL) (ImportMeta, error) {
	return c.meta, c.err
}

func TestPkgTemplate(t *testing.T) {
	buf := new(bytes.Buffer)
	err := pkgTemplate.Execute(buf, ImportMeta{})
	if err != nil {
		t.Fatalf("error executing package template: %v", err)
	}
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
