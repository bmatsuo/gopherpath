package main

import (
	"vendor/go-github/github"
	"vendor/goauth2/oauth"

	"appengine"
	"appengine/datastore"
	"appengine/memcache"
	"appengine/urlfetch"

	"fmt"
	"time"
)

const GitHubCredsMemcacheKey = "app:github:credentials"
const GitHubCredsDatastoreKind = "GitHubCreds"
const GitHubCredsDatastoreStringID = "credentials"

type GitHubToken struct {
	AccessToken  string
	RefreshToken string
	Expiry       time.Time
}

type GitHubUser struct {
	Token GitHubToken
	Login string
	URL   string
}

type GitHubCreds struct {
	ClientId     string
	ClientSecret string
	Description  string
	RedirectURL  string
}

func GetGitHubAuthUser(c appengine.Context, token *oauth.Token) (*GitHubUser, error) {
	// fetch the latest version of the user record and persist it.
	aeTrans := urlfetch.Transport{Context: c}
	authTrans := oauth.Transport{Token: token, Transport: &aeTrans}
	gh := github.NewClient(authTrans.Client())
	ghuser, _, err := gh.Users.Get("")
	if err != nil {
		c.Errorf("could not retreive authenticated user")
		return nil, err
	}
	user := &GitHubUser{
		Token: GitHubToken{
			AccessToken:  token.AccessToken,
			RefreshToken: token.RefreshToken,
			Expiry:       token.Expiry,
		},
		Login: *ghuser.Login,
		URL:   *ghuser.URL,
	}
	key := datastore.NewKey(c, "GitHubUsers", user.Login, 0, nil)
	_, err = datastore.Put(c, key, user)
	if err != nil {
		c.Errorf("unable to store user record: %v", err)
		return nil, err
	}
	return user, err
}

func GetGitHubCreds(c appengine.Context) (GitHubCreds, error) {
	var creds GitHubCreds
	_, err := memcache.Gob.Get(c, GitHubCredsMemcacheKey, &creds)
	if err == nil {
		return creds, nil
	}
	if err != memcache.ErrCacheMiss {
		c.Errorf("error looking up github app credentials in memcache: %v", err)
		return creds, err
	}

	key := datastore.NewKey(c, "GithubCreds", "credentials", 0, nil) // inconsistent kind name....
	err = datastore.Get(c, key, &creds)
	if err == datastore.ErrNoSuchEntity {
		c.Warningf("github app credentials are not in the datastore. add credentials in the admin console")
		_, _err := datastore.Put(c, key, &creds)
		if _err != nil {
			c.Errorf("unable to create credential stub: %v", _err)
		}
	} else if err == nil {
		c.Infof("github creds: %v", creds)
		if creds.ClientId == "" || creds.ClientSecret == "" {
			c.Errorf("incomplete github app credentials in datastore")
			return creds, fmt.Errorf("incomplete credentials")
		}
		// goroutine?
		_err := memcache.Gob.Set(c, &memcache.Item{
			Key:        GitHubCredsMemcacheKey,
			Object:     &creds,
			Expiration: time.Minute,
		})
		if _err != nil {
			c.Warningf("error caching github app credentials")
		}
	}
	return creds, err
}
