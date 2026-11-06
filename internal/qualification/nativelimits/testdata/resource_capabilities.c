#define _DARWIN_C_SOURCE 1

#include <dlfcn.h>
#include <errno.h>
#include <fcntl.h>
#include <inttypes.h>
#include <stdint.h>
#include <stdio.h>
#include <string.h>
#include <sys/resource.h>
#include <sys/stat.h>
#include <unistd.h>

/* This observation program never changes a process memory limit. The only
 * mutation is a 32 KiB preallocation on the creator's empty, private FD3.
 * These primitive results are not a whole-memory or total-storage lease. */
struct memory_limits {
    int32_t active;
    uint32_t active_attributes;
    int32_t inactive;
    uint32_t inactive_attributes;
};

typedef int (*memory_query_fn)(uint32_t, int32_t, uint32_t, void *, size_t);
_Static_assert(sizeof(struct memory_limits) == 16, "unexpected memory query ABI");

int main(int argc, char **argv) {
    (void)argv;
    if (argc != 1) return 64;
    const int root_context = geteuid() == 0;
    struct memory_limits memory = {0};
    memory_query_fn query = (memory_query_fn)dlsym(RTLD_DEFAULT, "memorystatus_control");
    const int present = query != NULL;
    int invoked = 0, memory_rc = -1, memory_error = present ? EPERM : ENOSYS;
    if (present && !root_context) {
        /* MEMORYSTATUS_CMD_GET_MEMLIMIT_PROPERTIES queries this process only.
         * No SET command, foreign PID, entitlement or privilege change exists. */
        invoked = 1;
        errno = 0;
        memory_rc = query(8, (int32_t)getpid(), 0, &memory, sizeof(memory));
        memory_error = memory_rc == 0 ? 0 : errno;
        if (memory_rc != 0) memset(&memory, 0, sizeof(memory));
    }

    struct rlimit address = {0};
    errno = 0;
    int address_rc = getrlimit(RLIMIT_AS, &address);
    int address_error = address_rc == 0 ? 0 : errno;
    if (address_rc != 0) memset(&address, 0, sizeof(address));

    struct stat before = {0}, after = {0};
    int eligible = fstat(3, &before) == 0 && S_ISREG(before.st_mode) &&
        before.st_uid == getuid() && (before.st_mode & 0077) == 0 &&
        before.st_nlink == 1 && before.st_size == 0 && before.st_blocks == 0;
    int allocated_rc = -1, allocated_error = 0, sync_rc = -1, sync_error = 0;
    struct fstore storage = {0};
    if (eligible) {
        storage.fst_flags = F_ALLOCATEALL;
        storage.fst_posmode = F_PEOFPOSMODE;
        storage.fst_length = 32 * 1024;
        errno = 0;
        allocated_rc = fcntl(3, F_PREALLOCATE, &storage);
        allocated_error = allocated_rc == 0 ? 0 : errno;
        if (allocated_rc == 0) {
            errno = 0;
            sync_rc = fsync(3);
            sync_error = sync_rc == 0 ? 0 : errno;
        }
    }
    int after_rc = fstat(3, &after);

    int printed = printf("{\"schema\":1,\"scope\":\"ordinary-self-query-owned-file\","
        "\"full_native_qualification\":false,\"root_context\":%s,"
        "\"memory_api_present\":%s,\"memory_query_invoked\":%s,"
        "\"memory_return\":%d,\"memory_errno\":%d,"
        "\"active_mb\":%" PRId32 ",\"active_attributes\":%" PRIu32 ","
        "\"inactive_mb\":%" PRId32 ",\"inactive_attributes\":%" PRIu32 ","
        "\"address_return\":%d,\"address_errno\":%d,"
        "\"address_soft\":\"%" PRIu64 "\",\"address_hard\":\"%" PRIu64 "\","
        "\"file_eligible\":%s,\"allocation_return\":%d,\"allocation_errno\":%d,"
        "\"allocated_bytes\":%" PRId64 ",\"sync_return\":%d,\"sync_errno\":%d,"
        "\"after_stat_return\":%d,\"after_size\":%" PRId64 ","
        "\"after_blocks\":%" PRId64 ",\"same_file\":%s}\n",
        root_context ? "true" : "false", present ? "true" : "false", invoked ? "true" : "false",
        memory_rc, memory_error, memory.active, memory.active_attributes,
        memory.inactive, memory.inactive_attributes, address_rc, address_error,
        (uint64_t)address.rlim_cur, (uint64_t)address.rlim_max,
        eligible ? "true" : "false", allocated_rc, allocated_error,
        (int64_t)storage.fst_bytesalloc, sync_rc, sync_error, after_rc,
        (int64_t)after.st_size, (int64_t)after.st_blocks,
        after_rc == 0 && before.st_dev == after.st_dev && before.st_ino == after.st_ino ? "true" : "false");
    return printed < 0 || fflush(stdout) != 0 ? 74 : 0;
}
