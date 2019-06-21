/**
 * Copyright 2019, Z Lab Corporation. All rights reserved.
 *
 * For the full copyright and license information, please view the LICENSE
 * file that was distributed with this source code.
 */

package openstack

import (
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"strings"

	"github.com/lestrrat-go/jwx/jwk"
)

const (
	defaultMetadataVersion = "latest"
	metadataURLTemplate    = "http://169.254.169.254/openstack/%s/meta_data.json"
	iidURLTemplate         = "http://169.254.169.254/openstack/%s/vendor_data2.json"
)

type MetadataClient interface {
	// SerVersion sets metadata version
	SetVersion(string)
	// SetMetadataAsTarget sets Medata Service as target
	SetMetadataAsTarget()
	// SetDynamicJSONAsTarget sets Vendordata Dynamic JSON Service as target
	SetDynamicJSONAsTarget()
	// GetTargetURL returns string of target URL
	GetTargetURL() string
	// GetMetadataFromMetadataService returns the result fetched from target URL.
	GetMetadataFromMetadataService() (interface{}, error)
}

// Client implements MetadataClient
type Client struct {
	client  *http.Client
	version string
	url     string
}

// Metadata represents the information fetched from OpenStack metadata service
type Metadata struct {
	UUID             string `json:"uuid"`
	Name             string `json:"name"`
	AvailabilityZone string `json:"availability_zone"`
	ProjectID        string `json:"project_id"`
	// we don't care any other fields.
}

// Vendordata2 represents the information fetched from OpenStack Vendordata Dynamic JSON service.
// In this module, it's assumed to return OpenStack IID(Instance Identity Documents) which considering cooperation
// with spire openstack plugin.
//
// see: https://docs.openstack.org/nova/latest/user/vendordata.html
// see: (TODO: add IID spec link)
type Vendordata2 struct {
	IID     *IID     `json:"iid"`
	IIDKeys *IIDKeys `json:"iid_keys"`
}

// IID contains the raw JWS String
type IID struct {
	Data string `json:"data"`
}

// IIDPayload is payload of IID
type IIDPayload struct {
	ProjectID  string            `json:"projectID"`
	InstanceID string            `json:"instanceID"`
	ImageID    string            `json:"imageID"`
	Hostname   string            `json:"hostname"`
	Metadata   map[string]string `json:"metadata"`
	IssuedAt   int64             `json:"iat"`
	ExpiresAt  int64             `json:"exp"`
}

// IIDKeys contains JWKSet
type IIDKeys struct {
	jwk.Set
}

// NewMetadataClient returns *Client with default http client and metadata version.
func NewMetadataClient() *Client {
	return &Client{
		client:  http.DefaultClient,
		version: defaultMetadataVersion,
	}
}

func (c *Client) SetVersion(ver string) {
	c.version = ver
}

func (c *Client) SetMetadataAsTarget() {
	c.url = fmt.Sprintf(metadataURLTemplate, c.version)
}

func (c *Client) SetDynamicJSONAsTarget() {
	c.url = fmt.Sprintf(iidURLTemplate, c.version)
}

func (c *Client) GetTargetURL() string {
	if c.url == "" {
		c.SetMetadataAsTarget()
	}
	return c.url
}

// GetMetadataFromMetadataService gets metadata from OpenStack Metadata service.
func (c *Client) GetMetadataFromMetadataService() (interface{}, error) {
	target := c.GetTargetURL()

	resp, err := c.client.Get(target)
	if err != nil {
		return nil, fmt.Errorf("error fetching metadata from %s: %v", c.url, err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("unexpected status code when reading metadata from %s: %s", c.url, resp.Status)
	}

	u, err := url.Parse(target)
	if err != nil {
		return nil, fmt.Errorf("failed to parse URL %s: %v", c.url, err)
	}

	elem := strings.Split(u.Path, "/")
	switch elem[3] {
	case "meta_data.json":
		return parseMetadata(resp.Body)
	case "vendor_data2.json":
		return parseIID(resp.Body)
	default:
		return nil, fmt.Errorf("unknown endpoint: %v", elem[2])
	}
}

func parseMetadata(r io.Reader) (*Metadata, error) {
	var metadata Metadata
	d := json.NewDecoder(r)
	if err := d.Decode(&metadata); err != nil {
		return nil, err
	}

	if metadata.UUID == "" {
		return nil, errors.New("invalid metadata, uuid seems empty")
	}

	return &metadata, nil
}

func parseIID(r io.Reader) (*Vendordata2, error) {
	var iid Vendordata2
	d := json.NewDecoder(r)
	if err := d.Decode(&iid); err != nil {
		return nil, err
	}
	return &iid, nil
}
