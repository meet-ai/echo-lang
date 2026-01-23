#include "../unit-tests/test_framework.h"
#include "../../domain/coroutine/entity/coroutine.h"
#include "../../domain/coroutine/service/coroutine_manager.h"
#include <string.h>

// 测试套件
TestSuite* coroutine_test_suite(void);

// 测试用例
void test_coroutine_creation(void);
void test_coroutine_execution(void);
void test_coroutine_yield_resume(void);
void test_coroutine_stack_management(void);
void test_coroutine_manager_basic(void);
void test_coroutine_concurrent_execution(void);

// 协程创建测试
void test_coroutine_creation(void) {
    // 创建协程
    Coroutine* coro = coroutine_create("test_coro", coroutine_function_example, NULL, 4096);

    TEST_ASSERT_NOT_NULL(coro);
    TEST_ASSERT_STR_EQUAL("test_coro", coro->name);
    TEST_ASSERT_EQUAL(COROUTINE_STATE_NEW, coro->state);
    TEST_ASSERT(coro->stack_size >= 4096);

    // 验证协程ID唯一性
    Coroutine* coro2 = coroutine_create("test_coro_2", coroutine_function_example, NULL, 4096);
    TEST_ASSERT(coro->id != coro2->id);

    // 清理
    coroutine_destroy(coro);
    coroutine_destroy(coro2);
}

// 协程执行测试
void test_coroutine_execution(void) {
    Coroutine* coro = coroutine_create("exec_test", coroutine_function_simple, NULL, 4096);

    // 启动协程
    TEST_ASSERT(coroutine_start(coro));
    TEST_ASSERT_EQUAL(COROUTINE_STATE_RUNNING, coro->state);

    // 等待协程完成
    TEST_ASSERT(coroutine_join(coro, 5000));
    TEST_ASSERT_EQUAL(COROUTINE_STATE_COMPLETED, coro->state);

    // 验证执行结果
    int result = 0;
    TEST_ASSERT(coroutine_get_result(coro, &result, sizeof(result)));
    TEST_ASSERT_EQUAL(42, result);

    coroutine_destroy(coro);
}

// 协程yield和resume测试
void test_coroutine_yield_resume(void) {
    Coroutine* coro = coroutine_create("yield_test", coroutine_function_yield, NULL, 4096);

    // 启动协程
    TEST_ASSERT(coroutine_start(coro));

    // 协程应该在第一次yield时暂停
    TEST_ASSERT_EQUAL(COROUTINE_STATE_SUSPENDED, coro->state);

    // resume协程
    TEST_ASSERT(coroutine_resume(coro));
    TEST_ASSERT_EQUAL(COROUTINE_STATE_SUSPENDED, coro->state);

    // 再次resume，协程应该完成
    TEST_ASSERT(coroutine_resume(coro));
    TEST_ASSERT_EQUAL(COROUTINE_STATE_COMPLETED, coro->state);

    // 验证yield计数
    int yield_count = 0;
    TEST_ASSERT(coroutine_get_yield_count(coro, &yield_count));
    TEST_ASSERT_EQUAL(2, yield_count);

    coroutine_destroy(coro);
}

// 协程栈管理测试
void test_coroutine_stack_management(void) {
    // 测试不同栈大小
    Coroutine* small_stack = coroutine_create("small", coroutine_function_stack_test, NULL, 1024);
    Coroutine* large_stack = coroutine_create("large", coroutine_function_stack_test, NULL, 65536);

    // 启动并等待完成
    TEST_ASSERT(coroutine_start(small_stack));
    TEST_ASSERT(coroutine_join(small_stack, 5000));

    TEST_ASSERT(coroutine_start(large_stack));
    TEST_ASSERT(coroutine_join(large_stack, 5000));

    // 验证栈使用情况
    size_t small_used = 0, large_used = 0;
    TEST_ASSERT(coroutine_get_stack_usage(small_stack, &small_used));
    TEST_ASSERT(coroutine_get_stack_usage(large_stack, &large_used));

    TEST_ASSERT(small_used > 0);
    TEST_ASSERT(large_used > 0);
    TEST_ASSERT(small_used <= 1024);
    TEST_ASSERT(large_used <= 65536);

    coroutine_destroy(small_stack);
    coroutine_destroy(large_stack);
}

