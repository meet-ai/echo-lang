#include "mark_service.h"
#include <stdlib.h>
#include <string.h>
#include <stdio.h>
#include <pthread.h>
#include <unistd.h>
#include "echo/gc.h"
#include "../aggregates/heap.h"
#include "../aggregates/garbage_collector.h"

// 工作线程函数
static void* mark_worker_thread(void* arg) {
    echo_gc_mark_service_t* service = (echo_gc_mark_service_t*)arg;

    while (service->running) {
        // 获取工作项
        pthread_mutex_lock(&service->work_queue_mutex);

        while (service->work_queue_head == NULL && service->running) {
            pthread_cond_wait(&service->work_queue_cond, &service->work_queue_mutex);
        }

        if (!service->running) {
            pthread_mutex_unlock(&service->work_queue_mutex);
            break;
        }

        // 获取工作项
        mark_work_item_t* work_item = service->work_queue_head;
        if (work_item) {
            service->work_queue_head = work_item->next;
            if (service->work_queue_head == NULL) {
                service->work_queue_tail = NULL;
            }
        }

        pthread_mutex_unlock(&service->work_queue_mutex);

        if (work_item) {
            // 处理对象标记
            echo_gc_process_mark_work(service, work_item->object);
            free(work_item);

            __atomic_fetch_add(&service->work_items_processed, 1, __ATOMIC_RELAXED);
        }
    }

    return NULL;
}

// 处理标记工作
static void echo_gc_process_mark_work(echo_gc_mark_service_t* service, void* obj) {
    if (!obj) return;

    // 标记对象为黑色（已处理）
    echo_gc_mark_object_black(service->heap, obj);

    // 扫描对象引用，生成新的工作项
    echo_gc_scan_object_references(service->heap, obj,
        echo_gc_mark_service_enqueue_work, service);

    __atomic_fetch_add(&service->objects_marked, 1, __ATOMIC_RELAXED);
}

// 创建标记服务
echo_gc_mark_service_t* echo_gc_mark_service_create(
    struct echo_gc_heap* heap,
    struct echo_gc_collector* collector
) {
    echo_gc_mark_service_t* service = calloc(1, sizeof(echo_gc_mark_service_t));
    if (!service) {
        return NULL;
    }

    service->heap = heap;
    service->collector = collector;

    // 初始化工作队列
    pthread_mutex_init(&service->work_queue_mutex, NULL);
    pthread_cond_init(&service->work_queue_cond, NULL);

    // 设置工作线程数
    service->worker_count = 4;  // 暂时固定
    service->workers = calloc(service->worker_count, sizeof(pthread_t));
    if (!service->workers) {
        free(service);
        return NULL;
    }

    // 初始化屏障
    pthread_barrier_init(&service->barrier, NULL, service->worker_count);

    return service;
}

// 销毁标记服务
void echo_gc_mark_service_destroy(echo_gc_mark_service_t* service) {
    if (!service) return;

    // 停止工作线程
    service->running = false;
    pthread_mutex_lock(&service->work_queue_mutex);
    pthread_cond_broadcast(&service->work_queue_cond);
    pthread_mutex_unlock(&service->work_queue_mutex);

    // 等待线程结束
    for (uint32_t i = 0; i < service->worker_count; i++) {
        if (service->workers[i]) {
            pthread_join(service->workers[i], NULL);
        }
    }

    // 清理资源
    free(service->workers);
    pthread_mutex_destroy(&service->work_queue_mutex);
    pthread_cond_destroy(&service->work_queue_cond);
    pthread_barrier_destroy(&service->barrier);

    free(service);
}

// 开始并发标记
echo_gc_error_t echo_gc_mark_service_start_concurrent_mark(
    echo_gc_mark_service_t* service
) {
    if (!service || service->running) {
        return ECHO_GC_ERROR_SYSTEM_ERROR;
    }

    // 重置状态
    service->running = true;
    service->completed = false;
    service->objects_marked = 0;
    service->bytes_processed = 0;
    service->work_items_processed = 0;

    // 启动工作线程
    for (uint32_t i = 0; i < service->worker_count; i++) {
        if (pthread_create(&service->workers[i], NULL, mark_worker_thread, service) != 0) {
            service->running = false;
            return ECHO_GC_ERROR_SYSTEM_ERROR;
        }
    }

    // 扫描根对象
    echo_gc_mark_service_scan_roots(service);

    return ECHO_GC_SUCCESS;
}

