SOURCES != find . -type f -name '*.go'

all: bin/ambient-glance

bin/ambient-glance: $(SOURCES)
	go build -o bin/ambient-glance .

bin/obactl: $(SOURCES)
	go build -o bin/obactl ./cmd/obactl

.PHONY: test
test:
	go test -v ./...

.PHONY: check
check:
	go tool ltag -check

.PHONY: clean
clean:
	rm -rf bin
