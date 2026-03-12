package provider

import "net/http"

type authedTransport struct {
	token   string
	wrapped http.RoundTripper
}

func (t *authedTransport) RoundTrip(req *http.Request) (*http.Response, error) {
	req.Header.Set("Authorization", "Bearer "+t.token)
	req.Header.Set("Project-Access-Token", t.token)

	return t.wrapped.RoundTrip(req)
}
