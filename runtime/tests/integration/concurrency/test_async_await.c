/**
 * @file test_async_await.c
 * @brief 异步/等待集成测试
 * 
 * 测试完整的 async/await 流程、多个异步操作并发执行和错误处理
 */

#include "../../unit-tests/test_framework.h"
#include "../../../domain/future/future.h"
#include "../../../domain/coroutine/coroutine.h"
#include "../../../domain/task/task.h"
#include "../../../domain/scheduler/scheduler.h"

// 声明 runtime 函数
extern void* coroutine_await(void* future_ptr);
extern void future_add_waiter(Future* future, struct Task* task);
#include <stdio.h>
#include <stdlib.h>
#include <string.h>
#include <assert.h>
#include <pthread.h>
#include <unistd.h>

// 测试辅助变量
static int async_operations_completed = 0;
static int async_operations_failed = 0;

// 异步操作函数（模拟）
void* async_operation_func(void* arg) {
    int delay_ms = *(int*)arg;
    
    // 模拟异步操作延迟
    usleep(delay_ms * 1000);
    
    int* result = malloc(sizeof(int));
    *result = delay_ms * 2;
    return result;
}

// 使用 async/await 的协程函数
void async_await_coroutine_func(void* arg) {
    Future* future = (Future*)arg;
    
    printf("  [Async Coroutine] Starting to await future %llu\n", future->id);
    
    // 等待 Future 完成
    void* result = coroutine_await(future);
    
    if (result) {
        async_operations_completed++;
        printf("  [Async Coroutine] Future completed with result %p\n", result);
    } else {
        async_operations_failed++;
        printf("  [Async Coroutine] Future failed or cancelled\n");
    }
}

/**
 * @brief 测试完整 async/await 流程
 */
void test_complete_async_await_flow(void) {
    printf("=== Testing Complete Async/Await Flow ===\n");
    
    // 创建 Future
    Future* future = future_new();
    TEST_ASSERT_NOT_NULL(future);
    TEST_ASSERT_EQUAL(FUTURE_PENDING, future->state);
    
    // 创建协程等待 Future
    Coroutine* coroutine = coroutine_create(
        "async_await_coroutine",
        async_await_coroutine_func,
        future,
        4096
    );
    TEST_ASSERT_NOT_NULL(coroutine);
    
    Task* task = task_create(NULL, NULL);
    task->coroutine = coroutine;
    coroutine->task = task;
    
    // 设置当前任务
    extern struct Task* current_task;
    struct Task* prev_task = current_task;
    current_task = task;
    
    // 启动协程（会阻塞等待 Future）
    coroutine_await(future);
    
    // 验证协程被挂起
    TEST_ASSERT(coroutine->state == COROUTINE_SUSPENDED || 
               coroutine->state == COROUTINE_READY);
    TEST_ASSERT(future->wait_count > 0);
    
    // 解析 Future（应该唤醒协程）
    int result_value = 42;
    future_resolve(future, &result_value);
    
    // 验证 Future 状态
    TEST_ASSERT_EQUAL(FUTURE_RESOLVED, future->state);
    TEST_ASSERT_EQUAL(0, future->wait_count);
    
    // 恢复当前任务
    current_task = prev_task;
    
    // 清理
    future_destroy(future);
    coroutine_destroy(coroutine);
    task_destroy(task);
    
    printf("✓ Complete async/await flow test passed\n");
}

/**
 * @brief 测试多个异步操作并发执行
 */
void test_multiple_async_operations_concurrent(void) {
    printf("=== Testing Multiple Async Operations Concurrent ===\n");
    
    const int num_operations = 5;
    Future* futures[num_operations];
    Coroutine* coroutines[num_operations];
    Task* tasks[num_operations];
    
    // 创建多个 Future 和协程
    for (int i = 0; i < num_operations; i++) {
        futures[i] = future_new();
        TEST_ASSERT_NOT_NULL(futures[i]);
        
        char name[64];
        snprintf(name, sizeof(name), "async_coroutine_%d", i);
        coroutines[i] = coroutine_create(
            name,
            async_await_coroutine_func,
            futures[i],
            4096
        );
        TEST_ASSERT_NOT_NULL(coroutines[i]);
        
        tasks[i] = task_create(NULL, NULL);
        tasks[i]->coroutine = coroutines[i];
        coroutines[i]->task = tasks[i];
        tasks[i]->status = TASK_WAITING;
        
        // 添加到 Future 等待队列
        future_add_waiter(futures[i], tasks[i]);
    }
    
    // 验证所有 Future 都有等待者
    for (int i = 0; i < num_operations; i++) {
        TEST_ASSERT_EQUAL(1, futures[i]->wait_count);
    }
    
    // 并发解析所有 Future
    for (int i = 0; i < num_operations; i++) {
        int* result_value = malloc(sizeof(int));
        *result_value = i * 10;
        future_resolve(futures[i], result_value);
    }
    
    // 验证所有 Future 都被解析
    for (int i = 0; i < num_operations; i++) {
        TEST_ASSERT_EQUAL(FUTURE_RESOLVED, futures[i]->state);
        TEST_ASSERT_EQUAL(0, futures[i]->wait_count);
    }
    
    // 验证所有任务被唤醒
    for (int i = 0; i < num_operations; i++) {
        pthread_mutex_lock(&tasks[i]->mutex);
        bool is_woken = (tasks[i]->status == TASK_READY);
        pthread_mutex_unlock(&tasks[i]->mutex);
        TEST_ASSERT(is_woken);
    }
    
    // 清理
    for (int i = 0; i < num_operations; i++) {
        future_destroy(futures[i]);
        coroutine_destroy(coroutines[i]);
        task_destroy(tasks[i]);
    }
    
    printf("✓ Multiple async operations concurrent test passed\n");
}

