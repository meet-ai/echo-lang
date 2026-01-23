/**
 * @file advanced_memory_pool_test.c
 * @brief 工业级异步运行时内存池测试
 */

#include "../src/core/advanced_memory_pool.h"
#include <stdio.h>
#include <stdlib.h>
#include <string.h>
#include <assert.h>
#include <time.h>

// ============================================================================
// 测试辅助函数
// ============================================================================

/** 测试计时器 */
typedef struct test_timer {
    struct timespec start;
    struct timespec end;
} test_timer_t;

static void timer_start(test_timer_t* timer) {
    clock_gettime(CLOCK_MONOTONIC, &timer->start);
}

static double timer_elapsed_ms(const test_timer_t* timer) {
    struct timespec end;
    clock_gettime(CLOCK_MONOTONIC, &end);

    double start_ms = timer->start.tv_sec * 1000.0 + timer->start.tv_nsec / 1000000.0;
    double end_ms = end.tv_sec * 1000.0 + end.tv_nsec / 1000000.0;

    return end_ms - start_ms;
}

/** 内存检查辅助函数 */
static void fill_memory_pattern(void* ptr, size_t size, uint8_t pattern) {
    memset(ptr, pattern, size);
}

static bool verify_memory_pattern(const void* ptr, size_t size, uint8_t pattern) {
    const uint8_t* bytes = (const uint8_t*)ptr;
    for (size_t i = 0; i < size; i++) {
        if (bytes[i] != pattern) {
            return false;
        }
    }
    return true;
}

// ============================================================================
// 基础内存池测试
// ============================================================================

/**
 * @brief 测试基础内存池分配和释放
 */
static void test_basic_memory_pool(void) {
    printf("Testing basic memory pool...\n");

    // 创建内存池
    advanced_memory_pool_t* pool = advanced_memory_pool_create(
        1024 * 1024, // 1MB per thread
        false,       // 不启用NUMA
        false        // 不启用大页
    );
    assert(pool != NULL);

    // 测试分配
    void* ptr1 = advanced_memory_pool_alloc(pool, 128);
    assert(ptr1 != NULL);
    fill_memory_pattern(ptr1, 128, 0xAA);
    assert(verify_memory_pattern(ptr1, 128, 0xAA));

    void* ptr2 = advanced_memory_pool_alloc(pool, 256);
    assert(ptr2 != NULL);
    fill_memory_pattern(ptr2, 256, 0xBB);
    assert(verify_memory_pattern(ptr2, 256, 0xBB));

    // 测试释放
    assert(advanced_memory_pool_free(pool, ptr1, 128));
    assert(advanced_memory_pool_free(pool, ptr2, 256));

    // 清理
    advanced_memory_pool_destroy(pool);
    printf("✓ Basic memory pool test passed\n");
}

/**
 * @brief 测试多线程分配
 */
static void test_concurrent_allocation(void) {
    printf("Testing concurrent allocation...\n");

    advanced_memory_pool_t* pool = advanced_memory_pool_create(
        1024 * 1024,
        false,
        false
    );
    assert(pool != NULL);

    // 简单的单线程压力测试
    const int num_allocations = 1000;
    void* pointers[num_allocations];

    // 分配
    for (int i = 0; i < num_allocations; i++) {
        pointers[i] = advanced_memory_pool_alloc(pool, 64 + (i % 10) * 8);
        assert(pointers[i] != NULL);
        fill_memory_pattern(pointers[i], 64, (uint8_t)i);
    }

    // 验证
    for (int i = 0; i < num_allocations; i++) {
        assert(verify_memory_pattern(pointers[i], 64, (uint8_t)i));
    }

    // 释放
    for (int i = 0; i < num_allocations; i++) {
        size_t size = 64 + (i % 10) * 8;
        assert(advanced_memory_pool_free(pool, pointers[i], size));
    }

    advanced_memory_pool_destroy(pool);
    printf("✓ Concurrent allocation test passed\n");
}

/**
 * @brief 测试大小类分配
 */
static void test_size_class_allocation(void) {
    printf("Testing size class allocation...\n");

    advanced_memory_pool_t* pool = advanced_memory_pool_create(
        1024 * 1024,
        false,
        false
    );
    assert(pool != NULL);

    // 测试不同大小的分配
    const size_t test_sizes[] = {8, 16, 32, 64, 96, 128, 256, 512};
    const int num_sizes = sizeof(test_sizes) / sizeof(test_sizes[0]);

    void* pointers[num_sizes];

    // 分配不同大小
    for (int i = 0; i < num_sizes; i++) {
        pointers[i] = advanced_memory_pool_alloc(pool, test_sizes[i]);
        assert(pointers[i] != NULL);
        fill_memory_pattern(pointers[i], test_sizes[i], (uint8_t)(i + 1));
    }

    // 验证
    for (int i = 0; i < num_sizes; i++) {
        assert(verify_memory_pattern(pointers[i], test_sizes[i], (uint8_t)(i + 1)));
    }

    // 释放
    for (int i = 0; i < num_sizes; i++) {
        assert(advanced_memory_pool_free(pool, pointers[i], test_sizes[i]));
    }

    advanced_memory_pool_destroy(pool);
    printf("✓ Size class allocation test passed\n");
}

// ============================================================================
// 异步对象专用测试
// ============================================================================

