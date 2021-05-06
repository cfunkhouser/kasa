.PHONY = all clean

PLATFORMS := linux-arm6 linux-arm7 linux-amd64 linux-386 darwin-amd64
VERSION := $(shell git describe --always --tags --dirty="-dev-$$(git rev-parse --short HEAD)")
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

kasautil-linux-%:
	env GOOS=linux GOARCH=$* $(BUILDCMD) $@ $(MAIN)

kasautil-darwin-%:
	env GOOS=darwin GOARCH=$* $(BUILDCMD) $@ $(MAIN)

SHA%SUM.txt: $(TARGETS)
	shasum -a $* $(TARGETS) > $@

clean:
	@rm -fv $(TARGETS) $(SUMS)
