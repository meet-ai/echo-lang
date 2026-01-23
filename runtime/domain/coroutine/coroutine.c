#include "coroutine.h"
#include "context.h"
#include "../../shared/types/common_types.h"
#include "../scheduler/scheduler.h"
#include <stdlib.h>
#include <stdio.h>
#include <string.h>
#include <stdatomic.h>
#include <pthread.h>
#include <time.h>
#include <errno.h>

// 全局协程ID计数器（原子操作）
static atomic_uint_fast64_t g_next_coroutine_id = ATOMIC_VAR_INIT(1);

// 当前正在执行的协程（线程本地存储）
__thread Coroutine* current_coroutine = NULL;

// 全局协程统计
static atomic_uint_fast64_t g_total_coroutines_created = ATOMIC_VAR_INIT(0);
static atomic_uint_fast64_t g_total_coroutines_completed = ATOMIC_VAR_INIT(0);

// 生成协程ID
uint64_t coroutine_generate_id(void) {
    return atomic_fetch_add(&g_next_coroutine_id, 1);
}

// 更新协程统计
static void update_coroutine_stats(Coroutine* coroutine, bool is_resume) {
    time_t now = time(NULL);

    if (is_resume) {
        coroutine->stats.resume_count++;
        coroutine->last_resume_at = now;
    } else {
        coroutine->stats.yield_count++;
        coroutine->last_yield_at = now;
    }
}

// 状态转换验证矩阵
static bool is_valid_state_transition(CoroutineState from, CoroutineState to) {
    switch (from) {
        case COROUTINE_NEW:
            return to == COROUTINE_READY;
        case COROUTINE_READY:
            return to == COROUTINE_RUNNING || to == COROUTINE_CANCELLED;
        case COROUTINE_RUNNING:
            return to == COROUTINE_SUSPENDED || to == COROUTINE_COMPLETED ||
                   to == COROUTINE_FAILED || to == COROUTINE_CANCELLED;
        case COROUTINE_SUSPENDED:
            return to == COROUTINE_RUNNING || to == COROUTINE_CANCELLED;
        case COROUTINE_COMPLETED:
        case COROUTINE_FAILED:
        case COROUTINE_CANCELLED:
            return false; // 终止状态不能转换
        default:
            return false;
    }
}

// 安全的协程状态转换
static bool coroutine_change_state(Coroutine* coroutine, CoroutineState new_state) {
    if (!coroutine) return false;

    if (!is_valid_state_transition(coroutine->state, new_state)) {
        fprintf(stderr, "ERROR: Invalid state transition from %d to %d for coroutine %llu\n",
                coroutine->state, new_state, coroutine->id);
        return false;
    }

    CoroutineState old_state = coroutine->state;
    coroutine->state = new_state;

    // 记录状态变更时间
    time_t now = time(NULL);
    if (new_state == COROUTINE_COMPLETED || new_state == COROUTINE_FAILED ||
        new_state == COROUTINE_CANCELLED) {
        coroutine->completed_at = now;
    } else if (new_state == COROUTINE_RUNNING) {
        coroutine->started_at = now;
    }

    printf("DEBUG: Coroutine %llu state changed from %d to %d\n",
           coroutine->id, old_state, new_state);

    return true;
}

// 计算协程执行时间
static uint64_t calculate_coroutine_execution_time(const Coroutine* coroutine) {
    if (coroutine->started_at == 0 || coroutine->completed_at == 0) {
        return 0;
    }
    return (uint64_t)(coroutine->completed_at - coroutine->started_at) * 1000000;
}

// 协程包装函数
void coroutine_wrapper(Coroutine* coroutine) {
    // 设置当前协程
    current_coroutine = coroutine;
    coroutine->started_at = time(NULL);

    // 执行协程入口函数
    if (coroutine->entry_point) {
        coroutine->entry_point(coroutine->arg);
    }

    // 协程执行完成
    coroutine->completed_at = time(NULL);
    coroutine->state = COROUTINE_COMPLETED;

    // 更新全局统计
    atomic_fetch_add(&g_total_coroutines_completed, 1);

    // 通知调度器协程已完成
    extern Scheduler* get_global_scheduler(void);
    extern void scheduler_notify_coroutine_completed(Scheduler* scheduler, Coroutine* coroutine);
    Scheduler* scheduler = get_global_scheduler();
    if (scheduler) {
        scheduler_notify_coroutine_completed(scheduler, coroutine);
    }

    // 调用完成回调（如果已设置）
    if (coroutine->on_complete) {
        coroutine->on_complete(coroutine);
    }
}

