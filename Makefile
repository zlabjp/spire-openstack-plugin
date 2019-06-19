binary_dirs := $(shell cd cmd && find */* -maxdepth 0 -type d)
out_dir := out/bin

uname := $(shell uname -s)
ifeq (${uname},Linux)
	OS=linux
endif
ifeq (${uname},Darwin)
	OS=darwin
endif

build: $(binary_dirs)

build-linux: OS=linux
build-linux: build

build-darwin: OS=darwin
build-darwin: build

$(binary_dirs): clean
	cd cmd/$@ && GO111MODULE=on GOOS=$(OS) GOARCH=amd64 go build -o ../../../$(out_dir)/$@  -i

test:
	GO111MODULE=on go test -race ./cmd/... ./pkg/...

clean:
	GO111MODULE=on go clean ./cmd/... ./pkg/...
	rm -rf out

noop:

.PHONY: all build build-linux build-darwin vendor utils test clean
