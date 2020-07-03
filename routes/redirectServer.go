package routes

import (
	"golang.org/x/crypto/acme/autocert"
	"net/http"
	"sr"
	"sr/config"
)

var redirectRouter = sr.BaseRouter()

var _ = redirectRouter.HandleFunc("/", redirectHTTP)

func redirectHTTP(response Response, request *Request) {
	logRequest(request)
	newURL := config.TlsHost + request.URL.String()
	logf(request, "-> %v %v", http.StatusMovedPermanently, newURL)
	http.Redirect(response, request, newURL, http.StatusMovedPermanently)
}

func MakeHTTPRedirectServer(certManager *autocert.Manager) *http.Server {
	server := makeServerFromHandler(redirectRouter)
	// Allow autocert to perform http challenges
	server.Handler = certManger.HTTPHandler(server.Handler)
	server.Addr = config.ServerHTTPPort
	return server
}
