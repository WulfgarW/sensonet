package sensonet

import (
	"context"
	"errors"
	"fmt"
	"io"
	"net/http"
	"net/http/cookiejar"
	"net/url"
	"regexp"
	"strings"
	"time"

	"github.com/coreos/go-oidc/v3/oidc"

	"golang.org/x/oauth2"
)

const REALM_GERMANY = "vaillant-germany-b2c"

// Timeout is the default request timeout used by the Helper
var timeout = 10 * time.Second

func Oauth2ConfigForRealm(realm string) *oauth2.Config {
	if realm == "" {
		realm = REALM_GERMANY
	}
	return &oauth2.Config{
		ClientID: CLIENT_ID,
		Endpoint: oauth2.Endpoint{
			AuthURL:  fmt.Sprintf(AUTH_URL, realm),
			TokenURL: fmt.Sprintf(TOKEN_URL, realm),
		},
		RedirectURL: REDIRECT_URL,
		Scopes:      []string{oidc.ScopeOpenID, oidc.ScopeOfflineAccess},
	}
}

type Identity struct {
	client *http.Client
	oc     *oauth2.Config
}

func NewIdentity(client *http.Client, realm string) (*Identity, error) {
	client.Jar, _ = cookiejar.New(nil)
	client.CheckRedirect = func(req *http.Request, via []*http.Request) error {
		return http.ErrUseLastResponse
	}

	v := &Identity{
		client: client,
		oc:     Oauth2ConfigForRealm(realm),
	}

	return v, nil
}

func (v *Identity) Login(user, password string) (oauth2.TokenSource, error) {
	cv := oauth2.GenerateVerifier()
	ctx := context.WithValue(context.TODO(), oauth2.HTTPClient, v.client)

	uri := v.oc.AuthCodeURL(cv, oauth2.S256ChallengeOption(cv), oauth2.SetAuthURLParam("code", "code_challenge"))
	resp, err := v.client.Get(uri)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, err
	}

	match := regexp.MustCompile(`action\s*=\s*"(.+?)"`).FindStringSubmatch(string(body))
	if len(match) < 2 {
		return nil, errors.New("missing login form action")
	}
	uri = match[1]

	params := url.Values{
		"username":     {user},
		"password":     {password},
		"credentialId": {""},
	}

	req, _ := http.NewRequest("POST", uri, strings.NewReader(params.Encode()))
	req.Header.Set("Content-Type", "application/x-www-form-urlencoded")

	resp, err = v.client.Do(req)
	if err != nil {
		return nil, err
	}

	location, _ := url.Parse(resp.Header.Get("Location"))
	code := location.Query().Get("code")
	if code == "" {
		return nil, errors.New("could not get code")
	}

	token, err := v.oc.Exchange(ctx, code, oauth2.VerifierOption(cv))
	if err != nil {
		return nil, fmt.Errorf("could not get token: %w", err)
	}

	ts := v.oc.TokenSource(ctx, token)

	return ts, nil
}