// 创建协程
Coroutine* coroutine_create(const char* name, void (*entry_point)(void*), void* arg, size_t stack_size) {
    if (!name || !entry_point) {
        return NULL;
    }

    // 验证参数
    if (strlen(name) == 0) {
        return NULL; // 名称不能为空
    }

    if (stack_size == 0) {
        stack_size = 64 * 1024; // 默认64KB栈大小
    } else if (stack_size < 4096) {
        return NULL; // 最小栈大小4KB
    }

    Coroutine* coroutine = (Coroutine*)malloc(sizeof(Coroutine));
    if (!coroutine) {
        return NULL;
    }

    memset(coroutine, 0, sizeof(Coroutine));

    // 初始化协程属性
    coroutine->id = coroutine_generate_id();
    strncpy(coroutine->name, name, sizeof(coroutine->name) - 1);
    coroutine->name[sizeof(coroutine->name) - 1] = '\0';

    coroutine->state = COROUTINE_NEW;
    coroutine->priority = COROUTINE_PRIORITY_NORMAL;
    coroutine->entry_point = entry_point;
    coroutine->arg = arg;
    coroutine->stack_size = stack_size;
    coroutine->stack_used = 0;

    // 初始化时间戳
    coroutine->created_at = time(NULL);
    coroutine->started_at = 0;
    coroutine->completed_at = 0;
    coroutine->last_yield_at = 0;
    coroutine->last_resume_at = 0;

    // 初始化调度相关
    coroutine->bound_processor = NULL;
    coroutine->task = NULL;
    coroutine->next = NULL;
    coroutine->schedule_count = 0;

    // 初始化统计信息
    memset(&coroutine->stats, 0, sizeof(CoroutineStats));

    // 分配栈内存
    coroutine->stack = (char*)malloc(stack_size);
    if (!coroutine->stack) {
        free(coroutine);
        return NULL;
    }

    // 创建上下文
    coroutine->context = context_create(stack_size);
    if (!coroutine->context) {
        free(coroutine->stack);
        free(coroutine);
        return NULL;
    }

    // 初始化上下文
    context_init(coroutine->context, (void (*)(void*))coroutine_wrapper, coroutine);

    // 初始化同步机制
    pthread_mutex_init(&coroutine->mutex, NULL);
    pthread_cond_init(&coroutine->condition, NULL);

    // 更新全局统计
    atomic_fetch_add(&g_total_coroutines_created, 1);

    return coroutine;
}

// 销毁协程
void coroutine_destroy(Coroutine* coroutine) {
    if (!coroutine) return;

    // 等待协程完成（如果正在运行）
    if (coroutine->state == COROUTINE_RUNNING) {
        coroutine_join(coroutine, 5000); // 等待最多5秒
    }

    // 清理栈内存
    if (coroutine->stack) {
        free(coroutine->stack);
        coroutine->stack = NULL;
    }

    // 清理上下文
    if (coroutine->context) {
        context_destroy(coroutine->context);
        coroutine->context = NULL;
    }

    // 清理结果数据
    if (coroutine->result_data) {
        free(coroutine->result_data);
        coroutine->result_data = NULL;
    }

    // 清理同步机制
    pthread_mutex_destroy(&coroutine->mutex);
    pthread_cond_destroy(&coroutine->condition);

    free(coroutine);
}

// 启动协程
bool coroutine_start(Coroutine* coroutine) {
    if (!coroutine) return false;

    pthread_mutex_lock(&coroutine->mutex);

    if (!coroutine_change_state(coroutine, COROUTINE_READY)) {
        pthread_mutex_unlock(&coroutine->mutex);
        return false; // 状态转换失败
    }

    coroutine->schedule_count++;

    pthread_mutex_unlock(&coroutine->mutex);

    return true;
}