// 等待标记完成
echo_gc_error_t echo_gc_mark_service_wait_for_completion(
    echo_gc_mark_service_t* service,
    uint64_t* objects_marked,
    uint64_t* bytes_processed
) {
    if (!service) {
        return ECHO_GC_ERROR_INVALID_CONFIG;
    }

    // 等待所有工作完成
    service->running = false;

    // 通知所有线程退出
    pthread_mutex_lock(&service->work_queue_mutex);
    pthread_cond_broadcast(&service->work_queue_cond);
    pthread_mutex_unlock(&service->work_queue_mutex);

    // 等待线程结束
    for (uint32_t i = 0; i < service->worker_count; i++) {
        if (service->workers[i]) {
            pthread_join(service->workers[i], NULL);
            service->workers[i] = 0;
        }
    }

    service->completed = true;

    // 返回统计信息
    if (objects_marked) {
        *objects_marked = service->objects_marked;
    }
    if (bytes_processed) {
        *bytes_processed = service->bytes_processed;
    }

    return ECHO_GC_SUCCESS;
}

// 扫描根对象
static void echo_gc_mark_service_scan_roots(echo_gc_mark_service_t* service) {
    if (service->scan_roots_callback) {
        service->scan_roots_callback(service->scan_roots_context);
    }

    // 默认根对象扫描（简化实现）
    // 实际需要扫描：栈、全局变量、线程本地存储等
}

// 设置根对象扫描器
void echo_gc_mark_service_set_root_scanner(
    echo_gc_mark_service_t* service,
    void (*callback)(void* context),
    void* context
) {
    if (service) {
        service->scan_roots_callback = callback;
        service->scan_roots_context = context;
    }
}

// 添加工作项到队列
static void echo_gc_mark_service_enqueue_work(void* obj, void* context) {
    echo_gc_mark_service_t* service = (echo_gc_mark_service_t*)context;

    if (!obj || !echo_gc_heap_is_marked(service->heap, obj)) {
        mark_work_item_t* work_item = calloc(1, sizeof(mark_work_item_t));
        if (work_item) {
            work_item->object = obj;

            pthread_mutex_lock(&service->work_queue_mutex);

            if (service->work_queue_tail) {
                service->work_queue_tail->next = work_item;
            } else {
                service->work_queue_head = work_item;
            }
            service->work_queue_tail = work_item;

            pthread_cond_signal(&service->work_queue_cond);
            pthread_mutex_unlock(&service->work_queue_mutex);
        }
    }
}

// 写屏障实现
echo_gc_write_barrier_t* echo_gc_write_barrier_create(struct echo_gc_heap* heap) {
    echo_gc_write_barrier_t* barrier = calloc(1, sizeof(echo_gc_write_barrier_t));
    if (!barrier) {
        return NULL;
    }

    barrier->heap = heap;
    barrier->buffer.max_count = 1024;
    barrier->enabled = false;

    pthread_mutex_init(&barrier->buffer.mutex, NULL);

    return barrier;
}

void echo_gc_write_barrier_destroy(echo_gc_write_barrier_t* barrier) {
    if (barrier) {
        pthread_mutex_destroy(&barrier->buffer.mutex);
        free(barrier);
    }
}

void echo_gc_write_barrier_enable(echo_gc_write_barrier_t* barrier) {
    if (barrier) {
        __atomic_store_n(&barrier->enabled, true, __ATOMIC_RELEASE);
    }
}

void echo_gc_write_barrier_disable(echo_gc_write_barrier_t* barrier) {
    if (barrier) {
        __atomic_store_n(&barrier->enabled, false, __ATOMIC_RELEASE);
        // 处理剩余缓冲区
        echo_gc_write_barrier_process_buffer(barrier);
    }
}

bool echo_gc_write_barrier_is_enabled(const echo_gc_write_barrier_t* barrier) {
    if (!barrier) return false;
    return __atomic_load_n(&barrier->enabled, __ATOMIC_ACQUIRE);
}

