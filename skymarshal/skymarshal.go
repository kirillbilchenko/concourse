package skymarshal

import (
	"crypto/rand"
	"crypto/rsa"
	"net/http"
	"net/url"

	"code.cloudfoundry.org/lager"
	"github.com/concourse/concourse/skymarshal/dexserver"
	"github.com/concourse/concourse/skymarshal/skycmd"
	"github.com/concourse/concourse/skymarshal/storage"
	"github.com/concourse/flag"
)

type Config struct {
	Logger      lager.Logger
	Flags       skycmd.AuthFlags
	ExternalURL *url.URL
	HTTPClient  *http.Client
	Storage     storage.Storage
}

type Server struct {
	http.Handler
}

func NewServer(config *Config) (*Server, error) {

	signingKey, err := loadOrGenerateSigningKey(config.Flags.SigningKey)
	if err != nil {
		return nil, err
	}

	issuerPath := "/sky/issuer"
	issuerURL := config.ExternalURL.String() + issuerPath
	redirectURL := config.ExternalURL.String() + "/sky/callback"

	dexServer, err := dexserver.NewDexServer(&dexserver.DexConfig{
		Logger:     config.Logger.Session("dex"),
		Flags:      config.Flags,
		IssuerURL:  issuerURL,
		WebHostURL: issuerPath,
		SigningKey: signingKey,
		Storage:    config.Storage,
		Clients: []*dexserver.DexClient{
			{
				ClientID:     "concourse-web",
				ClientSecret: "Y29uY291cnNlLXdlYgo=",
				RedirectURL:  redirectURL,
			},
			{
				ClientID:     "concourse-worker",
				ClientSecret: "Y29uY291cnNlLXdvcmtlcgo=",
			},
			{
				ClientID:     "fly",
				ClientSecret: "Zmx5Cg==",
				RedirectURL:  redirectURL,
			},
		},
	})
	if err != nil {
		return nil, err
	}

	handler := http.NewServeMux()
	handler.Handle("/sky/issuer/", dexServer)

	return &Server{handler}, nil
}

func loadOrGenerateSigningKey(keyFlag *flag.PrivateKey) (*rsa.PrivateKey, error) {
	if keyFlag != nil && keyFlag.PrivateKey != nil {
		return keyFlag.PrivateKey, nil
	}

	return rsa.GenerateKey(rand.Reader, 2048)
}
