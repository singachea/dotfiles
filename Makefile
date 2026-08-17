.PHONY: kit test install

kit:
	go build -o bin/kit ./cmd/kit

test:
	go test ./...

install: kit
	mkdir -p $(HOME)/bin
	cp bin/kit $(HOME)/bin/kit
	@echo "installed $(HOME)/bin/kit"
