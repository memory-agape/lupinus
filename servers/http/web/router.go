package web

import (
	"lupinus/servers/http"
	"lupinus/servers/http/web/controller"
)

func Connect(clientMeta http.HttpClientMeta) (*http.HttpBody, *http.HttpHeader, error) {
	responseBody := &http.HttpBody{}
	responseHeader := &http.HttpHeader{}

	switch clientMeta.Path.Path {
	case "/":
		responseBody, responseHeader = controller.RequestRoot(clientMeta)
		break
	case "/api/v1/login",
		"/api/v1/signin":
		responseBody, responseHeader = controller.RequestSignin(clientMeta)
		break

	case "/api/v1/signout":
		responseBody, responseHeader = controller.RequestSignout(clientMeta)
		break
	case "/api/v1/user":
		responseBody, responseHeader = controller.RequestUser(clientMeta)
		break
	case "/api/v1/info":
		responseBody, responseHeader = controller.RequestInfo(clientMeta)
		break
	case "/api/v1/favorite":
		responseBody, responseHeader = controller.RequestFavorite(clientMeta)
		break
	case "/api/v1/capture":
		responseBody, responseHeader = controller.RequestCapture(clientMeta)
		break
	case "/api/v1/image":
		responseBody, responseHeader = controller.RequestImage(clientMeta)
		break
	case "/api/v1/env":
		responseBody, responseHeader = controller.RequestEnv(clientMeta)
		break
	case "/favicon.ico":
		return nil, nil, nil
	case "/api/v1/slack":
		responseBody, responseHeader = controller.RequestSlack(clientMeta)
		break
	case "/api/v1/download":
		responseBody, responseHeader = controller.RequestDownload(clientMeta)
		break
	default:
		responseBody, responseHeader = controller.RequestFallback(clientMeta)
		break
	}

	return responseBody, responseHeader, nil
}
