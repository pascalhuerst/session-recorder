VERSION := $(shell git describe --all --long)
GO ?= go
GOBUILDFLAGS ?= -v

BINARIES = \
  chunk_sink

PACKAGES = \
  mdns

all: $(BINARIES)

chunk_sink:
	${GO} build -o bin/$@ $(GOBUILDFLAGS) -ldflags "-X main.version=$(VERSION)" ./cmd/$@

clean:
	cd bin && rm -f ${BINARIES}

