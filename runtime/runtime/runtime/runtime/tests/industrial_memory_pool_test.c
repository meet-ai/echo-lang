/**
 * @file industrial_memory_pool_test.c
 * @brief 工业级内存池系统测试
 *
 * 测试工业级异步运行时内存池系统的核心功能：
 * - 线程本地对象池分配/释放
 * - 全局Slab分配器多尺寸类分配
 * - 内存回收和压缩
 * - 异步对象专用池
 */

#include <stdio.h>
#include <stdlib.h>
#include <string.h>
#include <pthread.h>
#include <unistd.h>
#include <time.h>
#include <assert.h>

#include "industrial_memory_pool.h"

// ============================================================================
// 测试辅助函数
// ============================================================================

/** 测试断言 */
#define TEST_ASSERT(condition, message) \
    do { \
        if (!(condition)) { \
            fprintf(stderr, "TEST FAILED: %s at %s:%d\n", message, __FILE__, __LINE__); \
            exit(1); \
        } \
    } while (0)

/** 性能计时器 */
typedef struct {
    struct timespec start;
    struct timespec end;
} timer_t;

static inline void timer_start(timer_t* timer) {
    clock_gettime(CLOCK_MONOTONIC, &timer->start);
}

static inline double timer_elapsed_ms(timer_t* timer) {
    clock_gettime(CLOCK_MONOTONIC, &timer->end);
    return (timer->end.tv_sec - timer->start.tv_sec) * 1000.0 +
           (timer->end.tv_nsec - timer->start.tv_nsec) / 1000000.0;
}

/** 随机数生成器 */
static unsigned int seed = 12345;
static inline unsigned int rand_r_custom(void) {
    seed = seed * 1103515245 + 12345;
    return seed;
}

// ============================================================================
// 基础功能测试
// ============================================================================

/**
 * @brief 测试线程本地对象池基础功能
 */
void test_thread_local_object_pool_basic(void) {
    printf("Testing ThreadLocalObjectPool basic functionality...\n");

    // 创建对象池配置
    object_pool_config_t config = OBJECT_POOL_DEFAULT_CONFIG(64); // 64字节对象

    // 创建对象池
    thread_local_object_pool_t* pool = thread_local_object_pool_create(&config);
    TEST_ASSERT(pool != NULL, "Failed to create thread local object pool");

    // 测试分配
    void* obj1 = thread_local_object_pool_allocate(pool);
    TEST_ASSERT(obj1 != NULL, "Failed to allocate object 1");

    void* obj2 = thread_local_object_pool_allocate(pool);
    TEST_ASSERT(obj2 != NULL, "Failed to allocate object 2");

    TEST_ASSERT(obj1 != obj2, "Allocated objects should be different");

    // 测试释放
    bool success1 = thread_local_object_pool_deallocate(pool, obj1);
    TEST_ASSERT(success1, "Failed to deallocate object 1");

    bool success2 = thread_local_object_pool_deallocate(pool, obj2);
    TEST_ASSERT(success2, "Failed to deallocate object 2");

    // 再次分配，应该复用之前释放的对象
    void* obj3 = thread_local_object_pool_allocate(pool);
    TEST_ASSERT(obj3 != NULL, "Failed to allocate object 3 after deallocation");

    // 获取统计信息
    size_t allocated_count, free_count;
    double hit_rate;
    thread_local_object_pool_get_stats(pool, &allocated_count, &free_count, &hit_rate);

    printf("  Pool stats: allocated=%zu, free=%zu, hit_rate=%.2f%%\n",
           allocated_count, free_count, hit_rate * 100.0);

    // 清理
    thread_local_object_pool_destroy(pool);
    printf("  ✓ ThreadLocalObjectPool basic test passed\n");
}

/**
 * @brief 测试全局Slab分配器基础功能
 */
