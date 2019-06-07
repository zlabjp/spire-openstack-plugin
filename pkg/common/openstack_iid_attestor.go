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

	"github.com/hashicorp/go-hclog"
)

const (
	PluginName = "openstack_iid"

	defaultLogLevel = "INFO"
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

func GetLogLevelFromString(level string) hclog.Level {
	var logLevel string
	if level == "" {
		logLevel = defaultLogLevel
	} else {
		logLevel = level
	}
	return hclog.LevelFromString(logLevel)
}
