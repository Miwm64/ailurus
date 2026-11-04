BINS := ailurus-build ailurus-run

.PHONY: all build $(BINS) test lint clean

all: build

build: $(BINS)

$(BINS):
	@go build -o build/$@ ./cmd/$@
	@echo "Built $@ successfully"

test:
	@go test ./...

lint:
	@golangci-lint run

clean:
	@rm -rf build/