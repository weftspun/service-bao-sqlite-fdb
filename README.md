OpenBao secrets engine that runs precanned SQL against fabric-store SQLite databases whose pages live in FoundationDB, returning rows through Bao's KV interface.

## Placement

    OpenBao -> fdb -> s3

`service-openbao` runs OpenBao with FoundationDB as its storage backend. `datasource-store` (fabric-store) provides a SQLite VFS whose pages live in the same FoundationDB cluster, so a database opened through it has no local file. This plugin is a single-binary OpenBao secrets engine that embeds SQLite and fabric-store's `fdb_vfs.c`, mounts the database into Bao, and answers `bao read <mount>/query/<name>` by running a catalog-registered SQL statement. Because Bao and the database share one FoundationDB cluster and one S3 backup destination beneath it, a returned row exists in S3 exactly once — the FDB layer's own backup.

## Shape

    +------------------------------+
    |   bao-plugin-sqlite-fdb      |
    |                              |
    |  +------------------------+  |
    |  |  Go: openbao SDK       |  |
    |  |  paths, catalog, HCL   |  |
    |  +-----------+------------+  |
    |              |               |
    |  +-----------v------------+  |
    |  |  mattn/go-sqlite3      |  |
    |  |  (dynamic libsqlite3)  |  |
    |  +-----------+------------+  |
    |              | shared VFS    |
    |  +-----------v------------+  |
    |  |  fabric-store fdb_vfs  |  |
    |  |  (compiled inline)     |  |
    |  +-----------+------------+  |
    |              |               |
    |  +-----------v------------+  |
    |  |  libfdb_c              |  |
    |  |  (dynamic)             |  |
    |  +-----------+------------+  |
    +--------------|---------------+
                   v
              FoundationDB cluster

One process, no IPC. The plugin binary contains the openbao SDK glue, the SQLite client (via mattn/go-sqlite3 built with `-tags libsqlite3`), and fabric-store's `fdb_vfs.c` (built inline via a manifest linkfile). libsqlite3 and libfdb_c load dynamically from the deploy environment.

## What it does

- Serves `<mount>/query/<name>` — reads run one precanned SQL statement and return its rows.
- Serves `<mount>/queries` — lists registered query names.
- Rejects arbitrary SQL. Only queries defined in the startup catalog run.

## What it does not do

- Rotate credentials, mint short-lived tokens, or manage a role / lease lifecycle.
- Accept DDL or DML at request time.
- Cache to Bao's own storage backend. Every read hits SQLite live.

## Configuration

Environment variables read at plugin startup. Unmet preconditions fail startup rather than defaulting.

| variable | required | meaning |
|---|---|---|
| `BAO_SQLITE_FDB_CATALOG` | always | Path to the HCL catalog file. See `catalog.example.hcl`. |
| `BAO_SQLITE_FDB_CLUSTER` | fabric mode | Path to the FoundationDB cluster file, passed to `weft_fdb_start`. |
| `BAO_SQLITE_FDB_DB` | fabric mode | Actor / database name (becomes `weft/db/<name>/` in the FDB key space). |
| `BAO_SQLITE_FDB_DSN` | plain mode | Plain SQLite DSN (filesystem path or `:memory:`). For dev and tests. |

Exactly one mode. Fabric mode needs both `CLUSTER` and `DB`; plain mode needs `DSN`. Setting variables from both modes fails startup.

## Catalog shape

    query "greeting_by_lang" {
      sql  = "SELECT text FROM greetings WHERE lang = ?"
      args = ["lang"]
    }

`args` names bind positionally from the request data map in order. Duplicate names, missing `sql`, and unnamed queries are rejected at load.

## Build

Prerequisites:

- Go 1.27+
- libsqlite3 development headers (`libsqlite3-dev` on Debian)
- FoundationDB C client libraries (`foundationdb-clients` — the same package `service-openbao`'s Dockerfile.fdb installs)
- `thirdparty/store/` populated (see Manifest wiring)

    scripts/link-thirdparty.sh   # local dev: symlink thirdparty/store/ from 6-datasource/store
    make build               # produces ./bao-plugin-sqlite-fdb
    make sha256              # sha256 for `bao plugin register`
    make test                # unit tests, plain SQLite only

## Manifest wiring

The plugin sources fabric-store's `fdb_vfs.c` and `fdb_keys.h` through the goal manifest's linkfile mechanism, so a workspace checkout via `repo sync` populates `thirdparty/store/`:

    <project name="datasource-store" path="6-datasource/store" ...>
      <linkfile src="fdb_vfs.c"
                dest="7-service/service-bao-sqlite-fdb/thirdparty/store/fdb_vfs.c" />
      <linkfile src="fdb_keys.h"
                dest="7-service/service-bao-sqlite-fdb/thirdparty/store/fdb_keys.h" />
    </project>

Two files, two names, no drift — the same pattern this workspace uses for `CLAUDE.md` and `CITATION.cff` at the root. `scripts/link-thirdparty.sh` is the local-dev fallback for a checkout that does not have the manifest updates yet. The directory is `thirdparty/` rather than `vendor/` because Go treats a top-level `vendor/` as its own vendored-dependencies dir.

## Registering with OpenBao

    make build
    CHECKSUM=$(sha256sum bao-plugin-sqlite-fdb | cut -d' ' -f1)
    bao plugin register -sha256=$CHECKSUM secret sqlite-fdb
    bao secrets enable -path=sqlite-fdb sqlite-fdb

Bao spawns the plugin process, which reads the environment variables above at startup.

## Licence

Dual-licensed under Apache 2.0 or MIT, at your option, matching `datasource-store`.

`SPDX-License-Identifier: Apache-2.0 OR MIT`
