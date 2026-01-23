#ifndef ECHO_GC_MARK_SERVICE_H
#define ECHO_GC_MARK_SERVICE_H

#include <stdint.h>
#include <stdbool.h>
#include <pthread.h>
#include "echo/gc.h"

// 前向声明
struct echo_gc_heap;
struct echo_gc_collector;

// 标记工作项
typedef struct mark_work_item {
    void* object;                      // 要标记的对象
    struct mark_work_item* next;       // 链表下一个
} mark_work_item_t;

// 标记栈（每个线程一个）
typedef struct mark_stack {
    mark_work_item_t* items;           // 工作项数组
    uint32_t size;                     // 当前大小
    uint32_t capacity;                 // 容量
    uint32_t top;                      // 栈顶
} mark_stack_t;

// 并发标记服务
typedef struct echo_gc_mark_service {
    struct echo_gc_heap* heap;         // 关联的堆
    struct echo_gc_collector* collector; // 关联的收集器

    // 并发控制
    uint32_t worker_count;             // 工作线程数
    pthread_t* workers;                // 工作线程
    pthread_barrier_t barrier;         // 同步屏障

    // 工作队列（简单的全局队列）
    mark_work_item_t* work_queue_head;
    mark_work_item_t* work_queue_tail;
    pthread_mutex_t work_queue_mutex;
    pthread_cond_t work_queue_cond;

    // 统计信息
    uint64_t objects_marked;
    uint64_t bytes_processed;
    uint64_t work_items_processed;

    // 控制标志
    volatile bool running;
    volatile bool completed;

    // 根对象扫描器
    void (*scan_roots_callback)(void* context);
    void* scan_roots_context;
} echo_gc_mark_service_t;

// 写屏障缓冲区
typedef struct write_barrier_buffer {
    struct {
        void** slot;                   // 写操作的地址
        void* old_value;               // 旧值
        void* new_value;               // 新值
    } entries[1024];                   // 缓冲区大小

    uint32_t count;                    // 当前条目数
    uint32_t max_count;                // 最大条目数
    pthread_mutex_t mutex;             // 缓冲区锁
} write_barrier_buffer_t;

// 写屏障服务
typedef struct echo_gc_write_barrier {
    write_barrier_buffer_t buffer;
    struct echo_gc_heap* heap;
    volatile bool enabled;             // 是否启用写屏障
} echo_gc_write_barrier_t;

// 标记服务生命周期
echo_gc_mark_service_t* echo_gc_mark_service_create(
    struct echo_gc_heap* heap,
    struct echo_gc_collector* collector
);

void echo_gc_mark_service_destroy(echo_gc_mark_service_t* service);

// 并发标记执行
echo_gc_error_t echo_gc_mark_service_start_concurrent_mark(
    echo_gc_mark_service_t* service
);

echo_gc_error_t echo_gc_mark_service_wait_for_completion(
    echo_gc_mark_service_t* service,
    uint64_t* objects_marked,
    uint64_t* bytes_processed
);

// 根对象扫描设置
void echo_gc_mark_service_set_root_scanner(
    echo_gc_mark_service_t* service,
    void (*callback)(void* context),
    void* context
);

// 写屏障服务
echo_gc_write_barrier_t* echo_gc_write_barrier_create(struct echo_gc_heap* heap);
void echo_gc_write_barrier_destroy(echo_gc_write_barrier_t* barrier);

// 写屏障控制
void echo_gc_write_barrier_enable(echo_gc_write_barrier_t* barrier);
void echo_gc_write_barrier_disable(echo_gc_write_barrier_t* barrier);
bool echo_gc_write_barrier_is_enabled(const echo_gc_write_barrier_t* barrier);

// 写屏障执行
void echo_gc_write_barrier_write(
    echo_gc_write_barrier_t* barrier,
    void** slot,
    void* new_value
);

// 缓冲区处理
void echo_gc_write_barrier_process_buffer(echo_gc_write_barrier_t* barrier);

// 对象标记辅助函数
bool echo_gc_is_object_marked(struct echo_gc_heap* heap, void* obj);
echo_gc_error_t echo_gc_mark_object(struct echo_gc_heap* heap, void* obj);
echo_gc_error_t echo_gc_mark_object_gray(struct echo_gc_heap* heap, void* obj);
echo_gc_error_t echo_gc_mark_object_black(struct echo_gc_heap* heap, void* obj);

// 对象引用扫描
void echo_gc_scan_object_references(
    struct echo_gc_heap* heap,
    void* obj,
    void (*callback)(void* ref, void* context),
    void* context
);

// 根对象扫描辅助
void echo_gc_scan_stack_roots(void* context);
void echo_gc_scan_global_roots(void* context);
void echo_gc_scan_thread_roots(void* context);

#endif // ECHO_GC_MARK_SERVICE_H
