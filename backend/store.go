package backend

import (
	"context"
	"database/sql"
	"fmt"

	sqlite3 "github.com/mattn/go-sqlite3"
)

const (
	// FabricStoreVFS is the VFS name fabric-store's fdb_vfs.c registers.
	FabricStoreVFS = "weft_fdb"

	driverPlain  = "sqlite3"
	driverFabric = "sqlite3_fabric"

	pragmaJournal = "PRAGMA journal_mode=MEMORY"
	pragmaLocking = "PRAGMA locking_mode=EXCLUSIVE"
)

func init() {
	// A second driver that applies fabric-store's required pragmas on every
	// new connection. Registered even when no FDB is around; the driver only
	// dials sqlite3, and the pragmas are per-connection and benign locally.
	sql.Register(driverFabric, &sqlite3.SQLiteDriver{
		ConnectHook: func(conn *sqlite3.SQLiteConn) error {
			for _, p := range []string{pragmaJournal, pragmaLocking} {
				if _, err := conn.Exec(p, nil); err != nil {
					return fmt.Errorf("apply %q: %w", p, err)
				}
			}
			return nil
		},
	})
}

type Store interface {
	Query(ctx context.Context, sqlText string, args ...interface{}) ([]map[string]interface{}, error)
	Close() error
}

type sqlStore struct{ db *sql.DB }

// OpenStore opens a SQLite database at dsn using the default unix VFS. Used
// for local tests and non-fabric-store deployments.
func OpenStore(dsn string) (Store, error) {
	return openWith(driverPlain, dsn)
}

// OpenFabricStore opens a SQLite database whose pages live in FoundationDB
// via fabric-store's weft_fdb VFS. StartFabricStore must have been called
// first. dbName is the actor / database name — it becomes weft/db/<name>/
// in the FDB key space.
func OpenFabricStore(dbName string) (Store, error) {
	dsn := fmt.Sprintf("file:%s?vfs=%s", dbName, FabricStoreVFS)
	return openWith(driverFabric, dsn)
}

func openWith(driver, dsn string) (Store, error) {
	db, err := sql.Open(driver, dsn)
	if err != nil {
		return nil, err
	}
	if err := db.Ping(); err != nil {
		_ = db.Close()
		return nil, fmt.Errorf("ping %s: %w", dsn, err)
	}
	return &sqlStore{db: db}, nil
}

func (s *sqlStore) Query(ctx context.Context, sqlText string, args ...interface{}) ([]map[string]interface{}, error) {
	rows, err := s.db.QueryContext(ctx, sqlText, args...)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	cols, err := rows.Columns()
	if err != nil {
		return nil, err
	}
	var out []map[string]interface{}
	for rows.Next() {
		cells := make([]interface{}, len(cols))
		ptrs := make([]interface{}, len(cols))
		for i := range cells {
			ptrs[i] = &cells[i]
		}
		if err := rows.Scan(ptrs...); err != nil {
			return nil, err
		}
		row := make(map[string]interface{}, len(cols))
		for i, c := range cols {
			row[c] = cells[i]
		}
		out = append(out, row)
	}
	return out, rows.Err()
}

func (s *sqlStore) Close() error { return s.db.Close() }
