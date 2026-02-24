typedef unsigned char uint8_t;
typedef unsigned short uint16_t;
typedef unsigned int uint32_t;

typedef struct {
    char name[64];
    int age;
} Person;

typedef struct node {
    int value;
    struct node *next;
} Node;

typedef enum {
    STATUS_OK,
    STATUS_ERROR,
    STATUS_PENDING
} Status;

typedef void (*handler_t)(int, const char*);
typedef int (*comparator_t)(const void *, const void *);