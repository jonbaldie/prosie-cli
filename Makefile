.PHONY: all build test release-local clean

all: build

build:
	go build -trimpath -o bin/prosie .

test:
	go test -v -count=1 ./...

release-local:
	./scripts/build-release.sh

clean:
	rm -rf bin dist
