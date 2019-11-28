package proxyhandler

import (
	"net"
	"net/http"
	"net/http/httputil"
	"net/url"
	"time"

	"code.cloudfoundry.org/lager"
)

func NewAuthHandler(
	logger lager.Logger,
	target *url.URL,
) *authHandler {

	dialer := &net.Dialer{
		Timeout:   24 * time.Hour,
		KeepAlive: 24 * time.Hour,
	}

	transport := &http.Transport{
		Proxy:               http.ProxyFromEnvironment,
		Dial:                dialer.Dial,
		TLSHandshakeTimeout: 60 * time.Second,
	}

	handler := httputil.NewSingleHostReverseProxy(target)
	handler.FlushInterval = 100 * time.Millisecond
	handler.Transport = transport

	return &authHandler{
		Logger:  logger,
		Handler: handler,
	}
}

type authHandler struct {
	lager.Logger
	http.Handler
}

func (p *authHandler) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	p.Logger.Debug("proxy-auth-request", lager.Data{"path": r.URL.String()})

	p.Handler.ServeHTTP(w, r)
}
