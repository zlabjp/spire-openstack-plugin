binary_dirs := $(shell cd cmd && find */* -maxdepth 0 -type d)
out_dir := out/bin

uname := $(shell uname -s)
ifeq (${uname},Linux)
	OS=linux
endif
ifeq (${uname},Darwin)
	OS=darwin
endif

export GO111MODULE=on
export GOPROXY=https://proxy.golang.org

build: $(binary_dirs)

build-linux: OS=linux
build-linux: build

build-darwin: OS=darwin
build-darwin: build

$(binary_dirs): clean
	cd cmd/$@ && GOOS=$(OS) GOARCH=amd64 go build -o ../../../$(out_dir)/$@  -i

test:
	go test -race ./cmd/... ./pkg/...

clean:
	go clean ./cmd/... ./pkg/...
	rm -rf out

.PHONY: all build build-linux build-darwin test clean
