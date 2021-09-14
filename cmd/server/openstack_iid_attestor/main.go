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
	"sort"
	"sync"

	"github.com/gophercloud/gophercloud/openstack/compute/v2/extensions/secgroups"
	"github.com/gophercloud/gophercloud/openstack/compute/v2/servers"
	"github.com/hashicorp/go-hclog"
	"github.com/hashicorp/hcl"
	"github.com/mitchellh/mapstructure"
	"github.com/spiffe/spire-plugin-sdk/pluginmain"
	nodeattestorv1 "github.com/spiffe/spire-plugin-sdk/proto/spire/plugin/server/nodeattestor/v1"
	configv1 "github.com/spiffe/spire-plugin-sdk/proto/spire/service/common/config/v1"
	nodeattestorbase "github.com/spiffe/spire/pkg/server/plugin/nodeattestor/base"

	"github.com/zlabjp/spire-openstack-plugin/pkg/common"
	"github.com/zlabjp/spire-openstack-plugin/pkg/openstack"
)

// IIDAttestorPlugin implements the nodeattestor Plugin interface
type IIDAttestorPlugin struct {
	nodeattestorbase.Base
	nodeattestorv1.UnsafeNodeAttestorServer
	configv1.UnsafeConfigServer

	logger   hclog.Logger
	config   *IIDAttestorPluginConfig
	instance openstack.InstanceClient

	mtx *sync.RWMutex

	getInstanceHandler    func(string, hclog.Logger) (openstack.InstanceClient, error)
	attestedBeforeHandler func(ctx context.Context, p *IIDAttestorPlugin, agentID string) (bool, error)
}

type IIDAttestorPluginConfig struct {
	trustDomain        string
	CloudName          string   `hcl:"cloud_name"`
	ProjectIDAllowList []string `hcl:"projectid_allow_list"`
	// If CustomMetaData is not nil, the plugin makes custom metadata Selectors.
	//
	//  plugin_data {
	//     cloud_name = "test"
	//     // If you need custom metadata Selectors, specify the parameter as follows.
	//     custom_metadata = {}
	//     // If you need Selectors for specific metadata only, specify as follows.
	//     custom_metadata = {
	//         keys = ["alpha", "bravo"]
	//     }
	//  }
	//
	CustomMetaData *CustomMetadata `hcl:"custom_metadata"`
}

type CustomMetadata struct {
	// The plugin makes Selectors by given keys.
	// If the Keys is empty, the plugin will makes Selectors using with all custom metadata keys.
	Keys []string `hcl:"keys"`
}

func newPlugin() *IIDAttestorPlugin {
	return &IIDAttestorPlugin{
		mtx:                   &sync.RWMutex{},
		getInstanceHandler:    getOpenStackInstance,
		attestedBeforeHandler: attestedBefore,
	}
}

func (p *IIDAttestorPlugin) Attest(stream nodeattestorv1.NodeAttestor_AttestServer) error {
	p.logger.Info("Received attestation request")

	config, err := p.getConfig()
	if err != nil {
		return err
	}

	p.mtx.RLock()
	instance := p.instance
	p.mtx.RUnlock()

	if instance == nil {
		instance, err = p.getInstanceHandler(config.CloudName, p.logger)
		if err != nil {
			return fmt.Errorf("failed to prepare OpenStack Client: %v", err)
		}
		p.mtx.Lock()
		p.instance = instance
		p.mtx.Unlock()
	}

	req, err := stream.Recv()
	if err != nil {
		return err
	}

	iid := string(req.GetPayload())
	s, err := instance.Get(iid)
	if err != nil {
		return fmt.Errorf("failed to get instance information: %v", err)
	}

	p.logger.Debug("Got instance data successfully")

	agentID := common.GenerateSpiffeID(config.trustDomain, s.TenantID, iid)

	attested, err := p.attestedBeforeHandler(stream.Context(), p, agentID)
	switch {
	case err != nil:
		return err
	case attested:
		return fmt.Errorf("IID has already been used to attest an agent: %v", iid)
	}

	svs, err := p.makeSelectorValues(s)
	if err != nil {
		return err
	}

	for _, pid := range config.ProjectIDAllowList {
		if s.TenantID == pid {
			resp := &nodeattestorv1.AttestResponse{
				Response: &nodeattestorv1.AttestResponse_AgentAttributes{
					AgentAttributes: &nodeattestorv1.AgentAttributes{
						SpiffeId:       agentID,
						SelectorValues: svs,
					},
				},
			}
			return stream.Send(resp)
		}
	}

	return errors.New("invalid attestation request")
}

