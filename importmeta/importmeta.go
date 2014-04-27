package importmeta

import (
	"fmt"
	"html/template"
	"net/http"
	"net/url"
)

var pkgTemplate = template.Must(template.New("pkg").Funcs(template.FuncMap{
	"godoc": func(path string) string {
		return "http://godoc.org/" + path
	},
}).Parse(`
{{$godoc := .GodocURL}}
<html>
	<head>
		<meta http-equiv="refresh" content="0; URL='{{$godoc}}'">
		<meta name="go-import" content="{{.ImportPrefix}} {{.VCS}} {{.RepoPrefix}}">
	</head>
	<body>
		You are being redirected to <a href="{{$godoc}}">{{$godoc}}</a>.
	</body>
</html>
`))


// an HTTP middleware that responds to go-get requests.  the returned handler
// must be given an implementation of http.ResponseWriter which records whether
// the response headers have been sent over the wire.
func Middleware(codec Codec) http.Handler {
	return http.HandlerFunc(func(resp http.ResponseWriter, req *http.Request) {
		if IsGoGet(req) {
			m, err := codec.ImportMeta(req.URL)
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
			err = pkgTemplate.Execute(resp, m)
			if err != nil {
				logf("error rending/writing metadata response: %v", err)
				resp.Write(nil)
				return
			}
		}
	})
}

// true if req is a GET request with a "go-get" query parameter.
func IsGoGet(req *http.Request) bool {
	if req.Method != "GET" {
		return false
	}
	_, present := req.URL.Query()["go-get"]
	return present
}

// a Codec may return ErrNotFound if there is no import metadata corresponding
// to a request.
var ErrNotFound = fmt.Errorf("not found")

type ImportMeta struct {
	ImportPath   string
	ImportPrefix string
	VCS          string
	RepoPrefix   string
}

func (m *ImportMeta) GodocURL() string {
	return fmt.Sprintf("http://godoc.org/%s", m.ImportPath)
}

type Codec interface {
	ImportMeta(req *url.URL) (ImportMeta, error)
}

var Logger interface {
	Log(string)
}

func log(v ...interface{}) {
	if Logger != nil {
		Logger.Log(fmt.Sprint(v...))
	}
}

func logf(format string, v ...interface{}) {
	if Logger != nil {
		Logger.Log(fmt.Sprintf(format, v...))
	}
}
