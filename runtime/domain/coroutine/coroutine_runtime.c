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
#include "../scheduler/scheduler.h"  // 包含新的聚合根定义（通过scheduler.h）
#include "../task_scheduling/adapter/scheduler_adapter.h"  // 适配层辅助函数
#include "../task_execution/aggregate/task.h"  // 使用新的Task聚合根
#include "../async_computation/aggregate/future.h"  // 使用新的Future聚合根
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
    // 使用新的Task聚合根工厂方法
    size_t default_stack_size = 64 * 1024; // 默认栈大小64KB
    // 从适配层获取EventBus（通过scheduler_adapter，因为coroutine_runtime使用scheduler）
    extern EventBus* scheduler_adapter_get_event_bus(void);
    EventBus* event_bus = scheduler_adapter_get_event_bus();
    Task* task = task_factory_create(entry_func, args, default_stack_size, event_bus);
    if (!task) {
        fprintf(stderr, "ERROR: Failed to create task for spawn\n");
        return future_ptr;
    }

    // 如果提供了Future，设置任务等待该Future
    if (future_ptr) {
        Future* future = (Future*)future_ptr;
        FutureID future_id = future_get_id(future);
        // TODO: 调用task_wait_for_future(task, future_id)
        // 但由于current_task是旧的Task类型，暂时先不设置
        // 后续需要解决current_task类型问题
    }

    // 将任务添加到调度器
    // TODO: scheduler_add_task需要更新为接受TaskID而不是Task*
    // 暂时保留旧的调用方式
    if (!scheduler_add_task(scheduler, task)) {
        fprintf(stderr, "ERROR: Failed to add task to scheduler\n");
        // TODO: 需要Task聚合根的销毁方法
        // task_destroy(task);
        return future_ptr;
    }

    TaskID task_id = task_get_id(task);
    printf("DEBUG: Spawned task %llu with function %p\n", task_id, (void*)entry_func);
    fflush(stdout);

    return future_ptr;
}

