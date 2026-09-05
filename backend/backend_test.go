package backend

import (
	"context"
	"fmt"
	"os"
	"path/filepath"
	"testing"

	log "github.com/hashicorp/go-hclog"
	"github.com/openbao/openbao/sdk/v2/logical"
)

// Tests use plain mode (BAO_SQLITE_FDB_DSN) against a local SQLite file so
// they run without FoundationDB or fabric-store's linked C code. Fabric
// mode is exercised by an integration test outside `go test`.

func setupBackend(t *testing.T) (logical.Backend, *logical.InmemStorage) {
	t.Helper()
	dir := t.TempDir()
	dbPath := filepath.Join(dir, "test.db")
	catPath := filepath.Join(dir, "catalog.hcl")

	seed, err := OpenStore(dbPath)
	if err != nil {
		t.Fatal(err)
	}
	ss := seed.(*sqlStore)
	if _, err := ss.db.Exec(`CREATE TABLE greetings (lang TEXT, text TEXT)`); err != nil {
		t.Fatal(err)
	}
	if _, err := ss.db.Exec(`INSERT INTO greetings VALUES ('en', 'hello'), ('ja', 'konnichiwa')`); err != nil {
		t.Fatal(err)
	}
	_ = seed.Close()

	if err := os.WriteFile(catPath, []byte(`
query "greeting" {
  sql  = "SELECT text FROM greetings WHERE lang = ?"
  args = ["lang"]
}
`), 0o600); err != nil {
		t.Fatal(err)
	}

	t.Setenv(envCatalog, catPath)
	t.Setenv(envDB, "")
	t.Setenv(envCluster, "")
	t.Setenv(envDSN, dbPath)

	storage := &logical.InmemStorage{}
	b, err := Factory(context.Background(), &logical.BackendConfig{
		Logger:      log.NewNullLogger(),
		StorageView: storage,
	})
	if err != nil {
		t.Fatal(err)
	}
	return b, storage
}

func TestFactory_ReadRegisteredQuery(t *testing.T) {
	b, storage := setupBackend(t)

	resp, err := b.HandleRequest(context.Background(), &logical.Request{
		Operation: logical.ReadOperation,
		Path:      "query/greeting",
		Data:      map[string]interface{}{"lang": "en"},
		Storage:   storage,
	})
	if err != nil {
		t.Fatal(err)
	}
	if resp.IsError() {
		t.Fatalf("error resp: %v", resp.Data)
	}
	rows, ok := resp.Data["rows"].([]map[string]interface{})
	if !ok || len(rows) != 1 {
		t.Fatalf("rows shape %v", resp.Data)
	}
	if got := fmt.Sprint(rows[0]["text"]); got != "hello" {
		t.Fatalf("text=%q", got)
	}
}

func TestFactory_UnregisteredQueryReturnsError(t *testing.T) {
	b, storage := setupBackend(t)

	resp, err := b.HandleRequest(context.Background(), &logical.Request{
		Operation: logical.ReadOperation,
		Path:      "query/does_not_exist",
		Data:      map[string]interface{}{},
		Storage:   storage,
	})
	if err != nil {
		t.Fatal(err)
	}
	if !resp.IsError() {
		t.Fatalf("expected error response, got %v", resp.Data)
	}
}

func TestFactory_ListQueries(t *testing.T) {
	b, storage := setupBackend(t)

	resp, err := b.HandleRequest(context.Background(), &logical.Request{
		Operation: logical.ListOperation,
		Path:      "queries",
		Storage:   storage,
	})
	if err != nil {
		t.Fatal(err)
	}
	keys, ok := resp.Data["keys"].([]string)
	if !ok || len(keys) != 1 || keys[0] != "greeting" {
		t.Fatalf("keys %v", resp.Data)
	}
}

func TestFactory_NoModeConfigured(t *testing.T) {
	dir := t.TempDir()
	catPath := filepath.Join(dir, "catalog.hcl")
	_ = os.WriteFile(catPath, []byte(`query "q" { sql = "SELECT 1" }`), 0o600)
	t.Setenv(envCatalog, catPath)
	t.Setenv(envDB, "")
	t.Setenv(envCluster, "")
	t.Setenv(envDSN, "")
	_, err := Factory(context.Background(), &logical.BackendConfig{
		Logger:      log.NewNullLogger(),
		StorageView: &logical.InmemStorage{},
	})
	if err == nil {
		t.Fatal("expected error when no store mode is configured")
	}
}

func TestFactory_BothModesConfigured(t *testing.T) {
	dir := t.TempDir()
	catPath := filepath.Join(dir, "catalog.hcl")
	_ = os.WriteFile(catPath, []byte(`query "q" { sql = "SELECT 1" }`), 0o600)
	t.Setenv(envCatalog, catPath)
	t.Setenv(envDB, "actor1")
	t.Setenv(envCluster, "/etc/foundationdb/fdb.cluster")
	t.Setenv(envDSN, "/tmp/other.db")
	_, err := Factory(context.Background(), &logical.BackendConfig{
		Logger:      log.NewNullLogger(),
		StorageView: &logical.InmemStorage{},
	})
	if err == nil {
		t.Fatal("expected error when both fabric mode and plain mode are configured")
	}
}

func TestFactory_MissingCatalogEnv(t *testing.T) {
	t.Setenv(envCatalog, "")
	t.Setenv(envDSN, "/tmp/whatever.db")
	_, err := Factory(context.Background(), &logical.BackendConfig{
		Logger:      log.NewNullLogger(),
		StorageView: &logical.InmemStorage{},
	})
	if err == nil {
		t.Fatal("expected error when BAO_SQLITE_FDB_CATALOG is unset")
	}
}
