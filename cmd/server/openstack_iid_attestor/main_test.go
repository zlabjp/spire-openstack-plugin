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
	"encoding/json"
	"fmt"
	"strings"
	"sync"
	"testing"

	"github.com/hashicorp/go-hclog"
	"github.com/lestrrat-go/jwx/jwa"
	"github.com/spiffe/spire/proto/common/plugin"
	"github.com/spiffe/spire/proto/spire/common/plugin"
	"github.com/zlabjp/spire-openstack-plugin/pkg/openstack"
	"github.com/zlabjp/spire-openstack-plugin/pkg/testutil"
	"github.com/zlabjp/spire-openstack-plugin/pkg/util/fake"
)

const (
	testUUID      = "123"
	testProjectID = "abc"
	testKID       = "kid-xyz"

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
		mtx:    &sync.RWMutex{},
		logger: testutil.TestLogger(),
	}
}

func notAttestedBeforeHandler(p *IIDAttestorPlugin, ctx context.Context, agentID string) (bool, error) {
	return false, nil
}

func onceAttestedBeforeHandler(p *IIDAttestorPlugin, ctx context.Context, agentID string) (bool, error) {
	return true, nil
}

func TestConfigure(t *testing.T) {
	p := newTestPlugin()
	p.getInstanceHandler = func(n string, logger hclog.Logger) (openstack.InstanceClient, error) {
		return fake.NewInstance(testProjectID, nil, nil), nil
	}
	p.attestedBeforeHandler = notAttestedBeforeHandler

	ctx := context.Background()
	req := fake.NewFakeConfigureRequest(globalConfig, pluginConfig)

	_, err := p.Configure(ctx, req)
	if err != nil {
		t.Errorf("error from Configure(): %v", err)
	}
}

func TestConfigureError(t *testing.T) {
	p := newTestPlugin()
	p.getInstanceHandler = func(n string, logger hclog.Logger) (openstack.InstanceClient, error) {
		return fake.NewInstance(testProjectID, nil, nil), nil
	}
	p.attestedBeforeHandler = notAttestedBeforeHandler

	ctx := context.Background()
	req := fake.NewFakeConfigureRequest(globalConfig, "invalid config")

	_, err := p.Configure(ctx, req)
	if err == nil {
		t.Error("expected error, got nil")
	}
}

func TestConfigureEmptyProjectID(t *testing.T) {
	p := newTestPlugin()
	p.getInstanceHandler = func(n string, logger hclog.Logger) (openstack.InstanceClient, error) {
		return fake.NewInstance(testProjectID, nil, nil), nil
	}
	p.attestedBeforeHandler = notAttestedBeforeHandler

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
	p.attestedBeforeHandler = notAttestedBeforeHandler

	fs := fake.NewAttestStream(testUUID)

	if err := p.Attest(fs); err != nil {
		t.Errorf("Attestation error: %v", err)
	}
}

func TestAttestUseIID(t *testing.T) {
	fi := fake.NewInstance(testProjectID, nil, nil)
	p := newTestPlugin()
	p.instance = fi
	p.config.ProjectIDWhitelist = []string{testProjectID}
	p.config.UseIID = true

	privKey, err := ecdsa.GenerateKey(elliptic.P384(), rand.Reader)
	if err != nil {
		t.Errorf("Failed to generate test key")
	}
	s, err := fake.GenerateIIDByte(testUUID, testKID, jwa.ES384, privKey)
	if err != nil {
		t.Errorf("Failed to prepare test Vendordata2")
	}
	iid := &openstack.IID{}
	if err := json.Unmarshal(s, iid); err != nil {
		t.Errorf("Failed to unmarshal test IID data")
	}

	mc := &fake.MetadataClient{
		UUID:    testUUID,
		KID:     testKID,
		Alg:     jwa.ES384,
		PrivKey: privKey,
		PubKey:  &privKey.PublicKey,
		UseIID:  true,
	}
	p.metadata = mc
	p.verifyIIDHandler = verifyIIDSignature

	fs := fake.NewAttestStream(iid.Data, false)

	if err := p.Attest(fs); err != nil {
		t.Errorf("unexpected error from Attest(): %v", err)
	}
}

func TestAttestUseIIDWithInvalidData(t *testing.T) {
	fi := fake.NewInstance(testProjectID, nil, nil)
	p := newTestPlugin()
	p.instance = fi
	p.config.ProjectIDWhitelist = []string{testProjectID}
	p.config.UseIID = true

	privKey, err := ecdsa.GenerateKey(elliptic.P384(), rand.Reader)
	if err != nil {
		t.Errorf("Failed to generate test key")
	}
	s, err := fake.GenerateIIDByteWithInvalidSignature(testUUID, testKID, jwa.ES384, privKey)
	if err != nil {
		t.Errorf("Failed to prepare test Vendordata2")
	}
	iid := &openstack.IID{}
	if err := json.Unmarshal(s, iid); err != nil {
		t.Errorf("Failed to unmarshal test IID data")
	}

	mc := &fake.MetadataClient{
		UUID:    testUUID,
		KID:     testKID,
		Alg:     jwa.ES384,
		PrivKey: privKey,
		PubKey:  &privKey.PublicKey,
		UseIID:  true,
	}
	p.metadata = mc
	p.verifyIIDHandler = verifyIIDSignature

	fs := fake.NewAttestStream(iid.Data, false)

	if err := p.Attest(fs); err == nil {
		t.Errorf("an error expectd, got nil")
	} else if !strings.HasPrefix(err.Error(), "AttestationData is invalid") {
		t.Errorf("unexpected error messsage: %v", err)
	}
}

func TestAttestInvalidUUID(t *testing.T) {
	errMsg := "invalid uuid"
	fi := fake.NewErrorInstance(errMsg)

	p := newTestPlugin()
	p.instance = fi
	p.attestedBeforeHandler = notAttestedBeforeHandler

	fs := fake.NewAttestStream(testUUID)

	if err := p.Attest(fs); err == nil {
		t.Errorf("an error expected, got nil")
	} else if err.Error() != fmt.Sprintf("your IID is invalid: %v", errMsg) {
		t.Errorf("unexpected error messsage: %v", err)
	}
}

func TestAttestInvalidProjectID(t *testing.T) {
	fi := fake.NewInstance("invalid-project-id", nil, nil)

	p := newTestPlugin()
	p.instance = fi
	p.config.ProjectIDWhitelist = []string{testProjectID}
	p.attestedBeforeHandler = notAttestedBeforeHandler

	fs := fake.NewAttestStream(testUUID)

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

	p.attestedBeforeHandler = onceAttestedBeforeHandler

	fs := fake.NewAttestStream(testUUID)

	if err := p.Attest(fs); err == nil {
		t.Errorf("an error expected, got nil")
	} else if err.Error() != fmt.Sprintf("IID has already been used to attest an agent: %v", testUUID) {
		t.Errorf("an error expectd, got nil")
	}
}
