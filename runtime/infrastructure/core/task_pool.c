#include "task_pool.h"
#include "../../shared/types/common_types.h"
#include <stdlib.h>
#include <stdio.h>
#include <string.h>
#include <pthread.h>
#include <unistd.h>
#include <stdatomic.h>

// 内部结构体定义
struct TaskPoolWorker {
    uint64_t worker_id;          // 工作线程ID
    char worker_name[128];       // 工作线程名称
    char status[32];             // 工作线程状态
    time_t created_at;           // 创建时间
    time_t last_active_at;       // 最后活跃时间
    uint64_t total_tasks_executed; // 总执行任务数
    uint64_t current_task_id;    // 当前任务ID
    uint64_t execution_time_ms;  // 总执行时间
    double cpu_usage;            // CPU使用率
    uint64_t memory_usage;       // 内存使用量
    pthread_t thread;            // 线程句柄
    bool is_active;              // 是否激活
    void* user_data;             // 用户数据
};

typedef struct TaskPoolImpl {
    TaskPoolConfig config;              // 配置
    struct TaskPoolWorker** workers;    // 工作线程数组
    uint32_t active_workers;            // 活跃工作线程数
    uint32_t total_workers;             // 总工作线程数

    // 任务队列
    queue_t task_queue;                 // 任务队列
    pthread_mutex_t queue_mutex;        // 队列互斥锁
    pthread_cond_t queue_condition;     // 队列条件变量

    // 统计信息
    struct TaskPoolStats stats;         // 统计信息

    // 控制变量
    bool is_running;                    // 是否正在运行
    bool shutdown_requested;            // 是否请求关闭
    atomic_uint_fast64_t next_task_id;  // 下一个任务ID
    atomic_uint_fast64_t next_worker_id; // 下一个工作线程ID

    // 自动扩缩容
    time_t last_scale_check;            // 最后扩缩容检查时间
    uint32_t scale_check_interval;      // 扩缩容检查间隔

    // 同步
    pthread_mutex_t pool_mutex;         // 池互斥锁
} TaskPoolImpl;

// 任务状态定义
#define TASK_STATUS_PENDING    "pending"
#define TASK_STATUS_RUNNING    "running"
#define TASK_STATUS_COMPLETED  "completed"
#define TASK_STATUS_FAILED     "failed"
#define TASK_STATUS_CANCELLED  "cancelled"

// 工作线程状态定义
#define WORKER_STATUS_IDLE     "idle"
#define WORKER_STATUS_BUSY     "busy"
#define WORKER_STATUS_STOPPED  "stopped"

// 工作线程函数
static void* worker_thread_function(void* arg) {
    struct TaskPoolWorker* worker = (struct TaskPoolWorker*)arg;
    TaskPoolImpl* pool = (TaskPoolImpl*)worker->user_data;

    while (worker->is_active && !pool->shutdown_requested) {
        struct PooledTask* task = NULL;

        // 从队列获取任务
        pthread_mutex_lock(&pool->queue_mutex);

        while (pool->is_running && !pool->shutdown_requested &&
               queue_is_empty(&pool->task_queue)) {
            // 设置为空闲状态
            strcpy(worker->status, WORKER_STATUS_IDLE);
            worker->last_active_at = time(NULL);

            // 等待任务或超时
            struct timespec timeout;
            clock_gettime(CLOCK_REALTIME, &timeout);
            timeout.tv_sec += pool->config.worker_idle_timeout_seconds;

            int result = pthread_cond_timedwait(&pool->queue_condition,
                                              &pool->queue_mutex,
                                              &timeout);

            if (result == ETIMEDOUT) {
                // 空闲超时，检查是否需要退出
                if (pool->active_workers > pool->config.min_workers) {
                    break; // 退出线程
                }
            }
        }

        if (!queue_is_empty(&pool->task_queue)) {
            task = (struct PooledTask*)queue_dequeue(&pool->task_queue);
        }

        pthread_mutex_unlock(&pool->queue_mutex);

        if (task) {
            // 执行任务
            execute_pooled_task(worker, task);
        } else if (pool->shutdown_requested ||
                  (pool->active_workers > pool->config.min_workers &&
                   worker->last_active_at + pool->config.worker_idle_timeout_seconds < time(NULL))) {
            // 退出线程
            break;
        }
    }

    // 线程退出
    strcpy(worker->status, WORKER_STATUS_STOPPED);
    return NULL;
}

// 执行池化任务
static void execute_pooled_task(struct TaskPoolWorker* worker, struct PooledTask* task) {
    // 设置为忙碌状态
    strcpy(worker->status, WORKER_STATUS_BUSY);
    worker->current_task_id = task->task_id;
    task->started_at = time(NULL);
    strcpy(task->status, TASK_STATUS_RUNNING);

    // 执行任务函数
    void (*task_func)(void*) = (void (*)(void*))task->task_function;
    if (task_func) {
        task_func(task->task_arg);
        task->exit_code = 0;
        strcpy(task->status, TASK_STATUS_COMPLETED);
    } else {
        task->exit_code = -1;
        strcpy(task->error_message, "Invalid task function");
        strcpy(task->status, TASK_STATUS_FAILED);
    }

    task->completed_at = time(NULL);
    task->execution_time_ms = (uint32_t)(task->completed_at - task->started_at) * 1000;

    // 更新工作线程统计
    worker->total_tasks_executed++;
    worker->execution_time_ms += task->execution_time_ms;
    worker->last_active_at = time(NULL);
    worker->current_task_id = 0;

    // 更新池统计
    TaskPoolImpl* pool = (TaskPoolImpl*)worker->user_data;
    pthread_mutex_lock(&pool->pool_mutex);
    pool->stats.total_tasks_completed++;
    pthread_mutex_unlock(&pool->pool_mutex);

    // 清理任务（这里应该有更好的资源管理）
    // 暂时不释放，等待上层处理
}

