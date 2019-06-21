/**
 * Copyright 2019, Z Lab Corporation. All rights reserved.
 *
 * For the full copyright and license information, please view the LICENSE
 * file that was distributed with this source code.
 */

package fake

import (
	"encoding/json"

	"github.com/lestrrat-go/jwx/jwa"

	"github.com/zlabjp/spire-openstack-plugin/pkg/openstack"
)

type MetadataClient struct {
	UUID    string
	KID     string
	Alg     jwa.SignatureAlgorithm
	PrivKey interface{}
	PubKey  interface{}
	UseIID  bool
}

func (mc *MetadataClient) SetVersion(version string) {
}

func (mc *MetadataClient) SetMetadataAsTarget() {
}

func (mc *MetadataClient) SetDynamicJSONAsTarget() {
}

func (mc *MetadataClient) GetTargetURL() string {
	return "http://example.com/fake"
}

func (mc *MetadataClient) GetMetadataFromMetadataService() (interface{}, error) {
	if mc.UseIID {
		IIDByte, err := GenerateIIDByte(mc.UUID, mc.KID, mc.Alg, mc.PrivKey)
		if err != nil {
			return nil, err
		}
		IIDKeysByte, err := GenerateIIDKeysByte(mc.KID, mc.Alg, mc.PubKey)
		if err != nil {
			return nil, err
		}

		var data openstack.IID
		if err := json.Unmarshal(IIDByte, &data); err != nil {
			return nil, err
		}

		var iidKeys openstack.IIDKeys
		if err := json.Unmarshal(IIDKeysByte, &iidKeys); err != nil {
			return nil, err
		}

		resp := &openstack.Vendordata2{
			IID:     &data,
			IIDKeys: &iidKeys,
		}

		b, err := json.Marshal(resp)
		if err != nil {
			return nil, err
		}

		var iid openstack.Vendordata2
		if err := json.Unmarshal(b, &iid); err != nil {
			return nil, err
		}

		return &iid, nil
	}
	// TOOD: impl metadata
	return Instance{}, nil
}
