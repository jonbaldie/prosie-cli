.PHONY: all build test messgo mutago release-local clean

all: build

build:
	go build -trimpath -o bin/prosie .

test:
	go test -v -count=1 ./...

messgo:
	messgo . text quality-gates/messgo-ruleset.xml --ignore-tests

mutago:
	mutago --coverage --exec-timeout=300 --min-covered-msi=80 --quiet --no-diffs ./...

release-local:
	./scripts/build-release.sh

clean:
	rm -rf bin dist
