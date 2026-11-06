#define _DARWIN_C_SOURCE 1

#include <errno.h>
#include <fcntl.h>
#include <inttypes.h>
#include <signal.h>
#include <stdint.h>
#include <stdio.h>
#include <stdlib.h>
#include <string.h>
#include <sys/resource.h>
#include <unistd.h>

/* This fixed launcher performs no networking and starts exactly one program.
 * The owning Go parent supplies private copies and a finite common lifetime.
 * No setting here is a RAM reservation or a total-storage qualification. */
static int fail(void) {
    static const char message[] = "registry worker launcher unavailable\n";
    (void)write(STDERR_FILENO, message, sizeof(message) - 1);
    return 125;
}

static int number(const char *text, uint64_t maximum, uint64_t *out) {
    uint64_t value = 0;
    size_t n = strnlen(text, 21);
    if (n == 0 || n > 20 || text[0] == '0') return 0;
    for (size_t i = 0; i < n; i++) {
        if (text[i] < '0' || text[i] > '9') return 0;
        uint64_t digit = (uint64_t)(text[i] - '0');
        if (value > (maximum - digit) / 10) return 0;
        value = value * 10 + digit;
    }
    if (value == 0 || value > maximum) return 0;
    *out = value;
    return 1;
}

static int lower_limit(int resource, uint64_t value) {
    struct rlimit old, actual;
    if (getrlimit(resource, &old) != 0) return 0;
    if (old.rlim_max != RLIM_INFINITY && value > (uint64_t)old.rlim_max) return 0;
    struct rlimit selected = {(rlim_t)value, (rlim_t)value};
    if ((uint64_t)selected.rlim_cur != value || selected.rlim_cur == RLIM_INFINITY ||
        setrlimit(resource, &selected) != 0 || getrlimit(resource, &actual) != 0) return 0;
    return actual.rlim_cur == selected.rlim_cur && actual.rlim_max == selected.rlim_max;
}

int main(int argc, char **argv) {
    uint64_t cpu, file, descriptors;
    if (argc != 5 || argv[1][0] != '/' || strnlen(argv[1], 4097) > 4096 ||
        !number(argv[2], UINT64_C(31) * 24 * 60 * 60, &cpu) ||
        !number(argv[3], INT64_MAX, &file) ||
        !number(argv[4], 65536, &descriptors) || descriptors < 16) return fail();
    if (fcntl(3, F_SETFD, FD_CLOEXEC) != 0 ||
        signal(SIGXCPU, SIG_DFL) == SIG_ERR || signal(SIGXFSZ, SIG_DFL) == SIG_ERR ||
        !lower_limit(RLIMIT_CORE, 0) || !lower_limit(RLIMIT_CPU, cpu) ||
        !lower_limit(RLIMIT_FSIZE, file) || !lower_limit(RLIMIT_NOFILE, descriptors)) return fail();
    char ready[128];
    int n = snprintf(ready, sizeof(ready), "sage-registry-worker-v1 %" PRIu64 " %" PRIu64 " %" PRIu64 "\n",
                     cpu, file, descriptors);
    if (n <= 0 || (size_t)n >= sizeof(ready) || write(3, ready, (size_t)n) != n) return fail();
    char *arguments[] = {argv[1], "--sage-registry-history-worker", NULL};
    execv(argv[1], arguments);
    return fail();
}