// await Future
// 注意：此函数接收Future*指针，但内部使用FutureID和TaskID与新的聚合根交互
void* coroutine_await(void* future_ptr) {
    printf("DEBUG: coroutine_await ENTERED with future_ptr=%p\n", future_ptr);
    Future* future = (Future*)future_ptr;
    if (!future) {
        printf("DEBUG: coroutine_await called with NULL future\n");
        return NULL;
    }

    // 获取FutureID（使用新的聚合根方法）
    FutureID future_id = future_get_id(future);
    FutureState state = future_get_state(future);
    
    printf("DEBUG: coroutine_await called for future %llu, state=%d\n",
           future_id, state);

    // 获取当前任务的TaskID（使用新的Task聚合根方法）
    TaskID task_id = 0;
    if (current_task) {
        task_id = task_get_id(current_task);  // 使用新的聚合根方法
    }

    // 首先尝试poll，如果已经完成就直接返回
    printf("DEBUG: coroutine_await polling future %llu with task_id %llu\n", future_id, task_id);
    fflush(stdout);
    PollResult result = future_poll(future, task_id);  // 使用新的聚合根方法，传递TaskID
    printf("DEBUG: coroutine_await poll result: status=%d, value=%p\n", result.status, result.value);
    fflush(stdout);
    if (result.status == POLL_READY) {
        printf("DEBUG: Future already resolved, returning value %p\n", result.value);
        fflush(stdout);
        return result.value;
    }

    // Future还在等待中 - 真正的异步await行为
    // 在协程上下文中，我们需要挂起协程，让出控制权
    printf("DEBUG: Future %llu is pending, suspending coroutine to allow other work\n", future_id);

    // 获取当前协程 - 从当前任务中获取（使用新的Task聚合根方法）
    Coroutine* current_co = NULL;
    if (current_task) {
        const struct Coroutine* coroutine = task_get_coroutine(current_task);  // 使用新的聚合根方法
        current_co = (Coroutine*)coroutine;  // 类型转换（因为返回的是const指针，但我们需要修改）
    }

    if (!current_co) {
        // 如果没有协程上下文，回退到阻塞等待
        printf("DEBUG: No coroutine context, falling back to blocking wait\n");
        return future_wait(future);
    }

    // 将当前协程标记为等待状态
    current_co->state = COROUTINE_SUSPENDED;

    // 使用新的Task聚合根方法：task_wait_for_future
    if (current_task) {
        bool success = task_wait_for_future(current_task, future_id);  // 使用新的聚合根方法
        if (!success) {
            printf("DEBUG: Failed to set task waiting for future %llu\n", future_id);
            // 如果设置失败，仍然继续执行（可能是状态转换不允许）
        }
    }

    // future_poll已经调用了future_add_waiter（通过TaskID），所以任务已经在等待队列中
    printf("DEBUG: Yielding coroutine %llu (task %llu) waiting for future %llu\n",
           current_co->id, task_id, future_id);

    // 挂起协程，保存当前执行上下文
    // 当 Future 完成时，future_wake_waiters 会通过TaskRepository查找Task并唤醒
    coroutine_suspend(current_co);
    
    // 将任务放回调度队列，让调度器运行其他任务
    scheduler_yield();

    // 当Future完成时，任务会被重新调度，coroutine_resume 会恢复协程执行
    // 我们会从 coroutine_suspend 之后继续执行到这里
    printf("DEBUG: Coroutine %llu resumed after awaiting future %llu\n",
           current_co->id, future_id);

    // 再次检查Future状态（使用TaskID）
    result = future_poll(future, task_id);
    if (result.status == POLL_READY) {
        printf("DEBUG: Future %llu completed, returning value %p\n", future_id, result.value);
        return result.value;
    } else {
        // 这不应该发生，因为协程应该只在Future完成时被唤醒
        printf("DEBUG: Future %llu still not ready after resume, this indicates a bug\n", future_id);
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

    TaskID task_id = task_get_id(current_task);  // 使用新的聚合根方法
    printf("DEBUG: Yielding task %llu\n", task_id);

    // 如果任务有关联的协程，将协程状态设置为 SUSPENDED
    const struct Coroutine* coroutine = task_get_coroutine(current_task);  // 使用新的聚合根方法
    if (coroutine) {
        Coroutine* mutable_coroutine = (Coroutine*)coroutine;  // 类型转换（需要修改状态）
        mutable_coroutine->state = COROUTINE_SUSPENDED;
        printf("DEBUG: Set coroutine %llu state to SUSPENDED\n", mutable_coroutine->id);
    }

    // 将当前任务放回调度器队列，以便后续重新调度
    // 注意：这里需要将任务放回队列，但当前实现中任务执行是同步的
    // 对于单线程调度器，我们需要将任务标记为可重新调度，然后继续调度循环
    // 由于当前是同步执行模式，我们暂时将任务状态标记，让调度器知道可以重新调度
    
    // 获取当前任务的处理器（如果有）
    Processor* processor = NULL;
    // 重用上面已获取的coroutine变量
    if (coroutine) {
        Coroutine* mutable_coroutine = (Coroutine*)coroutine;  // 类型转换
        if (mutable_coroutine->bound_processor) {
            processor = mutable_coroutine->bound_processor;
        }
    }
    // 使用适配层辅助函数
    uint32_t num_processors = scheduler_get_num_processors(scheduler);
    if (!processor && num_processors > 0) {
        // 如果没有绑定处理器，使用第一个处理器
        // 使用适配层函数获取处理器，避免直接访问内部字段
        processor = scheduler_get_processor_by_index(scheduler, 0);
    }

    // 将任务放回处理器队列或全局队列
    // 注意：processor_push_local仍然接受Task*类型（Processor还未完全重构为TaskID）
    // 但全局队列已更新为使用TaskID，通过适配层函数添加
    if (processor) {
        processor_push_local(processor, current_task);  // 注意：Processor仍使用Task*，后续需要重构
        printf("DEBUG: Pushed task %llu back to processor %u queue\n", 
               task_id, processor->id);
    } else {
        // 如果没有处理器，放回全局队列
        // 使用适配层函数添加到全局队列（内部使用TaskID）
        int result = scheduler_add_task_to_global_queue(scheduler, current_task);
        if (result == 0) {
            printf("DEBUG: Pushed task %llu back to global queue\n", task_id);
        } else {
            fprintf(stderr, "ERROR: Failed to push task %llu back to global queue\n", task_id);
        }
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
        // 如果没有调度器，尝试创建一个单核调度器
        // 注意：对于简单的测试程序（如map迭代测试），可能不需要完整的调度器
        // 如果创建失败，静默返回（不影响程序正常执行）
        printf("DEBUG: No scheduler found, attempting to create single-core scheduler\n");
        scheduler = scheduler_create(1); // 创建单核调度器用于测试
        if (!scheduler) {
            // 对于简单程序，调度器创建失败不影响功能，只记录调试信息
            printf("DEBUG: Scheduler creation skipped (not required for simple programs)\n");
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
        // 使用适配层的辅助函数访问字段
        uint32_t num_processors = scheduler_get_num_processors(scheduler);
        uint32_t num_machines = scheduler_get_num_machines(scheduler);
        bool is_running = scheduler_is_running(scheduler);
        
        printf("DEBUG: Scheduler has %u processors, %u machines, is_running=%d\n",
               num_processors, num_machines, is_running);

        // 检查是否有任务在队列中
        bool has_work = false;

        // 检查全局队列（使用适配层辅助函数）
        if (scheduler_has_work_in_global_queue(scheduler)) {
            has_work = true;
            printf("DEBUG: Found work in global queue\n");
        }

        // 检查所有Processor的本地队列
        // 注意：Processor是内部实体，需要通过聚合根访问
        // 使用适配层函数获取Processor，避免直接访问内部字段
        if (!has_work) {
            // 通过适配层函数访问Processor队列
            for (uint32_t i = 0; i < num_processors; i++) {
                Processor* processor = scheduler_get_processor_by_index(scheduler, i);
                if (processor) {
                    uint32_t queue_size = processor_get_queue_size(processor);
                    printf("DEBUG: Processor %u queue size: %u\n", i, queue_size);
                    if (queue_size > 0) {
                        has_work = true;
                        break;
                    }
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

// 打印函数（与 LLVM 声明一致：print_int i32, print_float double, print_bool i1）
void print_int(int32_t value) {
    printf("%lld", (long long)value);
    fflush(stdout);
}

void print_float(double value) {
    printf("%g", value);
    fflush(stdout);
}

void print_bool(int value) {
    printf("%s", value ? "true" : "false");
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