binary_dirs := $(shell cd cmd && find */* -maxdepth 0 -type d)
out_dir := out/bin

uname := $(shell uname -s)
ifeq (${uname},Linux)
	OS=linux
endif
ifeq (${uname},Darwin)
	OS=darwin
endif

utils = github.com/goreleaser/goreleaser \
		github.com/golang/dep/cmd/dep

build: $(binary_dirs)

build-linux: OS=linux
build-linux: build

build-darwin: OS=darwin
build-darwin: build

$(binary_dirs): clean
	cd cmd/$@ && GOOS=$(OS) GOARCH=amd64 go build -o ../../../$(out_dir)/$@  -i

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
	rm -rf out

noop:

.PHONY: all build build-linux build-darwin vendor utils test clean
