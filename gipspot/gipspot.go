package main

import (
	"vendor/goauth2/oauth"
	"vendor/pat"

	"appengine"
	"appengine/urlfetch"

	"encoding/json"
	"fmt"
	"html/template"
	"io/ioutil"
	"net/http"
	"path"
)

func init() {
	mux := pat.New()
	mux.Del("/app/session", http.HandlerFunc(HandleSession))
	mux.Get("/app/github/auth/start", http.HandlerFunc(HandleGitHubAuthStart))
	mux.Get("/app/github/auth/finish", http.HandlerFunc(HandleGitHubAuthFinish))
	mux.Get("/pkg/", http.HandlerFunc(HandlePkg))
	mux.Post("/app/association", http.HandlerFunc(HandleAssociationCreate))
	mux.Get("/app/association", http.HandlerFunc(HandleAssociationList))
	mux.Get("/app/", http.HandlerFunc(HandleRoot))
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
	pkg := pat.Tail("/pkg/", req.URL.Path)
	pkgRoot := topLevelDir(pkg)
	query := req.URL.Query()
	goget := query.Get("go-get")
	if goget == "" {
		c.Warningf("missing go-get query parameter")
	}
	c.Infof("received request on host %v for pacakge %s", host, pkg)
	repoPrefix := path.Join("https://github.com", assoc.GitHubLogin, pkgRoot)
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

func HandleAssociationList(resp http.ResponseWriter, req *http.Request) {
	c := appengine.NewContext(req)

	var user GitHubUser
	codecs, err := GetSessionCodecs(c)
	if err != nil {
		http.Error(resp, "unable to authenticate user", http.StatusInternalServerError)
		return
	}
	err = GetSessionCookie(&user, resp, req, codecs)
	if err != nil {
		c.Warningf("unable to ")
		http.Error(resp, "unable to authenticate user", http.StatusInternalServerError)
		return
	}
	assocs, err := GetDomainAssocsGitHubLogin(c, user.Login)
	if err != nil {
		c.Errorf("error fetching associations by user: %v", err)
		http.Error(resp, "unable to retreive associations", http.StatusInternalServerError)
		return
	}
	entity := map[string]interface{}{
		"githubLogin":  user.Login,
		"associations": assocs,
	}
	p, err := json.Marshal(entity)
	if err != nil {
		c.Errorf("json marshaling error: %v", err)
		http.Error(resp, "something unexpected happened", http.StatusInternalServerError)
		return
	}
	_, err = resp.Write(p)
	if err != nil {
		c.Warningf("io error: %v", err)
	}
}

func HandleAssociationCreate(resp http.ResponseWriter, req *http.Request) {
	defer req.Body.Close()
	p, err := ioutil.ReadAll(req.Body)
	if err != nil {
		http.Error(resp, "unable to read complete request", http.StatusInternalServerError)
		return
	}
	var assoc DomainAssoc
	err = json.Unmarshal(p, &assoc)
	if err != nil {
		http.Error(resp, "invalid json in request entity", http.StatusBadRequest)
		return
	}

	c := appengine.NewContext(req)

	var user GitHubUser
	codecs, err := GetSessionCodecs(c)
	if err != nil {
		http.Error(resp, "unable to authenticate user", http.StatusInternalServerError)
		return
	}
	err = GetSessionCookie(&user, resp, req, codecs)
	if err != nil {
		c.Warningf("unable to ")
		http.Error(resp, "unable to authenticate user", http.StatusInternalServerError)
		return
	}

	// you can't create verified associations
	//if assoc.Verified {
	//	c.Warningf("attempt to create automatically verified association: %v", assoc)
	//}
	//assoc.Verified = false
	assoc.GitHubLogin = user.Login

	err = PutDomainAssoc(c, &assoc)
	if err != nil {
		c.Errorf("error writing domain association: %v", err)
		http.Error(resp, "unable to save domain association", http.StatusInternalServerError)
		return
	}

	resp.WriteHeader(http.StatusCreated)
	p, _ = json.Marshal(assoc)
	_, err = resp.Write(p)
	if err != nil {
		c.Errorf("io error: %v", err)
		return
	}
}

func HandleGitHubAuthStart(resp http.ResponseWriter, req *http.Request) {
	c := appengine.NewContext(req)

	creds, err := GetGitHubCreds(c)
	if err != nil {
		http.Redirect(resp, req, "/index.html", http.StatusFound)
		return
	}
	config := &oauth.Config{
		ClientId:     creds.ClientId,
		ClientSecret: creds.ClientSecret,
		// no explicit scopes
		AuthURL:     "https://github.com/login/oauth/authorize",
		TokenURL:    "https://github.com/login/oauth/access_token",
		RedirectURL: creds.RedirectURL,
	}
	http.Redirect(resp, req, config.AuthCodeURL("gipspot"), http.StatusFound)
}

func HandleGitHubAuthFinish(resp http.ResponseWriter, req *http.Request) {
	c := appengine.NewContext(req)
	creds, err := GetGitHubCreds(c)
	if err != nil {
		http.Redirect(resp, req, "/index.html", http.StatusFound)
		return
	}
	config := &oauth.Config{
		ClientId:     creds.ClientId,
		ClientSecret: creds.ClientSecret,
		// no explicit scopes
		AuthURL:     "https://github.com/login/oauth/authorize",
		TokenURL:    "https://github.com/login/oauth/access_token",
		RedirectURL: creds.RedirectURL,
	}
	aeTrans := &urlfetch.Transport{Context: c}
	trans := &oauth.Transport{
		Config:    config,
		Transport: aeTrans,
	}
	query := req.URL.Query()
	state := query.Get("state")
	if state != "gipspot" {
		c.Errorf("invalid state parameter passed to authorization finish endpoint")
		http.Redirect(resp, req, "/index.html", http.StatusFound)
		return
	}
	token, err := trans.Exchange(query.Get("code"))
	if err != nil {
		c.Errorf("unable to exchange code for access token")
		http.Redirect(resp, req, "/index.html", http.StatusFound)
		return
	}
	if token.AccessToken == "" {
		c.Warningf("no access token received")
		http.Redirect(resp, req, "/index.html", http.StatusFound)
		return
	}
	c.Infof("token exchange successful: %+v", token)
	user, err := GetGitHubAuthUser(c, token)
	if err != nil {
		http.Redirect(resp, req, "/index.html", http.StatusFound)
		return
	}

	// FIXME remove this when multitenancy and domain verification are ready
	if user.Login != "bmatsuo" {
		c.Infof("non-bmatsuo user attempted login: %v", user.Login)
		http.Error(resp, "you are not allowed to access this resource", http.StatusInternalServerError)
		return
	}

	codecs, err := GetSessionCodecs(c)
	if err != nil {
		http.Error(resp, "unable to get session codecs", http.StatusInternalServerError)
		return
	}
	err = SetSessionCookie(resp, user, codecs)
	if err != nil {
		http.Error(resp, "unable to initialize user session", http.StatusInternalServerError)
		return
	}

	http.Redirect(resp, req, "/association.html", http.StatusFound)
}

func HandleRoot(resp http.ResponseWriter, req *http.Request) {
	c := appengine.NewContext(req)

	codecs, err := GetSessionCodecs(c)
	if err != nil {
		http.Redirect(resp, req, "/app/github/auth/start", http.StatusFound)
		return
	}

	var user GitHubUser
	err = GetSessionCookie(&user, resp, req, codecs)
	if err != nil {
		if err != http.ErrNoCookie {
			c.Errorf("couldn't read session cookie: %v", err)
		}
		http.Redirect(resp, req, "/index.html", http.StatusFound)
		return
	}

	http.Redirect(resp, req, "/association.html", http.StatusFound)
}

func HandleSession(resp http.ResponseWriter, req *http.Request) {
	if req.Method != "DELETE" {
		resp.Header().Set("Allow", "DELETE")
		msg := fmt.Sprintf("%s requests are not allowed", req.Method)
		http.Error(resp, msg, http.StatusMethodNotAllowed)
		return
	}

	http.SetCookie(resp, &http.Cookie{
		Name:  "githubUser",
		Value: "",
		Path:  "/",
	})
	fmt.Fprint(resp, "the session has been deleted")
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
