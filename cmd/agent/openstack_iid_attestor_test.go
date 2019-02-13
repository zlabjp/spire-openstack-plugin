package main

import (
	"context"
	"errors"
	"fmt"
	"strings"
	"sync"
	"testing"

	"github.com/spiffe/spire/proto/common/plugin"

	"github.com/zlabjp/openstack-iid-attestor/pkg/openstack"
	"github.com/zlabjp/openstack-iid-attestor/pkg/util/fake"
)

func newTestPlugin() *IIDAttestorPlugin {
	return &IIDAttestorPlugin{
		config: &IIDAttestorPluginConfig{
			trustDomain: "example.com",
		},
		mtx: &sync.RWMutex{},
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
	if !strings.HasPrefix(err.Error(), "failed to parse configuration file") {
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
