/* Cgo compiles every .c file in this package directory. This shim pulls in
   fabric-store's fdb_vfs.c through the vendor/store include path (see
   fabricstore.go's CFLAGS). The source itself is owned by
   6-datasource/store and populated here by the goal manifest linkfile,
   or by scripts/link-vendor.sh in local development. */
#include "fdb_vfs.c"
