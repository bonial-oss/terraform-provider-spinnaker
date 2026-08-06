package api

import (
	"net/http"
	"net/http/httptest"
	"path/filepath"
	"strings"
	"sync"
	"testing"

	"github.com/stretchr/testify/require"
)

// application is the JSON body returned by the fake Gate servers below.
const applicationBody = `{"name":"myapp","email":"owner@example.com"}`

// tokenBody is the JSON body returned by the fake token endpoints below.
const tokenBody = `{"access_token":"the-access-token","token_type":"Bearer","expires_in":3600}`

// tokenRequest captures the parameters a token endpoint was called with.
type tokenRequest struct {
	path         string
	grantType    string
	scope        string
	clientID     string
	clientSecret string
}

// recorder collects what the fake servers observed. Requests are served on
// their own goroutines, so access is guarded by a mutex and all assertions are
// made from the test goroutine once the requests completed.
type recorder struct {
	mu             sync.Mutex
	tokenRequests  []tokenRequest
	authorizations []string
	gatePaths      []string
}

// recordToken parses a token request and responds with tokenBody.
func (r *recorder) recordToken(w http.ResponseWriter, req *http.Request) {
	_ = req.ParseForm()

	// Client credentials are sent either as basic auth or as form values,
	// depending on what the token endpoint turns out to accept.
	clientID, clientSecret, ok := req.BasicAuth()
	if !ok {
		clientID, clientSecret = req.PostForm.Get("client_id"), req.PostForm.Get("client_secret")
	}

	r.mu.Lock()
	r.tokenRequests = append(r.tokenRequests, tokenRequest{
		path:         req.URL.Path,
		grantType:    req.PostForm.Get("grant_type"),
		scope:        req.PostForm.Get("scope"),
		clientID:     clientID,
		clientSecret: clientSecret,
	})
	r.mu.Unlock()

	w.Header().Set("Content-Type", "application/json")
	_, _ = w.Write([]byte(tokenBody))
}

// recordGate records the Authorization header of a Gate request and responds
// with applicationBody.
func (r *recorder) recordGate(w http.ResponseWriter, req *http.Request) {
	r.mu.Lock()
	r.authorizations = append(r.authorizations, req.Header.Get("Authorization"))
	r.gatePaths = append(r.gatePaths, req.URL.Path)
	r.mu.Unlock()

	w.Header().Set("Content-Type", "application/json")
	_, _ = w.Write([]byte(applicationBody))
}

func (r *recorder) snapshot() ([]tokenRequest, []string) {
	r.mu.Lock()
	defer r.mu.Unlock()

	return r.tokenRequests, r.authorizations
}

// newTestClient builds a client against endpoint. It points the Gate config
// file at a location that does not exist, so that the test never picks up the
// config of whoever runs it.
func newTestClient(t *testing.T, endpoint string, oauth2Config *OAuth2Config) *Client {
	t.Helper()

	client, err := NewClient(Config{
		Endpoint:       endpoint,
		ConfigLocation: filepath.Join(t.TempDir(), "config"),
		OAuth2:         oauth2Config,
	})
	require.NoError(t, err)

	return client
}

func TestNewClient_withOAuth2(t *testing.T) {
	var rec recorder

	tokenServer := httptest.NewServer(http.HandlerFunc(rec.recordToken))
	defer tokenServer.Close()

	gateServer := httptest.NewServer(http.HandlerFunc(rec.recordGate))
	defer gateServer.Close()

	client := newTestClient(t, gateServer.URL, &OAuth2Config{
		ClientID:     "the-client-id",
		ClientSecret: "the-client-secret",
		TokenURL:     tokenServer.URL + "/oauth2/token",
		Scopes:       []string{"scope1", "scope2"},
	})

	// Tokens are fetched lazily, constructing the client must not talk to the
	// token endpoint yet.
	tokenRequests, _ := rec.snapshot()
	require.Empty(t, tokenRequests)

	var app struct {
		Name  string
		Email string
	}

	require.NoError(t, GetApplication(client, "myapp", &app))
	require.Equal(t, "myapp", app.Name)
	require.Equal(t, "owner@example.com", app.Email)

	// A second call must reuse the cached token instead of fetching a new one.
	require.NoError(t, GetApplication(client, "myapp", &app))

	tokenRequests, authorizations := rec.snapshot()

	require.Equal(t, []tokenRequest{{
		path:         "/oauth2/token",
		grantType:    "client_credentials",
		scope:        "scope1 scope2",
		clientID:     "the-client-id",
		clientSecret: "the-client-secret",
	}}, tokenRequests)

	require.Equal(t, []string{
		"Bearer the-access-token",
		"Bearer the-access-token",
	}, authorizations)
}

