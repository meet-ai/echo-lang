#define _XOPEN_SOURCE 600
#include <stdio.h>
#include <stdlib.h>
#include <stdint.h>
#include <unistd.h>
#include <pthread.h>
#include <sched.h>
#include <string.h>

// 统一并发运行时实现
// 基于新的DDD架构：任务控制块 + 中央调度器 + Future抽象

#include "coroutine_context.h"
#include "scheduler.h"
#include "task.h"
#include "future.h"
#include "coroutine.h"
#include "channel.h"
#include "processor.h"

// spawn协程 - 真正的并发执行
void* coroutine_spawn(void (*entry_func)(void*), int32_t arg_count, void* args, void* future_ptr) {
    printf("DEBUG: coroutine_spawn ENTERED - true concurrent execution\n");
    fflush(stdout);

    // 获取全局调度器
    Scheduler* scheduler = get_global_scheduler();
    if (!scheduler) {
        // 如果没有调度器，创建一个单核调度器
        printf("DEBUG: No scheduler available, creating one\n");
        scheduler = scheduler_create(1);
        if (!scheduler) {
            fprintf(stderr, "ERROR: Failed to create scheduler\n");
            return future_ptr;
        }
    }

    // 创建任务并设置entry_func为目标函数
    task_t* task = task_create(entry_func, future_ptr, 0); // 直接使用entry_func
    if (!task) {
        fprintf(stderr, "ERROR: Failed to create task for spawn\n");
        return future_ptr;
    }

    // 设置任务的执行函数为spawn的目标函数
    task->entry_point = entry_func;
    task->arg = future_ptr; // Future作为参数传递

    // 将任务添加到调度器
    if (!scheduler_add_task(scheduler, task)) {
        fprintf(stderr, "ERROR: Failed to add task to scheduler\n");
        task_destroy(task);
        return future_ptr;
    }

    printf("DEBUG: Spawned task %llu with function %p\n", task->id, entry_func);
    fflush(stdout);

    return future_ptr;
}

// await Future
void* coroutine_await(void* future_ptr) {
    printf("DEBUG: coroutine_await ENTERED with future_ptr=%p\n", future_ptr);
    struct Future* future = (struct Future*)future_ptr;
    if (!future) {
        printf("DEBUG: coroutine_await called with NULL future\n");
        return NULL;
    }

    printf("DEBUG: coroutine_await called for future %llu, state=%d\n",
           future->id, future->state);

    // 首先尝试poll，如果已经完成就直接返回
    printf("DEBUG: coroutine_await polling future %llu\n", future->id);
    fflush(stdout);
    PollResult result = future_poll(future, current_task);
    printf("DEBUG: coroutine_await poll result: status=%d, value=%p\n", result.status, result.value);
    fflush(stdout);
    if (result.status == POLL_READY) {
        printf("DEBUG: Future already resolved, returning value %p\n", result.value);
        fflush(stdout);
        return result.value;
    }

    // Future还在等待中 - 真正的异步await行为
    // 在协程上下文中，我们需要挂起协程，让出控制权
    printf("DEBUG: Future %llu is pending, suspending coroutine to allow other work\n", future->id);

    // 获取当前协程 - 从当前任务中获取
    Coroutine* current_co = NULL;
    if (current_task && current_task->coroutine) {
        current_co = current_task->coroutine;
    }

    if (!current_co) {
        // 如果没有协程上下文，回退到阻塞等待
        printf("DEBUG: No coroutine context, falling back to blocking wait\n");
        return future_wait(future);
    }

    // 将当前协程标记为等待状态
    current_co->state = COROUTINE_SUSPENDED;

    // 将当前协程与Future关联，这样Future完成时可以唤醒协程
    // 注意：这里需要更复杂的逻辑来管理协程唤醒
    // 暂时简化：让出控制权，期望调度器在Future完成时恢复协程

    printf("DEBUG: Yielding coroutine %llu waiting for future %llu\n",
           current_co->id, future->id);

    // 让出控制权，让调度器运行其他任务
    coroutine_suspend(current_co);

    // 当Future完成时，我们会回到这里继续执行
    printf("DEBUG: Coroutine %llu resumed after awaiting future %llu\n",
           current_co->id, future->id);

    // 再次检查Future状态
    result = future_poll(future, current_task);
    if (result.status == POLL_READY) {
        printf("DEBUG: Future %llu completed, returning value %p\n", future->id, result.value);
        return result.value;
    } else {
        // 这不应该发生，因为协程应该只在Future完成时被唤醒
        printf("DEBUG: Future %llu still not ready after resume, this indicates a bug\n", future->id);
        return NULL;
    }
}

// 让出调度权
void scheduler_yield() {
    printf("DEBUG: scheduler_yield ENTERED - using unified scheduler\n");

    // 获取全局调度器
    Scheduler* scheduler = get_global_scheduler();
    if (!scheduler) {
        printf("DEBUG: No scheduler available for yield\n");
        return;
    }

    // 简化实现：执行一次调度循环
    printf("DEBUG: Yielding control to scheduler (simplified)\n");
}

