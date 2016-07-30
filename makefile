export SHELL:=/bin/bash -O extglob -c
ARTIFACT = bilder
OS = $(shell uname | tr [:upper:] [:lower:])

all: dev

build: GOOS ?= ${OS}
build: GOARCH ?= amd64
build: clean
	GOOS=${GOOS} GOARCH=${GOARCH} CGO_ENABLED=0 go build -o ${ARTIFACT} -a *.go

release: GOOS ?= ${OS}
release: GOARCH ?= amd64
release: build
	tar Jcf ${ARTIFACT}-`git rev-parse HEAD | cut -c-7`-${GOOS}-${GOARCH}.xz ${ARTIFACT}

linux-release:
	GOOS=linux $(MAKE) release

darwin-release:
	GOOS=darwin $(MAKE) release

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
