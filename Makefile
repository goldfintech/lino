COMMIT := $(shell git log -1 --format='%H')
PACKAGES=$(shell go list ./... | grep -v '/vendor/')
LD_FLAGS := "-X github.com/tendermint/tendermint/version.GitCommit=$(COMMIT) -X vendor/github.com/cosmos/cosmos-sdk/types.DBBackend=boltdb"
GO_TAGS := "tendermint boltdb"

all: get_tools get_vendor_deps install build test

get_tools:
	go get github.com/golang/dep/cmd/dep

apply_patch:
	dep ensure
	(cd vendor/github.com/tendermint/tendermint && patch -p1 -t < ../../../../patches/fullnode/tendermint-cached-txindexer.patch); exit 0
	(cd vendor/github.com/cosmos/cosmos-sdk && patch -p1 -t < ../../../../patches/fixes/cosmos-boltdb-hack.patch); exit 0

_raw_build_cmd:
	go build -ldflags $(LD_FLAGS) -tags $(GO_TAGS) -o bin/linod   cmd/lino/main.go
	go build -ldflags $(LD_FLAGS) -tags $(GO_TAGS) -o bin/linocli cmd/linocli/main.go

_raw_install_cmd:
	cd cmd/lino    && go install -ldflags $(LD_FLAGS) -tags $(GO_TAGS)
	cd cmd/linocli && go install -ldflags $(LD_FLAGS) -tags $(GO_TAGS)

build: get_vendor_deps apply_patch
	make _raw_build_cmd

install: get_vendor_deps apply_patch
	make _raw_install_cmd

get_vendor_deps:
	@rm -rf vendor/
	@dep ensure

test:get_vendor_deps apply_patch
	go test -ldflags $(LD_FLAGS) -tags $(GO_TAGS) ./... -timeout 600s

benchmark:
	@go test -bench=. $(PACKAGES)

.PHONY: all get_tools get_vendor_deps install build test