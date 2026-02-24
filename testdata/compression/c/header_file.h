#include <stdio.h>
#include <stdlib.h>
#include <string.h>

#define MAX_BUFFER 4096
#define MIN(a, b) ((a) < (b) ? (a) : (b))

struct Config {
    char *host;
    int port;
    int max_connections;
};

enum LogLevel {
    LOG_DEBUG,
    LOG_INFO,
    LOG_WARN,
    LOG_ERROR
};

typedef void (*log_handler_t)(enum LogLevel, const char *);
typedef unsigned long size_t;

struct Config *config_new(const char *host, int port);
void config_free(struct Config *cfg);
int config_validate(const struct Config *cfg);