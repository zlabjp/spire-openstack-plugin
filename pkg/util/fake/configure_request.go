/**
 * Copyright 2019, Z Lab Corporation. All rights reserved.
 *
 * For the full copyright and license information, please view the LICENSE
 * file that was distributed with this source code.
 */

package fake

import "github.com/spiffe/spire/proto/spire/common/plugin"

func NewFakeConfigureRequest(g *plugin.ConfigureRequest_GlobalConfig, p string) *plugin.ConfigureRequest {
	return &plugin.ConfigureRequest{
		GlobalConfig:  g,
		Configuration: p,
	}
}