void test_global_slab_allocator_basic(void) {
    printf("Testing GlobalSlabAllocator basic functionality...\n");

    // 创建Slab分配器配置
    slab_allocator_config_t config = SLAB_ALLOCATOR_DEFAULT_CONFIG();

    // 创建分配器
    global_slab_allocator_t* allocator = global_slab_allocator_create(&config);
    TEST_ASSERT(allocator != NULL, "Failed to create global slab allocator");

    // 测试不同大小的分配
    const size_t test_sizes[] = {8, 16, 32, 64, 96, 128, 256, 512};
    const int num_sizes = sizeof(test_sizes) / sizeof(test_sizes[0]);

    void* allocated_ptrs[32];
    int allocated_count = 0;

    // 分配各种大小的对象
    for (int i = 0; i < num_sizes; i++) {
        for (int j = 0; j < 4; j++) { // 每种大小分配4个
            void* ptr = global_slab_allocator_allocate(allocator, test_sizes[i]);
            TEST_ASSERT(ptr != NULL, "Failed to allocate memory");

            // 写入测试数据
            memset(ptr, (char)(i * 10 + j), test_sizes[i]);

            allocated_ptrs[allocated_count++] = ptr;
        }
    }

    // 验证数据完整性
    for (int i = 0; i < allocated_count; i++) {
        int expected_value = (i / 4) * 10 + (i % 4);
        size_t size = test_sizes[i / 4];
        unsigned char* data = (unsigned char*)allocated_ptrs[i];

        for (size_t j = 0; j < size; j++) {
            TEST_ASSERT(data[j] == (unsigned char)expected_value,
                       "Data corruption detected");
        }
    }

    // 释放所有分配的内存
    for (int i = 0; i < allocated_count; i++) {
        size_t size = test_sizes[i / 4];
        bool success = global_slab_allocator_deallocate(allocator, allocated_ptrs[i], size);
        TEST_ASSERT(success, "Failed to deallocate memory");
    }

    // 获取统计信息
    size_t total_allocated, total_free;
    double fragmentation_rate;
    global_slab_allocator_get_stats(allocator, &total_allocated, &total_free,
                                   &fragmentation_rate);

    printf("  Slab stats: allocated=%zu, free=%zu, fragmentation=%.2f%%\n",
           total_allocated, total_free, fragmentation_rate * 100.0);

    // 清理
    global_slab_allocator_destroy(allocator);
    printf("  ✓ GlobalSlabAllocator basic test passed\n");
}

/**
 * @brief 测试内存回收器基础功能
 */
void test_memory_reclaimer_basic(void) {
    printf("Testing MemoryReclaimer basic functionality...\n");

    // 创建回收器配置
    memory_reclamation_config_t config = MEMORY_RECLAMATION_DEFAULT_CONFIG();

    // 创建回收器
    memory_reclaimer_t* reclaimer = memory_reclaimer_create(&config);
    TEST_ASSERT(reclaimer != NULL, "Failed to create memory reclaimer");

    // 执行垃圾回收
    size_t reclaimed1 = memory_reclaimer_gc(reclaimer, true); // 强制GC
    printf("  First GC reclaimed: %zu bytes\n", reclaimed1);

    // 执行内存压缩
    size_t compacted = memory_reclaimer_compact(reclaimer);
    printf("  Compaction reclaimed: %zu bytes\n", compacted);

    // 再次执行GC
    size_t reclaimed2 = memory_reclaimer_gc(reclaimer, false);
    printf("  Second GC reclaimed: %zu bytes\n", reclaimed2);

    // 获取统计信息
    uint64_t gc_count;
    size_t total_reclaimed;
    double avg_pause_us;
    memory_reclaimer_get_stats(reclaimer, &gc_count, &total_reclaimed, &avg_pause_us);

    printf("  Reclaimer stats: gc_count=%llu, total_reclaimed=%zu, avg_pause=%.2f us\n",
           gc_count, total_reclaimed, avg_pause_us);

    // 清理
    memory_reclaimer_destroy(reclaimer);
    printf("  ✓ MemoryReclaimer basic test passed\n");
}

// ============================================================================
// 工业级内存池系统测试
// ============================================================================

/**
 * @brief 测试工业级内存池系统基础功能
 */
