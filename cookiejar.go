package fasthttp

import (
	"bytes"
	"sync"
	"time"
)

// inspired by https://github.com/dgrr/cookiejar/blob/master/cookiejar.go
// but rewritten significantly

type CookieJar struct {
	mp    map[string]*Cookie
	mutex sync.Mutex
}

func NewCookieJar(initCookies []*Cookie) *CookieJar {
	cj := &CookieJar{
		mp: map[string]*Cookie{},
	}
	for _, v := range initCookies {
		keyCopy := string(v.key)
		cj.mp[keyCopy] = v
	}
	return cj
}

// ReadResponse gets all Response cookies reading Set-Cookie header.
func (cj *CookieJar) ReadResponse(r *Response) {
	cj.mutex.Lock()
	defer cj.mutex.Unlock()
	r.Header.VisitAllCookie(func(key, value []byte) {
		valueCopy := bytes.Clone(value)
		cookie := AcquireCookie()
		cookie.ParseBytes(valueCopy)
		if cookie.maxAge < 0 {
			return
		}
		if cookie.maxAge > 0 {
			cookie.expire = time.Now().Add(time.Duration(cookie.maxAge) * time.Second)
		}
		keyCopy := string(key)
		prev, ok := cj.mp[keyCopy]
		if ok {
			ReleaseCookie(prev)
		}
		cj.mp[keyCopy] = cookie
	})

	for k, v := range cj.mp {
		if !v.expire.IsZero() {
			if v.expire.After(time.Now()) {
				ReleaseCookie(v)
				delete(cj.mp, k)
			}
		}
	}
}

// FillRequest dumps all cookies stored in cj into Request adding this values to Cookie header.
func (cj *CookieJar) FillRequest(r *Request) {
	cj.mutex.Lock()
	defer cj.mutex.Unlock()
	for _, v := range cj.mp {
		r.Header.SetCookieBytesKV(bytes.Clone(v.Key()), bytes.Clone(v.Value()))
	}
}