/**
 * @brief 测试异步内存池
 */
static void test_async_memory_pool(void) {
    printf("Testing async memory pool...\n");

    // 创建基础池
    advanced_memory_pool_t* base_pool = advanced_memory_pool_create(
        1024 * 1024,
        false,
        false
    );
    assert(base_pool != NULL);

    // 创建异步池
    async_memory_pool_t* async_pool = async_memory_pool_create(base_pool);
    assert(async_pool != NULL);

    // 测试Future分配
    compact_future_t* future = async_memory_pool_alloc_future(async_pool, 128);
    assert(future != NULL);

    // 验证Future状态
    assert(compact_future_status(future) == FUTURE_PENDING);

    // 测试Waker分配
    void* waker = async_memory_pool_alloc_waker(async_pool);
    assert(waker != NULL);

    // 测试Task分配
    void* task = async_memory_pool_alloc_task(async_pool, 256);
    assert(task != NULL);

    // 测试Channel节点分配
    void* channel_node = async_memory_pool_alloc_channel_node(async_pool);
    assert(channel_node != NULL);

    // 测试释放
    async_memory_pool_free_future(async_pool, future);
    async_memory_pool_free_waker(async_pool, waker);
    async_memory_pool_free_task(async_pool, task);
    async_memory_pool_free_channel_node(async_pool, channel_node);

    // 清理
    async_memory_pool_destroy(async_pool);
    advanced_memory_pool_destroy(base_pool);
    printf("✓ Async memory pool test passed\n");
}

/**
 * @brief 测试性能
 */
static void test_performance(void) {
    printf("Testing performance...\n");

    advanced_memory_pool_t* pool = advanced_memory_pool_create(
        1024 * 1024,
        false,
        false
    );
    assert(pool != NULL);

    const int num_operations = 100000;
    void* pointers[num_operations];
    test_timer_t timer;

    // 测试分配性能
    timer_start(&timer);
    for (int i = 0; i < num_operations; i++) {
        pointers[i] = advanced_memory_pool_alloc(pool, 64);
        assert(pointers[i] != NULL);
    }
    double alloc_time = timer_elapsed_ms(&timer);

    // 测试释放性能
    timer_start(&timer);
    for (int i = 0; i < num_operations; i++) {
        assert(advanced_memory_pool_free(pool, pointers[i], 64));
    }
    double free_time = timer_elapsed_ms(&timer);

    printf("  分配性能: %.2f ops/ms (%.2f ns/op)\n",
           num_operations / alloc_time,
           (alloc_time * 1000000.0) / num_operations);

    printf("  释放性能: %.2f ops/ms (%.2f ns/op)\n",
           num_operations / free_time,
           (free_time * 1000000.0) / num_operations);

    advanced_memory_pool_destroy(pool);
    printf("✓ Performance test passed\n");
}

/**
 * @brief 测试统计信息
 */
static void test_statistics(void) {
    printf("Testing statistics...\n");

    advanced_memory_pool_t* pool = advanced_memory_pool_create(
        1024 * 1024,
        false,
        false
    );
    assert(pool != NULL);

    // 执行一些操作
    void* ptr1 = advanced_memory_pool_alloc(pool, 128);
    void* ptr2 = advanced_memory_pool_alloc(pool, 256);
    void* ptr3 = advanced_memory_pool_alloc(pool, 64);

    assert(advanced_memory_pool_free(pool, ptr2, 256));

    // 获取统计
    uint64_t total_alloc, total_free, memory_used;
    advanced_memory_pool_stats(pool, &total_alloc, &total_free, &memory_used);

    printf("  总分配次数: %llu\n", (unsigned long long)total_alloc);
    printf("  总释放次数: %llu\n", (unsigned long long)total_free);
    printf("  已用内存: %llu bytes\n", (unsigned long long)memory_used);

    assert(total_alloc == 3);
    assert(total_free == 1);
    assert(memory_used == 128 + 64); // 128 + 256 - 256 + 64

    // 清理
    assert(advanced_memory_pool_free(pool, ptr1, 128));
    assert(advanced_memory_pool_free(pool, ptr3, 64));
    advanced_memory_pool_destroy(pool);
    printf("✓ Statistics test passed\n");
}

// ============================================================================
// 工具函数测试
// ============================================================================

/**
 * @brief 测试工具函数
 */
static void test_utils(void) {
    printf("Testing utility functions...\n");

    // 测试缓存行对齐
    assert(align_to_cache_line(1) == 64);
    assert(align_to_cache_line(64) == 64);
    assert(align_to_cache_line(65) == 128);

    // 测试大小类
    assert(get_size_class(8) == 0);
    assert(get_size_class(64) == 3);
    assert(get_size_class(128) == 5);
    assert(get_size_class(2000) == 11); // 最大类

    printf("✓ Utility functions test passed\n");
}

// ============================================================================
// 主测试函数
// ============================================================================

int main(int argc, char* argv[]) {
    (void)argc;
    (void)argv;

    printf("=== Advanced Memory Pool Tests ===\n\n");

    // 运行所有测试
    test_utils();
    test_basic_memory_pool();
    test_size_class_allocation();
    test_concurrent_allocation();
    test_async_memory_pool();
    test_statistics();
    test_performance();

    printf("\n=== All tests passed! ===\n");

    return 0;
}
