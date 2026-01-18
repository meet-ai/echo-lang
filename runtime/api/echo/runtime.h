#ifndef ECHO_RUNTIME_H
#define ECHO_RUNTIME_H

#include <stdint.h>
#include <stdbool.h>
#include <stddef.h>

#ifdef __cplusplus
extern "C" {
#endif

// Version information
#define ECHO_RUNTIME_VERSION_MAJOR 0
#define ECHO_RUNTIME_VERSION_MINOR 1
#define ECHO_RUNTIME_VERSION_PATCH 0

// Platform detection
#if defined(_WIN32) || defined(_WIN64)
#define ECHO_PLATFORM_WINDOWS
#elif defined(__APPLE__)
#define ECHO_PLATFORM_MACOS
#elif defined(__linux__)
#define ECHO_PLATFORM_LINUX
#else
#define ECHO_PLATFORM_UNIX
#endif

// Export/import macros
#ifdef ECHO_PLATFORM_WINDOWS
#ifdef ECHO_RUNTIME_BUILD_SHARED
#define ECHO_API __declspec(dllexport)
#else
#define ECHO_API
#endif
#ifdef ECHO_RUNTIME_USE_SHARED
#define ECHO_API_IMPORT __declspec(dllimport)
#else
#define ECHO_API_IMPORT
#endif
#else
#define ECHO_API
#define ECHO_API_IMPORT
#endif

// Error codes
typedef enum {
    ECHO_SUCCESS = 0,
    ECHO_ERROR_INVALID_ARGUMENT = -1,
    ECHO_ERROR_OUT_OF_MEMORY = -2,
    ECHO_ERROR_NOT_SUPPORTED = -3,
    ECHO_ERROR_TIMEOUT = -4,
    ECHO_ERROR_IO = -5,
    ECHO_ERROR_PERMISSION_DENIED = -6,
    ECHO_ERROR_SYSTEM = -7
} echo_error_t;

// Runtime configuration
typedef struct {
    size_t heap_size;           // Initial heap size in bytes
    size_t stack_size;          // Default stack size for coroutines
    int max_processors;         // Maximum number of processors
    bool enable_gc;             // Enable garbage collection
    bool enable_debug;          // Enable debug features
    const char* log_level;      // Log level: "debug", "info", "warn", "error"
} echo_config_t;

// Runtime instance
typedef struct echo_runtime echo_runtime_t;

// Initialization and shutdown
ECHO_API echo_error_t echo_runtime_init(const echo_config_t* config, echo_runtime_t** runtime);
ECHO_API echo_error_t echo_runtime_shutdown(echo_runtime_t* runtime);

// Runtime information
ECHO_API echo_error_t echo_runtime_get_version(int* major, int* minor, int* patch);
ECHO_API echo_error_t echo_runtime_get_stats(echo_runtime_t* runtime, void* stats);

// Memory management
ECHO_API void* echo_alloc(size_t size);
ECHO_API void* echo_realloc(void* ptr, size_t size);
ECHO_API void echo_free(void* ptr);

// Error handling
ECHO_API const char* echo_error_string(echo_error_t error);

// Logging
typedef enum {
    ECHO_LOG_DEBUG,
    ECHO_LOG_INFO,
    ECHO_LOG_WARN,
    ECHO_LOG_ERROR
} echo_log_level_t;

typedef void (*echo_log_callback_t)(echo_log_level_t level, const char* message, void* user_data);

ECHO_API echo_error_t echo_set_log_callback(echo_log_callback_t callback, void* user_data);
ECHO_API echo_error_t echo_set_log_level(echo_log_level_t level);

// Thread and concurrency utilities
typedef void* echo_thread_t;
typedef void (*echo_thread_func_t)(void* arg);

ECHO_API echo_error_t echo_thread_create(echo_thread_func_t func, void* arg, echo_thread_t* thread);
ECHO_API echo_error_t echo_thread_join(echo_thread_t thread);
ECHO_API echo_error_t echo_thread_detach(echo_thread_t thread);

// Atomic operations
typedef volatile int32_t echo_atomic_int32_t;
typedef volatile int64_t echo_atomic_int64_t;

ECHO_API int32_t echo_atomic_load_int32(echo_atomic_int32_t* ptr);
ECHO_API void echo_atomic_store_int32(echo_atomic_int32_t* ptr, int32_t value);
ECHO_API int32_t echo_atomic_exchange_int32(echo_atomic_int32_t* ptr, int32_t value);
ECHO_API bool echo_atomic_compare_exchange_int32(echo_atomic_int32_t* ptr, int32_t* expected, int32_t desired);

// Time utilities
typedef struct {
    int64_t seconds;
    int32_t nanoseconds;
} echo_duration_t;

typedef struct {
    int64_t seconds_since_epoch;
    int32_t nanoseconds;
} echo_time_t;

ECHO_API echo_time_t echo_time_now();
ECHO_API echo_duration_t echo_duration_between(echo_time_t start, echo_time_t end);
ECHO_API void echo_sleep(echo_duration_t duration);

#ifdef __cplusplus
}
#endif

#endif // ECHO_RUNTIME_H
