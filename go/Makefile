VERSION := $(shell git describe --all --long)
GO ?= go
GOBUILDFLAGS ?= -v

BINARIES = \
  chunk_sink		\
  chunk_source		\
  observer			\
  chunk_sink_client

PACKAGES = \
  mdns

all: $(BINARIES)

chunk_sink:
	${GO} build -o bin/$@ $(GOBUILDFLAGS) -ldflags "-X main.version=$(VERSION)" ./cmd/$@

chunk_source:
	${GO} build -o bin/$@ $(GOBUILDFLAGS) -ldflags "-X main.version=$(VERSION)" ./cmd/$@

observer:
	${GO} build -o bin/$@ $(GOBUILDFLAGS) -ldflags "-X main.version=$(VERSION)" ./cmd/$@

chunk_sink_client:
	${GO} build -o bin/$@ $(GOBUILDFLAGS) -ldflags "-X main.version=$(VERSION)" ./cmd/$@

clean:
	cd bin && rm -f ${BINARIES}

