/**
 * Copyright 2021, Z Lab Corporation. All rights reserved.
 *
 * For the full copyright and license information, please view the LICENSE
 * file that was distributed with this source code.
 */

package common

import (
	configv1 "github.com/spiffe/spire-plugin-sdk/proto/spire/service/common/config/v1"
)

func NewConfigureRequest(g *configv1.CoreConfiguration, p string) *configv1.ConfigureRequest {
	return &configv1.ConfigureRequest{
		CoreConfiguration: g,
		HclConfiguration:  p,
	}
}
