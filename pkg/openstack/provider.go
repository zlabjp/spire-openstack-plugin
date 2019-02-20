/**
 * Copyright 2019, Z Lab Corporation. All rights reserved.
 *
 * For the full copyright and license information, please view the LICENSE
 * file that was distributed with this source code.
 */

package openstack

import (
	"github.com/gophercloud/gophercloud"
	"github.com/gophercloud/gophercloud/openstack"
	"github.com/gophercloud/utils/openstack/clientconfig"
)

// NewProvider returns a new authenticated ProviderClient
func NewProvider(cloudName string) (*gophercloud.ProviderClient, error) {
	opts := &clientconfig.ClientOpts{
		Cloud: cloudName,
	}
	authOpts, err := clientconfig.AuthOptions(opts)
	if err != nil {
		return nil, err
	}
	authOpts.AllowReauth = true

	provider, err := openstack.AuthenticatedClient(*authOpts)
	if err != nil {
		return nil, err
	}

	return provider, nil
}
