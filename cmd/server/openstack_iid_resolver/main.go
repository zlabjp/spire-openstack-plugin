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
	"regexp"
	"sync"

	"github.com/gophercloud/gophercloud/openstack/compute/v2/extensions/secgroups"
	"github.com/hashicorp/go-hclog"
	"github.com/hashicorp/hcl"
	"github.com/mitchellh/mapstructure"
	"github.com/spiffe/spire/pkg/common/catalog"
	"github.com/spiffe/spire/pkg/common/idutil"
	spu "github.com/spiffe/spire/pkg/common/util"
	"github.com/spiffe/spire/pkg/server/plugin/noderesolver"
	spc "github.com/spiffe/spire/proto/spire/common"
	spi "github.com/spiffe/spire/proto/spire/common/plugin"

	"github.com/zlabjp/spire-openstack-plugin/pkg/common"
	"github.com/zlabjp/spire-openstack-plugin/pkg/openstack"
)

var (
	regexpAgentIDPath = regexp.MustCompile(`^/spire/agent/openstack_iid/([^/]+)/([^/]+)$`)
)

// IIDResolverPlugin implements he noderesolver Plugin interface
type IIDResolverPlugin struct {
	logger   hclog.Logger
	config   *IIDResolverPluginConfig
	instance openstack.InstanceClient

	mtx                sync.RWMutex
	getInstanceHandler func(string, hclog.Logger) (openstack.InstanceClient, error)
}

type IIDResolverPluginConfig struct {
	// Name of cloud entry in clouds.yaml to use.
	CloudName string `hcl:"cloud_name"`
	// If true, the plugin makes Selector of Custom Meta Data.
	CustomMetaData bool `hcl:"custom_meta_data"`
	// If CustomMetaData is true, the Selector is generated using the specified keys.
	// If value is empty, use all entries
	MetaDataKeys []string `hcl:"meta_data_keys"`
}

// BuiltIn constructs a catalog Plugin using a new instance of this plugin.
func BuiltIn() catalog.Plugin {
	return builtin(New())
}

func builtin(p *IIDResolverPlugin) catalog.Plugin {
	return catalog.MakePlugin(common.PluginName, noderesolver.PluginServer(p))
}

// New returns a *IIDResolverPlugin with default getOpenStackHandler
func New() *IIDResolverPlugin {
	return &IIDResolverPlugin{
		mtx:                sync.RWMutex{},
		getInstanceHandler: getOpenStackInstance,
	}
}

func (p *IIDResolverPlugin) Configure(ctx context.Context, req *spi.ConfigureRequest) (*spi.ConfigureResponse, error) {
	if p.getInstanceHandler == nil {
		return nil, errors.New("handler not found, plugin not initialized")
	}

	config := new(IIDResolverPluginConfig)
	if err := hcl.Decode(config, req.Configuration); err != nil {
		return nil, fmt.Errorf("failed to decode configuration file: %v", err)
	}

	instance, err := p.getInstanceHandler(config.CloudName, p.logger)
	if err != nil {
		return nil, fmt.Errorf("failed to prepare OpenStack Client: %v", err)
	}

	p.instance = instance
	p.setConfig(config)

	return &spi.ConfigureResponse{}, nil
}

func (p *IIDResolverPlugin) Resolve(ctx context.Context, req *noderesolver.ResolveRequest) (*noderesolver.ResolveResponse, error) {
	p.logger.Info("Received resolve request")

	resp := &noderesolver.ResolveResponse{
		Map: make(map[string]*spc.Selectors),
	}

	for _, spiffeID := range req.BaseSpiffeIdList {
		selectors, err := p.makeSelectorFromSpiffeID(spiffeID)
		if err != nil {
			return nil, err
		}
		resp.Map[spiffeID] = selectors
	}
	p.logger.Info("Success in making Selectors")

	return resp, nil
}

