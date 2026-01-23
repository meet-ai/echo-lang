/**
 * @file test_future_wake.c
 * @brief Future 唤醒机制测试
 * 
 * 测试 Future 的创建、await 阻塞、完成时唤醒和多个协程等待同一个 Future
 */

#include "../../../unit-tests/test_framework.h"
#include "../../../../domain/future/future.h"
#include "../../../../domain/coroutine/coroutine.h"
#include "../../../../domain/task/task.h"
#include "../../../../domain/scheduler/scheduler.h"

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
static int waiters_woken = 0;
static void* future_result = NULL;

// 等待 Future 的协程函数
void await_future_coroutine_func(void* arg) {
    Future* future = (Future*)arg;
    
    printf("  [Coroutine] Starting to await future %llu\n", future->id);
    
    // 使用 coroutine_await 等待 Future
    void* result = coroutine_await(future);
    
    if (result) {
        waiters_woken++;
        future_result = result;
        printf("  [Coroutine] Future %llu resolved with value %p\n", future->id, result);
    } else {
        printf("  [Coroutine] Future %llu rejected or cancelled\n", future->id);
    }
}

/**
 * @brief 测试 Future 创建和 Waker 设置
 */
void test_future_creation_and_waker_setup(void) {
    printf("=== Testing Future Creation and Waker Setup ===\n");
    
    // 创建 Future
    Future* future = future_new();
    TEST_ASSERT_NOT_NULL(future);
    TEST_ASSERT_EQUAL(FUTURE_PENDING, future->state);
    TEST_ASSERT_EQUAL(0, future->wait_count);
    TEST_ASSERT_NULL(future->wait_queue_head);
    TEST_ASSERT_NULL(future->wait_queue_tail);
    
    // 创建任务并添加到等待队列
    Task* task = task_create(NULL, NULL);
    TEST_ASSERT_NOT_NULL(task);
    
    future_add_waiter(future, task);
    TEST_ASSERT_EQUAL(1, future->wait_count);
    TEST_ASSERT_NOT_NULL(future->wait_queue_head);
    TEST_ASSERT_EQUAL(task, future->wait_queue_head->task);
    
    // 清理
    future_destroy(future);
    task_destroy(task);
    
    printf("✓ Future creation and waker setup test passed\n");
}

/**
 * @brief 测试 await Future 阻塞
 */
void test_await_future_blocking(void) {
    printf("=== Testing Await Future Blocking ===\n");
    
    // 创建 Future
    Future* future = future_new();
    TEST_ASSERT_NOT_NULL(future);
    
    // 创建协程和任务
    Coroutine* coroutine = coroutine_create(
        "await_future_coroutine",
        await_future_coroutine_func,
        future,
        4096
    );
    TEST_ASSERT_NOT_NULL(coroutine);
    
    Task* task = task_create(NULL, NULL);
    task->coroutine = coroutine;
    coroutine->task = task;
    
    // 设置当前任务（模拟在协程上下文中）
    extern struct Task* current_task;
    struct Task* prev_task = current_task;
    current_task = task;
    
    // 调用 coroutine_await（应该阻塞）
    coroutine_await(future);
    
    // 验证协程被挂起（如果 Future 未完成）
    if (future->state == FUTURE_PENDING) {
        TEST_ASSERT(coroutine->state == COROUTINE_SUSPENDED || 
                   coroutine->state == COROUTINE_READY);
        TEST_ASSERT(task->status == TASK_WAITING || 
                   task->status == TASK_READY);
        TEST_ASSERT(future->wait_count > 0);
    }
    
    // 恢复当前任务
    current_task = prev_task;
    
    // 清理
    future_destroy(future);
    coroutine_destroy(coroutine);
    task_destroy(task);
    
    printf("✓ Await future blocking test passed\n");
}

/**
 * @brief 测试 Future 完成时唤醒
 */
