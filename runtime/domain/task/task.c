#include "task.h"
// #include "../async/entity/future.h" // 暂时禁用future支持
#include "../coroutine/coroutine.h"
#include <stdlib.h>
#include <stdio.h>
#include <string.h>
#include <unistd.h>

// 全局任务ID计数器
static uint64_t next_task_id = 1;
static pthread_mutex_t task_id_mutex = PTHREAD_MUTEX_INITIALIZER;

// 生成任务ID
uint64_t task_generate_id(void) {
    pthread_mutex_lock(&task_id_mutex);
    uint64_t id = next_task_id++;
    pthread_mutex_unlock(&task_id_mutex);
    return id;
}

// 创建新任务
Task* task_create(void (*entry_point)(void*), void* arg) {
    Task* task = (Task*)malloc(sizeof(Task));
    if (!task) {
        fprintf(stderr, "ERROR: Failed to allocate memory for task\n");
        return NULL;
    }

    memset(task, 0, sizeof(Task));

    // 初始化任务属性
    task->id = task_generate_id();
    task->status = TASK_READY;
    task->entry_point = entry_point;  // 设置入口点函数
    task->arg = arg;                  // 设置参数
    // task->future = NULL; // 暂时禁用future支持
    task->coroutine = NULL;
    task->result = NULL;
    task->stack = NULL;
    task->stack_size = 0;
    task->next = NULL;
    task->on_complete = NULL;
    task->user_data = NULL;

    // 初始化同步机制
    pthread_mutex_init(&task->mutex, NULL);
    pthread_cond_init(&task->cond, NULL);

    return task;
}

// 销毁任务
void task_destroy(Task* task) {
    if (!task) return;

    // 清理关联的协程
    if (task->coroutine) {
        coroutine_destroy(task->coroutine);
        task->coroutine = NULL;
    }

    // 清理关联的Future
    // if (task->future) {
    //     future_destroy(task->future);
    //     task->future = NULL; // 暂时禁用future支持
    // } // 暂时禁用future支持

    // 清理栈内存
    if (task->stack) {
        free(task->stack);
        task->stack = NULL;
    }

    // 清理同步机制
    pthread_mutex_destroy(&task->mutex);
    pthread_cond_destroy(&task->cond);

    printf("DEBUG: Destroyed task %llu\n", task->id);
    free(task);
}

// 执行任务
void task_execute(Task* task) {
    if (!task) return;

    pthread_mutex_lock(&task->mutex);

    if (task->status == TASK_READY) {
        task->status = TASK_RUNNING;
        printf("DEBUG: Starting execution of task %llu\n", task->id);

        // 如果有关联的协程，使用协程执行
        if (task->coroutine) {
            printf("DEBUG: Resuming coroutine for task %llu\n", task->id);
            coroutine_resume(task->coroutine);
        } else {
            // 直接执行函数（真正的并发执行）
            if (task->entry_point) {
                task->entry_point(task->arg);
                printf("DEBUG: Task %llu executed function\n", task->id);
            }

            // 任务执行完成，标记为完成并解决Future
            task->status = TASK_COMPLETED;

            // 如果有Future，解决它（spawn任务通常没有返回值，所以传递NULL）
            // if (task->future) {
            //     future_resolve(task->future, NULL);
            //     printf("DEBUG: Task %llu resolved its future %llu\n", task->id, task->future->id);
            // } // 暂时禁用future支持
        }

        // 调用完成回调
        if (task->status == TASK_COMPLETED && task->on_complete) {
            task->on_complete(task);
        }
    }

    pthread_mutex_unlock(&task->mutex);
}

// 检查任务是否完成
bool task_is_complete(Task* task) {
    if (!task) return true;

    pthread_mutex_lock(&task->mutex);
    bool complete = (task->status == TASK_COMPLETED || task->status == TASK_CANCELLED);
    pthread_mutex_unlock(&task->mutex);

    return complete;
}

// 等待任务完成
void task_wait(Task* task) {
    if (!task) return;

    pthread_mutex_lock(&task->mutex);
    while (!task_is_complete(task)) {
        pthread_cond_wait(&task->cond, &task->mutex);
    }
    pthread_mutex_unlock(&task->mutex);
}

// 设置任务的Future
// void task_set_future(Task* task, Future* future) {
//     if (!task || !future) return;

//     pthread_mutex_lock(&task->mutex);
//     task->future = future;
//     task->status = TASK_WAITING;
//     printf("DEBUG: Task %llu now waiting on future %llu\n", task->id, future->id);
//     pthread_mutex_unlock(&task->mutex);
// } // 暂时禁用future支持

// 唤醒任务
void task_wake(Task* task) {
    if (!task) return;

    pthread_mutex_lock(&task->mutex);

    if (task->status == TASK_WAITING) {
        task->status = TASK_READY;
        printf("DEBUG: Woke up task %llu\n", task->id);
        pthread_cond_signal(&task->cond);
    }

    pthread_mutex_unlock(&task->mutex);
}
