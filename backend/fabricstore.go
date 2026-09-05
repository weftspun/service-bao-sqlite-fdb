// The cgo bridge to fabric-store's fdb_vfs.c. Requires the
// thirdparty/store directory to hold fabric-store's fdb_vfs.c and
// fdb_keys.h (populated by the goal manifest's linkfile, or by
// scripts/link-thirdparty.sh in dev).
//
// Fabric-store's exported ABI, per its fdb_vfs.c:
//
//   int  weft_fdb_start(const char *cluster_file)
//   void weft_fdb_stop(void)
//   int  weft_vfs_register(int make_default)
//
// The VFS it installs is named "weft_fdb" (see FabricStoreVFS in store.go).

package backend

/*
// FDB_API_VERSION and the -I<prefix>/foundationdb path match
// datasource-store's CMakeLists.txt (find_path FDB_INCLUDE_DIR
// foundationdb/fdb_c.h + target_compile_definitions FDB_API_VERSION=730).
// Debian install path is /usr/include/foundationdb — override with
// CGO_CFLAGS on other systems.
#cgo CFLAGS: -I${SRCDIR}/../thirdparty/store -I/usr/include/foundationdb
#cgo CFLAGS: -DFDB_API_VERSION=730 -D_POSIX_C_SOURCE=200809L -Wno-unused-parameter
#cgo pkg-config: sqlite3
#cgo LDFLAGS: -lfdb_c -lpthread

extern int  weft_fdb_start(const char *cluster_file);
extern void weft_fdb_stop(void);
extern int  weft_vfs_register(int make_default);
*/
import "C"

import (
	"fmt"
	"unsafe"
)

// StartFabricStore initialises the FDB client and installs the weft_fdb VFS
// under SQLite. Call once at plugin startup, before any OpenFabricStore.
func StartFabricStore(clusterFile string) error {
	cf := C.CString(clusterFile)
	defer C.free(unsafe.Pointer(cf))
	if rc := C.weft_fdb_start(cf); rc != 0 {
		return fmt.Errorf("weft_fdb_start(%q) rc=%d", clusterFile, int(rc))
	}
	if rc := C.weft_vfs_register(0); rc != 0 {
		C.weft_fdb_stop()
		return fmt.Errorf("weft_vfs_register rc=%d", int(rc))
	}
	return nil
}

// StopFabricStore tears down the FDB network thread. Call once at plugin
// shutdown. After this returns, no OpenFabricStore may run.
func StopFabricStore() { C.weft_fdb_stop() }
