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
	"github.com/hashicorp/hcl"
	"github.com/spiffe/spire/pkg/common/catalog"
	"github.com/spiffe/spire/pkg/server/plugin/nodeattestor"
	nodeattestorbase "github.com/spiffe/spire/pkg/server/plugin/nodeattestor/base"
	spi "github.com/spiffe/spire/proto/spire/common/plugin"

	"github.com/zlabjp/spire-openstack-plugin/pkg/common"
	"github.com/zlabjp/spire-openstack-plugin/pkg/openstack"
)

// IIDAttestorPlugin implements the nodeattestor Plugin interface
type IIDAttestorPlugin struct {
	nodeattestorbase.Base

	logger   hclog.Logger
	config   *IIDAttestorPluginConfig
	instance openstack.InstanceClient

	mtx *sync.RWMutex

	getInstanceHandler    func(string, hclog.Logger) (openstack.InstanceClient, error)
	attestedBeforeHandler func(p *IIDAttestorPlugin, ctx context.Context, agentID string) (bool, error)
}

type IIDAttestorPluginConfig struct {
	trustDomain        string
	CloudName          string   `hcl:"cloud_name"`
	ProjectIDAllowList []string `hcl:"projectid_allow_list"`
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
		mtx:                   &sync.RWMutex{},
		getInstanceHandler:    getOpenStackInstance,
		attestedBeforeHandler: attestedBefore,
	}
}

func (p *IIDAttestorPlugin) Attest(stream nodeattestor.NodeAttestor_AttestServer) error {
	p.logger.Info("Received attestation request")

	config, err := p.getConfig()
	if err != nil {
		return err
	}

	if p.instance == nil {
		return errors.New("openstack instance client not configured")
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

	agentID := common.GenerateSpiffeID(config.trustDomain, s.TenantID, iid)

	attested, err := p.attestedBeforeHandler(p, stream.Context(), agentID)
	switch {
	case err != nil:
		return err
	case attested:
		return fmt.Errorf("IID has already been used to attest an agent: %v", iid)
	}

	for _, pid := range config.ProjectIDAllowList {
		if s.TenantID == pid {
			resp := &nodeattestor.AttestResponse{
				AgentId: agentID,
			}
			return stream.Send(resp)
		}
	}

	return errors.New("invalid attestation request")
}

func (p *IIDAttestorPlugin) Configure(ctx context.Context, req *spi.ConfigureRequest) (*spi.ConfigureResponse, error) {
	if p.getInstanceHandler == nil {
		return nil, errors.New("handler not found, plugin not initialized")
	}

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
	if len(config.ProjectIDAllowList) == 0 {
		return nil, errors.New("projectid_allow_list is required")
	}

	instance, err := p.getInstanceHandler(config.CloudName, p.logger)
	if err != nil {
		return nil, fmt.Errorf("failed to prepare OpenStack Client: %v", err)
	}

	p.instance = instance
	config.trustDomain = req.GlobalConfig.TrustDomain

	p.setConfig(config)

	return &spi.ConfigureResponse{}, nil
}

func (p *IIDAttestorPlugin) GetPluginInfo(context.Context, *spi.GetPluginInfoRequest) (*spi.GetPluginInfoResponse, error) {
	return &spi.GetPluginInfoResponse{}, nil
}

// attestedBefore returns true if given agentID attested before
func attestedBefore(p *IIDAttestorPlugin, ctx context.Context, agentID string) (bool, error) {
	return p.IsAttested(ctx, agentID)
}

// getOpenStackInstance returns authenticated openstack compute client.
func getOpenStackInstance(cloud string, logger hclog.Logger) (openstack.InstanceClient, error) {
	provider, err := openstack.NewProvider(cloud)
	if err != nil {
		return nil, err
	}
	return openstack.NewInstance(provider, logger)
}

func (p *IIDAttestorPlugin) SetLogger(log hclog.Logger) {
	p.logger = log
}

func (p *IIDAttestorPlugin) setConfig(config *IIDAttestorPluginConfig) {
	p.mtx.Lock()
	defer p.mtx.Unlock()
	p.config = config
}

func (p *IIDAttestorPlugin) getConfig() (*IIDAttestorPluginConfig, error) {
	p.mtx.RLock()
	defer p.mtx.RUnlock()

	if p.config == nil {
		return nil, errors.New("plugin not configured")
	}
	return p.config, nil
}

func main() {
	catalog.PluginMain(BuiltIn())
}
