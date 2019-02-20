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

	"github.com/spiffe/spire/proto/agent/nodeattestor"
)

var (
	ctx = context.Background()
)

type FakeFetchAttestationDataStream struct {
	req  *nodeattestor.FetchAttestationDataRequest
	resp *nodeattestor.FetchAttestationDataResponse
}

func NewFakeFetchAttestationStream() *FakeFetchAttestationDataStream {
	return &FakeFetchAttestationDataStream{
		req: new(nodeattestor.FetchAttestationDataRequest),
	}
}

func (f *FakeFetchAttestationDataStream) Context() context.Context {
	return ctx
}

func (f *FakeFetchAttestationDataStream) Recv() (*nodeattestor.FetchAttestationDataRequest, error) {
	req := f.req
	f.req = nil
	if req == nil {
		return nil, io.EOF
	}
	return req, nil
}

func (f *FakeFetchAttestationDataStream) Send(resp *nodeattestor.FetchAttestationDataResponse) error {
	if f.resp != nil {
		return io.EOF
	}
	f.resp = resp
	return nil
}
