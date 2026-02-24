#include <stdio.h>
#include <stdlib.h>
#include <string.h>

#define VERSION "1.0.0"
#define MAX(a, b) ((a) > (b) ? (a) : (b))

struct Point {
    int x;
    int y;
};

enum Color {
    RED,
    GREEN,
    BLUE
};

typedef void (*callback_t)(int);

static int counter = 0;

/**
 * Create a new point.
 */
struct Point *point_new(int x, int y) {
    struct Point *p = malloc(sizeof(struct Point));
    p->x = x;
    p->y = y;
    counter++;
    return p;
}

void point_free(struct Point *p) {
    free(p);
}

int point_distance(const struct Point *a, const struct Point *b) {
    int dx = a->x - b->x;
    int dy = a->y - b->y;
    return dx * dx + dy * dy;
}