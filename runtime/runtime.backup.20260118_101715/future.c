#include "future.h"
#include "task.h"
#include <stdlib.h>
#include <stdio.h>
#include <string.h>

// 全局Future ID计数器
static uint64_t next_future_id = 1;
static pthread_mutex_t future_id_mutex = PTHREAD_MUTEX_INITIALIZER;

// 函数声明
void future_add_waiter(Future* future, struct Task* task);
void future_wake_waiters(Future* future);

// 生成Future ID
uint64_t future_generate_id(void) {
    pthread_mutex_lock(&future_id_mutex);
    uint64_t id = next_future_id++;
    pthread_mutex_unlock(&future_id_mutex);
    return id;
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
    future->cancel = NULL;
    future->cleanup = NULL;

    // 初始化同步机制
    pthread_mutex_init(&future->mutex, NULL);
    pthread_cond_init(&future->cond, NULL);

    printf("DEBUG: Created future %llu\n", future->id);
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

    // 清理同步机制
    pthread_mutex_destroy(&future->mutex);
    pthread_cond_destroy(&future->cond);

    printf("DEBUG: Destroyed future %llu\n", future->id);
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

        // 这里应该唤醒任务，但由于我们的任务系统比较简单，
        // 暂时只打印信息，实际的唤醒逻辑在调度器中处理
        printf("DEBUG: Should wake task %llu\n", current->task->id);

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

    // 使用默认的poll实现
    result = future_default_poll(future, task);

    printf("DEBUG: Polled future %llu, state: %d, status: %d, value: %p\n",
           future->id, future->state, result.status, result.value);

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
    printf("DEBUG: future_resolve ENTERED for future %llu with value %p\n", future ? future->id : 0, value);
    fflush(stdout);

    if (!future) {
        printf("DEBUG: future_resolve called with NULL future\n");
        fflush(stdout);
        return;
    }

    printf("DEBUG: Resolving future %llu with value %p\n", future->id, value);
    fflush(stdout);

    pthread_mutex_lock(&future->mutex);

    if (future->state == FUTURE_PENDING) {
        future->state = FUTURE_RESOLVED;
        future->result = value;
        printf("DEBUG: Future %llu state changed to RESOLVED, waking waiters\n", future->id);
        fflush(stdout);
        future_wake_waiters(future);
    } else {
        printf("DEBUG: Future %llu already in state %d, not resolving\n", future->id, future->state);
        fflush(stdout);
    }

    pthread_mutex_unlock(&future->mutex);

    printf("DEBUG: future_resolve COMPLETED for future %llu\n", future->id);
    fflush(stdout);
}

// 拒绝Future（设置错误结果）
void future_reject(Future* future, void* error) {
    if (!future) return;

    printf("DEBUG: Rejecting future %llu with error %p\n", future->id, error);

    pthread_mutex_lock(&future->mutex);

    if (future->state == FUTURE_PENDING) {
        future->state = FUTURE_REJECTED;
        future->result = error;
        future_wake_waiters(future);
    }

    pthread_mutex_unlock(&future->mutex);
}

// 阻塞等待Future完成（用于同步等待）
void* future_wait(Future* future) {
    if (!future) return NULL;

    printf("DEBUG: Waiting for future %llu\n", future->id);

    pthread_mutex_lock(&future->mutex);

    while (future->state == FUTURE_PENDING) {
        printf("DEBUG: Future %llu still pending, blocking...\n", future->id);
        pthread_cond_wait(&future->cond, &future->mutex);
    }

    void* result = NULL;
    if (future->state == FUTURE_RESOLVED) {
        result = future->result;
        future->consumed = true;
    }

    printf("DEBUG: Future %llu completed, result: %p\n", future->id, result);
    pthread_mutex_unlock(&future->mutex);

    return result;
}