/**
 * @brief 测试异步操作错误处理
 */
void test_async_operation_error_handling(void) {
    printf("=== Testing Async Operation Error Handling ===\n");
    
    async_operations_completed = 0;
    async_operations_failed = 0;
    
    // 创建 Future
    Future* future = future_new();
    TEST_ASSERT_NOT_NULL(future);
    
    // 创建协程等待 Future
    Coroutine* coroutine = coroutine_create(
        "async_await_coroutine",
        async_await_coroutine_func,
        future,
        4096
    );
    TEST_ASSERT_NOT_NULL(coroutine);
    
    Task* task = task_create(NULL, NULL);
    task->coroutine = coroutine;
    coroutine->task = task;
    task->status = TASK_WAITING;
    
    // 添加到 Future 等待队列
    future_add_waiter(future, task);
    TEST_ASSERT_EQUAL(1, future->wait_count);
    
    // 拒绝 Future（模拟错误）
    char* error_msg = "Test error";
    future_reject(future, error_msg);
    
    // 验证 Future 状态
    TEST_ASSERT_EQUAL(FUTURE_REJECTED, future->state);
    TEST_ASSERT_EQUAL(0, future->wait_count);
    
    // 验证任务被唤醒
    pthread_mutex_lock(&task->mutex);
    bool is_woken = (task->status == TASK_READY);
    pthread_mutex_unlock(&task->mutex);
    TEST_ASSERT(is_woken);
    
    // 验证协程状态
    TEST_ASSERT(coroutine->state == COROUTINE_READY || 
               coroutine->state == COROUTINE_RUNNING);
    
    // 清理
    future_destroy(future);
    coroutine_destroy(coroutine);
    task_destroy(task);
    
    printf("✓ Async operation error handling test passed\n");
}

// 测试套件
TestSuite* async_await_test_suite(void) {
    TestSuite* suite = TEST_SUITE("Async/Await Integration Tests", "异步/等待集成测试套件");
    
    TEST_ADD(suite, "complete_async_await_flow", 
             "测试完整 async/await 流程", test_complete_async_await_flow);
    TEST_ADD(suite, "multiple_async_operations_concurrent", 
             "测试多个异步操作并发执行", test_multiple_async_operations_concurrent);
    TEST_ADD(suite, "async_operation_error_handling", 
             "测试异步操作错误处理", test_async_operation_error_handling);
    
    return suite;
}

// 主函数（用于独立运行测试）
#ifdef STANDALONE_TEST
int main(void) {
    printf("=== Async/Await Integration Test Suite ===\n");
    printf("====================================\n\n");
    
    int tests_run = 0;
    int tests_passed = 0;
    
    // 运行所有测试
    printf("\n[1/3] Running test_complete_async_await_flow...\n");
    tests_run++;
    test_complete_async_await_flow();
    tests_passed++;
    
    printf("\n[2/3] Running test_multiple_async_operations_concurrent...\n");
    tests_run++;
    test_multiple_async_operations_concurrent();
    tests_passed++;
    
    printf("\n[3/3] Running test_async_operation_error_handling...\n");
    tests_run++;
    test_async_operation_error_handling();
    tests_passed++;
    
    // 输出统计
    printf("\n====================================\n");
    printf("Test Summary:\n");
    printf("  Total:  %d\n", tests_run);
    printf("  Passed: %d\n", tests_passed);
    printf("  Failed: %d\n", tests_run - tests_passed);
    printf("====================================\n");
    
    return (tests_passed == tests_run) ? 0 : 1;
}
#endif

