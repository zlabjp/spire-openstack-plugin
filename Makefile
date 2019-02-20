attestor_dirs := agent server
out_dir := out/bin

utils = github.com/goreleaser/goreleaser \
		github.com/golang/dep/cmd/dep

build: build_attestor

build_attestor: $(attestor_dirs)

$(attestor_dirs): noop
	cd cmd/$@/openstack_iid_attestor && go build -o ../../../$(out_dir)/$@/openstack_iid_attestor  -i

utils: $(utils)

$(utils): noop
	go get $@

vendor: Gopkg.lock Gopkg.toml
	dep ensure

revendor:
	rm Gopkg.lock Gopkg.toml
	rm -rf vendor
	dep init

test:
	go test -race ./cmd/... ./pkg/...

release:
	goreleaser || true

clean:
	go clean ./cmd/... ./pkg/...
	rm -rf bin

noop:

.PHONY: all build vendor utils test clean
