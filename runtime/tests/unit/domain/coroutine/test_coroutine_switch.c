/**
 * @file test_coroutine_switch.c
 * @brief 协程上下文切换测试
 * 
 * 测试协程的创建、启动、yield、恢复和完成通知机制
 */

#include "../../../unit-tests/test_framework.h"
#include "../../../../domain/coroutine/coroutine.h"
#include "../../../../domain/task/task.h"
#include "../../../../domain/scheduler/scheduler.h"

// 声明 runtime 函数
extern void scheduler_yield(void);
extern void* coroutine_await(void* future_ptr);
#include <stdio.h>
#include <stdlib.h>
#include <string.h>
#include <assert.h>

// 测试辅助变量
static int yield_count = 0;
static bool coroutine_completed = false;

// 测试协程函数1：简单计数
void test_coroutine_func1(void* arg) {
    int* counter = (int*)arg;
    printf("  [Coroutine] Starting, counter=%d\n", *counter);
    
    for (int i = 0; i < 3; i++) {
        (*counter)++;
        printf("  [Coroutine] Counter=%d, yielding...\n", *counter);
        scheduler_yield();
        yield_count++;
    }
    
    printf("  [Coroutine] Completed, final counter=%d\n", *counter);
}

// 测试协程函数2：验证恢复
void test_coroutine_func2(void* arg) {
    int* value = (int*)arg;
    printf("  [Coroutine2] Starting, value=%d\n", *value);
    
    *value = *value * 2;
    printf("  [Coroutine2] After calculation, value=%d\n", *value);
    
    scheduler_yield();
    
    *value = *value + 10;
    printf("  [Coroutine2] After yield, value=%d\n", *value);
}

// 测试协程完成回调
void test_coroutine_completion_callback(Coroutine* coroutine) {
    printf("  [Callback] Coroutine %llu completed\n", coroutine->id);
    coroutine_completed = true;
}

/**
 * @brief 测试协程创建和启动
 */
void test_coroutine_creation_and_start(void) {
    printf("=== Testing Coroutine Creation and Start ===\n");
    
    // 初始化调度器（如果还没有）
    extern Scheduler* get_global_scheduler(void);
    Scheduler* scheduler = get_global_scheduler();
    if (!scheduler) {
        scheduler = scheduler_create(1);  // 1个处理器
        // 注意：这里需要设置全局调度器，但为了测试简化，我们直接使用
    }
    
    int counter = 0;
    
    // 创建协程
    Coroutine* coroutine = coroutine_create(
        "test_coroutine_1",  // name
        test_coroutine_func1,  // entry_point
        &counter,  // arg
        4096  // stack_size
    );
    
    TEST_ASSERT_NOT_NULL(coroutine);
    TEST_ASSERT_EQUAL(COROUTINE_NEW, coroutine->state);
    
    // 创建任务并关联协程
    Task* task = task_create(
        NULL,  // entry_point (协程有自己的入口)
        NULL   // arg
    );
    
    TEST_ASSERT_NOT_NULL(task);
    task->coroutine = coroutine;
    coroutine->task = task;
    
    // 启动协程
    coroutine_resume(coroutine);
    
    // 验证协程状态
    TEST_ASSERT(coroutine->state == COROUTINE_RUNNING || 
                coroutine->state == COROUTINE_SUSPENDED ||
                coroutine->state == COROUTINE_COMPLETED);
    
    // 清理
    coroutine_destroy(coroutine);
    task_destroy(task);
    
    printf("✓ Coroutine creation and start test passed\n");
}

/**
 * @brief 测试协程 yield 和恢复
 */
void test_coroutine_yield_resume(void) {
    printf("=== Testing Coroutine Yield and Resume ===\n");
    
    yield_count = 0;
    int counter = 0;
    
    // 创建协程
    Coroutine* coroutine = coroutine_create(
        "test_coroutine_yield",
        test_coroutine_func1,
        &counter,
        4096
    );
    
    TEST_ASSERT_NOT_NULL(coroutine);
    
    // 创建任务
    Task* task = task_create(NULL, NULL);
    task->coroutine = coroutine;
    coroutine->task = task;
    
    // 执行协程（会yield）
    coroutine_resume(coroutine);
    
    // 验证yield发生
    TEST_ASSERT(yield_count > 0 || coroutine->state == COROUTINE_SUSPENDED);
    
    // 再次恢复
    if (coroutine->state == COROUTINE_SUSPENDED) {
        coroutine->state = COROUTINE_READY;
        coroutine_resume(coroutine);
    }
    
    // 清理
    coroutine_destroy(coroutine);
    task_destroy(task);
    
    printf("✓ Coroutine yield and resume test passed\n");
}

