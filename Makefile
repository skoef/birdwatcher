GIT_BRANCH?= $(shell git rev-parse --abbrev-ref HEAD)
GIT_COMMIT?=$(shell git rev-parse --short HEAD)
GIT_DIRTY=$(shell git diff --quiet || echo '+')

# go build flags
LDFLAGS ?=
LDFLAGS += \
	-X main.buildVersion=$(GIT_COMMIT)$(GIT_DIRTY) \
	-X main.buildBranch=$(GIT_BRANCH)

all: release/birdwatcher

release/birdwatcher:
	mkdir -p release
	CGO_ENABLED=0 go build -a -ldflags '$(LDFLAGS)' -o release/birdwatcher .
