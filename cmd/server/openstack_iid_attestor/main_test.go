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
	"reflect"
	"sync"
	"testing"

	"github.com/hashicorp/go-hclog"
	"github.com/zlabjp/spire-openstack-plugin/pkg/openstack"

	configv1 "github.com/spiffe/spire-plugin-sdk/proto/spire/service/common/config/v1"
	fake_common "github.com/zlabjp/spire-openstack-plugin/pkg/testutil/fake/common"
	fake_openstack "github.com/zlabjp/spire-openstack-plugin/pkg/testutil/fake/openstack"
	fake_server "github.com/zlabjp/spire-openstack-plugin/pkg/testutil/fake/server"

	"github.com/zlabjp/spire-openstack-plugin/pkg/testutil"
)

const (
	testUUID      = "123"
	testProjectID = "abc"
)

var (
	globalConfig = &configv1.CoreConfiguration{
		TrustDomain: "example.com",
	}

	pluginConfig = `
	cloud_name = "test"
	projectid_allow_list = ["alpha", "bravo"]
    custom_metadata = { 
      keys = ["env", "role"]
    }
	`
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

func notAttestedBeforeHandler(_ context.Context, _ *IIDAttestorPlugin, _ string) (bool, error) {
	return false, nil
}

func onceAttestedBeforeHandler(_ context.Context, _ *IIDAttestorPlugin, _ string) (bool, error) {
	return true, nil
}

func TestConfigure(t *testing.T) {
	p := newTestPlugin()
	p.getInstanceHandler = func(n string, logger hclog.Logger) (openstack.InstanceClient, error) {
		return fake_openstack.NewInstance(testProjectID, nil, nil), nil
	}
	p.attestedBeforeHandler = notAttestedBeforeHandler

	ctx := context.Background()
	req := fake_common.NewConfigureRequest(globalConfig, pluginConfig)

	_, err := p.Configure(ctx, req)
	if err != nil {
		t.Errorf("error from Configure(): %v", err)
	}
}

func TestConfigureError(t *testing.T) {
	p := newTestPlugin()
	p.getInstanceHandler = func(n string, logger hclog.Logger) (openstack.InstanceClient, error) {
		return fake_openstack.NewInstance(testProjectID, nil, nil), nil
	}
	p.attestedBeforeHandler = notAttestedBeforeHandler

	ctx := context.Background()
	req := fake_common.NewConfigureRequest(globalConfig, "invalid config")

	_, err := p.Configure(ctx, req)
	if err == nil {
		t.Error("expected error, got nil")
	}
}

func TestConfigureEmptyProjectID(t *testing.T) {
	p := newTestPlugin()
	p.getInstanceHandler = func(n string, logger hclog.Logger) (openstack.InstanceClient, error) {
		return fake_openstack.NewInstance(testProjectID, nil, nil), nil
	}
	p.attestedBeforeHandler = notAttestedBeforeHandler

	conf := `
	cloud_name = "test"
	`

	ctx := context.Background()
	req := fake_common.NewConfigureRequest(globalConfig, conf)

	wantError := "projectid_allow_list is required"
	_, err := p.Configure(ctx, req)
	if err == nil {
		t.Error("expected error, got nil")
	} else if err.Error() != wantError {
		t.Errorf("got %v, wantPrefix %v", err, wantError)
	}
}

func TestAttest(t *testing.T) {
	p := newTestPlugin()
	p.getInstanceHandler = func(n string, logger hclog.Logger) (openstack.InstanceClient, error) {
		return fake_openstack.NewInstance(testProjectID, nil, nil), nil
	}
	p.config.ProjectIDAllowList = []string{testProjectID}
	p.attestedBeforeHandler = notAttestedBeforeHandler

	fs := fake_server.NewAttestStream(testUUID)

	if err := p.Attest(fs); err != nil {
		t.Errorf("Attestation error: %v", err)
	}
}

func TestMakeSelectorValues(t *testing.T) {
	for i, tc := range []struct {
		keys []string
		meta map[string]string
		sec  []map[string]interface{}
		want []string
	}{
		// 0: normal case
		{
			keys: []string{},
			meta: map[string]string{
				"env":  "test",
				"role": "my-role",
			},
			sec: []map[string]interface{}{
				{
					"id":   "123",
					"name": "my-sg",
				},
			},
			want: []string{
				"meta:env:test",
				"meta:role:my-role",
				"sg:id:123",
				"sg:name:my-sg",
			},
		},
		// 1: case has additional meta data
		{
			keys: []string{"env", "role"},
			meta: map[string]string{
				"env":     "test",
				"role":    "my-role",
				"charlie": "delta",
				"echo":    "foxtrot",
			},
			sec: []map[string]interface{}{
				{
					"id":   "123",
					"name": "my-sg",
				},
			},
			want: []string{
				"meta:env:test",
				"meta:role:my-role",
				"sg:id:123",
				"sg:name:my-sg",
			},
		},
		// 2: empty meta values
		{
			keys: []string{},
			sec: []map[string]interface{}{
				{
					"id":   "123",
					"name": "my-sg",
				},
			},
			want: []string{
				"sg:id:123",
				"sg:name:my-sg",
			},
		},
		// 3: empty sec values
		{
			keys: []string{},
			meta: map[string]string{
				"env":  "test",
				"role": "my-role",
			},
			want: []string{
				"meta:env:test",
				"meta:role:my-role",
			},
		},
		// 4: empty values
		{},
	} {
		fi := fake_openstack.NewInstance(testProjectID, tc.meta, tc.sec)

		p := newTestPlugin()
		p.logger = testutil.TestLogger()
		p.instance = fi
		p.config.CustomMetaData = &CustomMetadata{
			Keys: tc.keys,
		}

		server, _ := p.instance.Get(testUUID)
		resp, err := p.makeSelectorValues(server)
		if err != nil {
			t.Errorf("#%v: Error from makeSelectors(): %v", i, err)
		}

		if !reflect.DeepEqual(resp, tc.want) {
			t.Errorf("#%v: got %v, want %v", i, resp, tc.want)
		}
	}
}

func TestAttestInvalidUUID(t *testing.T) {
	errMsg := "invalid uuid"

	p := newTestPlugin()
	p.getInstanceHandler = func(n string, logger hclog.Logger) (openstack.InstanceClient, error) {
		return fake_openstack.NewErrorInstance(errMsg), nil
	}
	p.attestedBeforeHandler = notAttestedBeforeHandler

	fs := fake_server.NewAttestStream(testUUID)

	if err := p.Attest(fs); err == nil {
		t.Errorf("an error expected, got nil")
	} else if err.Error() != fmt.Sprintf("failed to get instance information: %v", errMsg) {
		t.Errorf("unexpected error messsage: %v", err)
	}
}

func TestAttestInvalidProjectID(t *testing.T) {
	p := newTestPlugin()
	p.getInstanceHandler = func(n string, logger hclog.Logger) (openstack.InstanceClient, error) {
		return fake_openstack.NewInstance("invalid-project-id", nil, nil), nil
	}
	p.config.ProjectIDAllowList = []string{testProjectID}
	p.attestedBeforeHandler = notAttestedBeforeHandler

	fs := fake_server.NewAttestStream(testUUID)

	if err := p.Attest(fs); err == nil {
		t.Errorf("an error expected, got nil")
	} else if err.Error() != "invalid attestation request" {
		t.Errorf("unexpected error messsage: %v", err)
	}
}

func TestAttestBefore(t *testing.T) {
	p := newTestPlugin()
	p.getInstanceHandler = func(n string, logger hclog.Logger) (openstack.InstanceClient, error) {
		return fake_openstack.NewInstance(testProjectID, nil, nil), nil
	}
	p.config.ProjectIDAllowList = []string{testProjectID}

	p.attestedBeforeHandler = onceAttestedBeforeHandler

	fs := fake_server.NewAttestStream(testUUID)

	if err := p.Attest(fs); err == nil {
		t.Errorf("an error expected, got nil")
	} else if err.Error() != fmt.Sprintf("IID has already been used to attest an agent: %v", testUUID) {
		t.Errorf("unexpected error messsage: %v", err)
	}
}
