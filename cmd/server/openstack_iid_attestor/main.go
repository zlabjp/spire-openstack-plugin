/**
 * Copyright 2019, Z Lab Corporation. All rights reserved.
 *
 * For the full copyright and license information, please view the LICENSE
 * file that was distributed with this source code.
 */

package main

import (
	"context"
	"encoding/base64"
	"encoding/json"
	"errors"
	"fmt"
	"strings"
	"sync"

	"github.com/hashicorp/go-hclog"
	"github.com/hashicorp/hcl"
	"github.com/lestrrat-go/jwx/jws"
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
	metadata openstack.MetadataClient

	mtx *sync.RWMutex

	getInstanceHandler    func(string, hclog.Logger) (openstack.InstanceClient, error)
	verifyIIDHandler      func([]byte, openstack.MetadataClient) ([]byte, error)
	attestedBeforeHandler func(p *IIDAttestorPlugin, ctx context.Context, agentID string) (bool, error)
}

type IIDAttestorPluginConfig struct {
	trustDomain        string
	CloudName          string   `hcl:"cloud_name"`
	ProjectIDWhitelist []string `hcl:"projectid_whitelist"`
	UseIID             bool     `hcl:"use_iid"`
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
		verifyIIDHandler:      verifyIIDSignature,
		attestedBeforeHandler: attestedBefore,
	}
}

func (p *IIDAttestorPlugin) Attest(stream nodeattestor.NodeAttestor_AttestServer) error {
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

	uuid := string(req.AttestationData.Data)
	if p.config.UseIID {
		payload, err := p.verifyIIDHandler(req.AttestationData.Data, p.metadata)
		if err != nil {
			return fmt.Errorf("AttestationData is invalid: %v", err)
		}

		p.logger.Debug("IID is valid", "data", string(req.AttestationData.Data))

		pl := &openstack.IIDPayload{}
		if err := json.Unmarshal(payload, pl); err != nil {
			return fmt.Errorf("failed to unmarshal IID payload: %v", err)
		}
		if pl.InstanceID == "" {
			return errors.New("InstanceID is empty in the Vendordata2")
		}
		uuid = pl.InstanceID
	}

	s, err := p.instance.Get(uuid)
	if err != nil {
		return fmt.Errorf("your InstanceID is invalid: %v", err)
	}

	p.logger.Debug("Got instance data successfully")

	agentID := common.GenerateSpiffeID(p.config.trustDomain, s.TenantID, uuid)

	attested, err := p.attestedBeforeHandler(p, stream.Context(), agentID)
	switch {
	case err != nil:
		return err
	case attested:
		return fmt.Errorf("IID has already been used to attest an agent: %v", uuid)
	}

	for _, pid := range p.config.ProjectIDWhitelist {
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

	instance, err := p.getInstanceHandler(config.CloudName, p.logger)
	if err != nil {
		return nil, fmt.Errorf("failed to prepare OpenStack Client: %v", err)
	}

	metadata := openstack.NewMetadataClient()
	if config.UseIID {
		metadata.SetDynamicJSONAsTarget()
	}

	p.instance = instance
	p.metadata = metadata
	config.trustDomain = req.GlobalConfig.TrustDomain
	p.config = config

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

// verifyIIDSignature verifies the input data as JWS.
// the verify key is fetched from OpenStack Vendordata Dynamic JSON.
func verifyIIDSignature(data []byte, client openstack.MetadataClient) ([]byte, error) {
	obj, err := client.GetMetadataFromMetadataService()
	if err != nil {
		return nil, err
	}
	vd := obj.(*openstack.Vendordata2)

	elem := strings.Split(string(data), ".")
	encHeader := elem[0]

	h := &jws.StandardHeaders{}
	header, err := base64.RawURLEncoding.DecodeString(encHeader)
	if err != nil {
		return nil, err
	}
	if err := json.Unmarshal([]byte(header), h); err != nil {
		return nil, err
	}
	var kid string
	if v, ok := h.Get(jws.KeyIDKey); ok {
		kid = v.(string)
	} else {
		return nil, fmt.Errorf("could not find the kid parameter in the header: %v", header)
	}

	return jws.VerifyWithJWK([]byte(data), vd.IIDKeys.LookupKeyID(kid)[0])
}

func main() {
	catalog.PluginMain(BuiltIn())
}
