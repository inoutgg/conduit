setup:
    lefthook install -f

mod:
    go mod download
    go mod tidy

lint-fix:
  golangci-lint run --fix ./...

format-sql:
  npx prettier -w **/*.sql

test-all:
  go test -race -count=1 -parallel=4 ./...

create-bin:
  go build -o bin/main ./cmd/conduit/main.go

create-debug-bin:
  go build -tags debug -o bin/main-debug ./cmd/conduit/main.go