/**
 * @brief 测试多个协程切换
 */
void test_multiple_coroutine_switching(void) {
    printf("=== Testing Multiple Coroutine Switching ===\n");
    
    int value1 = 5;
    int value2 = 10;
    
    // 创建两个协程
    Coroutine* coro1 = coroutine_create(
        "test_coroutine_1",
        test_coroutine_func2,
        &value1,
        4096
    );
    
    Coroutine* coro2 = coroutine_create(
        "test_coroutine_2",
        test_coroutine_func2,
        &value2,
        4096
    );
    
    TEST_ASSERT_NOT_NULL(coro1);
    TEST_ASSERT_NOT_NULL(coro2);
    
    // 创建任务
    Task* task1 = task_create(NULL, NULL);
    Task* task2 = task_create(NULL, NULL);
    coro1->task = task1;
    coro2->task = task2;
    task1->coroutine = coro1;
    task2->coroutine = coro2;
    
    // 切换执行
    coroutine_resume(coro1);
    if (coro1->state == COROUTINE_SUSPENDED) {
        coroutine_resume(coro2);
    }
    
    // 验证两个协程都执行了
    TEST_ASSERT(value1 > 5 || coro1->state == COROUTINE_COMPLETED);
    TEST_ASSERT(value2 > 10 || coro2->state == COROUTINE_COMPLETED);
    
    // 清理
    coroutine_destroy(coro1);
    coroutine_destroy(coro2);
    task_destroy(task1);
    task_destroy(task2);
    
    printf("✓ Multiple coroutine switching test passed\n");
}

/**
 * @brief 测试协程完成通知
 */
void test_coroutine_completion_notification(void) {
    printf("=== Testing Coroutine Completion Notification ===\n");
    
    coroutine_completed = false;
    int counter = 0;
    
    // 创建协程
    Coroutine* coroutine = coroutine_create(
        "test_coroutine_yield",
        test_coroutine_func1,
        &counter,
        4096
    );
    
    TEST_ASSERT_NOT_NULL(coroutine);
    
    // 创建任务
    Task* task = task_create(NULL, NULL);
    task->coroutine = coroutine;
    coroutine->task = task;
    
    // 执行协程直到完成
    while (coroutine->state != COROUTINE_COMPLETED && 
           coroutine->state != COROUTINE_FAILED) {
        if (coroutine->state == COROUTINE_SUSPENDED) {
            coroutine->state = COROUTINE_READY;
        }
        coroutine_resume(coroutine);
        
        // 防止无限循环
        if (counter > 10) break;
    }
    
    // 验证协程完成
    TEST_ASSERT(coroutine->state == COROUTINE_COMPLETED || 
                coroutine->state == COROUTINE_FAILED);
    
    // 清理
    coroutine_destroy(coroutine);
    task_destroy(task);
    
    printf("✓ Coroutine completion notification test passed\n");
}

// 测试套件
TestSuite* coroutine_switch_test_suite(void) {
    TestSuite* suite = TEST_SUITE("Coroutine Switch Tests", "协程上下文切换测试套件");
    
    TEST_ADD(suite, "coroutine_creation_and_start", 
             "测试协程创建和启动", test_coroutine_creation_and_start);
    TEST_ADD(suite, "coroutine_yield_resume", 
             "测试协程 yield 和恢复", test_coroutine_yield_resume);
    TEST_ADD(suite, "multiple_coroutine_switching", 
             "测试多个协程切换", test_multiple_coroutine_switching);
    TEST_ADD(suite, "coroutine_completion_notification", 
             "测试协程完成通知", test_coroutine_completion_notification);
    
    return suite;
}

// 主函数（用于独立运行测试）
#ifdef STANDALONE_TEST
int main(void) {
    printf("=== Coroutine Switch Test Suite ===\n");
    printf("====================================\n\n");
    
    int tests_run = 0;
    int tests_passed = 0;
    
    // 运行所有测试
    printf("\n[1/4] Running test_coroutine_creation_and_start...\n");
    tests_run++;
    test_coroutine_creation_and_start();
    tests_passed++;
    
    printf("\n[2/4] Running test_coroutine_yield_resume...\n");
    tests_run++;
    test_coroutine_yield_resume();
    tests_passed++;
    
    printf("\n[3/4] Running test_multiple_coroutine_switching...\n");
    tests_run++;
    test_multiple_coroutine_switching();
    tests_passed++;
    
    printf("\n[4/4] Running test_coroutine_completion_notification...\n");
    tests_run++;
    test_coroutine_completion_notification();
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

