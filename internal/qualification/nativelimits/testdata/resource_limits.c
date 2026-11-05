#define _DARWIN_C_SOURCE 1

#include <errno.h>
#include <fcntl.h>
#include <inttypes.h>
#include <mach/mach.h>
#include <signal.h>
#include <stdint.h>
#include <stdio.h>
#include <string.h>
#include <sys/mman.h>
#include <sys/resource.h>
#include <unistd.h>

/* Finite, owned child controls. This program issues no lease or witness and
 * performs no networking, privilege elevation or operating configuration. */
static int fail(const char *operation) {
    fprintf(stderr, "%s: errno=%d\n", operation, errno);
    return 1;
}

static int lower_limit(int resource, rlim_t value) {
    struct rlimit previous;
    if (getrlimit(resource, &previous) != 0) return fail("getrlimit");
    if (previous.rlim_max != RLIM_INFINITY && value > previous.rlim_max) {
        errno = EINVAL;
        return fail("insufficient inherited hard limit");
    }
    struct rlimit selected = {value, value};
    if (setrlimit(resource, &selected) != 0) return fail("setrlimit");
    struct rlimit actual;
    if (getrlimit(resource, &actual) != 0) return fail("readback limit");
    if (actual.rlim_cur != value || actual.rlim_max != value) {
        errno = EINVAL;
        return fail("limit readback mismatch");
    }
    return 0;
}

static int address_space(void) {
    mach_task_basic_info_data_t info;
    mach_msg_type_number_t count = MACH_TASK_BASIC_INFO_COUNT;
    if (task_info(mach_task_self(), MACH_TASK_BASIC_INFO,
                  (task_info_t)&info, &count) != KERN_SUCCESS) {
        errno = EINVAL;
        return fail("task address-space observation");
    }
    const uint64_t margin = UINT64_C(16) << 20;
    if ((uint64_t)info.virtual_size > (uint64_t)RLIM_INFINITY - margin) {
        errno = EOVERFLOW;
        return fail("address-space limit overflow");
    }
    rlim_t limit = (rlim_t)((uint64_t)info.virtual_size + margin);
    if (lower_limit(RLIMIT_AS, limit) != 0) return 1;
    size_t page = (size_t)getpagesize();
    if (page == 0 || page > (1U << 20)) {
        errno = EINVAL;
        return fail("page size");
    }
    void *small = mmap(NULL, page, PROT_READ | PROT_WRITE,
                       MAP_PRIVATE | MAP_ANON, -1, 0);
    if (small == MAP_FAILED) return fail("admitted small mapping");
    ((volatile unsigned char *)small)[0] = 1;
    ((volatile unsigned char *)small)[page - 1] = 1;
    if (munmap(small, page) != 0) return fail("small mapping cleanup");
    size_t rejected = 32U << 20;
    errno = 0;
    void *large = mmap(NULL, rejected, PROT_NONE,
                       MAP_PRIVATE | MAP_ANON, -1, 0);
    if (large != MAP_FAILED) {
        (void)munmap(large, rejected);
        errno = EINVAL;
        return fail("over-limit mapping unexpectedly admitted");
    }
    if (errno != ENOMEM) return fail("over-limit mapping error");
    printf("{\"primitive\":\"address-space\",\"passed\":true,"
           "\"initial_virtual_bytes\":%" PRIu64 ",\"limit_bytes\":%" PRIu64 "}\n",
           (uint64_t)info.virtual_size, (uint64_t)limit);
    return 0;
}

static int open_files(void) {
    if (lower_limit(RLIMIT_NOFILE, 32) != 0) return 1;
    int descriptors[64];
    size_t used = 0;
    int exhausted = 0;
    while (used < sizeof(descriptors) / sizeof(descriptors[0])) {
        int fd = open("/dev/null", O_RDONLY | O_CLOEXEC);
        if (fd < 0) {
            exhausted = errno == EMFILE;
            break;
        }
        descriptors[used++] = fd;
    }
    int closed = 1;
    for (size_t i = 0; i < used; i++) {
        if (close(descriptors[i]) != 0) closed = 0;
    }
    if (!exhausted || !closed || used == 0 || used > 32) {
        errno = EINVAL;
        return fail("open-file boundary or cleanup");
    }
    printf("{\"primitive\":\"open-files\",\"passed\":true,"
           "\"additional_descriptors\":%zu,\"limit\":32}\n", used);
    return 0;
}

static int file_size(const char *path) {
    if (signal(SIGXFSZ, SIG_DFL) == SIG_ERR) return fail("file signal disposition");
    if (lower_limit(RLIMIT_FSIZE, 4096) != 0) return 1;
    int fd = open(path, O_WRONLY | O_CREAT | O_EXCL | O_NOFOLLOW, 0600);
    if (fd < 0) return fail("create owned file");
    const unsigned char data[4096] = {0};
    if (write(fd, data, sizeof(data)) != (ssize_t)sizeof(data)) {
        (void)close(fd);
        return fail("admitted file write");
    }
    if (puts("ready:file-size") == EOF || fflush(stdout) != 0) {
        (void)close(fd);
        return fail("file boundary marker");
    }
    (void)write(fd, data, 1);
    (void)close(fd);
    errno = EINVAL;
    return fail("file boundary did not terminate child");
}

static int cpu_time(void) {
    if (signal(SIGXCPU, SIG_DFL) == SIG_ERR) return fail("CPU signal disposition");
    if (lower_limit(RLIMIT_CPU, 1) != 0) return 1;
    if (puts("ready:cpu-time") == EOF || fflush(stdout) != 0) return fail("CPU boundary marker");
    volatile uint64_t counter = 0;
    for (;;) counter++;
}

int main(int argc, char **argv) {
    if (argc != 3) {
        errno = EINVAL;
        return fail("fixed arguments");
    }
    if (lower_limit(RLIMIT_CORE, 0) != 0) return 1;
    if (strcmp(argv[1], "address-space") == 0) return address_space();
    if (strcmp(argv[1], "open-files") == 0) return open_files();
    if (strcmp(argv[1], "file-size") == 0) return file_size(argv[2]);
    if (strcmp(argv[1], "cpu-time") == 0) return cpu_time();
    errno = EINVAL;
    return fail("unknown primitive");
}