// 恢复协程执行
void coroutine_resume(Coroutine* coroutine) {
    if (!coroutine) return;

    pthread_mutex_lock(&coroutine->mutex);

    if (coroutine->state == COROUTINE_READY || coroutine->state == COROUTINE_SUSPENDED) {
        Coroutine* previous_coroutine = current_coroutine;
        current_coroutine = coroutine;

        if (!coroutine_change_state(coroutine, COROUTINE_RUNNING)) {
            pthread_mutex_unlock(&coroutine->mutex);
            return; // 状态转换失败
        }

        update_coroutine_stats(coroutine, true);

        // 上下文切换到协程
        context_switch(previous_coroutine->context, coroutine->context);

        // 从协程返回后恢复状态
        if (coroutine->state == COROUTINE_RUNNING) {
            coroutine_change_state(coroutine, COROUTINE_SUSPENDED); // 默认挂起
        }
    }

    pthread_mutex_unlock(&coroutine->mutex);
}

// 挂起协程执行
void coroutine_suspend(Coroutine* coroutine) {
    if (!coroutine) return;

    pthread_mutex_lock(&coroutine->mutex);

    if (coroutine->state == COROUTINE_RUNNING) {
        if (!coroutine_change_state(coroutine, COROUTINE_SUSPENDED)) {
            pthread_mutex_unlock(&coroutine->mutex);
            return; // 状态转换失败
        }
        update_coroutine_stats(coroutine, false);

        // 通知等待的线程
        pthread_cond_broadcast(&coroutine->condition);
    }

    pthread_mutex_unlock(&coroutine->mutex);
}

// 等待协程完成
bool coroutine_join(Coroutine* coroutine, uint32_t timeout_ms) {
    if (!coroutine) return false;

    struct timespec timeout_time;
    if (timeout_ms > 0) {
        clock_gettime(CLOCK_REALTIME, &timeout_time);
        timeout_time.tv_sec += timeout_ms / 1000;
        timeout_time.tv_nsec += (timeout_ms % 1000) * 1000000;
        if (timeout_time.tv_nsec >= 1000000000) {
            timeout_time.tv_sec++;
            timeout_time.tv_nsec -= 1000000000;
        }
    }

    pthread_mutex_lock(&coroutine->mutex);

    while (!coroutine_is_complete(coroutine)) {
        if (timeout_ms == 0) {
            // 无限等待
            pthread_cond_wait(&coroutine->condition, &coroutine->mutex);
        } else {
            // 带超时等待
            int result = pthread_cond_timedwait(&coroutine->condition,
                                              &coroutine->mutex,
                                              &timeout_time);
            if (result == ETIMEDOUT) {
                pthread_mutex_unlock(&coroutine->mutex);
                return false; // 超时
            }
        }
    }

    pthread_mutex_unlock(&coroutine->mutex);
    return true;
}

// 取消协程
bool coroutine_cancel(Coroutine* coroutine, const char* reason) {
    if (!coroutine) return false;

    pthread_mutex_lock(&coroutine->mutex);

    if (coroutine->state == COROUTINE_COMPLETED ||
        coroutine->state == COROUTINE_FAILED ||
        coroutine->state == COROUTINE_CANCELLED) {
        pthread_mutex_unlock(&coroutine->mutex);
        return false; // 已经完成
    }

    if (!coroutine_change_state(coroutine, COROUTINE_CANCELLED)) {
        pthread_mutex_unlock(&coroutine->mutex);
        return false; // 状态转换失败
    }

    if (reason) {
        coroutine_set_error(coroutine, reason);
    } else {
        coroutine_set_error(coroutine, "Coroutine cancelled");
    }

    // 通知等待的线程
    pthread_cond_broadcast(&coroutine->condition);

    pthread_mutex_unlock(&coroutine->mutex);

    return true;
}

// 协程yield函数（全局函数）
bool coroutine_yield(void) {
    if (!current_coroutine) {
        return false;
    }

    coroutine_suspend(current_coroutine);

    // 这里应该切换回调度器上下文
    // 暂时返回true
    return true;
}

// 检查协程是否完成
bool coroutine_is_complete(Coroutine* coroutine) {
    if (!coroutine) return true;

    pthread_mutex_lock(&coroutine->mutex);
    bool complete = (coroutine->state == COROUTINE_COMPLETED ||
                    coroutine->state == COROUTINE_FAILED ||
                    coroutine->state == COROUTINE_CANCELLED);
    pthread_mutex_unlock(&coroutine->mutex);

    return complete;
}

