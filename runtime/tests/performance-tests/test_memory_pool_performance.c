#include "../../infrastructure/core/industrial_memory_pool.h"
#include "../../shared/types/common_types.h"
#include <stdio.h>
#include <stdlib.h>
#include <string.h>
#include <time.h>
#include <pthread.h>
#include <unistd.h>
#include <sys/time.h>

// 测试配置
#define NUM_ITERATIONS 1000000
#define NUM_THREADS 8
#define ALLOCATION_SIZES_COUNT 5
#define ALLOCATION_SIZES {32, 128, 512, 2048, 8192}

// 性能测试结果
typedef struct {
    double allocation_time;
    double free_time;
    double total_time;
    size_t allocations_per_second;
    size_t memory_efficiency;
    size_t fragmentation_ratio;
} PerformanceResult;

// 线程参数
typedef struct {
    IndustrialMemoryPool* pool;
    size_t allocation_size;
    int thread_id;
    PerformanceResult* result;
} ThreadArgs;

// 获取当前时间（微秒）
static long long get_current_time_us(void) {
    struct timeval tv;
    gettimeofday(&tv, NULL);
    return (long long)tv.tv_sec * 1000000 + tv.tv_usec;
}

// 单线程性能测试
void* single_thread_test(void* arg) {
    ThreadArgs* args = (ThreadArgs*)arg;
    IndustrialMemoryPool* pool = args->pool;
    size_t size = args->allocation_size;
    PerformanceResult* result = args->result;

    // 预分配指针数组
    void** pointers = (void**)malloc(sizeof(void*) * NUM_ITERATIONS / NUM_THREADS);
    if (!pointers) {
        fprintf(stderr, "Failed to allocate pointers array\n");
        return NULL;
    }

    // 测试分配性能
    long long start_time = get_current_time_us();
    for (int i = 0; i < NUM_ITERATIONS / NUM_THREADS; i++) {
        pointers[i] = industrial_memory_pool_allocate(pool, size, "performance_test");
        if (!pointers[i]) {
            fprintf(stderr, "Allocation failed at iteration %d\n", i);
            break;
        }
        // 写入一些数据来模拟实际使用
        memset(pointers[i], 0xAA, size);
    }
    long long alloc_end_time = get_current_time_us();

    // 测试释放性能
    for (int i = 0; i < NUM_ITERATIONS / NUM_THREADS; i++) {
        if (pointers[i]) {
            industrial_memory_pool_free(pool, pointers[i], "performance_test");
        }
    }
    long long free_end_time = get_current_time_us();

    // 计算结果
    result->allocation_time = (alloc_end_time - start_time) / 1000000.0; // 秒
    result->free_time = (free_end_time - alloc_end_time) / 1000000.0;    // 秒
    result->total_time = result->allocation_time + result->free_time;
    result->allocations_per_second = (NUM_ITERATIONS / NUM_THREADS) / result->total_time;

    free(pointers);
    return NULL;
}

// 多线程性能测试
void run_multi_thread_performance_test(IndustrialMemoryPool* pool) {
    printf("=== 多线程性能测试 ===\n");

    pthread_t threads[NUM_THREADS];
    ThreadArgs thread_args[NUM_THREADS];
    PerformanceResult results[NUM_THREADS];

    size_t allocation_sizes[ALLOCATION_SIZES_COUNT] = ALLOCATION_SIZES;

    for (int size_idx = 0; size_idx < ALLOCATION_SIZES_COUNT; size_idx++) {
        size_t size = allocation_sizes[size_idx];
        printf("测试分配大小: %zu 字节\n", size);

        // 创建线程
        for (int i = 0; i < NUM_THREADS; i++) {
            thread_args[i].pool = pool;
            thread_args[i].allocation_size = size;
            thread_args[i].thread_id = i;
            thread_args[i].result = &results[i];

            if (pthread_create(&threads[i], NULL, single_thread_test, &thread_args[i]) != 0) {
                fprintf(stderr, "Failed to create thread %d\n", i);
                return;
            }
        }

        // 等待所有线程完成
        for (int i = 0; i < NUM_THREADS; i++) {
            pthread_join(threads[i], NULL);
        }

        // 汇总结果
        double total_alloc_time = 0;
        double total_free_time = 0;
        double total_time = 0;
        size_t total_allocations_per_second = 0;

        for (int i = 0; i < NUM_THREADS; i++) {
            total_alloc_time += results[i].allocation_time;
            total_free_time += results[i].free_time;
            total_time += results[i].total_time;
            total_allocations_per_second += results[i].allocations_per_second;
        }

        printf("  平均分配时间: %.3f 秒\n", total_alloc_time / NUM_THREADS);
        printf("  平均释放时间: %.3f 秒\n", total_free_time / NUM_THREADS);
        printf("  总时间: %.3f 秒\n", total_time / NUM_THREADS);
        printf("  每秒分配次数: %zu\n", total_allocations_per_second);
        printf("  平均每次分配时间: %.3f 微秒\n",
               (total_alloc_time / NUM_THREADS) * 1000000 / (NUM_ITERATIONS / NUM_THREADS));
    }
}

