package api

import (
	"context"
	"crypto/tls"
	"fmt"
	"net/http"
	"strings"

	"github.com/spinnaker/spin/cmd/gateclient"
	gateapi "github.com/spinnaker/spin/gateapi"
	"golang.org/x/oauth2"
	"golang.org/x/oauth2/clientcredentials"
)

const userAgent = "terraform-provider-spinnaker"

// OAuth2Config configures authentication against Gate using the OAuth2 client
// credentials flow.
type OAuth2Config struct {
	ClientID     string
	ClientSecret string

	// TokenURL is the OAuth2 token endpoint. Defaults to the /oauth2/token
	// endpoint of the Gate API if empty.
	TokenURL string

	// Scopes to request for the access token. May be empty.
	Scopes []string
}

// Config configures the Gate API client.
type Config struct {
	// Endpoint is the base URL of the Gate API.
	Endpoint string

	// DefaultHeaders holds comma separated key=value pairs which are sent with
	// every request.
	DefaultHeaders string

	// ConfigLocation is the path to the Gate config file. Defaults to
	// ~/.spin/config if empty. A missing file is not an error.
	ConfigLocation string

	// IgnoreCertErrors disables TLS certificate verification.
	IgnoreCertErrors bool

	// OAuth2 enables authentication via the OAuth2 client credentials flow if
	// non-nil. It takes precedence over the login based auth mechanisms from
	// the Gate config file.
	OAuth2 *OAuth2Config
}

// Client provides access to the Spinnaker Gate API.
type Client struct {
	*gateapi.APIClient

	// Context is passed to every Gate API call. It carries the credentials
	// that gateapi turns into request headers.
	Context context.Context
}

// NewClient creates a Gate API client from cfg.
//
// Credentials come either from cfg.OAuth2, which uses the OAuth2 client
// credentials flow, or from the auth section of the Gate config file, which
// supports the same mechanisms as the spin CLI.
func NewClient(cfg Config) (*Client, error) {
	defaultHeaders, err := parseDefaultHeaders(cfg.DefaultHeaders)
	if err != nil {
		return nil, err
	}

	gateConfig, err := loadGateConfig(cfg.ConfigLocation)
	if err != nil {
		return nil, err
	}

	authConfig := gateConfig.Auth
	endpoint := strings.TrimSuffix(cfg.Endpoint, "/")

	// Sets up client certificates from the x509 auth config, plus a cookie jar
	// to hold the session established by the login based auth mechanisms.
	httpClient, err := gateclient.InitializeHTTPClient(authConfig)
	if err != nil {
		return nil, fmt.Errorf("failed to initialize http client: %w", err)
	}

	if authConfig != nil && authConfig.IgnoreRedirects {
		httpClient.CheckRedirect = func(*http.Request, []*http.Request) error {
			return http.ErrUseLastResponse
		}
	}

	if cfg.IgnoreCertErrors {
		transport := httpClient.Transport.(*http.Transport)
		if transport.TLSClientConfig == nil {
			transport.TLSClientConfig = &tls.Config{}
		}

		// Explicitly opted into via the ignore_cert_errors attribute.
		transport.TLSClientConfig.InsecureSkipVerify = true //nolint:gosec
	}

	// Adds IAP and basic auth credentials to the context.
	ctx, err := gateclient.ContextWithAuth(context.Background(), authConfig)
	if err != nil {
		return nil, err
	}

	if cfg.OAuth2 != nil {
		ctx = withClientCredentials(ctx, httpClient, endpoint, cfg.OAuth2)
	} else if err := authenticate(httpClient, endpoint, gateConfig, cfg.ConfigLocation); err != nil {
		return nil, err
	}

	apiClient := gateapi.NewAPIClient(&gateapi.Configuration{
		BasePath:      endpoint,
		DefaultHeader: defaultHeaders,
		UserAgent:     userAgent,
		HTTPClient:    httpClient,
	})

	return &Client{APIClient: apiClient, Context: ctx}, nil
}

// withClientCredentials adds an OAuth2 token source using the client
// credentials flow to ctx. gateapi picks the token source up from there and
// sets the Authorization header on every request. Tokens are fetched lazily on
// the first request and refreshed once they expire.
func withClientCredentials(ctx context.Context, httpClient *http.Client, endpoint string, cfg *OAuth2Config) context.Context {
	tokenURL := cfg.TokenURL
	if tokenURL == "" {
		tokenURL = endpoint + "/oauth2/token"
	}

	// Fetching tokens through httpClient makes the token endpoint honour the
	// same TLS settings as the Gate requests.
	tokenCtx := context.WithValue(ctx, oauth2.HTTPClient, httpClient)

	oauth2Config := &clientcredentials.Config{
		ClientID:     cfg.ClientID,
		ClientSecret: cfg.ClientSecret,
		TokenURL:     tokenURL,
		Scopes:       cfg.Scopes,
	}

	return context.WithValue(ctx, gateapi.ContextOAuth2, oauth2Config.TokenSource(tokenCtx))
}

// parseDefaultHeaders parses comma separated key=value pairs.
func parseDefaultHeaders(defaultHeaders string) (map[string]string, error) {
	headers := make(map[string]string)

	if defaultHeaders == "" {
		return headers, nil
	}

	for _, pair := range strings.Split(defaultHeaders, ",") {
		name, value, found := strings.Cut(pair, "=")
		if !found {
			return nil, fmt.Errorf("bad default_headers value, use key=value form: %s", pair)
		}

		headers[strings.TrimSpace(name)] = strings.TrimSpace(value)
	}

	return headers, nil
}