void test_industrial_memory_pool_basic(void) {
    printf("Testing IndustrialMemoryPool basic functionality...\n");

    // 创建系统配置
    industrial_memory_pool_config_t config = INDUSTRIAL_MEMORY_POOL_DEFAULT_CONFIG();

    // 创建内存池系统
    industrial_memory_pool_t* pool = industrial_memory_pool_create(&config);
    TEST_ASSERT(pool != NULL, "Failed to create industrial memory pool");

    // 测试Task对象分配/释放
    void* task1 = industrial_memory_pool_allocate_task(pool);
    TEST_ASSERT(task1 != NULL, "Failed to allocate task");

    void* task2 = industrial_memory_pool_allocate_task(pool);
    TEST_ASSERT(task2 != NULL, "Failed to allocate second task");

    bool success1 = industrial_memory_pool_deallocate_task(pool, task1);
    TEST_ASSERT(success1, "Failed to deallocate task1");

    bool success2 = industrial_memory_pool_deallocate_task(pool, task2);
    TEST_ASSERT(success2, "Failed to deallocate task2");

    // 测试Waker对象分配/释放
    void* waker1 = industrial_memory_pool_allocate_waker(pool);
    TEST_ASSERT(waker1 != NULL, "Failed to allocate waker");

    void* waker2 = industrial_memory_pool_allocate_waker(pool);
    TEST_ASSERT(waker2 != NULL, "Failed to allocate second waker");

    bool success3 = industrial_memory_pool_deallocate_waker(pool, waker1);
    TEST_ASSERT(success3, "Failed to deallocate waker1");

    bool success4 = industrial_memory_pool_deallocate_waker(pool, waker2);
    TEST_ASSERT(success4, "Failed to deallocate waker2");

    // 测试Channel节点分配/释放
    void* node1 = industrial_memory_pool_allocate_channel_node(pool);
    TEST_ASSERT(node1 != NULL, "Failed to allocate channel node");

    void* node2 = industrial_memory_pool_allocate_channel_node(pool);
    TEST_ASSERT(node2 != NULL, "Failed to allocate second channel node");

    bool success5 = industrial_memory_pool_deallocate_channel_node(pool, node1);
    TEST_ASSERT(success5, "Failed to deallocate node1");

    bool success6 = industrial_memory_pool_deallocate_channel_node(pool, node2);
    TEST_ASSERT(success6, "Failed to deallocate node2");

    // 测试通用内存分配
    void* mem1 = industrial_memory_pool_allocate(pool, 256);
    TEST_ASSERT(mem1 != NULL, "Failed to allocate general memory");

    void* mem2 = industrial_memory_pool_allocate(pool, 1024);
    TEST_ASSERT(mem2 != NULL, "Failed to allocate second general memory");

    bool success7 = industrial_memory_pool_deallocate(pool, mem1, 256);
    TEST_ASSERT(success7, "Failed to deallocate general memory 1");

    bool success8 = industrial_memory_pool_deallocate(pool, mem2, 1024);
    TEST_ASSERT(success8, "Failed to deallocate general memory 2");

    // 测试GC
    size_t reclaimed = industrial_memory_pool_gc(pool);
    printf("  GC reclaimed: %zu bytes\n", reclaimed);

    // 获取系统统计信息
    size_t total_allocated, total_free;
    double fragmentation_rate;
    uint64_t gc_count;
    industrial_memory_pool_get_stats(pool, &total_allocated, &total_free,
                                   &fragmentation_rate, &gc_count);

    printf("  System stats: allocated=%zu, free=%zu, fragmentation=%.2f%%, gc_count=%llu\n",
           total_allocated, total_free, fragmentation_rate * 100.0, gc_count);

    // 清理
    industrial_memory_pool_destroy(pool);
    printf("  ✓ IndustrialMemoryPool basic test passed\n");
}

// ============================================================================
// 性能测试
// ============================================================================

/**
 * @brief 性能测试：线程本地对象池分配速度
 */