// 协程管理器基本测试
void test_coroutine_manager_basic(void) {
    CoroutineManager* manager = coroutine_manager_create();

    // 创建协程
    Coroutine* coro1 = coroutine_create("managed_1", coroutine_function_example, NULL, 4096);
    Coroutine* coro2 = coroutine_create("managed_2", coroutine_function_example, NULL, 4096);

    // 注册到管理器
    TEST_ASSERT(coroutine_manager_register(manager, coro1));
    TEST_ASSERT(coroutine_manager_register(manager, coro2));

    // 验证注册数量
    TEST_ASSERT_EQUAL(2, coroutine_manager_get_count(manager));

    // 查找协程
    Coroutine* found = coroutine_manager_find_by_id(manager, coro1->id);
    TEST_ASSERT_NOT_NULL(found);
    TEST_ASSERT_EQUAL(coro1->id, found->id);

    // 注销协程
    TEST_ASSERT(coroutine_manager_unregister(manager, coro1->id));
    TEST_ASSERT_EQUAL(1, coroutine_manager_get_count(manager));

    // 清理
    coroutine_destroy(coro1);
    coroutine_destroy(coro2);
    coroutine_manager_destroy(manager);
}

// 协程并发执行测试
void test_coroutine_concurrent_execution(void) {
    const int NUM_COROUTINES = 10;
    Coroutine* coroutines[NUM_COROUTINES];
    CoroutineManager* manager = coroutine_manager_create();

    // 创建多个协程
    for (int i = 0; i < NUM_COROUTINES; i++) {
        char name[32];
        sprintf(name, "concurrent_%d", i);
        coroutines[i] = coroutine_create(name, coroutine_function_concurrent, (void*)(uintptr_t)i, 4096);
        TEST_ASSERT(coroutine_manager_register(manager, coroutines[i]));
    }

    // 启动所有协程
    for (int i = 0; i < NUM_COROUTINES; i++) {
        TEST_ASSERT(coroutine_start(coroutines[i]));
    }

    // 等待所有协程完成
    for (int i = 0; i < NUM_COROUTINES; i++) {
        TEST_ASSERT(coroutine_join(coroutines[i], 10000));
        TEST_ASSERT_EQUAL(COROUTINE_STATE_COMPLETED, coroutines[i]->state);
    }

    // 验证并发执行没有问题
    TEST_ASSERT_EQUAL(NUM_COROUTINES, coroutine_manager_get_count(manager));

    // 清理
    for (int i = 0; i < NUM_COROUTINES; i++) {
        coroutine_destroy(coroutines[i]);
    }
    coroutine_manager_destroy(manager);
}

// 示例协程函数
void coroutine_function_example(void* arg) {
    // 简单的协程执行
    volatile int sum = 0;
    for (int i = 0; i < 100; i++) {
        sum += i;
        coroutine_yield(); // 主动让出控制权
    }
}

void coroutine_function_simple(void* arg) {
    // 设置结果
    int result = 42;
    coroutine_set_result(&result, sizeof(result));
}

void coroutine_function_yield(void* arg) {
    // 第一次yield
    coroutine_yield();

    // 第二次yield
    coroutine_yield();

    // 完成
}

void coroutine_function_stack_test(void* arg) {
    // 使用栈空间
    char buffer[1024];
    memset(buffer, 0xAA, sizeof(buffer));

    // 进行一些计算
    volatile int sum = 0;
    for (int i = 0; i < 1000; i++) {
        sum += buffer[i % sizeof(buffer)];
    }
}

void coroutine_function_concurrent(void* arg) {
    int id = (int)(uintptr_t)arg;

    // 模拟不同长度的计算
    volatile int iterations = 1000 + (id * 100);
    volatile int sum = 0;

    for (int i = 0; i < iterations; i++) {
        sum += i;
        if (i % 100 == 0) {
            coroutine_yield(); // 定期让出控制权
        }
    }
}

// 协程测试套件
TestSuite* coroutine_test_suite(void) {
    TestSuite* suite = TEST_SUITE("Coroutine Management", "协程管理相关测试");

    TEST_ADD(suite, "coroutine_creation", "测试协程创建功能", test_coroutine_creation);
    TEST_ADD(suite, "coroutine_execution", "测试协程执行功能", test_coroutine_execution);
    TEST_ADD(suite, "coroutine_yield_resume", "测试协程yield和resume", test_coroutine_yield_resume);
    TEST_ADD(suite, "coroutine_stack_management", "测试协程栈管理", test_coroutine_stack_management);
    TEST_ADD(suite, "coroutine_manager_basic", "测试协程管理器基本功能", test_coroutine_manager_basic);
    TEST_ADD(suite, "coroutine_concurrent_execution", "测试协程并发执行", test_coroutine_concurrent_execution);

    return suite;
}
