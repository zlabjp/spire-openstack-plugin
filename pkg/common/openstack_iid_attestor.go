/**
 * Copyright 2019, Z Lab Corporation. All rights reserved.
 *
 * For the full copyright and license information, please view the LICENSE
 * file that was distributed with this source code.
 */

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
