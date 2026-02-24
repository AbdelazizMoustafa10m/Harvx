#include "config.h"
#include <string.h>

static int internal_counter = 0;

/**
 * Create a new config.
 */
struct Config *config_new(const char *host, int port) {
    struct Config *cfg = malloc(sizeof(struct Config));
    if (!cfg) return NULL;
    cfg->host = strdup(host);
    cfg->port = port;
    cfg->max_connections = 100;
    internal_counter++;
    return cfg;
}

void config_free(struct Config *cfg) {
    if (cfg) {
        free(cfg->host);
        free(cfg);
    }
}

int config_validate(const struct Config *cfg) {
    if (!cfg) return -1;
    if (cfg->port <= 0 || cfg->port > 65535) return -1;
    if (!cfg->host) return -1;
    return 0;
}