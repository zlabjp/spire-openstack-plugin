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
	"reflect"
	"strings"
	"testing"

	"github.com/hashicorp/go-hclog"
	"github.com/spiffe/spire/pkg/common/util"
	spc "github.com/spiffe/spire/proto/common"
	"github.com/spiffe/spire/proto/common/plugin"
	"github.com/spiffe/spire/proto/server/noderesolver"

	"github.com/zlabjp/spire-openstack-plugin/pkg/common"
	"github.com/zlabjp/spire-openstack-plugin/pkg/openstack"
	"github.com/zlabjp/spire-openstack-plugin/pkg/testutil"
	"github.com/zlabjp/spire-openstack-plugin/pkg/util/fake"
)

const (
	testProjectID  = "alpha"
	testInstanceID = "1234"
)

type fakeInstance struct {
	projectID string
	metaData  map[string]string
	secGroup  []map[string]interface{}
	errMsg    string
}

func (i *fakeInstance) getFakeOpenStackInstance(cloud string, logger hclog.Logger) (openstack.InstanceClient, error) {
	if i.errMsg != "" {
		return nil, errors.New(i.errMsg)
	} else {
		return fake.NewInstance(i.projectID, i.metaData, i.secGroup), nil
	}
}

func getFakeConfigureRequest() *plugin.ConfigureRequest {
	return &plugin.ConfigureRequest{
		Configuration: `
		     cloud_name = "test"
             custom_meta_data = true
             meta_data_keys = ["env", "role"]
		`,
	}
}

func getFakeResolveRequest(list []string) *noderesolver.ResolveRequest {
	return &noderesolver.ResolveRequest{
		BaseSpiffeIdList: list,
	}
}

func TestConfigure(t *testing.T) {
	fi := &fakeInstance{
		projectID: testProjectID,
	}

	p := New()
	p.logger = testutil.TestLogger()
	p.getInstanceHandler = fi.getFakeOpenStackInstance

	ctx := context.Background()
	req := getFakeConfigureRequest()

	_, err := p.Configure(ctx, req)
	if err != nil {
		t.Errorf("error from Configure(): %v", err)
	}
}

func TestConfigureError(t *testing.T) {
	errMsg := "fake error"
	fi := &fakeInstance{
		errMsg: errMsg,
	}

	p := New()
	p.logger = testutil.TestLogger()
	p.getInstanceHandler = fi.getFakeOpenStackInstance

	ctx := context.Background()
	req := getFakeConfigureRequest()

	wantErr := fmt.Errorf("failed to prepare OpenStack Client: %v", errMsg)

	_, err := p.Configure(ctx, req)
	if err == nil {
		t.Error("want error but got nil")
	} else if err.Error() != wantErr.Error() {
		t.Errorf("got %v, want %v", err, wantErr)
	}
}

func TestResolve(t *testing.T) {

	tCase := []struct {
		meta map[string]string
		sec  []map[string]interface{}
		want []*spc.Selector
	}{
		// 0: normal case
		{
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
			want: []*spc.Selector{
				{
					Type:  common.PluginName,
					Value: "meta:env:test",
				},
				{
					Type:  common.PluginName,
					Value: "meta:role:my-role",
				},
				{
					Type:  common.PluginName,
					Value: "sg:id:123",
				},
				{
					Type:  common.PluginName,
					Value: "sg:name:my-sg",
				},
			},
		},
		// 1: case has additional meta data
		{
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
			want: []*spc.Selector{
				{
					Type:  common.PluginName,
					Value: "meta:env:test",
				},
				{
					Type:  common.PluginName,
					Value: "meta:role:my-role",
				},
				{
					Type:  common.PluginName,
					Value: "sg:id:123",
				},
				{
					Type:  common.PluginName,
					Value: "sg:name:my-sg",
				},
			},
		},
		// 2: empty meta values
		{
			sec: []map[string]interface{}{
				{
					"id":   "123",
					"name": "my-sg",
				},
			},
			want: []*spc.Selector{
				{
					Type:  common.PluginName,
					Value: "sg:id:123",
				},
				{
					Type:  common.PluginName,
					Value: "sg:name:my-sg",
				},
			},
		},
		// 3: empty sec values
		{
			meta: map[string]string{
				"env":  "test",
				"role": "my-role",
			},
			want: []*spc.Selector{
				{
					Type:  common.PluginName,
					Value: "meta:env:test",
				},
				{
					Type:  common.PluginName,
					Value: "meta:role:my-role",
				},
			},
		},
		// 4: empty values
		{},
	}

	for i, tc := range tCase {
		fi := &fakeInstance{
			projectID: testProjectID,
			metaData:  tc.meta,
			secGroup:  tc.sec,
		}

		p := New()
		p.logger = testutil.TestLogger()
		p.getInstanceHandler = fi.getFakeOpenStackInstance

		ctx := context.Background()
		_, err := p.Configure(ctx, getFakeConfigureRequest())
		if err != nil {
			t.Errorf("#%v: failed to configure testing: %v", i, err)
		}

		testSpiffeID := fmt.Sprintf("spiffe://acme.com/spire/agent/openstack_iid/%v/%v", testProjectID, testInstanceID)

		req := getFakeResolveRequest([]string{testSpiffeID})

		util.SortSelectors(tc.want)
		wantSelector := &spc.Selectors{
			Entries: tc.want,
		}

		resp, err := p.Resolve(ctx, req)
		if err != nil {
			t.Errorf("#%v: error from Resolve(): %v", i, err)
		} else {
			e := resp.Map[testSpiffeID]
			if !reflect.DeepEqual(e, wantSelector) {
				t.Errorf("#%v: got %v, want %v", i, e, wantSelector)
			}
		}
	}
}

func TestGetInstanceIDFromSpiffeID(t *testing.T) {
	tCase := []struct {
		spiffeID string
		wantID   string
		wantErr  string
	}{
		// 0: normal case
		{
			spiffeID: "spiffe://example.com/spire/agent/openstack_iid/test-pj/test-instance-id",
			wantID:   "test-instance-id",
		},
		// 1: invalid spiffe id format
		{
			spiffeID: "spiffe://example.com/spire/agent/unknown-plugin/test-pj/test-instance-id",
			wantErr:  "invalid spiffeID format",
		},
		// 2: invalid format
		{
			spiffeID: "invalid-format",
			wantErr:  "unable to parse spiffeID",
		},
	}

	for i, tc := range tCase {
		id, err := genInstanceIDFromSpiffeID(tc.spiffeID)
		if tc.wantErr == "" {
			if err != nil {
				t.Errorf("#%v: unexpected error: %v", i, err)
			} else if id != tc.wantID {
				t.Errorf("#%v: got %v, want %v", i, id, tc.wantID)
			}
		} else {
			if !strings.HasPrefix(err.Error(), tc.wantErr) {
				t.Errorf("#%v: got %v, want %v", i, err, tc.wantErr)
			}
		}
	}
}
