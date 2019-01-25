package main

import (
	"fmt"
	"sync"
	"testing"
	"time"

	"github.com/zlabjp/openstack-iid-attestor/pkg/util/fake"
)

const (
	testUUID      = "123"
	testProjectID = "abc"
	testPeriod    = "1h"
)

func newTestPlugin() *IIDAttestorPlugin {
	return &IIDAttestorPlugin{
		config: &IIDAttestorPluginConfig{
			trustDomain: "example.com",
		},
		mtx: &sync.RWMutex{},
	}
}

func TestAttest(t *testing.T) {
	fi := fake.NewInstance(testProjectID)

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
	fi := fake.NewInstance("invalid-project-id")

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
	fi := fake.NewInstance(testProjectID)

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