// 内存效率测试
void run_memory_efficiency_test(IndustrialMemoryPool* pool) {
    printf("=== 内存效率测试 ===\n");

    // 测试不同的分配模式
    const int test_sizes[] = {32, 64, 128, 256, 512, 1024, 2048, 4096};
    const int num_allocations = 10000;

    for (int size_idx = 0; size_idx < sizeof(test_sizes) / sizeof(test_sizes[0]); size_idx++) {
        size_t size = test_sizes[size_idx];
        printf("测试分配大小: %zu 字节\n", size);

        // 执行大量分配
        void** pointers = (void**)malloc(sizeof(void*) * num_allocations);
        if (!pointers) continue;

        long long start_time = get_current_time_us();
        for (int i = 0; i < num_allocations; i++) {
            pointers[i] = industrial_memory_pool_allocate(pool, size, "efficiency_test");
            if (!pointers[i]) {
                fprintf(stderr, "Allocation failed at %d\n", i);
                break;
            }
        }
        long long alloc_time = get_current_time_us();

        // 获取统计信息
        IndustrialMemoryPoolStats stats;
        industrial_memory_pool_get_stats(pool, &stats);

        printf("  已分配字节: %zu\n", stats.total_allocated_bytes);
        printf("  内存池总大小: %zu\n", stats.pool_total_size);
        printf("  内存利用率: %.2f%%\n",
               (double)stats.total_allocated_bytes / stats.pool_total_size * 100);
        printf("  分配速度: %.0f 次/秒\n",
               num_allocations / ((alloc_time - start_time) / 1000000.0));

        // 释放内存
        for (int i = 0; i < num_allocations; i++) {
            if (pointers[i]) {
                industrial_memory_pool_free(pool, pointers[i], "efficiency_test");
            }
        }

        free(pointers);
    }
}

// 内存碎片测试
void run_fragmentation_test(IndustrialMemoryPool* pool) {
    printf("=== 内存碎片测试 ===\n");

    const int num_blocks = 1000;
    void** blocks = (void**)malloc(sizeof(void*) * num_blocks);

    // 分配不同大小的块
    for (int i = 0; i < num_blocks; i++) {
        size_t size = 32 + (i % 10) * 32; // 32, 64, 96, ..., 352 字节循环
        blocks[i] = industrial_memory_pool_allocate(pool, size, "fragmentation_test");
    }

    // 随机释放一部分块（模拟真实使用模式）
    srand(time(NULL));
    for (int i = 0; i < num_blocks / 2; i++) {
        int index = rand() % num_blocks;
        if (blocks[index]) {
            industrial_memory_pool_free(pool, blocks[index], "fragmentation_test");
            blocks[index] = NULL;
        }
    }

    // 获取统计信息
    IndustrialMemoryPoolStats stats;
    industrial_memory_pool_get_stats(pool, &stats);

    printf("碎片测试结果:\n");
    printf("  总分配字节: %zu\n", stats.total_allocated_bytes);
    printf("  内存池总大小: %zu\n", stats.pool_total_size);
    printf("  当前利用率: %.2f%%\n",
           (double)stats.total_allocated_bytes / stats.pool_total_size * 100);

    // 清理
    for (int i = 0; i < num_blocks; i++) {
        if (blocks[i]) {
            industrial_memory_pool_free(pool, blocks[i], "fragmentation_test");
        }
    }
    free(blocks);
}

