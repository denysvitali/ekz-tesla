package ekz

import "net/http"

type ekzRoundTripper struct {
	inner  http.RoundTripper
	client *Client
}

func (e ekzRoundTripper) RoundTrip(request *http.Request) (*http.Response, error) {
	if e.client.token != "" {
		request.Header.Set("Authorization", "Token "+e.client.token)
	}
	request.Header.Set("User-Agent", "ekz-go")
	request.Header.Set("Device", "WEB")
	return e.inner.RoundTrip(request)
}

var _ http.RoundTripper = &ekzRoundTripper{}
