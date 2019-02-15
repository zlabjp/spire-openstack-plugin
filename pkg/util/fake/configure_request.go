package fake

import "github.com/spiffe/spire/proto/common/plugin"

func NewFakeConfigureRequest(g *plugin.ConfigureRequest_GlobalConfig, p string) *plugin.ConfigureRequest {
	return &plugin.ConfigureRequest{
		GlobalConfig:  g,
		Configuration: p,
	}
}
