package main

import (
	"vendor/securecookie"

	"appengine"
	"appengine/datastore"
	"appengine/memcache"

	"net/http"
	"time"
)

var SessionCookieName = "githubUser"

// secret session encryption keys
type SessionKeyPair struct {
	HashKey  []byte
	BlockKey []byte
	Created  time.Time
}

func GetSessionCookie(dst interface{}, resp http.ResponseWriter, req *http.Request, codecs []securecookie.Codec) error {
	enc, err := req.Cookie(SessionCookieName)
	if err != nil {
		if err != http.ErrNoCookie {
			DeleteSessionCookie(resp)
		}
		return err
	}

	err = securecookie.DecodeMulti(SessionCookieName, enc.Value, dst, codecs...)
	if err != nil {
		DeleteSessionCookie(resp)
		return err
	}

	return nil
}

func SetSessionCookie(resp http.ResponseWriter, v interface{}, codecs []securecookie.Codec) error {
	venc, err := securecookie.EncodeMulti(SessionCookieName, v, codecs...)
	if err != nil {
		return err
	}
	cookie := &http.Cookie{
		Name:  SessionCookieName,
		Value: venc,
		Path:  "/",
	}
	http.SetCookie(resp, cookie)
	return nil
}

func DeleteSessionCookie(resp http.ResponseWriter) {
	http.SetCookie(resp, &http.Cookie{
		Name:  SessionCookieName,
		Value: "",
		Path:  "/",
	})
}

func GetSessionCodecs(c appengine.Context) ([]securecookie.Codec, error) {
	skeys, err := GetSessionKeys(c)
	if err != nil {
		return nil, err
	}
	ps := make([][]byte, 2*len(skeys))
	for i := range skeys {
		hash := skeys[i].HashKey
		block := skeys[i].BlockKey
		ps[i] = hash
		if len(block) > 0 {
			ps[i+1] = block
		}
	}
	codecs := securecookie.CodecsFromPairs(ps...)
	return codecs, nil
}

func GetSessionKeys(c appengine.Context) ([]SessionKeyPair, error) {
	var skeys []SessionKeyPair

	_, err := memcache.Gob.Get(c, "session:keypairs", &skeys)
	if err != nil && err != memcache.ErrCacheMiss {
		c.Warningf("unable to get session keypairs from memcache: %v", err)
	}
	if err == nil {
		return skeys, nil
	}

	_, err = datastore.NewQuery("SessionKeyPairs").Order("-Created").Limit(3).GetAll(c, &skeys)
	if err != nil {
		c.Errorf("error fetching session key pairs: %v", err)
		return nil, err
	}
	if len(skeys) > 0 {
		return skeys, nil
	}

	skey := SessionKeyPair{
		HashKey: securecookie.GenerateRandomKey(64),
		// no encryption for now
	}
	newkey := datastore.NewIncompleteKey(c, "SessionKeyPairs", nil)
	_, err = datastore.Put(c, newkey, &skey)
	if err != nil {
		c.Errorf("error storing session key pairs: %v", err)
		return nil, err
	}

	skeys = append(skeys, skey)
	err = memcache.Gob.Set(c, &memcache.Item{
		Key:    "session:keypairs",
		Object: skeys,
	})
	if err != nil {
		return skeys, nil
	}

	return skeys, nil
}
