package common

import (
	"net/url"
	"path"
)

const (
	PluginName = "openstack_iid"
)

func GenerateSpiffeID(trustDomain, projectID, instanceID string) string {
	spiffePath := path.Join("spire", "agent", PluginName, projectID, instanceID)
	id := &url.URL{
		Scheme: "spiffe",
		Host:   trustDomain,
		Path:   spiffePath,
	}
	return id.String()
}
