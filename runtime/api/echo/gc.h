#ifndef ECHO_GC_H
#define ECHO_GC_H

#include "runtime.h"

#ifdef __cplusplus
extern "C" {
#endif

// GC configuration
typedef struct {
    size_t initial_heap_size;    // Initial heap size
    size_t max_heap_size;        // Maximum heap size
    double gc_trigger_ratio;     // GC trigger ratio (0.0-1.0)
    bool enable_concurrent;      // Enable concurrent GC
    bool enable_generational;    // Enable generational GC
    bool enable_compaction;      // Enable heap compaction
    int gc_threads;             // Number of GC threads
} echo_gc_config_t;

// GC statistics
typedef struct {
    size_t heap_size;           // Current heap size
    size_t heap_used;           // Used heap size
    size_t objects_count;       // Number of live objects
    uint64_t gc_cycles;         // Number of GC cycles completed
    echo_duration_t total_gc_time; // Total time spent in GC
    echo_duration_t last_gc_time;  // Time spent in last GC cycle
    double gc_efficiency;       // GC efficiency (0.0-1.0)
} echo_gc_stats_t;

// GC instance
typedef struct echo_gc echo_gc_t;

// GC initialization
ECHO_API echo_error_t echo_gc_init(const echo_gc_config_t* config, echo_gc_t** gc);
ECHO_API echo_error_t echo_gc_shutdown(echo_gc_t* gc);

// Memory allocation with GC
ECHO_API void* echo_gc_alloc(echo_gc_t* gc, size_t size);
ECHO_API void* echo_gc_realloc(echo_gc_t* gc, void* ptr, size_t size);
ECHO_API void echo_gc_free(echo_gc_t* gc, void* ptr);

// Manual GC operations
ECHO_API echo_error_t echo_gc_collect(echo_gc_t* gc);
ECHO_API echo_error_t echo_gc_collect_minor(echo_gc_t* gc);  // Minor GC (young generation)
ECHO_API echo_error_t echo_gc_collect_full(echo_gc_t* gc);   // Full GC

// GC statistics
ECHO_API echo_error_t echo_gc_get_stats(echo_gc_t* gc, echo_gc_stats_t* stats);

// GC configuration management
ECHO_API echo_error_t echo_gc_set_config(echo_gc_t* gc, const echo_gc_config_t* config);
ECHO_API echo_error_t echo_gc_get_config(echo_gc_t* gc, echo_gc_config_t* config);

// Memory pressure callbacks
typedef void (*echo_gc_pressure_callback_t)(echo_gc_t* gc, double pressure_ratio, void* user_data);

ECHO_API echo_error_t echo_gc_set_pressure_callback(echo_gc_t* gc,
                                                   echo_gc_pressure_callback_t callback,
                                                   void* user_data);

// Object pinning (prevent GC from moving objects)
ECHO_API echo_error_t echo_gc_pin_object(echo_gc_t* gc, void* object);
ECHO_API echo_error_t echo_gc_unpin_object(echo_gc_t* gc, void* object);

// Weak references
typedef struct echo_weak_ref echo_weak_ref_t;

ECHO_API echo_error_t echo_gc_create_weak_ref(echo_gc_t* gc, void* object, echo_weak_ref_t** weak_ref);
ECHO_API void* echo_gc_dereference_weak_ref(echo_weak_ref_t* weak_ref);
ECHO_API echo_error_t echo_gc_destroy_weak_ref(echo_weak_ref_t* weak_ref);

// Finalizers
typedef void (*echo_finalizer_t)(void* object, void* user_data);

ECHO_API echo_error_t echo_gc_set_finalizer(echo_gc_t* gc, void* object,
                                           echo_finalizer_t finalizer, void* user_data);

// GC debugging and monitoring
ECHO_API echo_error_t echo_gc_enable_profiling(echo_gc_t* gc, bool enable);
ECHO_API echo_error_t echo_gc_dump_heap(echo_gc_t* gc, const char* filename);
ECHO_API echo_error_t echo_gc_get_object_info(echo_gc_t* gc, void* object, void* info);

#ifdef __cplusplus
}
#endif

#endif // ECHO_GC_H

