/**
 * @file async_await.c
 * @brief 异步编程示例 - 演示异步闭环
 *
 * 这个示例展示了如何使用TimerFuture创建异步操作。
 */

#include "../include/echo/runtime.h"
#include "../include/echo/task.h"
#include "../include/echo/future.h"
#include "../src/core/runtime.h"
#include "../src/domain/reactor/timer_future.h"
#include <stdio.h>
#include <stdlib.h>

// ============================================================================
// 示例异步函数
// ============================================================================

/**
 * @brief 异步等待函数
 * 模拟一个异步操作，等待指定的毫秒数
 */
void* async_wait(int milliseconds) {
    printf("Starting async wait for %d ms...\n", milliseconds);

    // 创建TimerFuture
    timer_future_t* future = timer_future_create(milliseconds);
    if (!future) {
        fprintf(stderr, "Failed to create TimerFuture\n");
        return NULL;
    }

    // 等待Future完成（简化实现，实际应该由await语法糖处理）
    void* result = timer_future_await(future);

    printf("Async wait completed with result: %s\n", (char*)result);

    // 清理
    timer_future_destroy(future);

    return result;
}

/**
 * @brief 主任务函数
 * 演示异步操作的编排
 */
void main_task_function(void* arg) {
    (void)arg; // 未使用参数

    printf("=== Async/Await Example ===\n");

    // 这里演示基本的任务执行
    // 真正的异步await需要编译器支持，目前我们只是演示任务执行
    printf("Task executed successfully!\n");

    // 在真正的实现中，这里应该有：
    // let future1 = TimerFuture::new(100);
    // let result1 = await future1;
    // let future2 = TimerFuture::new(200);
    // let result2 = await future2;

    printf("All operations completed!\n");
}

// ============================================================================
// 主函数
// ============================================================================

int main(int argc, char* argv[]) {
    (void)argc;
    (void)argv;

    printf("Echo Runtime - Async/Await Example\n");

    // 创建Runtime
    runtime_t* runtime = runtime_create();
    if (!runtime) {
        fprintf(stderr, "Failed to create runtime\n");
        return 1;
    }

    // 创建主任务
    task_t* main_task = task_create(main_task_function, NULL, DEFAULT_STACK_SIZE);
    if (!main_task) {
        fprintf(stderr, "Failed to create main task\n");
        runtime_destroy(runtime);
        return 1;
    }

    // 运行Runtime
    int result = runtime_run(runtime, main_task);
    if (result != 0) {
        fprintf(stderr, "Runtime execution failed\n");
    }

    // 清理资源
    task_destroy(main_task);
    runtime_destroy(runtime);

    printf("Example completed successfully!\n");
    return result;
}
