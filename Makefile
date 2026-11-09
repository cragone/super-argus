GO_MODULES := . sdk/go agent testapp

.PHONY: fmt vet lint test test-race build

fmt:
	gofmt -l .
	@test -z "$$(gofmt -l .)" || (echo "run: gofmt -w \$$(gofmt -l .)" && exit 1)

vet:
	go vet ./...

lint: vet
	golangci-lint run

test:
	@for m in $(GO_MODULES); do \
		echo "== go test $$m =="; \
		(cd $$m && go test ./...) || exit 1; \
	done

test-race:
	@for m in $(GO_MODULES); do \
		echo "== go test -race $$m =="; \
		(cd $$m && go test -race ./...) || exit 1; \
	done

build:
	go build ./...
