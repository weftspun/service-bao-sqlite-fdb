package backend

import (
	"fmt"
	"os"
	"sort"
	"sync"

	"github.com/hashicorp/hcl/v2"
	"github.com/hashicorp/hcl/v2/hclsimple"
)

type Catalog struct {
	mu      sync.RWMutex
	entries map[string]*CatalogEntry
}

type CatalogEntry struct {
	Name string   `hcl:"name,label"`
	SQL  string   `hcl:"sql"`
	Args []string `hcl:"args,optional"`
}

type catalogFile struct {
	Queries []*CatalogEntry `hcl:"query,block"`
	Remain  hcl.Body        `hcl:",remain"`
}

func LoadCatalog(path string) (*Catalog, error) {
	if _, err := os.Stat(path); err != nil {
		return nil, err
	}
	var file catalogFile
	if err := hclsimple.DecodeFile(path, nil, &file); err != nil {
		return nil, fmt.Errorf("decode %s: %w", path, err)
	}
	c := &Catalog{entries: make(map[string]*CatalogEntry, len(file.Queries))}
	for _, q := range file.Queries {
		if q.Name == "" {
			return nil, fmt.Errorf("query in %s missing name", path)
		}
		if q.SQL == "" {
			return nil, fmt.Errorf("query %q in %s missing sql", q.Name, path)
		}
		if _, dup := c.entries[q.Name]; dup {
			return nil, fmt.Errorf("query %q defined twice in %s", q.Name, path)
		}
		c.entries[q.Name] = q
	}
	return c, nil
}

func (c *Catalog) Lookup(name string) (*CatalogEntry, bool) {
	c.mu.RLock()
	defer c.mu.RUnlock()
	e, ok := c.entries[name]
	return e, ok
}

func (c *Catalog) Names() []string {
	c.mu.RLock()
	defer c.mu.RUnlock()
	names := make([]string, 0, len(c.entries))
	for n := range c.entries {
		names = append(names, n)
	}
	sort.Strings(names)
	return names
}
