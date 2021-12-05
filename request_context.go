package main

import (
	"fmt"
	"net/http"
)

type RequestContext struct {
	User      *User
	Namespace string
}

func getRequestContext(r *http.Request) *RequestContext {
	authToken := r.Header.Get(authTokenHeader)
	user := users[authToken]
	namespace := r.Header.Get(namespaceHeader)
	if namespace == "" {
		namespace = globalNamespace
	}
	return &RequestContext{
		User:      &user,
		Namespace: namespace,
	}
}

func (rc *RequestContext) isAuthorized() bool {
	return rc.User.Username != ""
}

// item:global:qwer-asdf-1234asdf - global keys
// item:user:my:zcxv-asdf-qwer - user specific namespaces
func (rc *RequestContext) buidlKey(uid string) string {
	key := "item"
	if rc.Namespace != globalNamespace {
		key = fmt.Sprintf("%s:%s", key, rc.User.Username)
	}
	key = fmt.Sprintf("%s:%s", key, rc.Namespace)
	// if rc.Namespace != globalNamespace {
	// 	key = fmt.Sprintf("%s:", rc.User.Username)
	// key = fmt.Sprintf("%s:", rc.Namespace)
	if uid != "" {
		key = fmt.Sprintf("%s:%s", key, uid)
	}
	return key
}

func (rc *RequestContext) buildKeyPattern() string {
	// to make sure we are not building pattern with item uid
	key := rc.buidlKey("")
	return fmt.Sprintf("%s:*", key)
}