void test_performance_thread_local_pool(void) {
    printf("Testing ThreadLocalObjectPool performance...\n");

    object_pool_config_t config = OBJECT_POOL_DEFAULT_CONFIG(64);
    thread_local_object_pool_t* pool = thread_local_object_pool_create(&config);
    TEST_ASSERT(pool != NULL, "Failed to create pool for performance test");

    const int iterations = 1000000; // 100万次分配/释放
    void* objects[iterations];

    // 预热
    for (int i = 0; i < 1000; i++) {
        void* obj = thread_local_object_pool_allocate(pool);
        thread_local_object_pool_deallocate(pool, obj);
    }

    // 性能测试：连续分配
    timer_t timer;
    timer_start(&timer);

    for (int i = 0; i < iterations; i++) {
        objects[i] = thread_local_object_pool_allocate(pool);
        TEST_ASSERT(objects[i] != NULL, "Allocation failed in performance test");
    }

    double alloc_time = timer_elapsed_ms(&timer);
    printf("  Allocated %d objects in %.2f ms (%.2f ns per allocation)\n",
           iterations, alloc_time, (alloc_time * 1000000.0) / iterations);

    // 性能测试：连续释放
    timer_start(&timer);

    for (int i = 0; i < iterations; i++) {
        bool success = thread_local_object_pool_deallocate(pool, objects[i]);
        TEST_ASSERT(success, "Deallocation failed in performance test");
    }

    double dealloc_time = timer_elapsed_ms(&timer);
    printf("  Deallocated %d objects in %.2f ms (%.2f ns per deallocation)\n",
           iterations, dealloc_time, (dealloc_time * 1000000.0) / iterations);

    // 混合操作测试
    timer_start(&timer);

    for (int i = 0; i < iterations; i++) {
        void* obj = thread_local_object_pool_allocate(pool);
        thread_local_object_pool_deallocate(pool, obj);
    }

    double mixed_time = timer_elapsed_ms(&timer);
    printf("  Mixed operations: %d alloc/dealloc pairs in %.2f ms (%.2f ns per pair)\n",
           iterations, mixed_time, (mixed_time * 1000000.0) / iterations);

    thread_local_object_pool_destroy(pool);
    printf("  ✓ ThreadLocalObjectPool performance test completed\n");
}

/**
 * @brief 性能测试：全局Slab分配器
 */
void test_performance_global_slab(void) {
    printf("Testing GlobalSlabAllocator performance...\n");

    slab_allocator_config_t config = SLAB_ALLOCATOR_DEFAULT_CONFIG();
    global_slab_allocator_t* allocator = global_slab_allocator_create(&config);
    TEST_ASSERT(allocator != NULL, "Failed to create allocator for performance test");

    const int iterations = 500000; // 50万次分配/释放
    const size_t test_sizes[] = {64, 128, 256, 512};
    const int num_sizes = sizeof(test_sizes) / sizeof(test_sizes[0]);

    void* objects[iterations];

    // 预热
    for (int i = 0; i < 1000; i++) {
        void* obj = global_slab_allocator_allocate(allocator, 64);
        global_slab_allocator_deallocate(allocator, obj, 64);
    }

    // 性能测试
    timer_t timer;
    timer_start(&timer);

    for (int i = 0; i < iterations; i++) {
        size_t size = test_sizes[i % num_sizes];
        objects[i] = global_slab_allocator_allocate(allocator, size);
        TEST_ASSERT(objects[i] != NULL, "Allocation failed in performance test");
    }

    double alloc_time = timer_elapsed_ms(&timer);
    printf("  Allocated %d objects in %.2f ms (%.2f ns per allocation)\n",
           iterations, alloc_time, (alloc_time * 1000000.0) / iterations);

    // 释放测试
    timer_start(&timer);

    for (int i = 0; i < iterations; i++) {
        size_t size = test_sizes[i % num_sizes];
        bool success = global_slab_allocator_deallocate(allocator, objects[i], size);
        TEST_ASSERT(success, "Deallocation failed in performance test");
    }

    double dealloc_time = timer_elapsed_ms(&timer);
    printf("  Deallocated %d objects in %.2f ms (%.2f ns per deallocation)\n",
           iterations, dealloc_time, (dealloc_time * 1000000.0) / iterations);

    global_slab_allocator_destroy(allocator);
    printf("  ✓ GlobalSlabAllocator performance test completed\n");
}

// ============================================================================
// 并发测试
// ============================================================================

/** 并发测试参数 */
#define NUM_THREADS 4
#define OPS_PER_THREAD 100000

/** 线程参数结构 */
typedef struct {
    industrial_memory_pool_t* pool;
    int thread_id;
    int operations;
    double* results; // [alloc_time, dealloc_time, mixed_time]
} thread_args_t;

/**
 * @brief 并发测试线程函数
 */
