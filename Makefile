BINARY = pharmer 
GOTARGET = github.com/pharmer/pharmer
SYSTEM =
LINUX_ARCH := amd64 arm64 arm

VERBOSE_FLAG =

DOCKER ?= docker
DIR := ${CURDIR}
BUILDMNT = /go/src/$(GOTARGET)
BUILD_IMAGE ?= golang:1.10-alpine

DOCKER_BUILD ?= $(DOCKER) run --rm -v $(DIR):$(BUILDMNT) -w $(BUILDMNT) $(BUILD_IMAGE) /bin/sh -c

.PHONY: all pharmer clean

all: pharmer 

build_pharmer:
	$(DOCKER_BUILD) 'CGO_ENABLED=0 $(SYSTEM) go build -o $(BINARY) $(VERBOSE_FLAG) $(GOTARGET)'

pharmer:
	for arch in $(LINUX_ARCH); do \
		mkdir -p build/linux/$$arch; \
		echo Building: linux/$$arch; \
		$(MAKE) build_pharmer SYSTEM="GOOS=linux GOARCH=$$arch" BINARY="build/linux/$$arch/pharmer"; \
	done
	@echo Building: host
	make build_pharmer

clean:
	rm -f $(BINARY)
	rm -rf build
