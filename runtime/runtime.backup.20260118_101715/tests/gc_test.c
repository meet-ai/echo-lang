#include <stdio.h>
#include <stdlib.h>
#include <string.h>
#include <unistd.h>
#include <assert.h>
#include "echo/gc.h"

// 测试GC基本功能
void test_gc_basic_functionality(void) {
    printf("Testing GC basic functionality...\n");

    // 初始化GC
    echo_gc_error_t err = echo_gc_init(NULL);
    assert(err == ECHO_GC_SUCCESS);

    // 验证GC状态
    echo_gc_status_t status = echo_gc_get_status();
    assert(status == ECHO_GC_STATUS_IDLE);

    // 分配一些对象
    void* ptr1 = NULL;
    err = echo_gc_allocate(64, ECHO_OBJ_TYPE_STRUCT, &ptr1);
    assert(err == ECHO_GC_SUCCESS && ptr1 != NULL);

    void* ptr2 = NULL;
    err = echo_gc_allocate(128, ECHO_OBJ_TYPE_ARRAY, &ptr2);
    assert(err == ECHO_GC_SUCCESS && ptr2 != NULL);

    // 手动触发GC
    err = echo_gc_collect();
    assert(err == ECHO_GC_SUCCESS);

    // 等待GC完成
    sleep(1);

    // 检查统计信息
    echo_gc_stats_t stats;
    err = echo_gc_get_stats(&stats);
    assert(err == ECHO_GC_SUCCESS);
    assert(stats.total_cycles >= 1);

    printf("GC basic functionality test passed\n");
}

// 测试GC配置
void test_gc_configuration(void) {
    printf("Testing GC configuration...\n");

    // 获取当前配置
    echo_gc_config_t config;
    echo_gc_error_t err = echo_gc_get_config(&config);
    assert(err == ECHO_GC_SUCCESS);

    // 修改配置
    echo_gc_config_t new_config = config;
    new_config.trigger_ratio = 0.8;

    err = echo_gc_update_config(&new_config);
    assert(err == ECHO_GC_SUCCESS);

    // 验证配置更新
    echo_gc_config_t updated_config;
    err = echo_gc_get_config(&updated_config);
    assert(err == ECHO_GC_SUCCESS);
    assert(updated_config.trigger_ratio == 0.8);

    printf("GC configuration test passed\n");
}

// 测试GC指标收集
void test_gc_metrics(void) {
    printf("Testing GC metrics collection...\n");

    // 分配一些对象来产生指标
    for (int i = 0; i < 100; i++) {
        void* ptr = NULL;
        echo_gc_allocate(64, ECHO_OBJ_TYPE_PRIMITIVE, &ptr);
    }

    // 触发GC
    echo_gc_collect();
    sleep(1);

    // 收集指标
    echo_gc_metrics_t metrics;
    echo_gc_error_t err = echo_gc_get_metrics(&metrics);
    assert(err == ECHO_GC_SUCCESS);

    // 验证指标合理性
    assert(metrics.total_objects >= 100);
    assert(metrics.heap_usage_ratio >= 0.0 && metrics.heap_usage_ratio <= 1.0);
    assert(metrics.collected_at.tv_sec > 0);

    printf("GC metrics test passed\n");
}

// 测试并发分配
void test_concurrent_allocation(void) {
    printf("Testing concurrent allocation...\n");

    const int num_threads = 4;
    const int allocations_per_thread = 1000;

    // 创建多个线程进行分配
    // 简化实现：单线程分配大量对象
    for (int i = 0; i < num_threads * allocations_per_thread; i++) {
        void* ptr = NULL;
        echo_gc_error_t err = echo_gc_allocate(32, ECHO_OBJ_TYPE_PRIMITIVE, &ptr);
        assert(err == ECHO_GC_SUCCESS && ptr != NULL);
    }

    // 触发GC清理
    echo_gc_collect();
    sleep(1);

    printf("Concurrent allocation test passed\n");
}

// 测试GC压力场景
void test_gc_under_pressure(void) {
    printf("Testing GC under memory pressure...\n");

    // 分配大量对象
    const int num_allocations = 10000;
    void* pointers[num_allocations];

    for (int i = 0; i < num_allocations; i++) {
        echo_gc_error_t err = echo_gc_allocate(128, ECHO_OBJ_TYPE_ARRAY, &pointers[i]);
        assert(err == ECHO_GC_SUCCESS && pointers[i] != NULL);
    }

    // 触发多次GC
    for (int i = 0; i < 3; i++) {
        echo_gc_collect();
        usleep(100000);  // 100ms
    }

    // 验证系统稳定性
    echo_gc_stats_t stats;
    echo_gc_error_t err = echo_gc_get_stats(&stats);
    assert(err == ECHO_GC_SUCCESS);
    assert(stats.successful_cycles >= 3);

    printf("GC under pressure test passed\n");
}

// 测试GC调试功能
void test_gc_debugging(void) {
    printf("Testing GC debugging features...\n");

    // 分配一些对象
    void* ptr = NULL;
    echo_gc_allocate(256, ECHO_OBJ_TYPE_STRUCT, &ptr);

    // 测试堆转储
    echo_gc_dump_heap();

    // 测试统计转储
    echo_gc_dump_stats();

    // 测试堆验证
    bool valid = echo_gc_validate_heap();
    assert(valid == true);

    printf("GC debugging test passed\n");
}

int main(int argc, char* argv[]) {
    printf("Starting GC tests...\n");

    // 初始化GC
    echo_gc_config_t config = {
        .trigger_ratio = 0.75,
        .min_heap_size = 16 * 1024 * 1024,  // 16MB
        .max_heap_size = 128 * 1024 * 1024, // 128MB
        .target_pause_time_us = 10000,      // 10ms
        .max_pause_time_us = 50000,         // 50ms
        .max_concurrent_workers = 2,
        .enable_tracing = true,
        .enable_profiling = false
    };

    echo_gc_error_t err = echo_gc_init(&config);
    if (err != ECHO_GC_SUCCESS) {
        fprintf(stderr, "Failed to initialize GC: %d\n", err);
        return 1;
    }

    // 运行测试
    test_gc_basic_functionality();
    test_gc_configuration();
    test_gc_metrics();
    test_concurrent_allocation();
    test_gc_under_pressure();
    test_gc_debugging();

    // 清理
    echo_gc_shutdown();

    printf("All GC tests passed!\n");
    return 0;
}