// 运行调度器（启动调度循环）
void run_scheduler() {
    printf("DEBUG: run_scheduler ENTERED - using unified scheduler\n");

    // 获取全局调度器
    Scheduler* scheduler = get_global_scheduler();
    if (!scheduler) {
        // 如果没有调度器，创建一个单核调度器
        printf("DEBUG: No scheduler found, creating single-core scheduler\n");
        scheduler = scheduler_create(1); // 创建单核调度器用于测试
        if (!scheduler) {
            fprintf(stderr, "ERROR: Failed to create scheduler in run_scheduler\n");
            return;
        }
    }

    printf("DEBUG: Starting unified scheduler\n");
    scheduler_run(scheduler);

    printf("DEBUG: Unified scheduler stopped\n");
}

// 主函数入口
void echo_main() {
    printf("DEBUG: echo_main called\n");

    // 检查是否有异步任务需要调度
    Scheduler* scheduler = get_global_scheduler();
    printf("DEBUG: Scheduler pointer: %p\n", (void*)scheduler);

    if (scheduler) {
        printf("DEBUG: Scheduler has %u processors, %u machines, is_running=%d\n",
               scheduler->num_processors, scheduler->num_machines, scheduler->is_running);

        // 检查是否有任务在队列中
        bool has_work = false;

        // 检查全局队列
        if (scheduler->global_queue) {
            has_work = true;
            printf("DEBUG: Found work in global queue\n");
        }

        // 检查所有Processor的本地队列
        if (!has_work) {
            for (uint32_t i = 0; i < scheduler->num_processors; i++) {
                uint32_t queue_size = processor_get_queue_size(scheduler->processors[i]);
                printf("DEBUG: Processor %u queue size: %u\n", i, queue_size);
                if (queue_size > 0) {
                    has_work = true;
                    break;
                }
            }
        }

        if (has_work) {
            printf("DEBUG: Found async work, starting unified scheduler...\n");
            run_scheduler();
        } else {
            printf("DEBUG: No async work found, running synchronously\n");
        }
    } else {
        printf("DEBUG: No scheduler available, running synchronously\n");
    }

    printf("DEBUG: echo_main completed\n");
}

// 通道管理函数

// 创建通道
void* channel_create() {
    printf("DEBUG: channel_create called\n");
    fflush(stdout);
    return channel_create_impl();
}

// 发送数据到通道
void channel_send(void* channel_ptr, void* value) {
    printf("DEBUG: channel_send called for channel %p, value %p\n", channel_ptr, value);
    fflush(stdout);
    channel_send_impl((Channel*)channel_ptr, value);
}

// 从通道接收数据
void* channel_receive(void* channel_ptr) {
    printf("DEBUG: channel_receive called for channel %p\n", channel_ptr);
    fflush(stdout);
    return channel_receive_impl((Channel*)channel_ptr);
}

// select语句支持
int32_t channel_select(void** channels, int32_t* operations, int32_t count) {
    printf("DEBUG: channel_select called with %d cases\n", count);
    fflush(stdout);
    return channel_select_impl(count, (Channel**)channels, operations, NULL);
}

// 关闭通道
void channel_close(void* channel_ptr) {
    printf("DEBUG: channel_close called for channel %p\n", channel_ptr);
    fflush(stdout);
    channel_close_impl((Channel*)channel_ptr);
}

// 销毁通道
void channel_destroy(void* channel_ptr) {
    printf("DEBUG: channel_destroy called for channel %p\n", channel_ptr);
    fflush(stdout);
    channel_destroy_impl((Channel*)channel_ptr);
}

// 打印函数
void print_int(int32_t value) {
    printf("%lld", (long long)value);
    fflush(stdout);
}

void print_string(char* str) {
    printf("%s", str);
    fflush(stdout);
}

// 字符串拼接函数
char* string_concat(char* left, char* right) {
    if (!left && !right) return "";
    if (!left) return right;
    if (!right) return left;

    size_t left_len = strlen(left);
    size_t right_len = strlen(right);
    size_t total_len = left_len + right_len + 1; // +1 for null terminator

    char* result = (char*)malloc(total_len);
    if (!result) {
        fprintf(stderr, "ERROR: Failed to allocate memory for string concatenation\n");
        return "";
    }

    strcpy(result, left);
    strcpy(result + left_len, right);
    result[total_len - 1] = '\0';

    return result;
}

// 整数转字符串函数
char* int_to_string(int32_t value) {
    char* buffer = (char*)malloc(32); // 足够存放int32的字符串
    if (!buffer) {
        fprintf(stderr, "ERROR: Failed to allocate memory for int to string conversion\n");
        return "0";
    }
    sprintf(buffer, "%d", value);
    return buffer;
}