// 协程状态查询函数
CoroutineState coroutine_get_state(const Coroutine* coroutine) {
    return coroutine ? coroutine->state : COROUTINE_NEW;
}

bool coroutine_is_running(const Coroutine* coroutine) {
    return coroutine && coroutine->state == COROUTINE_RUNNING;
}

bool coroutine_is_cancelled(const Coroutine* coroutine) {
    return coroutine && coroutine->state == COROUTINE_CANCELLED;
}

// 协程属性管理函数
bool coroutine_set_name(Coroutine* coroutine, const char* name) {
    if (!coroutine || !name) return false;

    strncpy(coroutine->name, name, sizeof(coroutine->name) - 1);
    coroutine->name[sizeof(coroutine->name) - 1] = '\0';
    return true;
}

bool coroutine_set_description(Coroutine* coroutine, const char* description) {
    if (!coroutine) return false;

    if (description) {
        strncpy(coroutine->description, description, sizeof(coroutine->description) - 1);
        coroutine->description[sizeof(coroutine->description) - 1] = '\0';
    } else {
        coroutine->description[0] = '\0';
    }
    return true;
}

bool coroutine_set_priority(Coroutine* coroutine, CoroutinePriority priority) {
    if (!coroutine) return false;

    coroutine->priority = priority;
    return true;
}

bool coroutine_set_callbacks(Coroutine* coroutine,
                           void (*on_complete)(Coroutine*),
                           void (*on_error)(Coroutine*, const char*)) {
    if (!coroutine) return false;

    coroutine->on_complete = on_complete;
    coroutine->on_error = on_error;
    return true;
}

// 协程结果管理函数
bool coroutine_set_result(Coroutine* coroutine, const void* data, size_t size) {
    if (!coroutine) return false;

    // 释放旧的结果数据
    if (coroutine->result_data) {
        free(coroutine->result_data);
    }

    if (data && size > 0) {
        coroutine->result_data = malloc(size);
        if (!coroutine->result_data) return false;

        memcpy(coroutine->result_data, data, size);
        coroutine->result_size = size;
    } else {
        coroutine->result_data = NULL;
        coroutine->result_size = 0;
    }

    return true;
}

bool coroutine_get_result(Coroutine* coroutine, void** data, size_t* size) {
    if (!coroutine || !data || !size) return false;

    if (!coroutine_is_complete(coroutine)) {
        return false; // 协程未完成
    }

    *data = coroutine->result_data;
    *size = coroutine->result_size;
    return true;
}

bool coroutine_set_error(Coroutine* coroutine, const char* error_message) {
    if (!coroutine) return false;

    if (error_message) {
        strncpy(coroutine->error_message, error_message, sizeof(coroutine->error_message) - 1);
        coroutine->error_message[sizeof(coroutine->error_message) - 1] = '\0';
    } else {
        coroutine->error_message[0] = '\0';
    }

    return true;
}

const char* coroutine_get_error(const Coroutine* coroutine) {
    return coroutine ? coroutine->error_message : NULL;
}

// 协程统计和监控函数
bool coroutine_get_stats(const Coroutine* coroutine, CoroutineStats* stats) {
    if (!coroutine || !stats) return false;

    *stats = coroutine->stats;
    return true;
}

uint64_t coroutine_get_execution_time_us(const Coroutine* coroutine) {
    if (!coroutine) return 0;
    return calculate_coroutine_execution_time(coroutine);
}

uint64_t coroutine_get_total_time_us(const Coroutine* coroutine) {
    if (!coroutine || coroutine->created_at == 0) return 0;

    time_t end_time = coroutine->completed_at > 0 ? coroutine->completed_at : time(NULL);
    return (uint64_t)(end_time - coroutine->created_at) * 1000000;
}

size_t coroutine_get_stack_usage(const Coroutine* coroutine) {
    return coroutine ? coroutine->stack_used : 0;
}

uint32_t coroutine_get_yield_count(const Coroutine* coroutine) {
    return coroutine ? coroutine->stats.yield_count : 0;
}
