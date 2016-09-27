package handler

import "net/http"


const (
	tokenCreatePath = "auth/token/create"
	defaultTokenWrapTTL = "10m"
)

func wrapTokenCreation(operation, path string) string {
	if operation == http.MethodPost && path == tokenCreatePath {
		return defaultTokenWrapTTL
	}
	return ""
}
