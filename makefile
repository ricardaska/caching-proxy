BINARY=caching_proxy
MODULE=$(shell go mod edit -print | head -n1 | grep -oP '(?<=module ).*')

GOBUILD=go build
GC=$(shell which musl-gcc)
FLAGS='-linkmode=external -extldflags "-static -s -w"'

build:
	CC=$(GC) $(GOBUILD) -ldflags $(FLAGS) -o $(BINARY) ./cmd/$(MODULE)

build_pprof:
	CC=$(GC) $(GOBUILD) -tags pprof -ldflags $(FLAGS) -o $(BINARY) ./cmd/$(MODULE)

install:
	cp $(BINARY) /usr/bin/$(BINARY)
	cp $(BINARY).service /usr/lib/systemd/system/$(BINARY).service

uninstall:
	rm /usr/lib/systemd/system/$(BINARY).service
	rm /usr/bin/$(BINARY)
