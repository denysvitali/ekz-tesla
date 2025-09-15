package ekz

import (
	"bytes"
	"fmt"
	"io"
	"net/http"
)

type ekzRoundTripper struct {
	inner  http.RoundTripper
	client *Client
}

func (e ekzRoundTripper) RoundTrip(request *http.Request) (*http.Response, error) {
	// Clone the request in case we need to retry
	originalBody, err := cloneRequestBody(request)
	if err != nil {
		return nil, err
	}

	// Set auth headers
	token := e.client.getToken()
	if token != "" {
		request.Header.Set("Authorization", "Token "+token)
	}
	request.Header.Set("User-Agent", "ekz-go")
	request.Header.Set("Device", "WEB")

	// Make the request
	response, err := e.inner.RoundTrip(request)
	if err != nil {
		return response, err
	}

	// If we get 401 and have credentials, try to refresh token and retry
	if response.StatusCode == http.StatusUnauthorized && e.client.config.Username != "" && e.client.config.Password != "" {
		log.Debugf("Received 401, attempting to refresh token")

		// Close the original response body
		_ = response.Body.Close()

		// Attempt to refresh the token
		if err := e.client.refreshTokenIfNeeded(); err != nil {
			log.Errorf("Failed to refresh token: %v", err)
			// Return a proper error instead of the 401 response to stop infinite loops
			return nil, fmt.Errorf("authentication failed after token refresh attempts: %w", err)
		}

		// Clone the request again for retry
		retryReq, err := cloneRequest(request, originalBody)
		if err != nil {
			return nil, fmt.Errorf("failed to clone request for retry: %w", err)
		}

		// Set new token and retry
		newToken := e.client.getToken()
		if newToken != "" {
			retryReq.Header.Set("Authorization", "Token "+newToken)
		}

		log.Debugf("Retrying request with refreshed token")
		return e.inner.RoundTrip(retryReq)
	}

	return response, err
}

// cloneRequestBody reads and stores the request body for potential retry
func cloneRequestBody(req *http.Request) ([]byte, error) {
	if req.Body == nil || req.Body == http.NoBody {
		return nil, nil
	}

	body, err := io.ReadAll(req.Body)
	if err != nil {
		return nil, err
	}

	// Reset the body for the original request
	req.Body = io.NopCloser(bytes.NewReader(body))
	return body, nil
}

// cloneRequest creates a new request with the same properties
func cloneRequest(original *http.Request, body []byte) (*http.Request, error) {
	var bodyReader io.Reader
	if body != nil {
		bodyReader = bytes.NewReader(body)
	}

	newReq, err := http.NewRequest(original.Method, original.URL.String(), bodyReader)
	if err != nil {
		return nil, err
	}

	// Copy headers
	for key, values := range original.Header {
		for _, value := range values {
			newReq.Header.Add(key, value)
		}
	}

	return newReq, nil
}

var _ http.RoundTripper = &ekzRoundTripper{}
