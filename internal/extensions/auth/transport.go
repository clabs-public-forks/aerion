package auth

import (
	"errors"
	"fmt"
	"io"
	"net/http"
	"strings"
	"sync"
	"time"

	coreapi "github.com/hkdb/aerion/internal/core/api/v1"
	"github.com/hkdb/aerion/internal/credentials"
	"github.com/hkdb/aerion/internal/oauth2"
)

// bearerRefreshTransport is an http.RoundTripper that injects the current
// access token on each request and transparently refreshes it on 401
// responses. It serializes refreshes per (accountID, clientConfigID) so that
// N concurrent requests with the same expired token cause exactly one refresh.
type bearerRefreshTransport struct {
	base           http.RoundTripper
	credStore      *credentials.Store
	oauthManager   *oauth2.Manager
	accountID      string
	clientConfigID string
	// scopes the client was issued for; reported back if the refresh token
	// is rejected and the user must re-consent.
	scopes []coreapi.AuthScope

	mu sync.Mutex // guards token retrieval/refresh
}

// RoundTrip implements http.RoundTripper.
func (t *bearerRefreshTransport) RoundTrip(req *http.Request) (*http.Response, error) {
	tokens, err := t.credStore.GetOAuthTokensForClientConfig(t.accountID, t.clientConfigID)
	if err != nil {
		return nil, fmt.Errorf("auth broker: read tokens: %w", err)
	}

	resp, err := t.do(req, tokens.AccessToken)
	if err != nil {
		return nil, err
	}
	if resp.StatusCode != http.StatusUnauthorized {
		return resp, nil
	}

	// 401: drain + close body before retrying
	_, _ = io.Copy(io.Discard, resp.Body)
	_ = resp.Body.Close()

	// Refresh under lock to avoid thundering herd.
	t.mu.Lock()
	defer t.mu.Unlock()

	// Re-read tokens in case another goroutine refreshed already.
	tokens, err = t.credStore.GetOAuthTokensForClientConfig(t.accountID, t.clientConfigID)
	if err != nil {
		return nil, fmt.Errorf("auth broker: re-read tokens before refresh: %w", err)
	}

	provider, err := t.resolveProvider()
	if err != nil {
		return nil, err
	}

	refreshed, err := t.oauthManager.RefreshTokenWithProvider(provider, tokens.RefreshToken)
	if errors.Is(err, oauth2.ErrInvalidGrant) && slotSupportsIncrementalConsent(t.clientConfigID) {
		// The refresh token is dead (expired/revoked). Surface it as a consent
		// requirement so extensions offer their grant flow instead of a raw
		// error the user has no way to act on.
		return nil, &coreapi.ErrAdditionalConsentRequired{
			AccountID:      t.accountID,
			ClientConfigID: coreapi.ClientConfigID(t.clientConfigID),
			MissingScopes:  t.scopes,
		}
	}
	if err != nil {
		return nil, fmt.Errorf("auth broker: refresh: %w", err)
	}

	expiresAt := time.Now().Add(time.Duration(refreshed.ExpiresIn) * time.Second)
	// Persist the rotated refresh token too — providers like Microsoft and custom
	// OIDC return a NEW refresh_token on refresh; dropping it breaks the next one.
	if err := t.credStore.UpdateOAuthTokensForClientConfig(t.accountID, t.clientConfigID, refreshed.AccessToken, refreshed.RefreshToken, expiresAt); err != nil {
		return nil, fmt.Errorf("auth broker: persist refreshed tokens: %w", err)
	}

	return t.do(req, refreshed.AccessToken)
}

// slotSupportsIncrementalConsent reports whether an extension grant flow
// (StartIncrementalConsent) can write fresh tokens into the given slot. That
// flow resolves the slot via oauth2.GetProvider, which only knows the
// extension-owned slots (*-contacts, *-calendar). Mail slots — custom-mail
// and the google-/microsoft-mail slots extensions borrow core-routed scopes
// from — can only be repaired by mail re-authorization, so a dead refresh
// token there must keep its original error rather than prompt a grant that
// can't fix it.
func slotSupportsIncrementalConsent(clientConfigID string) bool {
	_, err := oauth2.GetProvider(clientConfigID)
	return err == nil
}

// resolveProvider returns the OAuth2 provider config for refreshing this account's
// token. Custom ("bring your own app") accounts store their endpoints/creds per account
// (no static client-config registration), so they resolve from GetCustomOAuthProvider;
// shipped providers use the static client-config lookup.
func (t *bearerRefreshTransport) resolveProvider() (oauth2.ProviderConfig, error) {
	if t.clientConfigID == "custom-mail" {
		cfg, ok, err := t.credStore.GetCustomOAuthProvider(t.accountID)
		if err != nil {
			return oauth2.ProviderConfig{}, fmt.Errorf("auth broker: get custom provider for %s: %w", t.accountID, err)
		}
		if !ok {
			return oauth2.ProviderConfig{}, fmt.Errorf("auth broker: no custom provider configured for account %s", t.accountID)
		}
		return oauth2.CustomProviderConfig(cfg.AuthURL, cfg.TokenURL, cfg.UserinfoEndpoint, cfg.Scopes, cfg.ClientID, cfg.ClientSecret), nil
	}

	provider, err := oauth2.GetProviderForClientConfig(t.clientConfigID)
	if err != nil {
		return oauth2.ProviderConfig{}, fmt.Errorf("auth broker: resolve provider for %s: %w", t.clientConfigID, err)
	}
	return provider, nil
}

// do clones the request, sets the bearer header, and dispatches via the base
// transport. Cloning is necessary because RoundTrip implementations must not
// mutate the input request.
func (t *bearerRefreshTransport) do(req *http.Request, accessToken string) (*http.Response, error) {
	cloned := req.Clone(req.Context())
	if cloned.Header == nil {
		cloned.Header = make(http.Header)
	}
	// Don't double-set if the caller already added one (rare; just in case).
	if strings.TrimSpace(cloned.Header.Get("Authorization")) == "" {
		cloned.Header.Set("Authorization", "Bearer "+accessToken)
	}
	base := t.base
	if base == nil {
		base = http.DefaultTransport
	}
	return base.RoundTrip(cloned)
}