// 创建任务池
TaskPool* task_pool_create(const struct TaskPoolConfig* config) {
    if (!config) return NULL;

    TaskPoolImpl* pool = (TaskPoolImpl*)malloc(sizeof(TaskPoolImpl));
    if (!pool) return NULL;

    memset(pool, 0, sizeof(TaskPoolImpl));
    pool->config = *config;

    // 初始化队列
    queue_init(&pool->task_queue);
    pthread_mutex_init(&pool->queue_mutex, NULL);
    pthread_cond_init(&pool->queue_condition, NULL);
    pthread_mutex_init(&pool->pool_mutex, NULL);

    // 初始化统计
    memset(&pool->stats, 0, sizeof(struct TaskPoolStats));
    pool->stats.created_at = time(NULL);

    // 初始化控制变量
    pool->is_running = false;
    pool->shutdown_requested = false;
    atomic_init(&pool->next_task_id, 1);
    atomic_init(&pool->next_worker_id, 1);

    // 分配工作线程数组
    pool->workers = (struct TaskPoolWorker**)malloc(sizeof(struct TaskPoolWorker*) * config->max_workers);
    if (!pool->workers) {
        task_pool_destroy((TaskPool*)pool);
        return NULL;
    }
    memset(pool->workers, 0, sizeof(struct TaskPoolWorker*) * config->max_workers);

    return (TaskPool*)pool;
}

// 销毁任务池
void task_pool_destroy(TaskPool* pool) {
    if (!pool) return;

    TaskPoolImpl* impl = (TaskPoolImpl*)pool;

    // 请求关闭
    impl->shutdown_requested = true;

    // 停止所有工作线程
    for (uint32_t i = 0; i < impl->total_workers; i++) {
        if (impl->workers[i] && impl->workers[i]->is_active) {
            impl->workers[i]->is_active = false;
            pthread_cond_broadcast(&impl->queue_condition);
            pthread_join(impl->workers[i]->thread, NULL);
            free(impl->workers[i]);
        }
    }

    // 清理资源
    free(impl->workers);
    pthread_mutex_destroy(&impl->queue_mutex);
    pthread_cond_destroy(&impl->queue_condition);
    pthread_mutex_destroy(&impl->pool_mutex);

    // 清理队列中的任务
    while (!queue_is_empty(&impl->task_queue)) {
        struct PooledTask* task = (struct PooledTask*)queue_dequeue(&impl->task_queue);
        if (task) {
            // 这里应该释放任务资源
            free(task);
        }
    }

    free(impl);
}

// 便捷构造函数
struct TaskPoolConfig* task_pool_config_create(void) {
    struct TaskPoolConfig* config = (struct TaskPoolConfig*)malloc(sizeof(struct TaskPoolConfig));
    if (config) {
        memset(config, 0, sizeof(struct TaskPoolConfig));
    }
    return config;
}

void task_pool_config_destroy(struct TaskPoolConfig* config) {
    free(config);
}

struct TaskPoolConfig* task_pool_config_create_default(void) {
    struct TaskPoolConfig* config = task_pool_config_create();
    if (config) {
        config->min_workers = 2;
        config->max_workers = 16;
        config->initial_workers = 4;
        config->task_queue_size = 1024;
        config->worker_idle_timeout_seconds = 60;
        config->task_timeout_seconds = 300;
        config->auto_scaling_enabled = true;
        config->scale_up_threshold = 100;
        config->scale_down_threshold = 2;
        strcpy(config->scheduling_policy, "fifo");
        config->enable_task_stats = true;
        config->enable_worker_stats = true;
    }
    return config;
}

// 其他API实现（简化版本）
bool task_pool_submit_task_function(TaskPool* pool, const char* task_name,
                                   void* task_function, void* task_arg, uint32_t priority) {
    if (!pool || !task_function) return false;

    TaskPoolImpl* impl = (TaskPoolImpl*)pool;

    struct PooledTask* task = (struct PooledTask*)malloc(sizeof(struct PooledTask));
    if (!task) return false;

    memset(task, 0, sizeof(struct PooledTask));
    task->task_id = atomic_fetch_add(&impl->next_task_id, 1);
    if (task_name) {
        strncpy(task->task_name, task_name, sizeof(task->task_name) - 1);
    }
    task->task_function = task_function;
    task->task_arg = task_arg;
    task->priority = priority;
    task->submitted_at = time(NULL);
    strcpy(task->status, TASK_STATUS_PENDING);

    // 提交到队列
    pthread_mutex_lock(&impl->queue_mutex);
    queue_enqueue(&impl->task_queue, task);
    impl->stats.total_tasks_submitted++;
    pthread_cond_signal(&impl->queue_condition);
    pthread_mutex_unlock(&impl->queue_mutex);

    return true;
}

bool task_pool_get_stats(const TaskPool* pool, struct TaskPoolStats* stats) {
    if (!pool || !stats) return false;

    const TaskPoolImpl* impl = (TaskPoolImpl*)pool;
    *stats = impl->stats;
    return true;
}

bool task_pool_shutdown(TaskPool* pool, uint32_t timeout_ms) {
    if (!pool) return false;

    TaskPoolImpl* impl = (TaskPoolImpl*)pool;

    impl->shutdown_requested = true;

    // 等待工作线程退出
    for (uint32_t i = 0; i < impl->total_workers; i++) {
        if (impl->workers[i] && impl->workers[i]->is_active) {
            pthread_cond_broadcast(&impl->queue_condition);
            // 这里应该有超时等待
        }
    }

    return true;
}