// 写屏障执行（编译器插入）
void echo_gc_write_barrier_write(
    echo_gc_write_barrier_t* barrier,
    void** slot,
    void* new_value
) {
    if (!barrier || !echo_gc_write_barrier_is_enabled(barrier)) {
        *slot = new_value;
        return;
    }

    void* old_value = *slot;

    // Dijkstra插入屏障：标记新引用
    if (new_value != NULL && !echo_gc_heap_is_marked(barrier->heap, new_value)) {
        echo_gc_mark_object_gray(barrier->heap, new_value);
    }

    // 更新缓冲区
    pthread_mutex_lock(&barrier->buffer.mutex);

    if (barrier->buffer.count < barrier->buffer.max_count) {
        barrier->buffer.entries[barrier->buffer.count].slot = slot;
        barrier->buffer.entries[barrier->buffer.count].old_value = old_value;
        barrier->buffer.entries[barrier->buffer.count].new_value = new_value;
        barrier->buffer.count++;
    }

    pthread_mutex_unlock(&barrier->buffer.mutex);

    // 实际写入
    *slot = new_value;
}

// 处理缓冲区（在安全点调用）
void echo_gc_write_barrier_process_buffer(echo_gc_write_barrier_t* barrier) {
    if (!barrier) return;

    pthread_mutex_lock(&barrier->buffer.mutex);

    for (uint32_t i = 0; i < barrier->buffer.count; i++) {
        void* old_value = barrier->buffer.entries[i].old_value;

        // Yuasa删除屏障：如果旧值未标记，则标记为灰色
        if (old_value != NULL && !echo_gc_heap_is_marked(barrier->heap, old_value)) {
            echo_gc_mark_object_gray(barrier->heap, old_value);
        }
    }

    barrier->buffer.count = 0;

    pthread_mutex_unlock(&barrier->buffer.mutex);
}

// 对象标记辅助函数
bool echo_gc_is_object_marked(struct echo_gc_heap* heap, void* obj) {
    return echo_gc_heap_is_marked(heap, obj);
}

echo_gc_error_t echo_gc_mark_object(struct echo_gc_heap* heap, void* obj) {
    return echo_gc_heap_mark_object(heap, obj);
}

echo_gc_error_t echo_gc_mark_object_gray(struct echo_gc_heap* heap, void* obj) {
    // 灰色对象：已标记但未扫描引用
    // 在简化实现中，标记为已标记即可
    return echo_gc_heap_mark_object(heap, obj);
}

echo_gc_error_t echo_gc_mark_object_black(struct echo_gc_heap* heap, void* obj) {
    // 黑色对象：已标记且已扫描引用
    // 在简化实现中，标记为已标记即可
    return echo_gc_heap_mark_object(heap, obj);
}

// 对象引用扫描（简化实现）
void echo_gc_scan_object_references(
    struct echo_gc_heap* heap,
    void* obj,
    void (*callback)(void* ref, void* context),
    void* context
) {
    if (!obj || !callback) return;

    // 获取对象头
    echo_gc_object_header_t* header = (echo_gc_object_header_t*)((char*)obj - sizeof(echo_gc_object_header_t));

    // 简化实现：假设对象可能包含引用
    // 实际需要基于对象类型进行精确扫描
    void** potential_refs = (void**)obj;
    size_t ref_count = (header->size - sizeof(echo_gc_object_header_t)) / sizeof(void*);

    for (size_t i = 0; i < ref_count; i++) {
        void* ref = potential_refs[i];
        if (ref != NULL && echo_gc_heap_contains(heap, ref)) {
            callback(ref, context);
        }
    }
}

// 根对象扫描辅助函数
void echo_gc_scan_stack_roots(void* context) {
    // 扫描当前线程栈
    // 简化实现，实际需要遍历栈帧
    printf("Scanning stack roots (simplified)\n");
}

void echo_gc_scan_global_roots(void* context) {
    // 扫描全局变量
    // 简化实现，实际需要遍历全局数据区
    printf("Scanning global roots (simplified)\n");
}

void echo_gc_scan_thread_roots(void* context) {
    // 扫描线程本地存储
    // 简化实现
    printf("Scanning thread roots (simplified)\n");
}
