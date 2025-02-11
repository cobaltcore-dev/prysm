// SPDX-FileCopyrightText: 2025 SAP SE or an SAP affiliate company and prysm contributors
//
// SPDX-License-Identifier: Apache-2.0
package rgwadmin

import (
	"context"
	"io"
	"net/http"
	"net/url"
	"time"

	"errors"

	"github.com/aws/aws-sdk-go/aws/credentials"
	v4 "github.com/aws/aws-sdk-go/aws/signer/v4"
)

const (
	authRegion        = "default"
	service           = "s3"
	connectionTimeout = 3 * time.Second
)

var (
	errNoEndpoint  = errors.New("endpoint not set")
	errNoAccessKey = errors.New("access key not set")
	errNoSecretKey = errors.New("secret key not set")
	errHTTPFailure = errors.New("failed to execute HTTP request")
)

// HTTPClient defines an interface for HTTP operations.
type HTTPClient interface {
	Do(req *http.Request) (*http.Response, error)
}

// AuthConfig holds authentication details.
type AuthConfig struct {
	AccessKey string
	SecretKey string
}

// API represents a Ceph RGW Admin Ops API client.
type API struct {
	Auth       AuthConfig
	Endpoint   string
	HTTPClient HTTPClient
}

// New creates a new Ceph RGW client with basic validation.
func New(endpoint, accessKey, secretKey string, httpClient HTTPClient) (*API, error) {
	if err := validateConfig(endpoint, accessKey, secretKey); err != nil {
		return nil, err
	}

	if httpClient == nil {
		httpClient = &http.Client{Timeout: connectionTimeout}
	}

	return &API{
		Endpoint: endpoint,
		Auth: AuthConfig{
			AccessKey: accessKey,
			SecretKey: secretKey,
		},
		HTTPClient: httpClient,
	}, nil
}

// validateConfig ensures required parameters are set.
func validateConfig(endpoint, accessKey, secretKey string) error {
	switch {
	case endpoint == "":
		return errNoEndpoint
	case accessKey == "":
		return errNoAccessKey
	case secretKey == "":
		return errNoSecretKey
	default:
		return nil
	}
}

// call performs a signed request to the RGW Admin Ops API.
func (api *API) call(ctx context.Context, method, path string, args url.Values, body io.Reader) ([]byte, error) {
	reqURL := buildQueryPath(api.Endpoint, path, args.Encode())

	// Create HTTP request
	req, err := http.NewRequestWithContext(ctx, method, reqURL, body)
	if err != nil {
		return nil, err
	}

	// Sign request using AWS v4 signing
	if err := api.signRequest(req); err != nil {
		return nil, err
	}

	// Perform request
	resp, err := api.HTTPClient.Do(req)
	if err != nil {
		return nil, errHTTPFailure
	}
	defer resp.Body.Close()

	return parseResponse(resp)
}

// signRequest signs an HTTP request using AWS v4 signing.
func (api *API) signRequest(req *http.Request) error {
	cred := credentials.NewStaticCredentials(api.Auth.AccessKey, api.Auth.SecretKey, "")
	signer := v4.NewSigner(cred)

	_, err := signer.Sign(req, nil, service, authRegion, time.Now())
	return err
}

// parseResponse reads and validates the HTTP response.
func parseResponse(resp *http.Response) ([]byte, error) {
	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, err
	}

	if resp.StatusCode >= 300 {
		return nil, handleStatusError(body)
	}

	return body, nil
}
