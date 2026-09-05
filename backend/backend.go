package backend

import (
	"context"
	"fmt"
	"os"
	"strings"

	log "github.com/hashicorp/go-hclog"
	"github.com/openbao/openbao/sdk/v2/framework"
	"github.com/openbao/openbao/sdk/v2/logical"
)

const backendHelp = `
The sqlite-fdb secrets engine answers reads by running a precanned SQL
statement from its catalog against a SQLite database. When the database is
opened through the fabric-store VFS, pages live in FoundationDB and no
local file is written.

Consumers cannot supply SQL. Only catalog-registered queries run.
`

const (
	envCatalog = "BAO_SQLITE_FDB_CATALOG"
	envDB      = "BAO_SQLITE_FDB_DB"
	envCluster = "BAO_SQLITE_FDB_CLUSTER"
	envDSN     = "BAO_SQLITE_FDB_DSN"
)

type backend struct {
	*framework.Backend

	logger      log.Logger
	catalog     *Catalog
	store       Store
	stopFabric  bool
}

// Factory wires the OpenBao logical.Backend.
//
// Two modes, decided at startup:
//
//   Fabric mode (production): BAO_SQLITE_FDB_CLUSTER and BAO_SQLITE_FDB_DB are
//   both set. The plugin calls StartFabricStore(cluster), registers the weft_fdb
//   VFS, and opens the named database whose pages live in FoundationDB.
//
//   Plain mode (dev / local tests): BAO_SQLITE_FDB_DSN is set to a filesystem
//   path or ":memory:". The plugin opens SQLite through the unix VFS. No FDB.
//
// Exactly one of the two modes must be selected. An unmet precondition is
// a startup failure, never a silent default.
func Factory(ctx context.Context, conf *logical.BackendConfig) (logical.Backend, error) {
	catalogPath := os.Getenv(envCatalog)
	if catalogPath == "" {
		return nil, fmt.Errorf("%s not set: plugin cannot start without a catalog", envCatalog)
	}
	catalog, err := LoadCatalog(catalogPath)
	if err != nil {
		return nil, fmt.Errorf("load catalog %s: %w", catalogPath, err)
	}

	store, stopFabric, err := openConfiguredStore()
	if err != nil {
		return nil, err
	}

	b := &backend{
		logger:     conf.Logger,
		catalog:    catalog,
		store:      store,
		stopFabric: stopFabric,
	}
	b.Backend = &framework.Backend{
		Help:        strings.TrimSpace(backendHelp),
		BackendType: logical.TypeLogical,
		Paths:       paths(b),
		Clean:       b.clean,
	}
	if err := b.Backend.Setup(ctx, conf); err != nil {
		_ = store.Close()
		if stopFabric {
			StopFabricStore()
		}
		return nil, err
	}
	return b, nil
}

func openConfiguredStore() (Store, bool, error) {
	cluster := os.Getenv(envCluster)
	dbName := os.Getenv(envDB)
	dsn := os.Getenv(envDSN)

	switch {
	case cluster != "" && dbName != "":
		if dsn != "" {
			return nil, false, fmt.Errorf("%s must not be set when %s/%s are set: pick one mode", envDSN, envCluster, envDB)
		}
		if err := StartFabricStore(cluster); err != nil {
			return nil, false, fmt.Errorf("start fabric-store: %w", err)
		}
		s, err := OpenFabricStore(dbName)
		if err != nil {
			StopFabricStore()
			return nil, false, fmt.Errorf("open fabric-store db %q: %w", dbName, err)
		}
		return s, true, nil

	case dsn != "":
		s, err := OpenStore(dsn)
		if err != nil {
			return nil, false, fmt.Errorf("open plain sqlite %q: %w", dsn, err)
		}
		return s, false, nil

	default:
		return nil, false, fmt.Errorf(
			"no database configured: set either %s+%s (fabric-store mode) or %s (plain mode)",
			envCluster, envDB, envDSN,
		)
	}
}

func (b *backend) clean(_ context.Context) {
	if err := b.store.Close(); err != nil {
		b.logger.Warn("store close", "error", err)
	}
	if b.stopFabric {
		StopFabricStore()
	}
}
