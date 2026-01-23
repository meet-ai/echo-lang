/**
 * @file basic_usage.c
 * @brief 工业级内存池系统基础使用示例
 *
 * 展示如何使用工业级异步运行时内存池系统进行基本的内存管理操作。
 */

#include <stdio.h>
#include <stdlib.h>
#include <string.h>
#include <time.h>

#include "echo/industrial_memory_pool.h"

/**
 * @brief 模拟异步Task结构
 */
typedef struct {
    int id;
    char name[32];
    int priority;
    void* data;
} async_task_t;

/**
 * @brief 模拟异步Future结构
 */
typedef struct {
    int result;
    char error_msg[64];
    int status;
} async_future_t;

/**
 * @brief 模拟Channel消息结构
 */
typedef struct {
    int sender_id;
    int message_type;
    char payload[128];
} channel_message_t;

/**
 * @brief 示例：基础内存池使用
 */
void example_basic_memory_pool(void) {
    printf("=== Basic Memory Pool Usage Example ===\n");

    // 1. 创建内存池系统
    industrial_memory_pool_config_t config = INDUSTRIAL_MEMORY_POOL_DEFAULT_CONFIG();
    industrial_memory_pool_t* pool = industrial_memory_pool_create(&config);

    if (!pool) {
        fprintf(stderr, "Failed to create memory pool\n");
        return;
    }

    printf("✓ Memory pool system created successfully\n");

    // 2. 分配和使用Task对象
    printf("\n--- Task Object Management ---\n");

    async_task_t* task = (async_task_t*)industrial_memory_pool_allocate_task(pool);
    if (task) {
        // 初始化Task
        task->id = 1;
        strcpy(task->name, "Example Task");
        task->priority = 5;
        task->data = NULL;

        printf("✓ Allocated Task: ID=%d, Name='%s', Priority=%d\n",
               task->id, task->name, task->priority);

        // 使用Task...
        // 模拟一些工作
        task->priority = 8; // 提升优先级

        // 释放Task
        industrial_memory_pool_deallocate_task(pool, task);
        printf("✓ Task deallocated\n");
    }

    // 3. 分配和使用Waker对象
    printf("\n--- Waker Object Management ---\n");

    void* waker = industrial_memory_pool_allocate_waker(pool);
    if (waker) {
        printf("✓ Allocated Waker object at %p\n", waker);

        // 在实际应用中，Waker用于唤醒异步任务
        // 这里我们只是演示分配/释放

        industrial_memory_pool_deallocate_waker(pool, waker);
        printf("✓ Waker deallocated\n");
    }

    // 4. 分配和使用Channel节点
    printf("\n--- Channel Node Management ---\n");

    channel_message_t* msg = (channel_message_t*)industrial_memory_pool_allocate_channel_node(pool);
    if (msg) {
        // 初始化消息
        msg->sender_id = 12345;
        msg->message_type = 1;
        strcpy(msg->payload, "Hello from memory pool!");

        printf("✓ Allocated Channel Message: Sender=%d, Type=%d, Payload='%s'\n",
               msg->sender_id, msg->message_type, msg->payload);

        industrial_memory_pool_deallocate_channel_node(pool, msg);
        printf("✓ Channel message deallocated\n");
    }

    // 5. 通用内存分配
    printf("\n--- General Memory Allocation ---\n");

    // 分配不同大小的内存块
    size_t sizes[] = {64, 256, 1024, 4096};
    void* blocks[4];

    for (int i = 0; i < 4; i++) {
        blocks[i] = industrial_memory_pool_allocate(pool, sizes[i]);
        if (blocks[i]) {
            // 初始化内存
            memset(blocks[i], 0xAA, sizes[i]);
            printf("✓ Allocated %zu bytes at %p\n", sizes[i], blocks[i]);
        }
    }

    // 释放内存块
    for (int i = 0; i < 4; i++) {
        if (blocks[i]) {
            industrial_memory_pool_deallocate(pool, blocks[i], sizes[i]);
            printf("✓ Deallocated %zu bytes\n", sizes[i]);
        }
    }

    // 6. 执行垃圾回收
    printf("\n--- Memory Reclamation ---\n");

    size_t reclaimed = industrial_memory_pool_gc(pool);
    printf("✓ Garbage collection reclaimed %zu bytes\n", reclaimed);

    // 7. 获取统计信息
    printf("\n--- System Statistics ---\n");

    size_t total_allocated, total_free;
    double fragmentation_rate;
    uint64_t gc_count;

    industrial_memory_pool_get_stats(pool, &total_allocated, &total_free,
                                   &fragmentation_rate, &gc_count);

    printf("System Stats:\n");
    printf("  Total Allocated: %zu bytes\n", total_allocated);
    printf("  Total Free: %zu bytes\n", total_free);
    printf("  Fragmentation Rate: %.2f%%\n", fragmentation_rate * 100.0);
    printf("  GC Cycles: %llu\n", gc_count);

    // 8. 清理资源
    industrial_memory_pool_destroy(pool);
    printf("\n✓ Memory pool system destroyed\n");
}

/**
 * @brief 示例：批量操作优化
 */
