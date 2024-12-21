package sensonet

import (
	"errors"
	"fmt"
	"html"
	"io"
	"log"
	"net/http"
	"net/http/cookiejar"
	"net/url"
	"strings"
	"sync"
	"time"

	"github.com/coreos/go-oidc/v3/oidc"

	"dario.cat/mergo"
	"golang.org/x/oauth2"
)

const REALM_GERMANY = "vaillant-germany-b2c"

// Timeout is the default request timeout used by the Helper
var timeout = 10 * time.Second

func Oauth2ConfigForRealm(realm string) *oauth2.Config {
	return &oauth2.Config{
		Endpoint: oauth2.Endpoint{
			AuthURL:  fmt.Sprintf(AUTH_URL, realm),
			TokenURL: fmt.Sprintf(TOKEN_URL, realm),
		},
		Scopes: []string{oidc.ScopeOpenID, oidc.ScopeOfflineAccess},
	}
}

type Identity struct {
	client   *Helper
	trclient *Helper // seperate client for token refresh
	user     string
	password string
	realm    string
	oc       *oauth2.Config
}

func NewIdentity(client *http.Client, credentials *CredentialsStruct) (*Identity, error) {
	client.Jar, _ = cookiejar.New(nil)
	client.CheckRedirect = func(req *http.Request, via []*http.Request) error {
		return http.ErrUseLastResponse
	}

	trclient := NewHelper(newClient())
	trclient.Jar, _ = cookiejar.New(nil)
	trclient.CheckRedirect = func(req *http.Request, via []*http.Request) error {
		return http.ErrUseLastResponse
	}

	v := &Identity{
		client:   NewHelper(client),
		trclient: trclient,
		user:     credentials.User,
		password: credentials.Password,
		realm:    credentials.Realm,
		oc:       Oauth2ConfigForRealm(credentials.Realm),
	}

	return v, nil
}

// newClient creates http client with default transport
// func newClient(log *log.Logger) *http.Client {
func newClient() *http.Client {
	return &http.Client{
		Timeout: timeout,
		// Transport: httplogger.NewLoggedTransport(http.DefaultTransport, newLogger(log)),
	}
}

func (v *Identity) Login() (oauth2.TokenSource, error) {
	cv := oauth2.GenerateVerifier()

	data := url.Values{
		"response_type":         {"code"},
		"client_id":             {CLIENT_ID},
		"code":                  {"code_challenge"},
		"redirect_uri":          {"enduservaillant.page.link://login"},
		"code_challenge_method": {"S256"},
		"code_challenge":        {oauth2.S256ChallengeFromVerifier(cv)},
	}

	uri := fmt.Sprintf(AUTH_URL, v.realm) + "?" + data.Encode()
	req, _ := http.NewRequest("GET", uri, nil)

	resp, err := v.client.Do(req)
	if err != nil {
		return nil, fmt.Errorf("could not get code: %w", err)
	}
	defer resp.Body.Close()

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, fmt.Errorf("could not read response body: %w", err)
	}

	var code string
	if val, ok := resp.Header["Location"]; ok {
		parsedUrl, _ := url.Parse(val[0])
		code = parsedUrl.Query()["code"][0]
	}

	if code != "" {
		return nil, errors.New("missing code")
	}

	uri = computeLoginUrl(string(body), v.realm)
	if uri == "" {
		return nil, errors.New("missing login url")
	}

	params := url.Values{
		"username":     {v.user},
		"password":     {v.password},
		"credentialId": {""},
	}

	req, _ = http.NewRequest("POST", uri, strings.NewReader(params.Encode()))
	req.Header.Set("Content-Type", "application/x-www-form-urlencoded")

	resp, err = v.client.Do(req)
	if err != nil {
		return nil, fmt.Errorf("could not get code: %w", err)
	}

	location := resp.Header.Get("Location")
	if location == "" {
		return nil, errors.New("could not find location header")
	}

	parsedUrl, _ := url.Parse(location)
	code = parsedUrl.Query()["code"][0]

	// get token
	var token TokenRequestStruct
	params = url.Values{
		"grant_type":    {"authorization_code"},
		"client_id":     {CLIENT_ID},
		"code":          {code},
		"code_verifier": {cv},
		"redirect_uri":  {"enduservaillant.page.link://login"},
	}

	uri = fmt.Sprintf(TOKEN_URL, v.realm)
	req, _ = http.NewRequest("POST", uri, strings.NewReader(params.Encode()))
	req.Header.Set("Content-Type", "application/x-www-form-urlencoded")

	if err := v.client.DoJSON(req, &token); err != nil {
		return nil, fmt.Errorf("could not get token: %w", err)
	}

	token.Expiry = time.Now().Add(time.Duration(token.ExpiresIn) * time.Second)

	ts := refreshTokenSource(&token.Token, v)

	return ts, nil
}

type tokenRefresher interface {
	RefreshToken(token *oauth2.Token) (*oauth2.Token, error)
}

type TokenSource struct {
	mu        sync.Mutex
	token     *oauth2.Token
	refresher tokenRefresher
}

func refreshTokenSource(token *oauth2.Token, refresher tokenRefresher) oauth2.TokenSource {
	if token == nil {
		// allocate an (expired) token or mergeToken will fail
		token = new(oauth2.Token)
	}

	ts := &TokenSource{
		token:     token,
		refresher: refresher,
	}

	return ts
}

func (ts *TokenSource) Token() (*oauth2.Token, error) {
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

func computeLoginUrl(body, realm string) string {
	url := fmt.Sprintf(LOGIN_URL, realm)
	index1 := strings.Index(body, "authenticate?")
	if index1 < 0 {
		return ""
	}
	index2 := strings.Index(body[index1:], "\"")
	if index2 < 0 {
		return ""
	}
	return html.UnescapeString(url + body[index1+12:index1+index2])
}

func (v *Identity) RefreshToken(token *oauth2.Token) (*oauth2.Token, error) {
	var res TokenRequestStruct
	params := url.Values{
		"grant_type":    {"refresh_token"},
		"client_id":     {CLIENT_ID},
		"refresh_token": {token.RefreshToken},
	}

	uri := fmt.Sprintf(TOKEN_URL, v.realm)
	req, _ := http.NewRequest("POST", uri, strings.NewReader(params.Encode()))
	req.Header.Set("Content-Type", "application/x-www-form-urlencoded")

	if err := v.trclient.DoJSON(req, &res); err != nil {
		return nil, err
	}

	res.Expiry = time.Now().Add(time.Duration(res.ExpiresIn) * time.Second)
	log.Println("RefreshToken successful. New expiry:", res.Expiry)

	return &res.Token, nil
}
