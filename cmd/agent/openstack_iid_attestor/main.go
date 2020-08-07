/**
 * Copyright 2019, Z Lab Corporation. All rights reserved.
 *
 * For the full copyright and license information, please view the LICENSE
 * file that was distributed with this source code.
 */

package main

import (
	"context"
	"errors"
	"fmt"

	"github.com/hashicorp/go-hclog"
	"github.com/spiffe/spire/pkg/agent/plugin/nodeattestor"
	"github.com/spiffe/spire/pkg/common/catalog"
	spc "github.com/spiffe/spire/proto/spire/common"
	spi "github.com/spiffe/spire/proto/spire/common/plugin"

	"github.com/zlabjp/spire-openstack-plugin/pkg/common"
	"github.com/zlabjp/spire-openstack-plugin/pkg/openstack"
)

// IIDAttestorPlugin implements the nodeattestor Plugin interface
type IIDAttestorPlugin struct {
	logger hclog.Logger

	getMetadataHandler func() (*openstack.Metadata, error)
}

// BuiltIn constructs a catalog Plugin using a new instance of this plugin.
func BuiltIn() catalog.Plugin {
	return builtin(New())
}

func builtin(p *IIDAttestorPlugin) catalog.Plugin {
	return catalog.MakePlugin(common.PluginName, nodeattestor.PluginServer(p))
}

func New() *IIDAttestorPlugin {
	return &IIDAttestorPlugin{
		getMetadataHandler: openstack.GetMetadataFromMetadataService,
	}
}

func (p *IIDAttestorPlugin) Configure(ctx context.Context, req *spi.ConfigureRequest) (*spi.ConfigureResponse, error) {
	return &spi.ConfigureResponse{}, nil
}

func (p *IIDAttestorPlugin) GetPluginInfo(context.Context, *spi.GetPluginInfoRequest) (*spi.GetPluginInfoResponse, error) {
	return &spi.GetPluginInfoResponse{}, nil
}

func (p *IIDAttestorPlugin) FetchAttestationData(stream nodeattestor.NodeAttestor_FetchAttestationDataServer) error {
	p.logger.Info("Prepare Attestation Request")

	if p.getMetadataHandler == nil {
		return errors.New("handler not found, plugin not initialized")
	}

	meta, err := p.getMetadataHandler()
	if err != nil {
		return fmt.Errorf("failed to retrieve openstack metadata: %v", err)
	}
	return stream.Send(&nodeattestor.FetchAttestationDataResponse{
		AttestationData: &spc.AttestationData{
			Type: common.PluginName,
			Data: []byte(meta.UUID),
		},
	})
}

func (p *IIDAttestorPlugin) SetLogger(log hclog.Logger) {
	p.logger = log
}

func main() {
	catalog.PluginMain(BuiltIn())
}
