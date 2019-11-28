package web

import (
	"net/http"
	"net/url"

	"code.cloudfoundry.org/lager"
	"github.com/concourse/concourse/skymarshal/legacyserver"
	"github.com/concourse/concourse/skymarshal/skycmd"
	"github.com/concourse/concourse/skymarshal/skyserver"
	"github.com/concourse/concourse/skymarshal/token"
	"github.com/concourse/concourse/web/indexhandler"
	"github.com/concourse/concourse/web/proxyhandler"
	"github.com/concourse/concourse/web/publichandler"
	"github.com/concourse/concourse/web/robotshandler"
	"golang.org/x/oauth2"
)

type WebConfig struct {
	XFrameOptions string
}

type ApiConfig struct {
	Target *url.URL
}

type AuthConfig struct {
	skycmd.AuthFlags
	Target      *url.URL
	OAuthConfig *oauth2.Config
	HTTPClient  *http.Client
}

func NewHandler(
	logger lager.Logger,
	webConfig WebConfig,
	apiConfig ApiConfig,
	authConfig AuthConfig,
) (http.Handler, error) {

	tokenMiddleware := token.NewMiddleware(authConfig.AuthFlags.SecureCookies)

	skyServer, err := skyserver.NewSkyServer(&skyserver.SkyConfig{
		Logger:          logger.Session("sky"),
		TokenMiddleware: tokenMiddleware,
		OAuthConfig:     authConfig.OAuthConfig,
		HTTPClient:      authConfig.HTTPClient,
	})
	if err != nil {
		return nil, err
	}

	apiProxy := proxyhandler.NewApiHandler(logger, apiConfig.Target, webConfig.XFrameOptions, tokenMiddleware)
	authProxy := proxyhandler.NewAuthHandler(logger, authConfig.Target)

	legacyServer, err := legacyserver.NewLegacyServer(&legacyserver.LegacyConfig{
		Logger: logger.Session("legacy"),
	})
	if err != nil {
		return nil, err
	}

	indexHandler, err := indexhandler.NewHandler(logger, tokenMiddleware)
	if err != nil {
		return nil, err
	}

	webMux := http.NewServeMux()
	webMux.Handle("/api/", apiProxy)
	webMux.Handle("/sky/issuer/", authProxy)
	webMux.Handle("/sky/", skyserver.NewSkyHandler(skyServer))
	webMux.Handle("/auth/", legacyServer)
	webMux.Handle("/login", legacyServer)
	webMux.Handle("/logout", legacyServer)
	webMux.Handle("/public/", publichandler.NewHandler())
	webMux.Handle("/robots.txt", robotshandler.NewHandler())
	webMux.Handle("/", indexHandler)
	return webMux, nil
}
