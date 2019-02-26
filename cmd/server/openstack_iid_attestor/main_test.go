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
	"strings"
	"sync"
	"testing"
	"time"

	"github.com/spiffe/spire/proto/common/plugin"

	"github.com/zlabjp/spire-openstack-plugin/pkg/openstack"
	"github.com/zlabjp/spire-openstack-plugin/pkg/util/fake"
)

const (
	testUUID      = "123"
	testProjectID = "abc"
	testPeriod    = "1h"

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

func TestConfigureInvalidPeriodValue(t *testing.T) {
	p := newTestPlugin()
	p.getInstanceHandler = func(n string) (openstack.InstanceClient, error) {
		return fake.NewInstance(testProjectID, nil, nil), nil
	}

	conf := `
	cloud_name = "test"
	projectid_whitelist = ["alpha", "bravo"]
	attestation_period = "invalid"
	`

	ctx := context.Background()
	req := fake.NewFakeConfigureRequest(globalConfig, conf)

	wantErrorPrefix := "invalid value for attestation_period"
	_, err := p.Configure(ctx, req)
	if err == nil {
		t.Error("expected error, got nil")
	} else if !strings.HasPrefix(err.Error(), wantErrorPrefix) {
		t.Errorf("got %v, wantPrefix %v", err, wantErrorPrefix)
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

func TestAttestPeriod(t *testing.T) {
	fi := fake.NewInstanceWithTime(testProjectID, time.Now().Add(-time.Second*6000))
	period, err := time.ParseDuration(testPeriod)
	if err != nil {
		t.Errorf("error before testing: %v", err)
	}

	p := newTestPlugin()
	p.instance = fi
	p.config.ProjectIDWhitelist = []string{testProjectID}
	p.period = time.Duration(period)

	fs := fake.NewAttestStream(testUUID, false)

	if err := p.Attest(fs); err == nil {
		t.Errorf("an error expectd, got nil")
	} else if err.Error() != "attestation period has expired" {
		t.Errorf("unexpected error message: %v", err)
	}
}
