/**
 * Copyright 2019, Z Lab Corporation. All rights reserved.
 *
 * For the full copyright and license information, please view the LICENSE
 * file that was distributed with this source code.
 */

package openstack

import (
	"bytes"
	"encoding/json"
	"io"
	"io/ioutil"
	"net/http"
	"strings"

	"github.com/gophercloud/gophercloud"
	"github.com/gophercloud/gophercloud/openstack"
	"github.com/gophercloud/utils/openstack/clientconfig"
	"github.com/hashicorp/go-hclog"
)

// NewProvider returns a new authenticated ProviderClient
func NewProvider(cloudName string, logger hclog.Logger) (*gophercloud.ProviderClient, error) {
	opts := &clientconfig.ClientOpts{
		Cloud: cloudName,
	}
	authOpts, err := clientconfig.AuthOptions(opts)
	if err != nil {
		return nil, err
	}
	authOpts.AllowReauth = true

	provider, err := openstack.AuthenticatedClient(*authOpts)
	if err != nil {
		return nil, err
	}
	provider.HTTPClient = newHTTPClient(logger)

	if provider.ReauthFunc == nil {
		logger.Debug("ReauthFunc is nil")
	} else {
		logger.Debug("Set log into ReauthFunc")
		rf := provider.ReauthFunc
		provider.ReauthFunc = func() error {
			logger.Debug("Proceed to re-authenticate")
			return rf()
		}
	}

	return provider, nil
}

// https://github.com/gophercloud/gophercloud/blob/master/auth_options.go#L73
// LogRoundTripper satisfies the http.RoundTripper interface
type LogRoundTripper struct {
	Logger hclog.Logger
	rt     http.RoundTripper
}

// newHTTPClient return a custom HTTP client that allows for logging relevant
// information before and after the HTTP request.
func newHTTPClient(logger hclog.Logger) http.Client {
	return http.Client{
		Transport: &LogRoundTripper{
			Logger: logger,
			rt:     http.DefaultTransport,
		},
	}
}

// RoundTrip performs a round-trip HTTP request and logs relevant information about it.
func (lrt *LogRoundTripper) RoundTrip(request *http.Request) (*http.Response, error) {
	var err error

	if lrt.Logger.IsDebug() && request.Body != nil {
		lrt.Logger.Debug("Logging request body...")
		request.Body, err = lrt.logRequestBody(request.Body)
		if err != nil {
			return nil, err
		}
	}

	info, err := json.Marshal(request.Header)
	if err != nil {
		lrt.Logger.Debug("Error logging request headers", "error", err)
	}
	lrt.Logger.Debug("Request Headers", "value", string(info))

	lrt.Logger.Info("Request URL", "value", request.URL)

	response, err := lrt.rt.RoundTrip(request)
	if response == nil {
		return nil, err
	}

	lrt.Logger.Debug("Response Status", "value", response.Status)

	info, err = json.Marshal(response.Header)
	if err != nil {
		lrt.Logger.Debug("Error logging response headers", "error", err)
	}
	lrt.Logger.Debug("Response Headers", "value", string(info))

	if lrt.Logger.IsDebug() && response.Body != nil {
		response.Body, err = lrt.logResponseBody(response.Body)
		if err != nil {
			return nil, err
		}
	}

	return response, err
}

func (lrt *LogRoundTripper) logRequestBody(original io.ReadCloser) (io.ReadCloser, error) {
	bs, err := getCopyBuffer(original)
	if err != nil {
		return nil, err
	}

	lrt.Logger.Debug("Request Options", "value", bs.String())

	return ioutil.NopCloser(strings.NewReader(bs.String())), nil
}

func (lrt *LogRoundTripper) logResponseBody(original io.ReadCloser) (io.ReadCloser, error) {
	bs, err := getCopyBuffer(original)
	if err != nil {
		return nil, err
	}

	lrt.Logger.Debug("Response Body", "value", bs.String())

	return ioutil.NopCloser(strings.NewReader(bs.String())), nil
}

func getCopyBuffer(original io.ReadCloser) (bytes.Buffer, error) {
	defer original.Close()

	var bs bytes.Buffer
	_, err := io.Copy(&bs, original)
	if err != nil {
		return bs, err
	}

	return bs, nil

}
