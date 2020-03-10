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
	"github.com/spiffe/spire/pkg/agent/plugin/nodeattestor"
	"github.com/spiffe/spire/pkg/common/catalog"
	spc "github.com/spiffe/spire/proto/spire/common"
	spi "github.com/spiffe/spire/proto/spire/common/plugin"

	"github.com/zlabjp/spire-openstack-plugin/pkg/common"
	"github.com/zlabjp/spire-openstack-plugin/pkg/openstack"
)

// IIDAttestorPlugin implements the nodeattestor Plugin interface
type IIDAttestorPlugin struct {
	logger     hclog.Logger
	config     *IIDAttestorPluginConfig
	metaData   *openstack.Metadata
	vendordata *openstack.Vendordata2

	mtx *sync.RWMutex

	getMetadataHandler   func() (*openstack.Metadata, error)
	getVendordataHandler func() (*openstack.Vendordata2, error)
}

type IIDAttestorPluginConfig struct {
	trustDomain string
	UseIID      bool `hcl:"use_iid"`
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
		mtx:                  &sync.RWMutex{},
		getMetadataHandler:   getMetadata,
		getVendordataHandler: getVendordata,
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

	meta, err := p.getMetadataHandler()
	if err != nil {
		return nil, fmt.Errorf("failed to retrieve openstack metadta: %v", err)
	}
	p.metaData = meta

	if config.UseIID {
		v, err := p.getVendordataHandler()
		if err != nil {
			return nil, fmt.Errorf("failed to retrieve openstack metadta: %v", err)
		}
		p.vendordata = v
	}

	config.trustDomain = req.GlobalConfig.TrustDomain
	p.config = config

	return &spi.ConfigureResponse{}, nil
}

func (p *IIDAttestorPlugin) GetPluginInfo(context.Context, *spi.GetPluginInfoRequest) (*spi.GetPluginInfoResponse, error) {
	return &spi.GetPluginInfoResponse{}, nil
}

func (p *IIDAttestorPlugin) FetchAttestationData(stream nodeattestor.NodeAttestor_FetchAttestationDataServer) error {
	p.logger.Info("Prepare Attestation Request")

	p.mtx.RLock()
	defer p.mtx.RUnlock()

	if p.config == nil || (p.metaData == nil && p.vendordata == nil) {
		return errors.New("plugin not configured")
	}

	var data []byte
	if p.vendordata != nil {
		data = []byte(p.vendordata.IID.Data)
	} else {
		data = []byte(p.metaData.UUID)
	}

	return stream.Send(&nodeattestor.FetchAttestationDataResponse{
		AttestationData: &spc.AttestationData{
			Type: common.PluginName,
			Data: data,
		},
	})
}

func (p *IIDAttestorPlugin) SetLogger(log hclog.Logger) {
	p.logger = log
}

func getMetadata() (*openstack.Metadata, error) {
	c := openstack.NewMetadataClient()
	c.SetMetadataAsTarget()

	obj, err := c.GetMetadataFromMetadataService()
	if err != nil {
		return nil, err
	}
	return obj.(*openstack.Metadata), nil
}

func getVendordata() (*openstack.Vendordata2, error) {
	c := openstack.NewMetadataClient()
	c.SetDynamicJSONAsTarget()

	obj, err := c.GetMetadataFromMetadataService()
	if err != nil {
		return nil, err
	}
	return obj.(*openstack.Vendordata2), nil
}

func main() {
	catalog.PluginMain(BuiltIn())
}
