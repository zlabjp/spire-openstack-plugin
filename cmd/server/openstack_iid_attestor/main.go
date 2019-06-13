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
	spi "github.com/spiffe/spire/proto/common/plugin"
	"github.com/spiffe/spire/proto/server/nodeattestor"

	"github.com/zlabjp/spire-openstack-plugin/pkg/common"
	"github.com/zlabjp/spire-openstack-plugin/pkg/openstack"
)

// IIDAttestorPlugin implements the nodeattestor Plugin interface
type IIDAttestorPlugin struct {
	logger   hclog.Logger
	config   *IIDAttestorPluginConfig
	instance openstack.InstanceClient

	mtx *sync.RWMutex

	getInstanceHandler func(string, hclog.Logger) (openstack.InstanceClient, error)
}

type IIDAttestorPluginConfig struct {
	trustDomain        string
	LogLevel           string   `hcl:"log_level"`
	CloudName          string   `hcl:"cloud_name"`
	ProjectIDWhitelist []string `hcl:"projectid_whitelist"`
}

func New() *IIDAttestorPlugin {
	return &IIDAttestorPlugin{
		mtx:                &sync.RWMutex{},
		getInstanceHandler: getOpenStackInstance,
	}
}

func (p *IIDAttestorPlugin) Attest(stream nodeattestor.Attest_PluginStream) error {
	p.logger.Info("Received attestation request")

	p.mtx.RLock()
	defer p.mtx.RUnlock()

	if p.instance == nil {
		return errors.New("plugin not configured")
	}

	req, err := stream.Recv()
	if err != nil {
		return err
	}

	iid := string(req.AttestationData.Data)
	s, err := p.instance.Get(iid)
	if err != nil {
		return fmt.Errorf("your IID is invalid: %v", err)
	}

	p.logger.Debug("Got instance data successfully")

	if req.AttestedBefore {
		return fmt.Errorf("the IID has been used and is no longer valid: %v", iid)
	}

	for _, pid := range p.config.ProjectIDWhitelist {
		if s.TenantID == pid {
			resp := &nodeattestor.AttestResponse{
				Valid:        true,
				BaseSPIFFEID: common.GenerateSpiffeID(p.config.trustDomain, s.TenantID, iid),
			}
			return stream.Send(resp)
		}
	}

	return errors.New("invalid attestation request")
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
	if len(config.ProjectIDWhitelist) == 0 {
		return nil, errors.New("projectid_whitelist is required")
	}

	p.mtx.Lock()
	defer p.mtx.Unlock()

	p.logger.SetLevel(common.GetLogLevelFromString(config.LogLevel))

	instance, err := p.getInstanceHandler(config.CloudName, p.logger)
	if err != nil {
		return nil, fmt.Errorf("failed to prepare OpenStack Client: %v", err)
	}

	p.instance = instance
	config.trustDomain = req.GlobalConfig.TrustDomain
	p.config = config

	return &spi.ConfigureResponse{}, nil
}

func (p *IIDAttestorPlugin) GetPluginInfo(context.Context, *spi.GetPluginInfoRequest) (*spi.GetPluginInfoResponse, error) {
	return &spi.GetPluginInfoResponse{}, nil
}

// getOpenStackInstance returns authenticated openstack compute client.
func getOpenStackInstance(cloud string, logger hclog.Logger) (openstack.InstanceClient, error) {
	provider, err := openstack.NewProvider(cloud, logger)
	if err != nil {
		return nil, err
	}
	return openstack.NewInstance(provider, logger)
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
		HandshakeConfig: nodeattestor.Handshake,
		GRPCServer:      plugin.DefaultGRPCServer,
		Logger:          logger,
	})
}
