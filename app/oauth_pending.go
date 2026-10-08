package app

import "github.com/hkdb/aerion/internal/oauth2"

// The pending OAuth fields are written by the callback goroutine and read
// and cleared by bindings, which Wails runs on their own goroutines, so all
// access goes through these helpers under pendingOAuthMu.

func (a *App) setPendingOAuth(tokens *oauth2.TokenResponse, email string) {
	a.pendingOAuthMu.Lock()
	defer a.pendingOAuthMu.Unlock()
	a.pendingOAuthTokens, a.pendingOAuthEmail = tokens, email
}

func (a *App) pendingOAuth() (*oauth2.TokenResponse, string, *oauth2.ProviderConfig) {
	a.pendingOAuthMu.Lock()
	defer a.pendingOAuthMu.Unlock()
	return a.pendingOAuthTokens, a.pendingOAuthEmail, a.pendingCustomProvider
}

// clearPendingOAuth drops the pending tokens if they are still tokens, so a
// newer flow that finished meanwhile keeps its own.
func (a *App) clearPendingOAuth(tokens *oauth2.TokenResponse) {
	a.pendingOAuthMu.Lock()
	defer a.pendingOAuthMu.Unlock()
	if a.pendingOAuthTokens == tokens {
		a.pendingOAuthTokens, a.pendingOAuthEmail = nil, ""
	}
}

func (a *App) setPendingCustomProvider(p *oauth2.ProviderConfig) {
	a.pendingOAuthMu.Lock()
	defer a.pendingOAuthMu.Unlock()
	a.pendingCustomProvider = p
}

func (a *App) setPendingContactSourceOAuth(tokens *oauth2.TokenResponse, email, provider string) {
	a.pendingOAuthMu.Lock()
	defer a.pendingOAuthMu.Unlock()
	a.setPendingContactSourceOAuthLocked(tokens, email, provider)
}

func (a *App) pendingContactSourceOAuth() (*oauth2.TokenResponse, string, string) {
	a.pendingOAuthMu.Lock()
	defer a.pendingOAuthMu.Unlock()
	return a.pendingContactSourceOAuthTokens, a.pendingContactSourceOAuthEmail, a.pendingContactSourceOAuthProvider
}

// clearPendingContactSourceOAuth drops the pending contact-source tokens if
// they are still tokens.
func (a *App) clearPendingContactSourceOAuth(tokens *oauth2.TokenResponse) {
	a.pendingOAuthMu.Lock()
	defer a.pendingOAuthMu.Unlock()
	if a.pendingContactSourceOAuthTokens == tokens {
		a.setPendingContactSourceOAuthLocked(nil, "", "")
	}
}

func (a *App) setPendingContactSourceOAuthLocked(tokens *oauth2.TokenResponse, email, provider string) {
	a.pendingContactSourceOAuthTokens = tokens
	a.pendingContactSourceOAuthEmail = email
	a.pendingContactSourceOAuthProvider = provider
}
