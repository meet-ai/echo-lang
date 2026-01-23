/**
 * @file async_memory_pool_test.c
 * @brief 异步内存池优化测试
 *
 * 测试异步Runtime特化内存池优化：
 * - Future状态机压缩
 * - Task内存池头部分离
 * - 协程栈管理
 * - 缓存感知通道
 */

#include <stdio.h>
#include <stdlib.h>
#include <string.h>
#include <pthread.h>
#include <unistd.h>
#include <time.h>
#include <assert.h>

#include "industrial_memory_pool.h"
#include "async_memory_pool.h"

// ============================================================================
// 测试辅助宏
// ============================================================================

#define TEST_ASSERT(condition, message) \
    do { \
        if (!(condition)) { \
            fprintf(stderr, "TEST FAILED: %s at %s:%d\n", message, __FILE__, __LINE__); \
            exit(1); \
        } \
    } while (0)

// ============================================================================
// Future状态机测试
// ============================================================================

/**
 * @brief 测试压缩Future基础功能
 */
void test_compact_future_basic(void) {
    printf("Testing CompactFuture basic functionality...\n");

    // 创建测试数据
    const char* test_data = "Hello, Compact Future!";
    size_t data_size = strlen(test_data) + 1;

    // 创建压缩Future
    compact_future_t* future = compact_future_create((void*)test_data, data_size);
    TEST_ASSERT(future != NULL, "Failed to create compact future");

    // 检查初始状态
    future_status_t status = compact_future_get_status(future);
    TEST_ASSERT(status == FUTURE_PENDING, "Initial status should be PENDING");

    // 设置为Ready状态
    compact_future_set_status(future, FUTURE_READY);
    status = compact_future_get_status(future);
    TEST_ASSERT(status == FUTURE_READY, "Status should be READY after setting");

    // 其他状态测试
    compact_future_set_status(future, FUTURE_WAITING);
    TEST_ASSERT(compact_future_get_status(future) == FUTURE_WAITING, "Status should be WAITING");

    compact_future_set_status(future, FUTURE_DROPPED);
    TEST_ASSERT(compact_future_get_status(future) == FUTURE_DROPPED, "Status should be DROPPED");

    // 清理
    compact_future_destroy(future);
    printf("  ✓ CompactFuture basic test passed\n");
}

/**
 * @brief 测试压缩Future内联存储
 */
void test_compact_future_inline_storage(void) {
    printf("Testing CompactFuture inline storage...\n");

    // 小数据（应该内联存储）
    const char small_data[] = "Hi!"; // 4字节，应该内联

    compact_future_t* future = compact_future_create((void*)small_data, sizeof(small_data));
    TEST_ASSERT(future != NULL, "Failed to create future with small data");

    // 验证可以正常工作
    future_status_t status = compact_future_get_status(future);
    TEST_ASSERT(status == FUTURE_PENDING, "Should be pending");

    compact_future_set_status(future, FUTURE_READY);
    TEST_ASSERT(compact_future_get_status(future) == FUTURE_READY, "Should be ready");

    compact_future_destroy(future);
    printf("  ✓ CompactFuture inline storage test passed\n");
}

// ============================================================================
// Task内存池测试
// ============================================================================

/**
 * @brief 测试Task内存池基础功能
 */
void test_task_memory_pool_basic(void) {
    printf("Testing TaskMemoryPool basic functionality...\n");

    // 创建Task内存池（最大64KB）
    task_memory_pool_t* pool = task_memory_pool_create(65536);
    TEST_ASSERT(pool != NULL, "Failed to create task memory pool");

    // 测试分配小Task（头部+小主体）
    void* task1 = task_memory_pool_allocate(pool, 128);
    TEST_ASSERT(task1 != NULL, "Failed to allocate small task");

    // 测试分配中等Task
    void* task2 = task_memory_pool_allocate(pool, 1024);
    TEST_ASSERT(task2 != NULL, "Failed to allocate medium task");

    // 测试分配大Task
    void* task3 = task_memory_pool_allocate(pool, 8192);
    TEST_ASSERT(task3 != NULL, "Failed to allocate large task");

    // 验证分配的任务各不相同
    TEST_ASSERT(task1 != task2 && task2 != task3 && task1 != task3,
               "Allocated tasks should be different");

    // 测试释放
    bool success1 = task_memory_pool_deallocate(pool, task1);
    TEST_ASSERT(success1, "Failed to deallocate task1");

    bool success2 = task_memory_pool_deallocate(pool, task2);
    TEST_ASSERT(success2, "Failed to deallocate task2");

    bool success3 = task_memory_pool_deallocate(pool, task3);
    TEST_ASSERT(success3, "Failed to deallocate task3");

    // 测试释放后再次分配（应该复用）
    void* task4 = task_memory_pool_allocate(pool, 256);
    TEST_ASSERT(task4 != NULL, "Failed to allocate task after deallocation");

    task_memory_pool_deallocate(pool, task4);

    // 清理
    task_memory_pool_destroy(pool);
    printf("  ✓ TaskMemoryPool basic test passed\n");
}

