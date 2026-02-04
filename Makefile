build:
	go build -ldflags="-s -w" -o goarc.exe ./cmd/goarc/main.go

test:
	go test -v ./internal/engine/...

bench:
	go test -bench=. -benchmem ./internal/engine/...

benchcpu:
	go test -bench=. -cpu=1,4,8 ./internal/engine

install:
	go install ./cmd/goarc
