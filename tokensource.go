package sensonet

// Copied from https://github.com/evcc-io/evcc

import (
	"errors"
	"sync"

	"dario.cat/mergo"
	"golang.org/x/oauth2"
)

type tokenRefresher interface {
	RefreshToken(token *oauth2.Token) (*oauth2.Token, error)
}

type tokenSource struct {
	mu        sync.Mutex
	token     *oauth2.Token
	refresher tokenRefresher
}

func refreshTokenSource(token *oauth2.Token, refresher tokenRefresher) oauth2.TokenSource {
	if token == nil {
		// allocate an (expired) token or mergeToken will fail
		token = new(oauth2.Token)
	}

	ts := &tokenSource{
		token:     token,
		refresher: refresher,
	}

	return ts
}

func (ts *tokenSource) Token() (*oauth2.Token, error) {
	ts.mu.Lock()
	defer ts.mu.Unlock()

	if ts.token.Valid() {
		return ts.token, nil
	}

	token, err := ts.refresher.RefreshToken(ts.token)
	if err != nil {
		return ts.token, err
	}

	if token.AccessToken == "" {
		err = errors.New("token refresh failed to obtain access token")
	} else {
		err = mergo.Merge(ts.token, token, mergo.WithOverride)
	}

	return ts.token, err
}