func (p *IIDResolverPlugin) GetPluginInfo(ctx context.Context, req *spi.GetPluginInfoRequest) (*spi.GetPluginInfoResponse, error) {
	return &spi.GetPluginInfoResponse{}, nil
}

// makeSelectorFromSpiffeID returns Selector sets related to instance
func (p *IIDResolverPlugin) makeSelectorFromSpiffeID(spiffeID string) (*spc.Selectors, error) {
	config, err := p.getConfig()
	if err != nil {
		return nil, err
	}
	if p.instance == nil {
		return nil, errors.New("openstack client not configured")
	}

	iid, err := genInstanceIDFromSpiffeID(spiffeID)
	if err != nil {
		return nil, err
	}

	s, err := p.instance.Get(iid)
	if err != nil {
		return nil, fmt.Errorf("failed to get instance information: %v", err)
	}

	sgSelector, err := genSGSelector(s.SecurityGroups)
	var selectors spc.Selectors
	selectors.Entries = sgSelector

	if config.CustomMetaData {
		metaSelector := genCustomMetaSelector(s.Metadata, config.MetaDataKeys)
		selectors.Entries = append(selectors.Entries, metaSelector...)
	}

	spu.SortSelectors(selectors.Entries)

	return &selectors, nil
}

// genSGSelector generates Selector list about SecurityGroup.
func genSGSelector(sgMapList []map[string]interface{}) ([]*spc.Selector, error) {
	var sList []*spc.Selector
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
			sList = append(sList,
				&spc.Selector{
					Type:  common.PluginName,
					Value: fmt.Sprintf("sg:id:%s", sg.ID),
				})
		}
		if sg.Name != "" {
			sList = append(sList,
				&spc.Selector{
					Type:  common.PluginName,
					Value: fmt.Sprintf("sg:name:%s", sg.Name),
				})
		}
	}
	return sList, nil
}

// genCustomMetaSelector generates Selector list about Custom Meta Data.
func genCustomMetaSelector(meta map[string]string, acceptKeys []string) []*spc.Selector {
	var sList []*spc.Selector

	if acceptKeys != nil {
		for _, key := range acceptKeys {
			if v, ok := meta[key]; ok && v != "" {
				sList = append(sList,
					&spc.Selector{
						Type:  common.PluginName,
						Value: fmt.Sprintf("meta:%s:%s", key, v),
					})
			}
		}
	} else {
		for k, v := range meta {
			if k != "" && v != "" {
				sList = append(sList,
					&spc.Selector{
						Type:  common.PluginName,
						Value: fmt.Sprintf("meta:%s:%s", k, v),
					})
			}
		}
	}

	return sList
}

// genInstanceIDFromSpiffeID returns InstanceID which is included spiffeID
func genInstanceIDFromSpiffeID(spiffeID string) (string, error) {
	u, err := idutil.ParseSpiffeID(spiffeID, idutil.AllowAnyTrustDomainAgent())
	if err != nil {
		return "", fmt.Errorf("unable to parse spiffeID %v: %v", spiffeID, err)
	}
	m := regexpAgentIDPath.FindStringSubmatch(u.Path)
	if m == nil {
		return "", fmt.Errorf("invalid spiffeID format: %v", spiffeID)
	}
	return m[2], nil
}

// getOpenStackInstance returns authenticated openstack compute client.
func getOpenStackInstance(cloud string, logger hclog.Logger) (openstack.InstanceClient, error) {
	provider, err := openstack.NewProvider(cloud)
	if err != nil {
		return nil, err
	}
	return openstack.NewInstance(provider, logger)
}

func (p *IIDResolverPlugin) SetLogger(log hclog.Logger) {
	p.logger = log
}

func (p *IIDResolverPlugin) setConfig(config *IIDResolverPluginConfig) {
	p.mtx.Lock()
	defer p.mtx.Unlock()
	p.config = config
}

func (p *IIDResolverPlugin) getConfig() (*IIDResolverPluginConfig, error) {
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
