package main

import (
	"appengine"
	"appengine/datastore"

	"fmt"
)

var errNotFound = fmt.Errorf("not found")

type DomainAssoc struct {
	Key         *datastore.Key `json:"key" datastore:"-"`
	GitHubLogin string         `json:"githubLogin"`
	Domain      string         `json:"domain"`
}

func GetDomainAssocsGitHubLogin(c appengine.Context, login string) ([]DomainAssoc, error) {
	var assocs []DomainAssoc
	q := datastore.NewQuery("DomainAssocs").Filter("GitHubLogin = ", login)
	keys, err := q.GetAll(c, &assocs)
	if err != nil {
		return assocs, err
	}
	for i := range keys {
		assocs[i].Key = keys[i]
	}
	if len(keys) == 0 {
		return assocs, errNotFound
	}
	return assocs, nil
}

func GetDomainAssocs(c appengine.Context, domain string) ([]DomainAssoc, error) {
	var assocs []DomainAssoc
	q := datastore.NewQuery("DomainAssocs").
		Filter("Domain = ", domain)
	keys, err := q.GetAll(c, &assocs)
	if err != nil {
		return assocs, err
	}
	for i := range keys {
		assocs[i].Key = keys[i]
	}
	if len(keys) == 0 {
		return assocs, errNotFound
	}
	return assocs, nil
}

func PutDomainAssoc(c appengine.Context, assoc *DomainAssoc) error {
	if assoc.GitHubLogin == "" {
		return fmt.Errorf("unknown github login for association")
	}
	if assoc.Domain == "" {
		return fmt.Errorf("unknown domain for association")
	}

	var key *datastore.Key
	if assoc.Key != nil {
		key = assoc.Key
	} else {
		key = datastore.NewIncompleteKey(c, "DomainAssocs", nil)
	}
	// XXX this is not safe UNLESS assoc.domain has already been verified as belonging to assoc.GitHubLogin
	key, err := datastore.Put(c, key, assoc)
	if err != nil {
		return err
	}
	assoc.Key = key
	return nil
}
