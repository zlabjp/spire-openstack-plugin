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
	"github.com/spiffe/spire-plugin-sdk/pluginmain"
	nodeattestorv1 "github.com/spiffe/spire-plugin-sdk/proto/spire/plugin/agent/nodeattestor/v1"
	configv1 "github.com/spiffe/spire-plugin-sdk/proto/spire/service/common/config/v1"

	"github.com/zlabjp/spire-openstack-plugin/pkg/openstack"
)

// IIDAttestorPlugin implements the nodeattestor Plugin interface
type IIDAttestorPlugin struct {
	nodeattestorv1.UnsafeNodeAttestorServer
	configv1.UnsafeConfigServer

	logger hclog.Logger

	getMetadataHandler func() (*openstack.Metadata, error)
}

func newPlugin() *IIDAttestorPlugin {
	return &IIDAttestorPlugin{
		getMetadataHandler: openstack.GetMetadataFromMetadataService,
	}
}

func (p *IIDAttestorPlugin) Configure(_ context.Context, _ *configv1.ConfigureRequest) (*configv1.ConfigureResponse, error) {
	return &configv1.ConfigureResponse{}, nil
}

func (p *IIDAttestorPlugin) AidAttestation(stream nodeattestorv1.NodeAttestor_AidAttestationServer) error {
	p.logger.Info("Prepare Attestation Request")

	if p.getMetadataHandler == nil {
		return errors.New("handler not found, plugin not initialized")
	}

	meta, err := p.getMetadataHandler()
	if err != nil {
		return fmt.Errorf("failed to retrieve openstack metadata: %v", err)
	}

	return stream.Send(&nodeattestorv1.PayloadOrChallengeResponse{
		Data: &nodeattestorv1.PayloadOrChallengeResponse_Payload{
			Payload: []byte(meta.UUID),
		},
	})
}

func (p *IIDAttestorPlugin) SetLogger(log hclog.Logger) {
	p.logger = log
}

func main() {
	p := newPlugin()
	pluginmain.Serve(
		nodeattestorv1.NodeAttestorPluginServer(p),
		configv1.ConfigServiceServer(p),
	)
}
