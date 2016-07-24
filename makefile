export SHELL:=/bin/bash -O extglob -c
ARTIFACT = bilder
OS = $(shell uname | tr [:upper:] [:lower:])

all: dev

build: GOOS ?= ${OS}
build: GOARCH ?= amd64
build: clean
	GOOS=${GOOS} GOARCH=${GOARCH} CGO_ENABLED=0 go build -o ${ARTIFACT} -a !(gen_assets*).go

release: assets build

run:
	./${ARTIFACT} -config config.json

clean:
	rm -f ${ARTIFACT}

dev: test build
	./${ARTIFACT} -config config.json

.PHONY: assets
assets:
	go run gen_assets.go

.PHONY: test
test:
	go test !(gen_assets*).go |& tee quickfix

test-loop:
	ls !(gen_assets*).go | entr bash -O extglob -c 'go test !(gen_assets*).go |& tee quickfix'
