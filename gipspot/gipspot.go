package main

import (
	"importmeta"

	"appengine"

	"fmt"
	"net/http"
	"path"
)

func init() {
	http.HandleFunc("/", HandleRoot)
}

var MetaCodec = importmeta.CodecFunc(func(req *http.Request) (importmeta.ImportMeta, error) {
	var meta importmeta.ImportMeta
	c := appengine.NewContext(req)
	host := req.Host
	assocs, err := GetDomainAssocs(c, host)
	if err != nil {
		c.Errorf("error retrieving host association: %v", host)
		return meta, err
	}
	if len(assocs) == 0 {
		c.Warningf("request for unknown host: %v", host)
		return meta, importmeta.ErrNotFound
	}
	pkgRootBase := topLevelDir(req.URL.Path)
	meta.Pkg = path.Join(host, req.URL.Path)
	meta.RootPkg = path.Join(host, pkgRootBase)
	meta.VCS = "git"
	meta.Repo = fmt.Sprintf("https://github.com/%v/%v", assocs[0].GitHubLogin, pkgRootBase)
	return meta, nil
})
var MetaHandler = importmeta.Handler(MetaCodec)

func topLevelDir(reqpath string) string {
	dir, file := path.Split(reqpath)
	if dir == "" || dir == "/" {
		return file
	}
	return topLevelDir(dir)
}

func HandleRoot(resp http.ResponseWriter, req *http.Request) {
	c := appengine.NewContext(req)

	if req.URL.Path != "/" {
		MetaHandler.ServeHTTP(resp, req)
		return
	}

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
		fmt.Fprintln(resp, "unrecognized host: ", host)
		fmt.Fprintln(resp)
		fmt.Fprintf(resp, "associate a github login with datastore DomainAssocs entity %s using the GAE admin dashboard\n", assoc.Key)
		return
	}

	fmt.Fprintf(resp, "%v directs clients to source repositories at https://github.com/%v", host, assoc.GitHubLogin)
}
