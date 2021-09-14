/**
 * Copyright 2019, Z Lab Corporation. All rights reserved.
 *
 * For the full copyright and license information, please view the LICENSE
 * file that was distributed with this source code.
 */

package main

import (
	"errors"
	"fmt"
	"testing"

	"github.com/zlabjp/spire-openstack-plugin/pkg/openstack"
	"github.com/zlabjp/spire-openstack-plugin/pkg/testutil"
	fake_agent "github.com/zlabjp/spire-openstack-plugin/pkg/testutil/fake/agent"
)

func newTestPlugin() *IIDAttestorPlugin {
	return &IIDAttestorPlugin{
		logger: testutil.TestLogger(),
	}
}

func TestAidAttestation(t *testing.T) {
	p := newTestPlugin()
	p.getMetadataHandler = func() (*openstack.Metadata, error) {
		return &openstack.Metadata{
			UUID:      "alpha",
			Name:      "bravo",
			ProjectID: "charlie",
		}, nil
	}

	f := fake_agent.NewAidAttestationStream()

	if err := p.AidAttestation(f); err != nil {
		t.Errorf("unexpected error from FetchAttestationData(): %v", err)
	}
	if _, err := f.Recv(); err != nil {
		t.Errorf("unexptected error from stream.Recv(): %v", err)
	}
}

func TestFetchAttestationDataMetadataHandlerFailed(t *testing.T) {
	p := newTestPlugin()
	errMsg := "fake error"
	p.getMetadataHandler = func() (*openstack.Metadata, error) {
		return nil, errors.New(errMsg)
	}

	f := fake_agent.NewAidAttestationStream()
	wantErr := fmt.Sprintf("failed to retrieve openstack metadata: %v", errMsg)

	if err := p.AidAttestation(f); err == nil {
		t.Errorf("Expected an error, got nil: %v", err)
	} else {
		if err.Error() != wantErr {
			t.Errorf("got %v, want %v", err.Error(), wantErr)
		}
	}

}

func TestFetchAttestationDataMetadataHandlerNotFound(t *testing.T) {
	p := newTestPlugin()

	errMsg := "handler not found, plugin not initialized"

	f := fake_agent.NewAidAttestationStream()

	err := p.AidAttestation(f)
	if err == nil {
		t.Error("expected an error is occurred but got nil")
	} else {
		if err.Error() != errMsg {
			t.Errorf("got %v, want %v", err.Error(), errMsg)
		}
	}
}
