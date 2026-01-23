#include "future.h"
#include "../../shared/types/common_types.h"
#include "../task/task.h"
#include "../coroutine/coroutine.h"
#include "../scheduler/scheduler.h"
#include "../scheduler/processor.h"
#include <stdlib.h>
#include <stdio.h>
#include <string.h>
#include <stdatomic.h>
#include <time.h>

// 全局Future ID计数器（原子操作）
static atomic_uint_fast64_t g_next_future_id = ATOMIC_VAR_INIT(1);

// 全局Future统计
static atomic_uint_fast64_t g_total_futures_created = ATOMIC_VAR_INIT(0);
static atomic_uint_fast64_t g_total_futures_resolved = ATOMIC_VAR_INIT(0);
static atomic_uint_fast64_t g_total_futures_rejected = ATOMIC_VAR_INIT(0);

// 函数声明
void future_add_waiter(Future* future, struct Task* task);
void future_wake_waiters(Future* future);

// 生成Future ID
uint64_t future_generate_id(void) {
    return atomic_fetch_add(&g_next_future_id, 1);
}

// 获取Future统计信息
void future_get_stats(uint64_t* created, uint64_t* resolved, uint64_t* rejected) {
    if (created) *created = atomic_load(&g_total_futures_created);
    if (resolved) *resolved = atomic_load(&g_total_futures_resolved);
    if (rejected) *rejected = atomic_load(&g_total_futures_rejected);
}

// 创建新Future
Future* future_new(void) {
    Future* future = (Future*)malloc(sizeof(Future));
    if (!future) {
        fprintf(stderr, "ERROR: Failed to allocate memory for future\n");
        return NULL;
    }

    memset(future, 0, sizeof(Future));

    // 初始化Future属性
    future->id = future_generate_id();
    future->state = FUTURE_PENDING;
    future->result = NULL;
    future->consumed = false;
    future->wait_queue_head = NULL;
    future->wait_queue_tail = NULL;
    future->wait_count = 0;

    // 初始化时间戳
    future->created_at = time(NULL);
    future->resolved_at = 0;
    future->rejected_at = 0;

    // 初始化统计信息
    future->poll_count = 0;
    future->wait_time_ms = 0;
    future->priority = FUTURE_PRIORITY_NORMAL;

    // 初始化同步机制
    pthread_mutex_init(&future->mutex, NULL);
    pthread_cond_init(&future->cond, NULL);

    // 设置默认方法
    future->cancel = future_cancel;
    future->cleanup = NULL;

    // 更新全局统计
    atomic_fetch_add(&g_total_futures_created, 1);

    return future;
}

// 销毁Future
void future_destroy(Future* future) {
    if (!future) return;

    // 调用清理函数
    if (future->cleanup) {
        future->cleanup(future);
    }

    // 清理等待队列
    WaitNode* current = future->wait_queue_head;
    while (current) {
        WaitNode* next = current->next;
        free(current);
        current = next;
    }

    // 清理结果数据（如果需要）
    if (future->result && future->result_needs_free) {
        free(future->result);
        future->result = NULL;
    }

    // 清理同步机制
    pthread_mutex_destroy(&future->mutex);
    pthread_cond_destroy(&future->cond);

    free(future);
}

// 默认的poll实现（真正的异步等待）
struct PollResult future_default_poll(Future* future, struct Task* task) {
    struct PollResult result = { POLL_PENDING, NULL };

    // 检查Future状态
    if (future->state == FUTURE_RESOLVED) {
        if (!future->consumed) {
            result.status = POLL_READY;
            result.value = future->result;
            future->consumed = true; // 标记为已消费
        } else {
            result.status = POLL_COMPLETED;
            result.value = future->result;
        }
    } else if (future->state == FUTURE_REJECTED) {
        if (!future->consumed) {
            result.status = POLL_READY;
            result.value = future->result; // 这里是错误信息
            future->consumed = true;
        } else {
            result.status = POLL_COMPLETED;
            result.value = future->result;
        }
    } else {
        // 仍然等待中
        result.status = POLL_PENDING;
        result.value = NULL;

        // 如果有任务在等待，将其加入等待队列
        if (task) {
            future_add_waiter(future, task);
        }
    }

    return result;
}

// 添加等待任务到队列
void future_add_waiter(Future* future, struct Task* task) {
    if (!future || !task) return;

    WaitNode* node = (WaitNode*)malloc(sizeof(WaitNode));
    if (!node) {
        fprintf(stderr, "ERROR: Failed to allocate wait node\n");
        return;
    }

    node->task = task;
    node->next = NULL;

    if (future->wait_queue_tail) {
        future->wait_queue_tail->next = node;
        future->wait_queue_tail = node;
    } else {
        future->wait_queue_head = node;
        future->wait_queue_tail = node;
    }

    future->wait_count++;
    printf("DEBUG: Added task %llu to wait queue of future %llu\n", task->id, future->id);
}

