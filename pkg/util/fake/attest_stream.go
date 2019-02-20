/**
 * Copyright 2019, Z Lab Corporation. All rights reserved.
 *
 * For the full copyright and license information, please view the LICENSE
 * file that was distributed with this source code.
 */

package fake

import (
	"context"
	"io"

	spc "github.com/spiffe/spire/proto/common"
	"github.com/spiffe/spire/proto/server/nodeattestor"

	"github.com/zlabjp/spire-openstack-plugin/pkg/common"
)

type AttestPluginStream struct {
	req  *nodeattestor.AttestRequest
	resp *nodeattestor.AttestResponse
}

func NewAttestStream(uuid string, attestedBefore bool) *AttestPluginStream {
	return &AttestPluginStream{
		req: &nodeattestor.AttestRequest{
			AttestationData: &spc.AttestationData{
				Type: common.PluginName,
				Data: []byte(uuid),
			},
			AttestedBefore: attestedBefore,
		},
	}
}

func (f *AttestPluginStream) Context() context.Context {
	return ctx
}

func (f *AttestPluginStream) Recv() (*nodeattestor.AttestRequest, error) {
	req := f.req
	f.req = nil
	if req == nil {
		return nil, io.EOF
	}
	return req, nil
}

func (f *AttestPluginStream) Send(resp *nodeattestor.AttestResponse) error {
	if f.resp != nil {
		return io.EOF
	}
	f.resp = resp
	return nil
}
