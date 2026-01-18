/**
 * @file runtime.c
 * @brief Runtime 组装服务实现
 *
 * Runtime 负责组装Executor和Reactor，提供统一的异步运行时入口。
 */

#include "runtime.h"
#include "../domain/executor/executor.h"
#include "../domain/reactor/reactor.h"
#include "../domain/reactor/timer_future.h"
#include <stdlib.h>
#include <stdio.h>
#include <pthread.h>

// ============================================================================
// Runtime 组装服务实现
// ============================================================================

/**
 * @brief 创建Runtime实例
 * 核心业务逻辑：组装异步运行时的各个组件
 */
runtime_t* runtime_create(void) {
    runtime_t* runtime = (runtime_t*)malloc(sizeof(runtime_t));
    if (!runtime) {
        fprintf(stderr, "Failed to allocate Runtime\n");
        return NULL;
    }

    runtime->initialized = false;

    // 创建基础组件
    // 简化实现：创建临时的事件循环
    struct EventLoop* event_loop = NULL; // 暂时为NULL，由Reactor创建

    // 创建Reactor
    runtime->reactor = reactor_create(event_loop);
    if (!runtime->reactor) {
        fprintf(stderr, "Failed to create Reactor\n");
        free(runtime);
        return NULL;
    }

    // 设置TimerFuture的Reactor引用
    timer_future_set_reactor(runtime->reactor);

    // 创建Executor
    runtime->executor = executor_create(NULL, runtime->reactor); // TODO: 创建真正的GMP调度器
    if (!runtime->executor) {
        fprintf(stderr, "Failed to create Executor\n");
        reactor_destroy(runtime->reactor);
        free(runtime);
        return NULL;
    }

    runtime->initialized = true;
    printf("DEBUG: Created Runtime with Executor and Reactor\n");
    return runtime;
}

/**
 * @brief 销毁Runtime实例
 */
void runtime_destroy(runtime_t* runtime) {
    if (!runtime) return;

    runtime->initialized = false;

    if (runtime->executor) {
        executor_destroy(runtime->executor);
        runtime->executor = NULL;
    }

    if (runtime->reactor) {
        reactor_destroy(runtime->reactor);
        runtime->reactor = NULL;
    }

    printf("DEBUG: Destroyed Runtime\n");
    free(runtime);
}

/**
 * @brief 运行主任务
 * 核心业务逻辑：实现异步闭环的启动和运行
 */
int runtime_run(runtime_t* runtime, task_t* main_task) {
    if (!runtime || !runtime->initialized || !main_task) {
        fprintf(stderr, "Invalid Runtime or main task\n");
        return -1;
    }

    printf("DEBUG: Starting Runtime with main task %llu\n", task_get_id(main_task));

    // 生成主任务
    executor_spawn_task(runtime->executor, main_task);

    // 启动执行器（在当前线程中运行）
    // 简化实现：直接调用执行循环
    executor_run_loop(runtime->executor);

    printf("DEBUG: Runtime execution completed\n");
    return 0;
}

/**
 * @brief 停止Runtime
 */
void runtime_shutdown(runtime_t* runtime) {
    if (runtime && runtime->initialized) {
        printf("DEBUG: Shutting down Runtime\n");

        if (runtime->executor) {
            executor_shutdown(runtime->executor);
        }

        runtime->initialized = false;
    }
}

/**
 * @brief 检查Runtime是否正在运行
 */
bool runtime_is_running(const runtime_t* runtime) {
    return runtime && runtime->initialized && executor_is_running(runtime->executor);
}

// ============================================================================
// 高级功能（预留接口）
// ============================================================================

/**
 * @brief 创建多线程Runtime
 * TODO: 实现真正的GMP模型 - 当前只实现了基本的执行器
 */
runtime_t* runtime_create_multi_threaded(int num_threads) {
    // 暂时返回单线程版本
    (void)num_threads;
    fprintf(stderr, "Multi-threaded runtime not implemented yet\n");
    return runtime_create();
}

/**
 * @brief 获取Runtime统计信息
 * TODO: 实现性能监控
 */
void runtime_get_stats(const runtime_t* runtime, void* stats) {
    // 暂时无实现
    (void)runtime;
    (void)stats;
    printf("DEBUG: Runtime stats not implemented yet\n");
}
