package testutil

import (
	"bytes"

	"github.com/hashicorp/go-hclog"
	"github.com/zlabjp/spire-openstack-plugin/pkg/common"
)

func TestLogger() hclog.Logger {
	return hclog.New(&hclog.LoggerOptions{
		Output: new(bytes.Buffer),
		Name:   common.PluginName,
		Level:  hclog.Debug,
	})
}
