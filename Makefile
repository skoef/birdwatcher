# go build flags
LDFLAGS ?=-w -s

all: release/birdwatcher

release/birdwatcher:
	mkdir -p release
	CGO_ENABLED=0 go build -a -ldflags '$(LDFLAGS)' -o release/birdwatcher .