func (p *IIDAttestorPlugin) Configure(_ context.Context, req *configv1.ConfigureRequest) (*configv1.ConfigureResponse, error) {
	if p.getInstanceHandler == nil {
		return nil, errors.New("handler not found, plugin not initialized")
	}

	config := &IIDAttestorPluginConfig{}
	if err := hcl.Decode(config, req.HclConfiguration); err != nil {
		return nil, fmt.Errorf("failed to decode configuration file: %w", err)
	}
	if req.CoreConfiguration == nil {
		return nil, errors.New("global configuration is required")
	}
	if req.CoreConfiguration.TrustDomain == "" {
		return nil, errors.New("trust_domain is required")
	}
	if len(config.ProjectIDAllowList) == 0 {
		return nil, errors.New("projectid_allow_list is required")
	}

	config.trustDomain = req.CoreConfiguration.TrustDomain

	p.setConfig(config)

	return &configv1.ConfigureResponse{}, nil
}

// attestedBefore returns true if given agentID attested before
func attestedBefore(ctx context.Context, p *IIDAttestorPlugin, agentID string) (bool, error) {
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

// makeSelectorValues returns Selector sets related to instance
func (p *IIDAttestorPlugin) makeSelectorValues(server *servers.Server) ([]string, error) {
	sgSelector, err := genSGSelectorValues(server.SecurityGroups)
	if err != nil {
		return nil, err
	}
	var svs []string
	svs = append(svs, sgSelector...)

	if p.config.CustomMetaData != nil {
		metaSelector := genCustomMetaSelectorValues(server.Metadata, p.config.CustomMetaData.Keys)
		svs = append(svs, metaSelector...)
	}

	sort.Strings(svs)

	return svs, nil
}

// genSGSelectorValues generates Selector list about SecurityGroup.
func genSGSelectorValues(sgMapList []map[string]interface{}) ([]string, error) {
	var sList []string
	for i := range sgMapList {
		m := sgMapList[i]
		if m == nil {
			continue
		}

		var sg secgroups.SecurityGroup
		if err := mapstructure.Decode(m, &sg); err != nil {
			return nil, fmt.Errorf("failed to decode SecurityGroup info: %v", err)
		}

		if sg.ID != "" {
			sList = append(sList, fmt.Sprintf("sg:id:%s", sg.ID))
		}
		if sg.Name != "" {
			sList = append(sList, fmt.Sprintf("sg:name:%s", sg.Name))
		}
	}
	return sList, nil
}

// genCustomMetaSelectorValues generates Selector list about Custom Metadata.
func genCustomMetaSelectorValues(meta map[string]string, acceptKeys []string) []string {
	var sList []string

	if len(acceptKeys) > 0 {
		for _, key := range acceptKeys {
			if v, ok := meta[key]; ok && v != "" {
				sList = append(sList, fmt.Sprintf("meta:%s:%s", key, v))
			}
		}
	} else {
		for k, v := range meta {
			if k != "" && v != "" {
				sList = append(sList, fmt.Sprintf("meta:%s:%s", k, v))
			}
		}
	}

	return sList
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
	p := newPlugin()
	pluginmain.Serve(
		nodeattestorv1.NodeAttestorPluginServer(p),
		configv1.ConfigServiceServer(p),
	)
}
