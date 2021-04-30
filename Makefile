.PHONY = all clean

PLATFORMS := linux-arm6 linux-arm7 linux-amd64 darwin-amd64
VERSION := $(shell git describe --always --dirty="-dev-$$(git rev-parse --short HEAD)")
MAIN := ./cmd/kasautil

BUILDCMD := go build -o
ifneq ($(strip $(VERSION)),)
	BUILDCMD := go build -ldflags="-X 'main.Version=$(VERSION)'" -o
endif


TARGETS := $(foreach ku,$(PLATFORMS),kasautil-$(ku))
SUMS := SHA1SUM.txt SHA256SUM.txt

all: $(TARGETS) $(SUMS)
	@chmod +x $(TARGETS)

kasautil-linux-arm%:
	env GOOS=linux GOARCH=arm GOARM=$* $(BUILDCMD) $@ $(MAIN)

kasautil-linux-amd64:
	env GOOS=linux GOARCH=amd64 $(BUILDCMD) $@ $(MAIN)

kasautil-darwin-%:
	env GOOS=darwin GOARCH=$* $(BUILDCMD) $@ $(MAIN)

SHA%SUM.txt: $(TARGETS)
	shasum -a $* $(TARGETS) > $@

clean:
	@rm -fv $(TARGETS) $(SUMS)