func TestNewClient_withoutOAuth2(t *testing.T) {
	var rec recorder

	gateServer := httptest.NewServer(http.HandlerFunc(rec.recordGate))
	defer gateServer.Close()

	client := newTestClient(t, gateServer.URL, nil)

	var app struct {
		Name string
	}

	require.NoError(t, GetApplication(client, "myapp", &app))
	require.Equal(t, "myapp", app.Name)

	// Without OAuth2 no credentials are attached to the request.
	_, authorizations := rec.snapshot()
	require.Equal(t, []string{""}, authorizations)
}

func TestNewClient_oauth2DefaultTokenURL(t *testing.T) {
	var rec recorder

	// One server acts as both Gate and token endpoint to assert that the token
	// URL defaults to the /oauth2/token endpoint of the Gate URL.
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, req *http.Request) {
		if strings.HasSuffix(req.URL.Path, "/oauth2/token") {
			rec.recordToken(w, req)
			return
		}

		rec.recordGate(w, req)
	}))
	defer server.Close()

	client := newTestClient(t, server.URL, &OAuth2Config{
		ClientID:     "the-client-id",
		ClientSecret: "the-client-secret",
	})

	var app struct {
		Name string
	}

	require.NoError(t, GetApplication(client, "myapp", &app))

	tokenRequests, authorizations := rec.snapshot()

	require.Len(t, tokenRequests, 1)
	require.Equal(t, "/oauth2/token", tokenRequests[0].path)
	// No scopes are requested by default.
	require.Empty(t, tokenRequests[0].scope)
	require.Equal(t, []string{"Bearer the-access-token"}, authorizations)
}

func TestNewClient_trailingSlashInEndpoint(t *testing.T) {
	var rec recorder

	server := httptest.NewServer(http.HandlerFunc(rec.recordGate))
	defer server.Close()

	client := newTestClient(t, server.URL+"/", nil)

	var app struct {
		Name string
	}

	require.NoError(t, GetApplication(client, "myapp", &app))

	// The trailing slash must not end up doubled in the request path.
	require.Equal(t, []string{"/applications/myapp"}, rec.gatePaths)
}

func TestParseDefaultHeaders(t *testing.T) {
	tests := []struct {
		name        string
		input       string
		expected    map[string]string
		expectedErr string
	}{
		{
			name:     "empty",
			input:    "",
			expected: map[string]string{},
		},
		{
			name:     "single header",
			input:    "Api-Key=abc123",
			expected: map[string]string{"Api-Key": "abc123"},
		},
		{
			name:     "multiple headers with surrounding whitespace",
			input:    "header1=value1, header2 = value2",
			expected: map[string]string{"header1": "value1", "header2": "value2"},
		},
		{
			name:     "value containing an equals sign",
			input:    "Authorization=Basic dXNlcjpwYXNz==",
			expected: map[string]string{"Authorization": "Basic dXNlcjpwYXNz=="},
		},
		{
			name:        "missing equals sign",
			input:       "header1=value1,header2",
			expectedErr: "bad default_headers value, use key=value form: header2",
		},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			headers, err := parseDefaultHeaders(test.input)

			if test.expectedErr != "" {
				require.EqualError(t, err, test.expectedErr)
				return
			}

			require.NoError(t, err)
			require.Equal(t, test.expected, headers)
		})
	}
}
