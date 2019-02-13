package main

import (
	"context"
	"errors"
	"fmt"
	"sync"
	"time"

	"github.com/hashicorp/go-plugin"
	"github.com/hashicorp/hcl"
	spi "github.com/spiffe/spire/proto/common/plugin"
	"github.com/spiffe/spire/proto/server/nodeattestor"
	"github.com/zlabjp/openstack-iid-attestor/pkg/common"
	"github.com/zlabjp/openstack-iid-attestor/pkg/openstack"
)

// IIDAttestorPlugin implements the nodeattestor Plugin interface
type IIDAttestorPlugin struct {
	config   *IIDAttestorPluginConfig
	instance openstack.InstanceClient
	period   time.Duration

	mtx *sync.RWMutex

	getInstanceHandler func(string) (openstack.InstanceClient, error)
}

type IIDAttestorPluginConfig struct {
	trustDomain        string
	AttestationPeriod  string   `hcl:"attestation_period"`
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

	if req.AttestedBefore {
		return fmt.Errorf("the IID has been used and is no longer valid: %v", iid)
	}

	if p.period != time.Duration(0) {
		deadline := s.Created.Add(p.period)
		if time.Now().After(deadline) {
			return errors.New("attestation period has expired")
		}
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

	instance, err := p.getInstanceHandler(config.CloudName)
	if err != nil {
		return nil, fmt.Errorf("failed to prepare OpenStack Client: %v", err)
	}

	if config.AttestationPeriod != "" {
		period, err := time.ParseDuration(config.AttestationPeriod)
		if err != nil {
			return nil, fmt.Errorf("invalid value for attestation_period: %v", err)
		}
		p.period = period
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
func getOpenStackInstance(cloud string) (openstack.InstanceClient, error) {
	provider, err := openstack.NewProvider(cloud)
	if err != nil {
		return nil, err
	}
	return openstack.NewInstance(provider)
}

func main() {
	plugin.Serve(&plugin.ServeConfig{
		Plugins: map[string]plugin.Plugin{
			common.PluginName: nodeattestor.GRPCPlugin{
				ServerImpl: &nodeattestor.GRPCServer{
					Plugin: New(),
				},
			},
		},
		HandshakeConfig: nodeattestor.Handshake,
		GRPCServer:      plugin.DefaultGRPCServer,
	})
}
