/**
 * Copyright 2019, Z Lab Corporation. All rights reserved.
 *
 * For the full copyright and license information, please view the LICENSE
 * file that was distributed with this source code.
 */

package common

import (
	"fmt"
	"testing"
)

func TestGenerateSpiffeID(t *testing.T) {
	domain := "example.com"
	projectID := "alpha"
	instanceID := "bravo"

	want := fmt.Sprintf("spiffe://%v/spire/agent/%v/%v/%v", domain, PluginName, projectID, instanceID)

	if got := GenerateSpiffeID(domain, projectID, instanceID); got != want {
		t.Errorf("got %v, want %v", got, want)
	}
}
