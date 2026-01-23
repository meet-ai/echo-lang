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

#include "context.h"
#include "../scheduler/scheduler.h"
#include "../task/task.h"
#include "../future/future.h"
#include "coroutine.h"
#include "../channel/channel.h"
#include "../scheduler/processor.h"

// 前向声明
void scheduler_yield(void);

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
    Task* task = task_create(entry_func, future_ptr); // 直接使用entry_func
    if (!task) {
        fprintf(stderr, "ERROR: Failed to create task for spawn\n");
        return future_ptr;
    }

    // 设置任务的执行函数为spawn的目标函数
    // task_create已经设置了entry_point和arg

    // 将任务添加到调度器
    if (!scheduler_add_task(scheduler, task)) {
        fprintf(stderr, "ERROR: Failed to add task to scheduler\n");
        task_destroy(task);
        return future_ptr;
    }

    printf("DEBUG: Spawned task %llu with function %p\n", task->id, (void*)entry_func);
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

    // 将当前任务状态设置为等待，并添加到 Future 的等待队列
    // 这样 Future 完成时可以唤醒任务和协程
    pthread_mutex_lock(&current_task->mutex);
    current_task->status = TASK_WAITING;
    current_task->future = future;  // 关联 Future
    pthread_mutex_unlock(&current_task->mutex);

    // 将任务添加到 Future 的等待队列
    // future_poll 会在 POLL_PENDING 时自动添加任务到等待队列
    // 但我们需要确保任务已经被添加
    pthread_mutex_lock(&future->mutex);
    if (future->state == FUTURE_PENDING) {
        // 确保任务在等待队列中
        // future_poll 已经调用了 future_add_waiter，但为了安全，我们再次检查
        // 实际上 future_poll 已经处理了，这里只是确保状态正确
    }
    pthread_mutex_unlock(&future->mutex);

    printf("DEBUG: Yielding coroutine %llu (task %llu) waiting for future %llu\n",
           current_co->id, current_task->id, future->id);

    // 挂起协程，保存当前执行上下文
    // 当 Future 完成时，future_wake_waiters 会唤醒任务并重新加入调度队列
    // 任务被重新调度时，coroutine_resume 会恢复协程执行，从这里继续
    coroutine_suspend(current_co);
    
    // 将任务放回调度队列，让调度器运行其他任务
    scheduler_yield();

    // 当Future完成时，任务会被重新调度，coroutine_resume 会恢复协程执行
    // 我们会从 coroutine_suspend 之后继续执行到这里
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
void scheduler_yield(void) {
    printf("DEBUG: scheduler_yield ENTERED - using unified scheduler\n");

    // 获取全局调度器
    Scheduler* scheduler = get_global_scheduler();
    if (!scheduler) {
        printf("DEBUG: No scheduler available for yield\n");
        return;
    }

    // 获取当前任务
    if (!current_task) {
        printf("DEBUG: No current task to yield\n");
        return;
    }

    printf("DEBUG: Yielding task %llu\n", current_task->id);

    // 如果任务有关联的协程，将协程状态设置为 SUSPENDED
    if (current_task->coroutine) {
        Coroutine* coroutine = current_task->coroutine;
        coroutine->state = COROUTINE_SUSPENDED;
        printf("DEBUG: Set coroutine %llu state to SUSPENDED\n", coroutine->id);
    }

    // 将当前任务放回调度器队列，以便后续重新调度
    // 注意：这里需要将任务放回队列，但当前实现中任务执行是同步的
    // 对于单线程调度器，我们需要将任务标记为可重新调度，然后继续调度循环
    // 由于当前是同步执行模式，我们暂时将任务状态标记，让调度器知道可以重新调度
    
    // 获取当前任务的处理器（如果有）
    Processor* processor = NULL;
    if (current_task->coroutine && current_task->coroutine->bound_processor) {
        processor = current_task->coroutine->bound_processor;
    } else if (scheduler->num_processors > 0) {
        // 如果没有绑定处理器，使用第一个处理器
        processor = scheduler->processors[0];
    }

    // 将任务放回处理器队列
    if (processor) {
        processor_push_local(processor, current_task);
        printf("DEBUG: Pushed task %llu back to processor %u queue\n", 
               current_task->id, processor->id);
    } else {
        // 如果没有处理器，放回全局队列
        pthread_mutex_lock(&scheduler->global_lock);
        current_task->next = scheduler->global_queue;
        scheduler->global_queue = current_task;
        pthread_mutex_unlock(&scheduler->global_lock);
        printf("DEBUG: Pushed task %llu back to global queue\n", current_task->id);
    }

    // 清空当前任务，让调度器选择下一个任务
    current_task = NULL;

    printf("DEBUG: Yield completed, scheduler will pick next task\n");
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