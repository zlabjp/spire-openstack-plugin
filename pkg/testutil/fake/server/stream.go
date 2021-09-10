/**
 * Copyright 2021, Z Lab Corporation. All rights reserved.
 *
 * For the full copyright and license information, please view the LICENSE
 * file that was distributed with this source code.
 */

package server

import (
	"context"
	"io"

	nodeattestorv1 "github.com/spiffe/spire-plugin-sdk/proto/spire/plugin/server/nodeattestor/v1"
	"google.golang.org/grpc"
)

var (
	ctx = context.Background()
)

type AttestPluginStream struct {
	req  *nodeattestorv1.AttestRequest
	resp *nodeattestorv1.AttestResponse
	grpc.ServerStream
}

func NewAttestStream(data string) *AttestPluginStream {
	return &AttestPluginStream{
		req: &nodeattestorv1.AttestRequest{
			Request: &nodeattestorv1.AttestRequest_Payload{
				Payload: []byte(data),
			},
		},
	}
}

func (f *AttestPluginStream) Context() context.Context {
	return ctx
}

func (f *AttestPluginStream) Recv() (*nodeattestorv1.AttestRequest, error) {
	req := f.req
	f.req = nil
	if req == nil {
		return nil, io.EOF
	}
	return req, nil
}

func (f *AttestPluginStream) Send(resp *nodeattestorv1.AttestResponse) error {
	if f.resp != nil {
		return io.EOF
	}
	f.resp = resp
	return nil
}
