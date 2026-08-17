.PHONY: setup server test benchmark vet build vuln check

setup:
	go mod download

server:
	go run ./cmd/urlshortener

test:
	go test -race -coverpkg=./... -cover ./...

benchmark:
	go test -run '^$$' -bench . -benchmem ./tests

vet:
	go vet ./...

build:
	go build ./...

vuln:
	go run golang.org/x/vuln/cmd/govulncheck@latest ./...

check: test vet build vuln
