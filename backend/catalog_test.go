package backend

import (
	"os"
	"path/filepath"
	"testing"
)

func writeCatalog(t *testing.T, body string) string {
	t.Helper()
	p := filepath.Join(t.TempDir(), "catalog.hcl")
	if err := os.WriteFile(p, []byte(body), 0o600); err != nil {
		t.Fatal(err)
	}
	return p
}

func TestLoadCatalog(t *testing.T) {
	p := writeCatalog(t, `
query "roles_by_agent" {
  sql  = "SELECT role FROM roles WHERE agent = ?"
  args = ["agent"]
}
query "all_agents" {
  sql = "SELECT id FROM agents"
}
`)
	c, err := LoadCatalog(p)
	if err != nil {
		t.Fatal(err)
	}
	got := c.Names()
	if len(got) != 2 || got[0] != "all_agents" || got[1] != "roles_by_agent" {
		t.Fatalf("names %v", got)
	}
	q, ok := c.Lookup("roles_by_agent")
	if !ok {
		t.Fatal("expected lookup hit")
	}
	if q.SQL != "SELECT role FROM roles WHERE agent = ?" {
		t.Fatalf("sql %q", q.SQL)
	}
	if len(q.Args) != 1 || q.Args[0] != "agent" {
		t.Fatalf("args %v", q.Args)
	}
}

func TestLoadCatalog_DuplicateName(t *testing.T) {
	p := writeCatalog(t, `
query "dup" { sql = "SELECT 1" }
query "dup" { sql = "SELECT 2" }
`)
	if _, err := LoadCatalog(p); err == nil {
		t.Fatal("expected duplicate error")
	}
}

func TestLoadCatalog_MissingSQL(t *testing.T) {
	p := writeCatalog(t, `query "empty" { sql = "" }`)
	if _, err := LoadCatalog(p); err == nil {
		t.Fatal("expected missing-sql error")
	}
}

func TestLoadCatalog_MissingFile(t *testing.T) {
	if _, err := LoadCatalog(filepath.Join(t.TempDir(), "no.hcl")); err == nil {
		t.Fatal("expected missing-file error")
	}
}
