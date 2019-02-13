package fake

import (
	"context"
	"io"

	"github.com/zlabjp/openstack-iid-attestor/pkg/common"

	spc "github.com/spiffe/spire/proto/common"

	"github.com/spiffe/spire/proto/server/nodeattestor"
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