void example_batch_operations(void) {
    printf("\n=== Batch Operations Example ===\n");

    industrial_memory_pool_config_t config = INDUSTRIAL_MEMORY_POOL_DEFAULT_CONFIG();
    industrial_memory_pool_t* pool = industrial_memory_pool_create(&config);

    if (!pool) {
        fprintf(stderr, "Failed to create memory pool for batch operations\n");
        return;
    }

    const int BATCH_SIZE = 1000;

    // 批量分配Task对象
    printf("Performing batch allocation of %d tasks...\n", BATCH_SIZE);

    async_task_t* tasks[BATCH_SIZE];

    clock_t start = clock();
    for (int i = 0; i < BATCH_SIZE; i++) {
        tasks[i] = (async_task_t*)industrial_memory_pool_allocate_task(pool);
        if (tasks[i]) {
            tasks[i]->id = i;
            sprintf(tasks[i]->name, "Task_%d", i);
            tasks[i]->priority = i % 10;
        }
    }
    clock_t alloc_end = clock();

    // 模拟使用
    for (int i = 0; i < BATCH_SIZE; i++) {
        if (tasks[i]) {
            tasks[i]->priority += 1; // 简单操作
        }
    }

    // 批量释放
    printf("Performing batch deallocation of %d tasks...\n", BATCH_SIZE);

    for (int i = 0; i < BATCH_SIZE; i++) {
        if (tasks[i]) {
            industrial_memory_pool_deallocate_task(pool, tasks[i]);
        }
    }
    clock_t dealloc_end = clock();

    // 计算性能
    double alloc_time = (double)(alloc_end - start) / CLOCKS_PER_SEC * 1000.0;
    double dealloc_time = (double)(dealloc_end - alloc_end) / CLOCKS_PER_SEC * 1000.0;

    printf("Performance Results:\n");
    printf("  Allocation: %.2f ms (%d tasks)\n", alloc_time, BATCH_SIZE);
    printf("  Deallocation: %.2f ms (%d tasks)\n", dealloc_time, BATCH_SIZE);
    printf("  Avg alloc time: %.2f ns per task\n", (alloc_time * 1000000.0) / BATCH_SIZE);
    printf("  Avg dealloc time: %.2f ns per task\n", (dealloc_time * 1000000.0) / BATCH_SIZE);

    industrial_memory_pool_destroy(pool);
    printf("✓ Batch operations example completed\n");
}

/**
 * @brief 示例：内存池生命周期管理
 */
void example_lifecycle_management(void) {
    printf("\n=== Lifecycle Management Example ===\n");

    // 创建配置
    industrial_memory_pool_config_t config = INDUSTRIAL_MEMORY_POOL_DEFAULT_CONFIG();

    // 可以自定义配置
    config.task_pool_config.object_size = 128;  // Task大小128B
    config.waker_pool_config.object_size = 64;  // Waker大小64B
    config.channel_node_pool_config.object_size = 96; // Channel节点96B
    config.slab_config.enable_huge_pages = true; // 启用大页支持
    config.reclamation_config.gc_threshold = 0.7; // 70%使用率触发GC

    printf("Creating memory pool with custom configuration...\n");
    industrial_memory_pool_t* pool = industrial_memory_pool_create(&config);

    if (!pool) {
        fprintf(stderr, "Failed to create memory pool with custom config\n");
        return;
    }

    printf("✓ Memory pool created with custom configuration\n");

    // 模拟应用程序运行周期
    printf("Simulating application lifecycle...\n");

    for (int cycle = 1; cycle <= 5; cycle++) {
        printf("\n-- Cycle %d --\n", cycle);

        // 分配一些对象
        void* tasks[10];
        void* wakers[5];
        void* nodes[3];

        for (int i = 0; i < 10; i++) {
            tasks[i] = industrial_memory_pool_allocate_task(pool);
        }
        for (int i = 0; i < 5; i++) {
            wakers[i] = industrial_memory_pool_allocate_waker(pool);
        }
        for (int i = 0; i < 3; i++) {
            nodes[i] = industrial_memory_pool_allocate_channel_node(pool);
        }

        printf("  Allocated: %d tasks, %d wakers, %d nodes\n", 10, 5, 3);

        // 模拟工作负载
        usleep(10000); // 10ms

        // 释放部分对象（模拟对象生命周期）
        for (int i = 0; i < 7; i++) { // 释放70%的tasks
            industrial_memory_pool_deallocate_task(pool, tasks[i]);
        }
        for (int i = 0; i < 3; i++) { // 释放60%的wakers
            industrial_memory_pool_deallocate_waker(pool, wakers[i]);
        }

        printf("  Deallocated: %d tasks, %d wakers\n", 7, 3);

        // 周期性GC
        if (cycle % 2 == 0) {
            size_t reclaimed = industrial_memory_pool_gc(pool);
            printf("  GC reclaimed: %zu bytes\n", reclaimed);
        }

        // 获取统计信息
        size_t allocated, free;
        double fragmentation;
        uint64_t gc_count;

        industrial_memory_pool_get_stats(pool, &allocated, &free, &fragmentation, &gc_count);
        printf("  Stats: allocated=%zu, free=%zu, fragmentation=%.1f%%, gc_count=%llu\n",
               allocated, free, fragmentation * 100.0, gc_count);
    }

    printf("\nApplication simulation completed.\n");

    // 最终清理
    industrial_memory_pool_destroy(pool);
    printf("✓ Lifecycle management example completed\n");
}

/**
 * @brief 主函数
 */
int main(int argc, char* argv[]) {
    printf("Industrial Memory Pool System - Basic Usage Examples\n");
    printf("====================================================\n\n");

    // 运行各个示例
    example_basic_memory_pool();
    example_batch_operations();
    example_lifecycle_management();

    printf("\n🎉 All examples completed successfully!\n");
    printf("The industrial memory pool system is ready for use in your async runtime.\n");

    return 0;
}
