package main

import (
	"fmt"
	"net/http"
	"strings"
)

type RequestContext struct {
	User            *User
	NamespacePrefix string
	Namespace       string
}

func getRequestContext(r *http.Request) *RequestContext {
	authToken := r.Header.Get(authTokenHeader)
	user := users[authToken]
	namespace := r.Header.Get(namespaceHeader)
	namespacePrefix := r.Header.Get(namespacePrefixHeader)
	return &RequestContext{
		User:            &user,
		Namespace:       namespace,
		NamespacePrefix: namespacePrefix,
	}
}

func (rc *RequestContext) isAuthorized() bool {
	return rc.User.Username != ""
}

// item:g:default:qwer-asdf-1234asdf - global keys in default namespace
// item:my:user:work:zcxv-asdf-qwer - user specific keys in work namespace
func (rc *RequestContext) buidlKey(uid string) string {
	keyParts := []string{"item", rc.NamespacePrefix}
	if rc.NamespacePrefix == "my" {
		keyParts = append(keyParts, rc.User.Username)
	}
	keyParts = append(keyParts, rc.Namespace)
	if uid != "" {
		keyParts = append(keyParts, uid)
	}
	return strings.Join(keyParts, ":")
}

func (rc *RequestContext) buildKeyPattern() string {
	// to make sure we are not building pattern with item uid
	key := rc.buidlKey("")
	return fmt.Sprintf("%s:*", key)
}
