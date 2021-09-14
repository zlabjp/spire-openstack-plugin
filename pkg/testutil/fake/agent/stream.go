/**
 * Copyright 2021, Z Lab Corporation. All rights reserved.
 *
 * For the full copyright and license information, please view the LICENSE
 * file that was distributed with this source code.
 */

package agent

import (
	"context"
	"io"

	"google.golang.org/grpc"

	nodeattestorv1 "github.com/spiffe/spire-plugin-sdk/proto/spire/plugin/agent/nodeattestor/v1"
)

var (
	ctx = context.Background()
)

type AidAttestationServer struct {
	req  *nodeattestorv1.Challenge
	resp *nodeattestorv1.PayloadOrChallengeResponse
	grpc.ServerStream
}

func NewAidAttestationStream() *AidAttestationServer {
	return &AidAttestationServer{
		req: new(nodeattestorv1.Challenge),
	}
}

func (f *AidAttestationServer) Context() context.Context {
	return ctx
}

func (f *AidAttestationServer) Recv() (*nodeattestorv1.Challenge, error) {
	req := f.req
	f.req = nil
	if req == nil {
		return nil, io.EOF
	}
	return req, nil
}

func (f *AidAttestationServer) Send(resp *nodeattestorv1.PayloadOrChallengeResponse) error {
	if f.resp != nil {
		return io.EOF
	}
	f.resp = resp
	return nil
}