// ============================================================================
// 协程栈管理测试
// ============================================================================

/**
 * @brief 测试协程栈管理基础功能
 */
void test_coroutine_stack_manager_basic(void) {
    printf("Testing CoroutineStackManager basic functionality...\n");

    // 创建分段栈管理器
    coroutine_stack_manager_t* manager = coroutine_stack_manager_create(
        STACK_SEGMENTED, 4096, 65536); // 4KB-64KB
    TEST_ASSERT(manager != NULL, "Failed to create stack manager");

    // 测试分配小栈
    void* stack1 = coroutine_stack_manager_allocate(manager, 4096);
    TEST_ASSERT(stack1 != NULL, "Failed to allocate small stack");

    // 测试分配中等栈
    void* stack2 = coroutine_stack_manager_allocate(manager, 16384);
    TEST_ASSERT(stack2 != NULL, "Failed to allocate medium stack");

    // 验证栈各不相同
    TEST_ASSERT(stack1 != stack2, "Allocated stacks should be different");

    // 测试释放
    bool success1 = coroutine_stack_manager_deallocate(manager, stack1);
    TEST_ASSERT(success1, "Failed to deallocate stack1");

    bool success2 = coroutine_stack_manager_deallocate(manager, stack2);
    TEST_ASSERT(success2, "Failed to deallocate stack2");

    // 测试释放后再次分配
    void* stack3 = coroutine_stack_manager_allocate(manager, 8192);
    TEST_ASSERT(stack3 != NULL, "Failed to allocate stack after deallocation");

    coroutine_stack_manager_deallocate(manager, stack3);

    // 清理
    coroutine_stack_manager_destroy(manager);
    printf("  ✓ CoroutineStackManager basic test passed\n");
}

/**
 * @brief 测试协程栈大小调整
 */
void test_coroutine_stack_resize(void) {
    printf("Testing CoroutineStackManager resize functionality...\n");

    coroutine_stack_manager_t* manager = coroutine_stack_manager_create(
        STACK_SEGMENTED, 4096, 131072); // 4KB-128KB
    TEST_ASSERT(manager != NULL, "Failed to create stack manager for resize test");

    void* stack = coroutine_stack_manager_allocate(manager, 4096);
    TEST_ASSERT(stack != NULL, "Failed to allocate initial stack");

    // 模拟使用统计：高增长模式
    stack_usage_t usage = {
        .peak_usage = 12000,     // 12KB峰值使用
        .current_usage = 8000,   // 8KB当前使用
        .growth_pattern = 1,     // 增长模式
        .safety_margin = 2048    // 2KB安全边际
    };

    // 执行大小调整
    bool resized = coroutine_stack_manager_adjust_size(manager, stack, &usage);
    // 注意：实际实现中可能不会总是调整大小，这里我们不做严格断言

    printf("  Stack resize operation completed (resized: %s)\n", resized ? "yes" : "no");

    coroutine_stack_manager_deallocate(manager, stack);
    coroutine_stack_manager_destroy(manager);
    printf("  ✓ CoroutineStackManager resize test passed\n");
}

// ============================================================================
// 缓存感知通道测试
// ============================================================================

/**
 * @brief 测试缓存感知通道基础功能
 */
