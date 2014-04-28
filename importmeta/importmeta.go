package importmeta

import (
	"fmt"
	"html/template"
	"net/http"
)

// PkgTemplate describes the content served by Handler and Middleware return
// values.  It is invoked with an ImportMeta type as its context.
var PkgTemplate = template.Must(template.New("pkg").Parse(`
{{$godoc := .GodocURL}}
<html>
	<head>
		<meta http-equiv="refresh" content="0; URL='{{$godoc}}'">
		<meta name="go-import" content="{{.RootPkg}} {{.VCS}} {{.Repo}}">
	</head>
	<body>
		You are being redirected to <a href="{{$godoc}}">{{$godoc}}</a>.
	</body>
</html>
`))

// Handler creates an http.Handler that serves all requests as go-get requests.
func Handler(codec Codec) http.Handler {
	return http.HandlerFunc(func(resp http.ResponseWriter, req *http.Request) {
		m, err := codec.ImportMeta(req)
		if err == ErrNotFound {
			logf("no metadata for url %q: %v", req.URL, err)
			http.Error(resp, "unrecognized package", http.StatusNotFound)
			return
		}
		if err != nil {
			logf("error locating metadata for url %q: %v", req.URL, err)
			http.Error(resp, "something went wrong", http.StatusInternalServerError)
			return
		}
		err = PkgTemplate.Execute(resp, m)
		if err != nil {
			logf("error rendering/writing metadata response: %v", err)
			resp.Write(nil)
			return
		}
	})
}

// Middleware is like Handler, but the returned http.Handler only writes
// responses to go-get requests.  to be useful the returned handler must be
// given an implementation of http.ResponseWriter which records whether the
// response headers have been sent over the wire.
func Middleware(codec Codec) http.Handler {
	handler := Handler(codec)
	return http.HandlerFunc(func(resp http.ResponseWriter, req *http.Request) {
		if IsGoGet(req) {
			handler.ServeHTTP(resp, req)
		}
	})
}

// Render executes PkgTemplate with meta as its context.
func Render(resp http.ResponseWriter, meta ImportMeta) error {
	return PkgTemplate.Execute(resp, meta)
}

// IsGoGet returns true if req is a GET request with a "go-get" query parameter.
func IsGoGet(req *http.Request) bool {
	if req.Method != "GET" {
		return false
	}
	_, present := req.URL.Query()["go-get"]
	return present
}

// ErrNotFound is may returned by a Codec if there is no import metadata
// corresponding to a request.
var ErrNotFound = fmt.Errorf("not found")

// ImportMeta contains information needed for go-get to find a package.
type ImportMeta struct {
	Pkg     string // fully qualified package import path (e.g. foo.io/bar/baz)
	RootPkg string // fully qualified root package import path (e.g. foo.io/bar)
	VCS     string // repository VCS (e.g. git)
	Repo    string // repository URL (e.g. https://github.com/someuser/bar)
}

func (m ImportMeta) GodocURL() string {
	return fmt.Sprintf("http://godoc.org/%s", m.Pkg)
}

// Codec defines the interface required of implementation specific backend stores.
type Codec interface {
	ImportMeta(*http.Request) (ImportMeta, error)
}

type CodecFunc func(*http.Request) (ImportMeta, error)

func (fn CodecFunc) ImportMeta(req *http.Request) (ImportMeta, error) {
	return fn(req)
}

var Logger interface {
	Log(string)
}

func logf(format string, v ...interface{}) {
	if Logger != nil {
		Logger.Log(fmt.Sprintf(format, v...))
	}
}
