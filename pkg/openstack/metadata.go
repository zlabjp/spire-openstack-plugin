package openstack

import (
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/http"
)

const (
	defaultMetadataVersion = "latest"
	metadataURLTemplate    = "http://169.254.169.254/openstack/%s/meta_data.json"
)

// Metadata represents the information fetched from OpenStack metadata service
type Metadata struct {
	UUID             string `json:"uuid"`
	Name             string `json:"name"`
	AvailabilityZone string `json:"availability_zone"`
	ProjectID        string `json:"project_id"`
	// we don't care any other fields.
}

// GetMetadataFromMetadataService gets metadata from OpenStack Metadata service.
func GetMetadataFromMetadataService() (*Metadata, error) {
	metadataURL := getMetadataURL(defaultMetadataVersion)
	resp, err := http.Get(metadataURL)
	if err != nil {
		return nil, fmt.Errorf("error fetching metadata from %s: %v", metadataURL, err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		err = fmt.Errorf("unexpected status code when reading metadata from %s: %s", metadataURL, resp.Status)
		return nil, err
	}

	return parseMetadata(resp.Body)
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

func getMetadataURL(metadataVersion string) string {
	return fmt.Sprintf(metadataURLTemplate, metadataVersion)
}
