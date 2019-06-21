/**
 * Copyright 2019, Z Lab Corporation. All rights reserved.
 *
 * For the full copyright and license information, please view the LICENSE
 * file that was distributed with this source code.
 */

package main

import (
	"context"
	"crypto/ecdsa"
	"crypto/elliptic"
	"crypto/rand"
	"errors"
	"fmt"
	"strings"
	"sync"
	"testing"

	"github.com/lestrrat-go/jwx/jwk"
	"github.com/spiffe/spire/proto/common/plugin"
	"github.com/spiffe/spire/proto/spire/common/plugin"

	"github.com/zlabjp/spire-openstack-plugin/pkg/openstack"
	"github.com/zlabjp/spire-openstack-plugin/pkg/testutil"
	"github.com/zlabjp/spire-openstack-plugin/pkg/util/fake"
)

func newTestPlugin() *IIDAttestorPlugin {
	return &IIDAttestorPlugin{
		config: &IIDAttestorPluginConfig{
			trustDomain: "example.com",
		},
		mtx:    &sync.RWMutex{},
		logger: testutil.TestLogger(),
	}
}

func newConfigureRequest() *plugin.ConfigureRequest {
	return &plugin.ConfigureRequest{
		GlobalConfig: &plugin.ConfigureRequest_GlobalConfig{
			TrustDomain: "example.com",
		},
	}
}

func TestConfigure(t *testing.T) {
	p := newTestPlugin()
	p.getMetadataHandler = func() (*openstack.Metadata, error) {
		return &openstack.Metadata{
			UUID:      "alpha",
			Name:      "bravo",
			ProjectID: "charlie",
		}, nil
	}

	ctx := context.Background()
	cReq := newConfigureRequest()

	_, err := p.Configure(ctx, cReq)
	if err != nil {
		t.Errorf("unexpected error from Configure(): %v", err)
	}
}

func TestConfigureUseIID(t *testing.T) {
	p := newTestPlugin()
	p.config.UseIID = true

	privKey, err := ecdsa.GenerateKey(elliptic.P384(), rand.Reader)
	if err != nil {
		t.Errorf("Failed to generate test key")
	}
	pubKey := &privKey.PublicKey

	var keys []jwk.Key
	k, err := jwk.New(pubKey)
	if err != nil {
		t.Errorf("Failed to generate jwk")
	}
	keys = append(keys, k)

	p.getMetadataHandler = func() (*openstack.Metadata, error) {
		return &openstack.Metadata{
			UUID:      "alpha",
			Name:      "bravo",
			ProjectID: "charlie",
		}, nil
	}

	p.getVendordataHandler = func() (*openstack.Vendordata2, error) {
		return &openstack.Vendordata2{
			IID: &openstack.IID{
				Data: "test-data",
			},
			IIDKeys: &openstack.IIDKeys{
				jwk.Set{
					Keys: keys,
				},
			},
		}, nil
	}

	ctx := context.Background()
	cReq := newConfigureRequest()
	cReq.Configuration = `
		use_iid = true
	`

	_, err = p.Configure(ctx, cReq)
	if err != nil {
		t.Errorf("unexpected error from Configure(): %v", err)
	}
}

func TestConfigureInvalidConfig(t *testing.T) {
	p := newTestPlugin()
	p.getMetadataHandler = func() (*openstack.Metadata, error) {
		return &openstack.Metadata{
			UUID:      "alpha",
			Name:      "bravo",
			ProjectID: "charlie",
		}, nil
	}

	ctx := context.Background()
	cReq := newConfigureRequest()
	cReq.Configuration = "invalid string"

	_, err := p.Configure(ctx, cReq)
	if !strings.HasPrefix(err.Error(), "failed to decode configuration file") {
		t.Errorf("unexpected error from Configure(): %v", err)
	}
}

func TestConfigureMetadataFailed(t *testing.T) {
	p := newTestPlugin()
	errMsg := "fake error"
	p.getMetadataHandler = func() (*openstack.Metadata, error) {
		return nil, errors.New(errMsg)
	}

	ctx := context.Background()
	cReq := newConfigureRequest()

	_, err := p.Configure(ctx, cReq)
	wantErr := fmt.Sprintf("failed to retrieve openstack metadta: %v", errMsg)
	if err.Error() != wantErr {
		t.Errorf("got %v, want %v", err.Error(), wantErr)
	}
}

func TestFetchAttestationData(t *testing.T) {
	p := newTestPlugin()
	p.metaData = &openstack.Metadata{
		UUID:      "alpha",
		ProjectID: "bravo",
	}

	f := fake.NewFakeFetchAttestationStream()

	if err := p.FetchAttestationData(f); err != nil {
		t.Errorf("unexpected error from FetchAttestationData(): %v", err)
	}
	if _, err := f.Recv(); err != nil {
		t.Errorf("unexptected error from stream.Recv(): %v", err)
	}
}

func TestFetchAttestationDataNoConfigure(t *testing.T) {
	p := newTestPlugin()
	p.config = nil

	errMsg := "plugin not configured"

	f := fake.NewFakeFetchAttestationStream()

	err := p.FetchAttestationData(f)
	if err == nil {
		t.Error("expected an error is occurred but got nil")
	}
	if err.Error() != errMsg {
		t.Errorf("got %v, want %v", err.Error(), errMsg)
	}
}

func TestFetchAttestationDataMetadataError(t *testing.T) {
	p := newTestPlugin()

	errMsg := "plugin not configured"

	f := fake.NewFakeFetchAttestationStream()

	err := p.FetchAttestationData(f)
	if err == nil {
		t.Error("expected an error is occurred but got nil")
	}
	if err.Error() != errMsg {
		t.Errorf("got %v, want %v", err.Error(), errMsg)
	}
}