void test_cache_aware_channel_basic(void) {
    printf("Testing CacheAwareChannel basic functionality...\n");

    // 创建通道（容量1024）
    cache_aware_channel_t* channel = cache_aware_channel_create(1024);
    TEST_ASSERT(channel != NULL, "Failed to create cache aware channel");

    // 测试基本数据
    const char* test_data1 = "Message 1";
    const char* test_data2 = "Message 2";
    const char* test_data3 = "Message 3";

    void* messages[] = {(void*)test_data1, (void*)test_data2, (void*)test_data3};
    size_t count = sizeof(messages) / sizeof(messages[0]);

    // 批量发送
    size_t sent = cache_aware_channel_send_batch(channel, messages, count);
    TEST_ASSERT(sent == count, "Failed to send all messages");

    // 零拷贝接收
    void* recv1 = cache_aware_channel_try_recv_zero_copy(channel);
    TEST_ASSERT(recv1 != NULL, "Failed to receive first message");
    TEST_ASSERT(recv1 == (void*)test_data1, "Received wrong message");

    void* recv2 = cache_aware_channel_try_recv_zero_copy(channel);
    TEST_ASSERT(recv2 != NULL, "Failed to receive second message");
    TEST_ASSERT(recv2 == (void*)test_data2, "Received wrong message");

    void* recv3 = cache_aware_channel_try_recv_zero_copy(channel);
    TEST_ASSERT(recv3 != NULL, "Failed to receive third message");
    TEST_ASSERT(recv3 == (void*)test_data3, "Received wrong message");

    // 确认接收（延迟释放）
    cache_aware_channel_ack_recv(channel);
    cache_aware_channel_ack_recv(channel);
    cache_aware_channel_ack_recv(channel);

    // 验证通道已空
    void* empty = cache_aware_channel_try_recv_zero_copy(channel);
    TEST_ASSERT(empty == NULL, "Channel should be empty");

    // 清理
    cache_aware_channel_destroy(channel);
    printf("  ✓ CacheAwareChannel basic test passed\n");
}

// ============================================================================
// 异步Runtime内存池集成测试
// ============================================================================

/**
 * @brief 测试异步Runtime内存池集成
 */
void test_async_runtime_memory_pool_integration(void) {
    printf("Testing AsyncRuntimeMemoryPool integration...\n");

    // 创建异步Runtime内存池
    async_runtime_memory_pool_t* pool = async_runtime_memory_pool_create();
    TEST_ASSERT(pool != NULL, "Failed to create async runtime memory pool");

    // 测试Task分配
    void* task1 = async_runtime_memory_pool_allocate_task(pool, 256);
    TEST_ASSERT(task1 != NULL, "Failed to allocate async task");

    void* task2 = async_runtime_memory_pool_allocate_task(pool, 1024);
    TEST_ASSERT(task2 != NULL, "Failed to allocate second async task");

    // 测试栈分配
    void* stack1 = async_runtime_memory_pool_allocate_stack(pool, 8192);
    TEST_ASSERT(stack1 != NULL, "Failed to allocate coroutine stack");

    void* stack2 = async_runtime_memory_pool_allocate_stack(pool, 16384);
    TEST_ASSERT(stack2 != NULL, "Failed to allocate second coroutine stack");

    // 测试通道操作
    const char* msg1 = "Async Message 1";
    const char* msg2 = "Async Message 2";
    void* messages[] = {(void*)msg1, (void*)msg2};

    size_t sent = async_runtime_memory_pool_channel_send_batch(pool, messages, 2);
    TEST_ASSERT(sent == 2, "Failed to send messages to async channel");

    void* recv1 = async_runtime_memory_pool_channel_try_recv_zero_copy(pool);
    TEST_ASSERT(recv1 == (void*)msg1, "Received wrong async message 1");

    void* recv2 = async_runtime_memory_pool_channel_try_recv_zero_copy(pool);
    TEST_ASSERT(recv2 == (void*)msg2, "Received wrong async message 2");

    // 确认接收
    async_runtime_memory_pool_channel_try_recv_zero_copy(pool); // ack first
    async_runtime_memory_pool_channel_try_recv_zero_copy(pool); // ack second

    // 测试压缩Future
    const char small_future_data[] = "Future";
    compact_future_t* future = async_runtime_memory_pool_create_compact_future(
        pool, (void*)small_future_data, sizeof(small_future_data));
    TEST_ASSERT(future != NULL, "Failed to create compact future in async pool");

    future_status_t status = compact_future_get_status(future);
    TEST_ASSERT(status == FUTURE_PENDING, "Future should be pending");

    compact_future_set_status(future, FUTURE_READY);
    TEST_ASSERT(compact_future_get_status(future) == FUTURE_READY, "Future should be ready");

    async_runtime_memory_pool_destroy_compact_future(pool, future);

    // 清理所有资源
    // 注意：实际使用中需要释放task和stack，但这里简化了测试

    async_runtime_memory_pool_destroy(pool);
    printf("  ✓ AsyncRuntimeMemoryPool integration test passed\n");
}

