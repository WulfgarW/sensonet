package sensonet

import "net/http"

type transport struct {
	http.RoundTripper
}

func (t *transport) RoundTrip(req *http.Request) (*http.Response, error) {
	for k, v := range (http.Header{
		"Accept-Language":           {"en-GB"},
		"Accept":                    {"application/json, text/plain, */*"},
		"x-app-identifier":          {"VAILLANT"},
		"x-client-locale":           {"en-GB"},
		"x-idm-identifier":          {"KEYCLOAK"},
		"ocp-apim-subscription-key": {"1e0a2f3511fb4c5bbb1c7f9fedd20b1c"},
	}) {
		for _, vv := range v {
			req.Header.Add(k, vv)
		}
	}

	resp, err := t.RoundTripper.RoundTrip(req)
	if err == nil {
		err = ResponseError(resp)
	}

	return resp, err
}
