package sensonet

import (
	"bytes"
	"context"
	"crypto/pbkdf2"
	"crypto/sha256"
	"crypto/sha512"
	"encoding/base64"
	"encoding/binary"
	"encoding/hex"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/http"
	"net/http/cookiejar"
	"net/url"
	"regexp"
	"slices"
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

	altcha, altchaErr := getAndSolveAltchaChallenge(client)
	if altcha != "" {
		params.Set("altcha", altcha)
	}

	req, _ := http.NewRequest("POST", uri, strings.NewReader(params.Encode()))
	req.Header.Set("Content-Type", "application/x-www-form-urlencoded")

	resp, err = client.Do(req)
	if err != nil {
		if altchaErr != nil {
			return nil, errors.New("Could not fetch or solve ALTCHA challenge. Tried to continue without it, but without success.")
		}
		return nil, err
	}

	location, _ := url.Parse(resp.Header.Get("Location"))
	code := location.Query().Get("code")
	if code == "" {
		return nil, errors.New("could not get code")
	}

	return oc.Exchange(ctx, code, oauth2.VerifierOption(cv))
}

func getAndSolveAltchaChallenge(client *http.Client) (string, error) {
	resp, err := client.Get(ALTCHA_CHALLENGE_URL)
	if err != nil {
		return "", err
	}
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusOK {
		return "", fmt.Errorf("status %s", resp.Status)
	}
	challenge, err := io.ReadAll(resp.Body)
	if err != nil {
		return "", err
	}

	return solveAltchaChallenge(challenge)
}

func solveAltchaChallenge(challenge []byte) (string, error) {
	var challengeStruct struct {
		Parameters json.RawMessage `json:"parameters"`
		Signature  string          `json:"signature"`
	}
	if err := json.Unmarshal(challenge, &challengeStruct); err != nil {
		return "", err
	}

	var parametersStruct struct {
		Algorithm string `json:"algorithm"`
		Cost      int    `json:"cost"`
		KeyLength int    `json:"keyLength"`
		KeyPrefix string `json:"keyPrefix"`
		Nonce     string `json:"nonce"`
		Salt      string `json:"salt"`
	}
	if err := json.Unmarshal(challengeStruct.Parameters, &parametersStruct); err != nil {
		return "", err
	}

	nonce, err := hex.DecodeString(parametersStruct.Nonce)
	if err != nil {
		return "", err
	}
	salt, err := hex.DecodeString(parametersStruct.Salt)
	if err != nil {
		return "", err
	}
	prefix, err := hex.DecodeString(parametersStruct.KeyPrefix)
	if err != nil {
		return "", err
	}

	newHash := sha256.New
	switch parametersStruct.Algorithm {
	case "PBKDF2/SHA-512":
		newHash = sha512.New
	case "PBKDF2/SHA-384":
		newHash = sha512.New384
	}

	keyLength := parametersStruct.KeyLength
	if keyLength == 0 {
		keyLength = 32
	}

	for counter := uint32(0); ; counter++ {
		password := binary.BigEndian.AppendUint32(slices.Clone(nonce), counter)

		key, err := pbkdf2.Key(newHash, string(password), salt, parametersStruct.Cost, keyLength)
		if err != nil {
			return "", err
		}

		if !bytes.HasPrefix(key, prefix) {
			continue
		}

		payload := struct {
			Challenge struct {
				Parameters json.RawMessage `json:"parameters"`
				Signature  string          `json:"signature"`
			} `json:"challenge"`
			Solution struct {
				Counter    uint32 `json:"counter"`
				DerivedKey string `json:"derivedKey"`
				Time       int    `json:"time"`
			} `json:"solution"`
		}{}
		payload.Challenge.Parameters = challengeStruct.Parameters
		payload.Challenge.Signature = challengeStruct.Signature
		payload.Solution.Counter = counter
		payload.Solution.DerivedKey = hex.EncodeToString(key)

		challengeResult, err := json.Marshal(payload)
		if err != nil {
			return "", err
		}

		return base64.StdEncoding.EncodeToString(challengeResult), nil
	}
}
