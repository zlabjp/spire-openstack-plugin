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
	"github.com/gophercloud/gophercloud/openstack/compute/v2/servers"
)

type InstanceClient interface {
	// Get retrieves a instance information from Provider
	Get(uuid string) (*servers.Server, error)
}

// Instance represents a OpenStack Compute Service client
type Instance struct {
	serviceClient *gophercloud.ServiceClient
}

// NewInstance returns a new OpenStack Compute Service client with given provider
func NewInstance(client *gophercloud.ProviderClient) (InstanceClient, error) {
	sc, err := openstack.NewComputeV2(client, gophercloud.EndpointOpts{})
	if err != nil {
		return nil, err
	}
	return &Instance{
		serviceClient: sc,
	}, nil
}

func (i *Instance) Get(uuid string) (*servers.Server, error) {
	return servers.Get(i.serviceClient, uuid).Extract()
}
