.PHONY: setup server test benchmark vet build vuln check

COVERAGE_PACKAGES := ./config,./core/...,./web
COVERAGE_MINIMUM := 98.0

setup:
	go mod download

server:
	go run ./cmd/urlshortener

test:
	go test -race ./...
	go test -coverpkg=$(COVERAGE_PACKAGES) -coverprofile=coverage.out ./tests
	@go tool cover -func=coverage.out | awk -v minimum=$(COVERAGE_MINIMUM) '/^total:/ { gsub("%", "", $$3); if ($$3 + 0 < minimum) { printf "coverage %.1f%% is below %.1f%%\n", $$3, minimum; exit 1 }; printf "coverage %.1f%% meets %.1f%% minimum\n", $$3, minimum }'

benchmark:
	go test -run '^$$' -bench . -benchmem ./tests

vet:
	go vet ./...

build:
	go build ./...

vuln:
	go run golang.org/x/vuln/cmd/govulncheck@latest ./...

check: test vet build vuln
