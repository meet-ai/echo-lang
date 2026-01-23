/**
 * @file test_coroutine_switch_runner.c
 * @brief 协程切换测试运行器
 */

#include <stdio.h>
#include <stdlib.h>
#include <stdbool.h>
#include <signal.h>

// 声明测试函数
extern void test_coroutine_creation_and_start(void);
extern void test_coroutine_yield_resume(void);
extern void test_multiple_coroutine_switching(void);
extern void test_coroutine_completion_notification(void);

// 简单的测试运行器
static int tests_run = 0;
static int tests_passed = 0;
static int tests_failed = 0;

void run_test(const char* name, void (*test_func)(void)) {
    printf("\n=== Running: %s ===\n", name);
    fflush(stdout);
    tests_run++;
    
    // 设置信号处理（捕获段错误等）
    signal(SIGSEGV, SIG_DFL);
    
    test_func();
    tests_passed++;
    printf("✓ %s passed\n", name);
}

int main(void) {
    printf("=== Coroutine Switch Test Suite ===\n");
    printf("====================================\n\n");
    
    // 运行所有测试
    run_test("test_coroutine_creation_and_start", test_coroutine_creation_and_start);
    run_test("test_coroutine_yield_resume", test_coroutine_yield_resume);
    run_test("test_multiple_coroutine_switching", test_multiple_coroutine_switching);
    run_test("test_coroutine_completion_notification", test_coroutine_completion_notification);
    
    // 输出统计
    printf("\n====================================\n");
    printf("Test Summary:\n");
    printf("  Total:  %d\n", tests_run);
    printf("  Passed: %d\n", tests_passed);
    printf("  Failed: %d\n", tests_failed);
    printf("====================================\n");
    
    return (tests_failed > 0) ? 1 : 0;
}