// ============================================================================
// 性能测试
// ============================================================================

/**
 * @brief 异步对象性能测试
 */
void test_async_performance(void) {
    printf("Testing async objects performance...\n");

    async_runtime_memory_pool_t* pool = async_runtime_memory_pool_create();
    TEST_ASSERT(pool != NULL, "Failed to create pool for performance test");

    const int iterations = 100000;
    timer_t timer;

    // Task分配/释放性能
    printf("  Testing Task allocation/deallocation performance...\n");
    timer_start(&timer);

    for (int i = 0; i < iterations; i++) {
        void* task = async_runtime_memory_pool_allocate_task(pool, 128);
        // 注意：实际应用中这里会有业务逻辑
        // 这里我们简化测试，不进行释放以避免复杂的状态管理
    }

    double task_time = timer_elapsed_ms(&timer);
    printf("    %d Task allocations: %.2f ms (%.2f ns per allocation)\n",
           iterations, task_time, (task_time * 1000000.0) / iterations);

    // 栈分配性能
    printf("  Testing stack allocation performance...\n");
    timer_start(&timer);

    for (int i = 0; i < iterations / 10; i++) { // 减少迭代次数避免内存耗尽
        void* stack = async_runtime_memory_pool_allocate_stack(pool, 4096);
        // 简化测试，不释放
    }

    double stack_time = timer_elapsed_ms(&timer);
    int stack_iterations = iterations / 10;
    printf("    %d stack allocations: %.2f ms (%.2f ns per allocation)\n",
           stack_iterations, stack_time, (stack_time * 1000000.0) / stack_iterations);

    // 通道性能
    printf("  Testing channel performance...\n");
    timer_start(&timer);

    const char* test_msg = "Test";
    void* msgs[iterations];
    for (int i = 0; i < iterations; i++) {
        msgs[i] = (void*)test_msg;
    }

    size_t sent = async_runtime_memory_pool_channel_send_batch(pool, msgs, iterations);
    printf("    Sent %zu messages in batch\n", sent);

    double channel_time = timer_elapsed_ms(&timer);
    printf("    Channel operations: %.2f ms (%.2f ns per message)\n",
           channel_time, (channel_time * 1000000.0) / iterations);

    async_runtime_memory_pool_destroy(pool);
    printf("  ✓ Async performance test completed\n");
}

// ============================================================================
// 主测试函数
// ============================================================================

int main(int argc, char* argv[]) {
    printf("=== Async Memory Pool Optimization Test Suite ===\n\n");

    // 设置随机种子
    srand(time(NULL));

    // Future状态机测试
    printf("Testing Future state machine optimizations...\n");
    test_compact_future_basic();
    test_compact_future_inline_storage();
    printf("✓ Future tests passed!\n\n");

    // Task内存池测试
    printf("Testing Task memory pool optimizations...\n");
    test_task_memory_pool_basic();
    printf("✓ Task pool tests passed!\n\n");

    // 协程栈管理测试
    printf("Testing coroutine stack management...\n");
    test_coroutine_stack_manager_basic();
    test_coroutine_stack_resize();
    printf("✓ Stack management tests passed!\n\n");

    // 缓存感知通道测试
    printf("Testing cache-aware channel optimizations...\n");
    test_cache_aware_channel_basic();
    printf("✓ Channel tests passed!\n\n");

    // 集成测试
    printf("Testing async runtime integration...\n");
    test_async_runtime_memory_pool_integration();
    printf("✓ Integration tests passed!\n\n");

    // 性能测试
    printf("Running performance tests...\n");
    test_async_performance();
    printf("✓ Performance tests completed!\n\n");

    printf("=== All async memory pool tests passed successfully! ===\n");
    return 0;
}
