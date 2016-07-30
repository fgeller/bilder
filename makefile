export SHELL:=/bin/bash -O extglob -c
ARTIFACT = bilder
OS = $(shell uname | tr [:upper:] [:lower:])

all: dev

build: GOOS ?= ${OS}
build: GOARCH ?= amd64
build: clean
	GOOS=${GOOS} GOARCH=${GOARCH} CGO_ENABLED=0 go build -o ${ARTIFACT} -a *.go

linux-release:
	GOOS=linux $(MAKE) build

darwin-release:
	GOOS=darwin $(MAKE) build

releases: linux-release darwin-release

run:
	./${ARTIFACT} -config config.json

clean:
	rm -f ${ARTIFACT}

dev: build
	./${ARTIFACT} -config config.json

.PHONY: assets
assets:
	go run tools/gen_assets.go