void test_future_completion_wakeup(void) {
    printf("=== Testing Future Completion Wakeup ===\n");
    
    // 创建 Future
    Future* future = future_new();
    TEST_ASSERT_NOT_NULL(future);
    
    // 创建多个等待的任务
    Task* tasks[3];
    for (int i = 0; i < 3; i++) {
        tasks[i] = task_create(NULL, NULL);
        TEST_ASSERT_NOT_NULL(tasks[i]);
        tasks[i]->status = TASK_WAITING;
        future_add_waiter(future, tasks[i]);
    }
    
    TEST_ASSERT_EQUAL(3, future->wait_count);
    
    // 解析 Future（应该唤醒所有等待者）
    int result_value = 42;
    future_resolve(future, &result_value);
    
    TEST_ASSERT_EQUAL(FUTURE_RESOLVED, future->state);
    TEST_ASSERT_EQUAL(0, future->wait_count);  // 等待队列应该被清空
    TEST_ASSERT_NULL(future->wait_queue_head);
    TEST_ASSERT_NULL(future->wait_queue_tail);
    
    // 验证所有任务被唤醒
    for (int i = 0; i < 3; i++) {
        pthread_mutex_lock(&tasks[i]->mutex);
        bool is_woken = (tasks[i]->status == TASK_READY);
        pthread_mutex_unlock(&tasks[i]->mutex);
        TEST_ASSERT(is_woken);
    }
    
    // 清理
    future_destroy(future);
    for (int i = 0; i < 3; i++) {
        task_destroy(tasks[i]);
    }
    
    printf("✓ Future completion wakeup test passed\n");
}

/**
 * @brief 测试多个协程等待同一个 Future
 */
void test_multiple_coroutines_waiting_same_future(void) {
    printf("=== Testing Multiple Coroutines Waiting Same Future ===\n");
    
    // 创建 Future
    Future* future = future_new();
    TEST_ASSERT_NOT_NULL(future);
    
    waiters_woken = 0;
    
    // 创建多个协程等待同一个 Future
    Coroutine* coroutines[3];
    Task* tasks[3];
    
    for (int i = 0; i < 3; i++) {
        char name[64];
        snprintf(name, sizeof(name), "await_future_coroutine_%d", i);
        coroutines[i] = coroutine_create(
            name,
            await_future_coroutine_func,
            future,
            4096
        );
        TEST_ASSERT_NOT_NULL(coroutines[i]);
        
        tasks[i] = task_create(NULL, NULL);
        tasks[i]->coroutine = coroutines[i];
        coroutines[i]->task = tasks[i];
        tasks[i]->status = TASK_WAITING;
        
        // 添加到 Future 等待队列
        future_add_waiter(future, tasks[i]);
    }
    
    TEST_ASSERT_EQUAL(3, future->wait_count);
    
    // 解析 Future（应该唤醒所有协程）
    int result_value = 100;
    future_resolve(future, &result_value);
    
    // 验证所有协程被唤醒
    TEST_ASSERT_EQUAL(0, future->wait_count);
    
    // 验证协程状态
    for (int i = 0; i < 3; i++) {
        TEST_ASSERT(coroutines[i]->state == COROUTINE_READY || 
                   coroutines[i]->state == COROUTINE_RUNNING);
    }
    
    // 清理
    future_destroy(future);
    for (int i = 0; i < 3; i++) {
        coroutine_destroy(coroutines[i]);
        task_destroy(tasks[i]);
    }
    
    printf("✓ Multiple coroutines waiting same future test passed\n");
}

// 测试套件
TestSuite* future_wake_test_suite(void) {
    TestSuite* suite = TEST_SUITE("Future Wake Tests", "Future 唤醒机制测试套件");
    
    TEST_ADD(suite, "future_creation_and_waker_setup", 
             "测试 Future 创建和 Waker 设置", test_future_creation_and_waker_setup);
    TEST_ADD(suite, "await_future_blocking", 
             "测试 await Future 阻塞", test_await_future_blocking);
    TEST_ADD(suite, "future_completion_wakeup", 
             "测试 Future 完成时唤醒", test_future_completion_wakeup);
    TEST_ADD(suite, "multiple_coroutines_waiting_same_future", 
             "测试多个协程等待同一个 Future", test_multiple_coroutines_waiting_same_future);
    
    return suite;
}

// 主函数（用于独立运行测试）
#ifdef STANDALONE_TEST
int main(void) {
    printf("=== Future Wake Test Suite ===\n");
    printf("====================================\n\n");
    
    int tests_run = 0;
    int tests_passed = 0;
    
    // 运行所有测试
    printf("\n[1/4] Running test_future_creation_and_waker_setup...\n");
    tests_run++;
    test_future_creation_and_waker_setup();
    tests_passed++;
    
    printf("\n[2/4] Running test_await_future_blocking...\n");
    tests_run++;
    test_await_future_blocking();
    tests_passed++;
    
    printf("\n[3/4] Running test_future_completion_wakeup...\n");
    tests_run++;
    test_future_completion_wakeup();
    tests_passed++;
    
    printf("\n[4/4] Running test_multiple_coroutines_waiting_same_future...\n");
    tests_run++;
    test_multiple_coroutines_waiting_same_future();
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