// 唤醒所有等待的任务
void future_wake_waiters(Future* future) {
    if (!future) return;

    printf("DEBUG: Waking %d waiters for future %llu\n", future->wait_count, future->id);

    WaitNode* current = future->wait_queue_head;
    while (current) {
        WaitNode* next = current->next;

        // 唤醒等待的任务
        struct Task* task = current->task;
        if (task) {
            printf("DEBUG: Waking task %llu waiting for future %llu\n", task->id, future->id);
            
            // 唤醒任务（将状态从 TASK_WAITING 改为 TASK_READY）
            task_wake(task);
            
            // 如果任务有关联的协程，恢复协程状态
            if (task->coroutine) {
                Coroutine* coroutine = task->coroutine;
                coroutine->state = COROUTINE_READY;
                printf("DEBUG: Set coroutine %llu state to READY\n", coroutine->id);
                
                // 将任务重新加入调度器队列，以便继续执行
                extern Scheduler* get_global_scheduler(void);
                Scheduler* scheduler = get_global_scheduler();
                if (scheduler) {
                    // 获取协程绑定的处理器
                    Processor* processor = coroutine->bound_processor;
                    if (!processor && scheduler->num_processors > 0) {
                        processor = scheduler->processors[0];
                    }
                    
                    if (processor) {
                        extern bool processor_push_local(Processor* processor, Task* task);
                        processor_push_local(processor, task);
                        printf("DEBUG: Pushed task %llu back to processor %u queue\n", 
                               task->id, processor->id);
                    } else {
                        // 如果没有处理器，放回全局队列
                        pthread_mutex_lock(&scheduler->global_lock);
                        task->next = scheduler->global_queue;
                        scheduler->global_queue = task;
                        pthread_mutex_unlock(&scheduler->global_lock);
                        printf("DEBUG: Pushed task %llu back to global queue\n", task->id);
                    }
                }
            }
        }

        free(current);
        current = next;
    }

    future->wait_queue_head = NULL;
    future->wait_queue_tail = NULL;
    future->wait_count = 0;

    // 广播条件变量唤醒所有等待者
    pthread_cond_broadcast(&future->cond);
}

// 轮询Future状态
struct PollResult future_poll(Future* future, struct Task* task) {
    struct PollResult result = { POLL_COMPLETED, NULL };

    if (!future) {
        return result;
    }

    pthread_mutex_lock(&future->mutex);

    // 更新轮询统计
        future->poll_count++;

    // 使用默认的poll实现
    result = future_default_poll(future, task);

    pthread_mutex_unlock(&future->mutex);

    return result;
}

// 添加等待任务到队列

// 取消Future
void future_cancel(Future* future) {
    if (!future) return;

    pthread_mutex_lock(&future->mutex);
    // 取消Future，唤醒所有等待者
    if (future->state == FUTURE_PENDING) {
        future->state = FUTURE_REJECTED;
        future->result = NULL; // 取消没有具体错误信息
        future_wake_waiters(future);
    }
    printf("DEBUG: Cancelled future %llu\n", future->id);
    pthread_mutex_unlock(&future->mutex);
}

// 解决Future（设置成功结果）
void future_resolve(Future* future, void* value) {
    if (!future) return;

    time_t now = time(NULL);

    pthread_mutex_lock(&future->mutex);

    if (future->state == FUTURE_PENDING) {
        future->state = FUTURE_RESOLVED;
        future->result = value;
        future->resolved_at = now;

        // 计算等待时间
        if (future->created_at > 0) {
            future->wait_time_ms = (uint64_t)(now - future->created_at) * 1000;
        }

        // 唤醒所有等待者
        future_wake_waiters(future);

        // 更新全局统计
        atomic_fetch_add(&g_total_futures_resolved, 1);
    }

    pthread_mutex_unlock(&future->mutex);
}

// 拒绝Future（设置错误结果）
void future_reject(Future* future, void* error) {
    if (!future) return;

    time_t now = time(NULL);

    pthread_mutex_lock(&future->mutex);

    if (future->state == FUTURE_PENDING) {
        future->state = FUTURE_REJECTED;
        future->result = error;
        future->rejected_at = now;

        // 计算等待时间
        if (future->created_at > 0) {
            future->wait_time_ms = (uint64_t)(now - future->created_at) * 1000;
        }

        // 唤醒所有等待者
        future_wake_waiters(future);

        // 更新全局统计
        atomic_fetch_add(&g_total_futures_rejected, 1);
    }

    pthread_mutex_unlock(&future->mutex);
}

// 阻塞等待Future完成（用于同步等待）
void* future_wait(Future* future) {
    if (!future) return NULL;

    time_t start_time = time(NULL);

    pthread_mutex_lock(&future->mutex);

    while (future->state == FUTURE_PENDING) {
        pthread_cond_wait(&future->cond, &future->mutex);
    }

    void* result = NULL;
    if (future->state == FUTURE_RESOLVED) {
        result = future->result;
        future->consumed = true;
    }

    // 记录等待时间（如果还没有记录）
    if (future->wait_time_ms == 0) {
        time_t end_time = time(NULL);
        future->wait_time_ms = (uint64_t)(end_time - start_time) * 1000;
    }

    pthread_mutex_unlock(&future->mutex);

    return result;
}

// 带超时的等待
void* future_wait_timeout(Future* future, uint32_t timeout_ms) {
    if (!future) return NULL;

    struct timespec timeout_time;
    clock_gettime(CLOCK_REALTIME, &timeout_time);
    timeout_time.tv_sec += timeout_ms / 1000;
    timeout_time.tv_nsec += (timeout_ms % 1000) * 1000000;

    time_t start_time = time(NULL);

    pthread_mutex_lock(&future->mutex);

    int wait_result = 0;
    while (future->state == FUTURE_PENDING && wait_result == 0) {
        wait_result = pthread_cond_timedwait(&future->cond, &future->mutex, &timeout_time);
    }

    void* result = NULL;
    if (future->state == FUTURE_RESOLVED) {
        result = future->result;
        future->consumed = true;
    }

    // 记录等待时间
    if (future->wait_time_ms == 0) {
        time_t end_time = time(NULL);
        future->wait_time_ms = (uint64_t)(end_time - start_time) * 1000;
    }

    pthread_mutex_unlock(&future->mutex);

    return result;
}

