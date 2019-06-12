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
	"sync"

	"github.com/hashicorp/go-hclog"
	"github.com/hashicorp/go-plugin"
	"github.com/hashicorp/hcl"
	"github.com/spiffe/spire/proto/agent/nodeattestor"
	spc "github.com/spiffe/spire/proto/common"
	spi "github.com/spiffe/spire/proto/common/plugin"

	"github.com/zlabjp/spire-openstack-plugin/pkg/common"
	"github.com/zlabjp/spire-openstack-plugin/pkg/openstack"
)

// IIDAttestorPlugin implements the nodeattestor Plugin interface
type IIDAttestorPlugin struct {
	logger   hclog.Logger
	config   *IIDAttestorPluginConfig
	metaData *openstack.Metadata

	mtx *sync.RWMutex

	getMetadataHandler func() (*openstack.Metadata, error)
}

type IIDAttestorPluginConfig struct {
	trustDomain string
	LogLevel    string `hcl:"log_level"`
}

func New() *IIDAttestorPlugin {
	return &IIDAttestorPlugin{
		mtx:                &sync.RWMutex{},
		getMetadataHandler: openstack.GetMetadataFromMetadataService,
	}
}

func (p *IIDAttestorPlugin) Configure(ctx context.Context, req *spi.ConfigureRequest) (*spi.ConfigureResponse, error) {
	config := &IIDAttestorPluginConfig{}
	if err := hcl.Decode(config, req.Configuration); err != nil {
		return nil, fmt.Errorf("failed to decode configuration file: %v", err)
	}

	if req.GlobalConfig == nil {
		return nil, errors.New("global configuration is required")
	}
	if req.GlobalConfig.TrustDomain == "" {
		return nil, errors.New("trust_domain is required")
	}

	p.mtx.Lock()
	defer p.mtx.Unlock()

	p.logger.SetLevel(common.GetLogLevelFromString(config.LogLevel))

	meta, err := p.getMetadataHandler()
	if err != nil {
		return nil, fmt.Errorf("failed to retrieve openstack metadta: %v", err)
	}

	p.metaData = meta
	config.trustDomain = req.GlobalConfig.TrustDomain
	p.config = config

	return &spi.ConfigureResponse{}, nil
}

func (p *IIDAttestorPlugin) GetPluginInfo(context.Context, *spi.GetPluginInfoRequest) (*spi.GetPluginInfoResponse, error) {
	return &spi.GetPluginInfoResponse{}, nil
}

func (p *IIDAttestorPlugin) FetchAttestationData(stream nodeattestor.FetchAttestationData_PluginStream) error {
	p.logger.Info("Prepare Attestation Request")

	p.mtx.RLock()
	defer p.mtx.RUnlock()

	if p.config == nil || p.metaData == nil {
		return errors.New("plugin not configured")
	}

	return stream.Send(&nodeattestor.FetchAttestationDataResponse{
		AttestationData: &spc.AttestationData{
			Type: common.PluginName,
			Data: []byte(p.metaData.UUID),
		},
		SpiffeId: common.GenerateSpiffeID(p.config.trustDomain, p.metaData.ProjectID, p.metaData.UUID),
	})
}

func main() {
	logger := hclog.New(&hclog.LoggerOptions{
		Name: common.PluginName,
	})

	p := New()
	p.logger = logger

	plugin.Serve(&plugin.ServeConfig{
		Plugins: map[string]plugin.Plugin{
			common.PluginName: nodeattestor.GRPCPlugin{
				ServerImpl: &nodeattestor.GRPCServer{
					Plugin: p,
				},
			},
		},
		GRPCServer:      plugin.DefaultGRPCServer,
		HandshakeConfig: nodeattestor.Handshake,
		Logger:          logger,
	})

}
