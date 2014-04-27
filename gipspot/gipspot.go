package main

import (
	"vendor/pat"

	"appengine"

	"fmt"
	"html/template"
	"net/http"
	"path"
	"strings"
)

func init() {
	mux := pat.New()
	mux.Get("/:pkgRoot", http.HandlerFunc(HandlePkg))
	mux.Get("/:pkgRoot/", http.HandlerFunc(HandlePkg))
	mux.Get("/", http.HandlerFunc(HandleRoot))
	http.Handle("/", mux)
}

var pkgTemplate = template.Must(template.New("pkg").Funcs(template.FuncMap{
	"godoc": func(path string) string {
		return "http://godoc.org/" + path
	},
}).Parse(`
<html>
	<head>
		<!-- uncomment this when this thing actually works -->
		<!--meta http-equiv="refresh" content="0; URL='http://godoc.org/{{.ImportPath}}'"-->
		<meta name="go-import" content="{{.ImportPrefix}} {{.VCS}} {{.RepoPrefix}}">
	</head>
	<body>
		<strong>you are being redirected to <a href="{{godoc .ImportPath}}">{{godoc .ImportPath}}</a>.</strong>
	</body>
</html>
`))

type PkgMeta struct {
	ImportPath   string
	ImportPrefix string
	VCS          string
	RepoPrefix   string
}

func HandlePkg(resp http.ResponseWriter, req *http.Request) {
	c := appengine.NewContext(req)
	host := req.Host
	assocs, err := GetDomainAssocs(c, host)
	if err != nil {
		http.Error(resp, "an error occurred", http.StatusInternalServerError)
		return
	}
	if len(assocs) == 0 {
		http.Error(resp, "unknown host: "+host, http.StatusNotFound)
		return
	}
	assoc := assocs[0]
	query := req.URL.Query()
	goget := query.Get("go-get")
	if goget == "" {
		c.Warningf("missing go-get query parameter")
	}
	pkgRoot := query.Get(":pkgRoot")
	pkg := strings.TrimRight(path.Join(pkgRoot, pat.Tail("/:pkgRoot/", req.URL.Path)), "/")
	c.Infof("request for package %v/%v", host, pkg)
	repoPrefix := fmt.Sprintf("https://github.com/%v/%v", assoc.GitHubLogin, pkgRoot)
	meta := &PkgMeta{
		ImportPath:   path.Join(host, pkg),
		ImportPrefix: host,
		VCS:          "git",
		RepoPrefix:   repoPrefix,
	}
	err = pkgTemplate.Execute(resp, meta)
	if err != nil {
		c.Errorf("couldn't render package template")
	}
}

func HandleRoot(resp http.ResponseWriter, req *http.Request) {
	c := appengine.NewContext(req)

	host := req.Host
	assocs, err := GetDomainAssocs(c, host)
	if err != nil {
		c.Errorf("unable to lookup hostname: %v", err)
		http.Error(resp, "an error occurred", http.StatusInternalServerError)
		return
	}
	var assoc DomainAssoc
	if len(assocs) == 0 {
		assoc := DomainAssoc{Domain: host}
		err := PutDomainAssoc(c, &assoc)
		if err != nil {
			c.Errorf("unable to create stub entity: %v", err)
			http.Error(resp, "an error occurred. check the logs for more information", http.StatusInternalServerError)
			return
		}
	} else {
		assoc = assocs[0]
	}

	if assoc.GitHubLogin == "" {
		resp.WriteHeader(http.StatusNotFound)
		fmt.Fprintln(resp, "unrecognized host")
		fmt.Fprintln(resp)
		fmt.Fprintf(resp, "use the GAE admin panel to associate a github login the datastore DomainAssocs entity %s\n", assoc.Key)
		return
	}

	fmt.Fprintf(resp, "%v directs clients to source repositories at https://github.com/%v", host, assoc.GitHubLogin)
}

func topLevelDir(fullpath string) string {
	p1 := fullpath
	p2 := path.Dir(fullpath)
	for {
		if p2 == "/" || p2 == "." {
			return path.Base(p1)
		}
		p1 = p2
		p2 = path.Dir(p2)
	}
	panic("unreachable")
}
