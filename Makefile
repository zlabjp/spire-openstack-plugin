binary_dirs := agent server
out_dir := bin

utils = github.com/goreleaser/goreleaser \
		github.com/golang/dep/cmd/dep

build: $(binary_dirs)

$(binary_dirs): noop
	cd cmd/$@ && go build -o ../../$(out_dir)/$@/openstack_iid_attestor  -i

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