void* concurrent_test_thread(void* arg) {
    thread_args_t* args = (thread_args_t*)arg;
    industrial_memory_pool_t* pool = args->pool;

    void* objects[OPS_PER_THREAD];
    timer_t timer;

    // 分配测试
    timer_start(&timer);
    for (int i = 0; i < args->operations; i++) {
        objects[i] = industrial_memory_pool_allocate_task(pool);
    }
    double alloc_time = timer_elapsed_ms(&timer);

    // 释放测试
    timer_start(&timer);
    for (int i = 0; i < args->operations; i++) {
        industrial_memory_pool_deallocate_task(pool, objects[i]);
    }
    double dealloc_time = timer_elapsed_ms(&timer);

    // 混合操作测试
    timer_start(&timer);
    for (int i = 0; i < args->operations; i++) {
        void* obj = industrial_memory_pool_allocate_task(pool);
        industrial_memory_pool_deallocate_task(pool, obj);
    }
    double mixed_time = timer_elapsed_ms(&timer);

    args->results[0] = alloc_time;
    args->results[1] = dealloc_time;
    args->results[2] = mixed_time;

    return NULL;
}

/**
 * @brief 并发性能测试
 */
void test_concurrent_performance(void) {
    printf("Testing concurrent performance with %d threads...\n", NUM_THREADS);

    industrial_memory_pool_config_t config = INDUSTRIAL_MEMORY_POOL_DEFAULT_CONFIG();
    industrial_memory_pool_t* pool = industrial_memory_pool_create(&config);
    TEST_ASSERT(pool != NULL, "Failed to create pool for concurrent test");

    pthread_t threads[NUM_THREADS];
    thread_args_t thread_args[NUM_THREADS];
    double results[NUM_THREADS][3]; // [alloc_time, dealloc_time, mixed_time]

    // 创建线程
    for (int i = 0; i < NUM_THREADS; i++) {
        thread_args[i].pool = pool;
        thread_args[i].thread_id = i;
        thread_args[i].operations = OPS_PER_THREAD;
        thread_args[i].results = results[i];

        int ret = pthread_create(&threads[i], NULL, concurrent_test_thread, &thread_args[i]);
        TEST_ASSERT(ret == 0, "Failed to create thread");
    }

    // 等待所有线程完成
    for (int i = 0; i < NUM_THREADS; i++) {
        pthread_join(threads[i], NULL);
    }

    // 汇总结果
    double total_alloc_time = 0, total_dealloc_time = 0, total_mixed_time = 0;
    for (int i = 0; i < NUM_THREADS; i++) {
        total_alloc_time += results[i][0];
        total_dealloc_time += results[i][1];
        total_mixed_time += results[i][2];
    }

    int total_ops = NUM_THREADS * OPS_PER_THREAD;
    printf("  Concurrent performance (%d threads, %d ops each):\n", NUM_THREADS, OPS_PER_THREAD);
    printf("    Total allocations: %.2f ms (%.2f ns/op)\n",
           total_alloc_time, (total_alloc_time * 1000000.0) / total_ops);
    printf("    Total deallocations: %.2f ms (%.2f ns/op)\n",
           total_dealloc_time, (total_dealloc_time * 1000000.0) / total_ops);
    printf("    Total mixed operations: %.2f ms (%.2f ns/op)\n",
           total_mixed_time, (total_mixed_time * 1000000.0) / total_ops);

    // 计算吞吐量
    double total_time_seconds = (total_mixed_time / 1000.0);
    double throughput = total_ops / total_time_seconds;
    printf("    Throughput: %.0f ops/sec\n", throughput);

    industrial_memory_pool_destroy(pool);
    printf("  ✓ Concurrent performance test completed\n");
}

// ============================================================================
// 主测试函数
// ============================================================================

int main(int argc, char* argv[]) {
    printf("=== Industrial Memory Pool System Test Suite ===\n\n");

    // 设置随机种子
    srand(time(NULL));

    // 基础功能测试
    printf("Running basic functionality tests...\n");
    test_thread_local_object_pool_basic();
    test_global_slab_allocator_basic();
    test_memory_reclaimer_basic();
    test_industrial_memory_pool_basic();
    printf("✓ All basic tests passed!\n\n");

    // 性能测试
    printf("Running performance tests...\n");
    test_performance_thread_local_pool();
    test_performance_global_slab();
    printf("✓ Performance tests completed!\n\n");

    // 并发测试
    printf("Running concurrent tests...\n");
    test_concurrent_performance();
    printf("✓ Concurrent tests completed!\n\n");

    printf("=== All tests passed successfully! ===\n");
    return 0;
}
