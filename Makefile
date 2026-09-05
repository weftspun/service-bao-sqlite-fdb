PLUGIN := bao-plugin-sqlite-fdb
SHASUM := $(shell command -v sha256sum >/dev/null 2>&1 && echo sha256sum || echo shasum -a 256)

# Build requirements:
#   - Go 1.27+
#   - libsqlite3 development headers (Debian: libsqlite3-dev)
#   - FoundationDB C client libraries (foundationdb-clients)
#   - thirdparty/store populated (scripts/link-thirdparty.sh or manifest linkfile)

.PHONY: build sha256 test vet tidy clean thirdparty

build: thirdparty
	CGO_ENABLED=1 go build -tags libsqlite3 -trimpath -o $(PLUGIN) .

sha256: build
	$(SHASUM) $(PLUGIN)

# Unit tests use plain SQLite (no FDB, no fabric-store). They still need
# cgo because mattn/go-sqlite3 is cgo-only. Fabric-store integration tests
# live outside `go test`.
test:
	CGO_ENABLED=1 go test -tags libsqlite3 -race -count=1 ./...

vet:
	CGO_ENABLED=1 go vet -tags libsqlite3 ./...

tidy:
	go mod tidy

thirdparty:
	@[ -f thirdparty/store/fdb_vfs.c ] || (echo "thirdparty/store/ not populated; run scripts/link-thirdparty.sh"; exit 1)

clean:
	rm -f $(PLUGIN)
