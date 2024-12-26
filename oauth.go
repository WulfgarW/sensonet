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

	"github.com/coreos/go-oidc/v3/oidc"

	"golang.org/x/oauth2"
)

const REALM_GERMANY = "vaillant-germany-b2c"

type Oauth2Config struct {
	*oauth2.Config
}

func Oauth2ConfigForRealm(realm string) *Oauth2Config {
	if realm == "" {
		realm = REALM_GERMANY
	}
	return &Oauth2Config{
		Config: &oauth2.Config{
			ClientID: CLIENT_ID,
			Endpoint: oauth2.Endpoint{
				AuthURL:  fmt.Sprintf(AUTH_URL, realm),
				TokenURL: fmt.Sprintf(TOKEN_URL, realm),
			},
			RedirectURL: REDIRECT_URL,
			Scopes:      []string{oidc.ScopeOpenID, oidc.ScopeOfflineAccess},
		},
	}
}

func (oc *Oauth2Config) PasswordCredentialsToken(ctx context.Context, username string, password string) (*oauth2.Token, error) {
	client, ok := ctx.Value(oauth2.HTTPClient).(*http.Client)
	if !ok {
		client = new(http.Client)
	}

	client.Jar, _ = cookiejar.New(nil)
	client.CheckRedirect = func(req *http.Request, via []*http.Request) error {
		return http.ErrUseLastResponse
	}

	cv := oauth2.GenerateVerifier()

	uri := oc.AuthCodeURL(cv, oauth2.S256ChallengeOption(cv), oauth2.SetAuthURLParam("code", "code_challenge"))
	resp, err := client.Get(uri)
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
		"username":     {username},
		"password":     {password},
		"credentialId": {""},
	}

	req, _ := http.NewRequest("POST", uri, strings.NewReader(params.Encode()))
	req.Header.Set("Content-Type", "application/x-www-form-urlencoded")

	resp, err = client.Do(req)
	if err != nil {
		return nil, err
	}

	location, _ := url.Parse(resp.Header.Get("Location"))
	code := location.Query().Get("code")
	if code == "" {
		return nil, errors.New("could not get code")
	}

	return oc.Exchange(ctx, code, oauth2.VerifierOption(cv))
}
