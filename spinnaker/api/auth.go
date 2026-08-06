package api

import (
	"fmt"
	"log"
	"net/http"
	"os"
	"path/filepath"

	"github.com/spinnaker/spin/cmd/gateclient"
	"github.com/spinnaker/spin/config"
	"github.com/spinnaker/spin/config/auth"
	"sigs.k8s.io/yaml"
)

// gateConfigFileMode is the file mode used when creating a Gate config file.
// It matches the mode used by the spin CLI so that cached tokens are only
// readable by the user owning the file.
const gateConfigFileMode os.FileMode = 0600

// loadGateConfig reads the Gate config file from location, defaulting to
// ~/.spin/config. A missing file yields an empty configuration, as most
// provider setups do not have one.
func loadGateConfig(location string) (config.Config, error) {
	var cfg config.Config

	location, err := gateConfigLocation(location)
	if err != nil {
		return cfg, err
	}

	contents, err := os.ReadFile(location)
	if os.IsNotExist(err) {
		return cfg, nil
	} else if err != nil {
		return cfg, fmt.Errorf("failed to read Gate config file %s: %w", location, err)
	}

	// Environment variables are expanded to allow keeping credentials out of
	// the config file.
	if err := yaml.UnmarshalStrict([]byte(os.ExpandEnv(string(contents))), &cfg); err != nil {
		return cfg, fmt.Errorf("failed to parse Gate config file %s: %w", location, err)
	}

	return cfg, nil
}

// authenticate performs the login based authentication mechanisms configured
// in the Gate config file, that is the OAuth2 authorization code flow, Google
// service accounts and LDAP. Tokens obtained along the way are cached back
// into the config file, just like the spin CLI does.
func authenticate(httpClient *http.Client, endpoint string, cfg config.Config, location string) error {
	if cfg.Auth == nil {
		return nil
	}

	discardUnusableToken(cfg.Auth)

	updated, err := gateclient.Authenticate(logOutput, httpClient, endpoint, cfg.Auth)
	if err != nil {
		return fmt.Errorf("authentication failed: %w", err)
	}

	if !updated {
		return nil
	}

	if err := cacheGateConfig(cfg, location); err != nil {
		// Not being able to cache the token only costs us a round trip on the
		// next run, so this must not fail the operation.
		log.Printf("[WARN] failed to cache token in Gate config file: %v", err)
	}

	return nil
}

// discardUnusableToken drops a cached OAuth2 token that is expired and cannot
// be refreshed. Keeping it would make the token refresh fail hard, whereas
// dropping it triggers a new authorization round trip.
func discardUnusableToken(authConfig *auth.Config) {
	if authConfig.OAuth2 == nil || authConfig.OAuth2.CachedToken == nil {
		return
	}

	token := authConfig.OAuth2.CachedToken
	if token.AccessToken != "" && !token.Valid() && token.RefreshToken == "" {
		authConfig.OAuth2.CachedToken = nil
	}
}

// cacheGateConfig writes cfg back to the Gate config file at location,
// preserving the file mode if the file already exists.
func cacheGateConfig(cfg config.Config, location string) error {
	location, err := gateConfigLocation(location)
	if err != nil {
		return err
	}

	buf, err := yaml.Marshal(&cfg)
	if err != nil {
		return err
	}

	mode := gateConfigFileMode

	if info, err := os.Stat(location); err == nil {
		mode = info.Mode()
	} else if !os.IsNotExist(err) {
		return err
	}

	return os.WriteFile(location, buf, mode)
}

// gateConfigLocation returns location, or the default Gate config file
// location if it is empty.
func gateConfigLocation(location string) (string, error) {
	if location != "" {
		return location, nil
	}

	home, err := os.UserHomeDir()
	if err != nil {
		return "", fmt.Errorf("failed to determine user home directory: %w", err)
	}

	return filepath.Join(home, ".spin", "config"), nil
}

// logOutput receives status messages and prompts from the spin authentication
// helpers.
func logOutput(msg string) {
	log.Printf("[INFO] %s", msg)
}
