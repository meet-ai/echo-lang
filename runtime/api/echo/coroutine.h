#ifndef ECHO_COROUTINE_H
#define ECHO_COROUTINE_H

#include "runtime.h"

#ifdef __cplusplus
extern "C" {
#endif

// Coroutine handle
typedef struct echo_coroutine echo_coroutine_t;

// Coroutine function type
typedef void (*echo_coroutine_func_t)(void* arg);

// Coroutine status
typedef enum {
    ECHO_COROUTINE_READY,      // Ready to run
    ECHO_COROUTINE_RUNNING,    // Currently running
    ECHO_COROUTINE_SUSPENDED,  // Suspended (yielded)
    ECHO_COROUTINE_COMPLETED,  // Completed execution
    ECHO_COROUTINE_CANCELLED   // Cancelled
} echo_coroutine_status_t;

// Coroutine configuration
typedef struct {
    size_t stack_size;         // Stack size in bytes
    const char* name;          // Coroutine name (for debugging)
    int priority;              // Priority (higher value = higher priority)
    bool enable_cancellation;  // Enable cancellation support
} echo_coroutine_config_t;

// Create and destroy coroutines
ECHO_API echo_error_t echo_coroutine_create(echo_coroutine_func_t func, void* arg,
                                           const echo_coroutine_config_t* config,
                                           echo_coroutine_t** coroutine);
ECHO_API echo_error_t echo_coroutine_destroy(echo_coroutine_t* coroutine);

// Coroutine lifecycle management
ECHO_API echo_error_t echo_coroutine_resume(echo_coroutine_t* coroutine);
ECHO_API echo_error_t echo_coroutine_yield(echo_coroutine_t* coroutine);
ECHO_API echo_error_t echo_coroutine_cancel(echo_coroutine_t* coroutine);

// Status and information
ECHO_API echo_coroutine_status_t echo_coroutine_get_status(echo_coroutine_t* coroutine);
ECHO_API echo_error_t echo_coroutine_get_name(echo_coroutine_t* coroutine, const char** name);
ECHO_API echo_error_t echo_coroutine_get_id(echo_coroutine_t* coroutine, uint64_t* id);

// Stack management
ECHO_API echo_error_t echo_coroutine_get_stack_size(echo_coroutine_t* coroutine, size_t* size);
ECHO_API echo_error_t echo_coroutine_get_stack_usage(echo_coroutine_t* coroutine, size_t* usage);

// Exception handling in coroutines
ECHO_API echo_error_t echo_coroutine_set_exception_handler(echo_coroutine_t* coroutine,
                                                          echo_exception_handler_t handler,
                                                          void* user_data);

// Coroutine local storage
typedef void* echo_cls_key_t;

ECHO_API echo_error_t echo_cls_create_key(echo_cls_key_t* key);
ECHO_API echo_error_t echo_cls_set_value(echo_cls_key_t key, void* value);
ECHO_API void* echo_cls_get_value(echo_cls_key_t key);
ECHO_API echo_error_t echo_cls_delete_key(echo_cls_key_t key);

// Coroutine pools for efficient management
typedef struct echo_coroutine_pool echo_coroutine_pool_t;

typedef struct {
    size_t max_coroutines;     // Maximum number of coroutines
    size_t stack_size;         // Default stack size
    size_t prealloc_count;     // Number of coroutines to pre-allocate
} echo_coroutine_pool_config_t;

ECHO_API echo_error_t echo_coroutine_pool_create(const echo_coroutine_pool_config_t* config,
                                                echo_coroutine_pool_t** pool);
ECHO_API echo_error_t echo_coroutine_pool_destroy(echo_coroutine_pool_t* pool);

ECHO_API echo_error_t echo_coroutine_pool_acquire(echo_coroutine_pool_t* pool,
                                                 echo_coroutine_func_t func, void* arg,
                                                 const echo_coroutine_config_t* config,
                                                 echo_coroutine_t** coroutine);
ECHO_API echo_error_t echo_coroutine_pool_release(echo_coroutine_pool_t* pool,
                                                 echo_coroutine_t* coroutine);

// Statistics
typedef struct {
    uint64_t total_created;    // Total coroutines created
    uint64_t total_destroyed;  // Total coroutines destroyed
    uint64_t currently_active; // Currently active coroutines
    uint64_t max_concurrent;   // Maximum concurrent coroutines
    size_t total_stack_allocated; // Total stack memory allocated
} echo_coroutine_stats_t;

ECHO_API echo_error_t echo_coroutine_get_stats(echo_coroutine_stats_t* stats);

// Advanced features
// Channel-based communication between coroutines
typedef struct echo_channel echo_channel_t;

ECHO_API echo_error_t echo_channel_create(size_t capacity, echo_channel_t** channel);
ECHO_API echo_error_t echo_channel_destroy(echo_channel_t* channel);

ECHO_API echo_error_t echo_channel_send(echo_channel_t* channel, void* data);
ECHO_API echo_error_t echo_channel_receive(echo_channel_t* channel, void** data);
ECHO_API echo_error_t echo_channel_try_send(echo_channel_t* channel, void* data);
ECHO_API echo_error_t echo_channel_try_receive(echo_channel_t* channel, void** data);

// Select operation for multiple channels
typedef struct {
    echo_channel_t* channel;
    void* data;
    bool is_send;
} echo_select_case_t;

ECHO_API echo_error_t echo_channel_select(echo_select_case_t* cases, size_t case_count,
                                         size_t* selected_index);

#ifdef __cplusplus
}
#endif

#endif // ECHO_COROUTINE_H

