/**
 * Copyright 2019, Z Lab Corporation. All rights reserved.
 *
 * For the full copyright and license information, please view the LICENSE
 * file that was distributed with this source code.
 */

package main

import (
	"context"
	"fmt"
	"sync"
	"testing"

	"github.com/spiffe/spire/proto/common/plugin"
	"github.com/zlabjp/spire-openstack-plugin/pkg/openstack"
	"github.com/zlabjp/spire-openstack-plugin/pkg/util/fake"
)

const (
	testUUID      = "123"
	testProjectID = "abc"

	pluginConfig = `
	cloud_name = "test"
	projectid_whitelist = ["alpha", "bravo"]
	`
)

var (
	globalConfig = &plugin.ConfigureRequest_GlobalConfig{
		TrustDomain: "example.com",
	}
)

func newTestPlugin() *IIDAttestorPlugin {
	return &IIDAttestorPlugin{
		config: &IIDAttestorPluginConfig{
			trustDomain: "example.com",
		},
		mtx: &sync.RWMutex{},
	}
}

func TestConfigure(t *testing.T) {
	p := newTestPlugin()
	p.getInstanceHandler = func(n string) (openstack.InstanceClient, error) {
		return fake.NewInstance(testProjectID, nil, nil), nil
	}

	ctx := context.Background()
	req := fake.NewFakeConfigureRequest(globalConfig, pluginConfig)

	_, err := p.Configure(ctx, req)
	if err != nil {
		t.Errorf("error from Configure(): %v", err)
	}
}

func TestConfigureError(t *testing.T) {
	p := newTestPlugin()
	p.getInstanceHandler = func(n string) (openstack.InstanceClient, error) {
		return fake.NewInstance(testProjectID, nil, nil), nil
	}

	ctx := context.Background()
	req := fake.NewFakeConfigureRequest(globalConfig, "invalid config")

	_, err := p.Configure(ctx, req)
	if err == nil {
		t.Error("expected error, got nil")
	}
}

func TestConfigureEmptyProjectID(t *testing.T) {
	p := newTestPlugin()
	p.getInstanceHandler = func(n string) (openstack.InstanceClient, error) {
		return fake.NewInstance(testProjectID, nil, nil), nil
	}

	conf := `
	cloud_name = "test"
	`

	ctx := context.Background()
	req := fake.NewFakeConfigureRequest(globalConfig, conf)

	wantError := "projectid_whitelist is required"
	_, err := p.Configure(ctx, req)
	if err == nil {
		t.Error("expected error, got nil")
	} else if err.Error() != wantError {
		t.Errorf("got %v, wantPrefix %v", err, wantError)
	}
}

func TestAttest(t *testing.T) {
	fi := fake.NewInstance(testProjectID, nil, nil)

	p := newTestPlugin()
	p.instance = fi
	p.config.ProjectIDWhitelist = []string{testProjectID}

	fs := fake.NewAttestStream(testUUID, false)

	if err := p.Attest(fs); err != nil {
		t.Errorf("Attestation error: %v", err)
	}
}

func TestAttestInvalidUUID(t *testing.T) {
	errMsg := "invalid uuid"
	fi := fake.NewErrorInstance(errMsg)

	p := newTestPlugin()
	p.instance = fi

	fs := fake.NewAttestStream(testUUID, false)

	if err := p.Attest(fs); err == nil {
		t.Errorf("unexpected error from Attest(): %v", err)
	} else if err.Error() != fmt.Sprintf("your IID is invalid: %v", errMsg) {
		t.Errorf("unexpected error messsage: %v", err)
	}
}

func TestAttestInvalidProjectID(t *testing.T) {
	fi := fake.NewInstance("invalid-project-id", nil, nil)

	p := newTestPlugin()
	p.instance = fi
	p.config.ProjectIDWhitelist = []string{testProjectID}

	fs := fake.NewAttestStream(testUUID, false)

	if err := p.Attest(fs); err == nil {
		t.Errorf("an error expectd, got nil")
	} else if err.Error() != "invalid attestation request" {
		t.Errorf("unexpected error messsage: %v", err)
	}
}

func TestAttestBefore(t *testing.T) {
	fi := fake.NewInstance(testProjectID, nil, nil)

	p := newTestPlugin()
	p.instance = fi
	p.config.ProjectIDWhitelist = []string{testProjectID}

	fs := fake.NewAttestStream(testUUID, true)

	if err := p.Attest(fs); err == nil {
		t.Errorf("an error expectd, got nil")
	} else if err.Error() != fmt.Sprintf("the IID has been used and is no longer valid: %v", testUUID) {
		t.Errorf("unexpected error messsage: %v", err)
	}
}