// 长时间运行稳定性测试
void run_stability_test(IndustrialMemoryPool* pool) {
    printf("=== 稳定性测试 (30秒) ===\n");

    const int duration_seconds = 30;
    time_t start_time = time(NULL);
    int allocation_count = 0;
    int free_count = 0;

    // 使用一个动态数组来跟踪分配的指针
    void** active_allocations = NULL;
    size_t active_count = 0;
    size_t active_capacity = 1000;

    active_allocations = (void**)malloc(sizeof(void*) * active_capacity);

    while (time(NULL) - start_time < duration_seconds) {
        // 随机决定是分配还是释放
        if (active_count == 0 || (rand() % 100) < 70) { // 70% 概率分配
            // 分配
            size_t size = 32 + (rand() % 10) * 32; // 32-352 字节随机
            void* ptr = industrial_memory_pool_allocate(pool, size, "stability_test");

            if (ptr) {
                if (active_count >= active_capacity) {
                    active_capacity *= 2;
                    active_allocations = (void**)realloc(active_allocations,
                                                        sizeof(void*) * active_capacity);
                }
                active_allocations[active_count++] = ptr;
                allocation_count++;
            }
        } else {
            // 释放
            if (active_count > 0) {
                int index = rand() % active_count;
                industrial_memory_pool_free(pool, active_allocations[index], "stability_test");
                active_allocations[index] = active_allocations[--active_count];
                free_count++;
            }
        }

        // 短暂休眠以避免CPU占用过高
        usleep(1000); // 1ms
    }

    // 清理剩余分配
    for (size_t i = 0; i < active_count; i++) {
        industrial_memory_pool_free(pool, active_allocations[i], "stability_test");
    }

    free(active_allocations);

    IndustrialMemoryPoolStats stats;
    industrial_memory_pool_get_stats(pool, &stats);

    printf("稳定性测试结果:\n");
    printf("  总分配次数: %d\n", allocation_count);
    printf("  总释放次数: %d\n", free_count);
    printf("  最终内存使用: %zu 字节\n", stats.total_allocated_bytes);
    printf("  测试持续时间: %d 秒\n", duration_seconds);
    printf("  平均每秒操作: %.1f 次\n", (allocation_count + free_count) / (double)duration_seconds);
}

int main(int argc, char* argv[]) {
    printf("工业级内存池性能测试\n");
    printf("================================\n");

    // 创建内存池配置
    IndustrialMemoryPoolConfig config = {
        .initial_size = 64 * 1024 * 1024,  // 64MB
        .max_size = 512 * 1024 * 1024,     // 512MB
        .slab_sizes = {32, 64, 128, 256, 512, 1024, 2048, 4096, 8192, 16384},
        .num_slab_sizes = 10,
        .enable_gc = true,
        .gc_threshold = 0.8,
        .enable_metrics = true
    };

    // 创建内存池
    IndustrialMemoryPool* pool = industrial_memory_pool_create(&config);
    if (!pool) {
        fprintf(stderr, "Failed to create industrial memory pool\n");
        return 1;
    }

    printf("内存池创建成功\n");

    // 运行各种测试
    run_multi_thread_performance_test(pool);
    printf("\n");

    run_memory_efficiency_test(pool);
    printf("\n");

    run_fragmentation_test(pool);
    printf("\n");

    run_stability_test(pool);
    printf("\n");

    // 获取最终统计信息
    IndustrialMemoryPoolStats final_stats;
    industrial_memory_pool_get_stats(pool, &final_stats);

    printf("=== 最终统计信息 ===\n");
    printf("总分配字节: %zu\n", final_stats.total_allocated_bytes);
    printf("总释放字节: %zu\n", final_stats.total_freed_bytes);
    printf("当前使用字节: %zu\n", final_stats.current_allocated_bytes);
    printf("内存池总大小: %zu\n", final_stats.pool_total_size);
    printf("内存利用率: %.2f%%\n",
           (double)final_stats.current_allocated_bytes / final_stats.pool_total_size * 100);
    printf("GC 执行次数: %zu\n", final_stats.gc_cycles);
    printf("平均GC 暂停时间: %.3f 毫秒\n", final_stats.avg_gc_pause_ms);

    // 销毁内存池
    industrial_memory_pool_destroy(pool);

    printf("性能测试完成\n");
    return 0;
}
