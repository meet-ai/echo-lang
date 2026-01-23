#include "../unit-tests/test_framework.h"
#include "../../application/services/runtime_application_service.h"
#include <unistd.h>
#include <pthread.h>

// 测试套件
TestSuite* runtime_stress_test_suite(void);

// 压力测试用例
void test_high_concurrency_task_creation(void);
void test_memory_pressure_handling(void);
void test_long_running_operations(void);
void test_resource_exhaustion_recovery(void);
void test_network_failure_simulation(void);

// 高并发任务创建压力测试
void test_high_concurrency_task_creation(void) {
    RuntimeApplicationService* runtime = runtime_application_service_create();
    TEST_ASSERT(runtime_application_service_initialize(runtime));

    const int NUM_THREADS = 10;
    const int TASKS_PER_THREAD = 100;
    const int TOTAL_TASKS = NUM_THREADS * TASKS_PER_THREAD;

    StressTestThread threads[NUM_THREADS];
    pthread_t thread_handles[NUM_THREADS];

    // 创建线程
    for (int i = 0; i < NUM_THREADS; i++) {
        threads[i].runtime = runtime;
        threads[i].thread_id = i;
        threads[i].num_tasks = TASKS_PER_THREAD;
        threads[i].successful_creations = 0;
        threads[i].errors = 0;

        pthread_create(&thread_handles[i], NULL, concurrent_task_creation_thread, &threads[i]);
    }

    // 等待所有线程完成
    for (int i = 0; i < NUM_THREADS; i++) {
        pthread_join(thread_handles[i], NULL);
    }

    // 统计结果
    int total_successful = 0;
    int total_errors = 0;

    for (int i = 0; i < NUM_THREADS; i++) {
        total_successful += threads[i].successful_creations;
        total_errors += threads[i].errors;
    }

    // 验证结果
    TEST_ASSERT(total_successful >= TOTAL_TASKS * 0.95);  // 至少95%成功率
    TEST_ASSERT(total_errors < TOTAL_TASKS * 0.05);      // 错误率低于5%

    printf("High concurrency test: %d successful, %d errors\n", total_successful, total_errors);

    // 等待所有任务完成
    TEST_ASSERT(runtime_application_service_wait_for_all_tasks(runtime, 60000));  // 最多等待60秒

    // 验证最终状态
    TaskStatisticsDTO* stats = runtime_application_service_get_task_statistics(runtime, NULL);
    TEST_ASSERT_NOT_NULL(stats);
    TEST_ASSERT_EQUAL(TOTAL_TASKS, stats->total_tasks);
    TEST_ASSERT_EQUAL(TOTAL_TASKS, stats->completed_tasks);

    task_statistics_dto_destroy(stats);
    runtime_application_service_destroy(runtime);
}

// 内存压力处理测试
void test_memory_pressure_handling(void) {
    RuntimeApplicationService* runtime = runtime_application_service_create();
    TEST_ASSERT(runtime_application_service_initialize(runtime));

    const int NUM_MEMORY_TASKS = 1000;
    const size_t MEMORY_PER_TASK = 1024 * 1024;  // 1MB per task

    // 创建大量内存密集型任务
    for (int i = 0; i < NUM_MEMORY_TASKS; i++) {
        CreateTaskCommand cmd = {
            .name = "memory_pressure_task",
            .description = "内存压力测试任务",
            .entry_point = memory_intensive_task,
            .arg = (void*)(uintptr_t)MEMORY_PER_TASK,
            .stack_size = 2048
        };

        TaskCreationResultDTO* result = runtime_application_service_create_task(runtime, &cmd);
        TEST_ASSERT(result->success);
        task_creation_result_dto_destroy(result);
    }

    // 监控内存使用
    MemoryStatusDTO* initial_memory = runtime_application_service_get_memory_status(runtime);
    TEST_ASSERT_NOT_NULL(initial_memory);

    // 等待任务执行并监控内存
    bool memory_stable = true;
    uint64_t max_memory_used = 0;

    for (int i = 0; i < 30; i++) {  // 监控30秒
        MemoryStatusDTO* current_memory = runtime_application_service_get_memory_status(runtime);
        if (current_memory->currently_used > max_memory_used) {
            max_memory_used = current_memory->currently_used;
        }

        // 检查内存使用是否异常增长
        if (current_memory->currently_used > initial_memory->total_allocated * 1.5) {
            memory_stable = false;
            break;
        }

        memory_status_dto_destroy(current_memory);
        sleep(1);
    }

    TEST_ASSERT(memory_stable);  // 内存使用应该稳定

    // 等待所有任务完成
    TEST_ASSERT(runtime_application_service_wait_for_all_tasks(runtime, 120000));  // 最多等待2分钟

    // 验证内存回收
    MemoryStatusDTO* final_memory = runtime_application_service_get_memory_status(runtime);
    TEST_ASSERT(final_memory->currently_used < max_memory_used * 1.2);  // 内存应该得到回收

    printf("Memory pressure test: max used %llu bytes, final used %llu bytes\n",
           max_memory_used, final_memory->currently_used);

    memory_status_dto_destroy(initial_memory);
    memory_status_dto_destroy(final_memory);
    runtime_application_service_destroy(runtime);
}

// 长运行操作测试
void test_long_running_operations(void) {
    RuntimeApplicationService* runtime = runtime_application_service_create();
    TEST_ASSERT(runtime_application_service_initialize(runtime));

    const int NUM_LONG_TASKS = 5;
    const int LONG_TASK_DURATION_SECONDS = 30;

    // 创建长运行任务
    for (int i = 0; i < NUM_LONG_TASKS; i++) {
        CreateTaskCommand cmd = {
            .name = "long_running_task",
            .description = "长运行操作测试任务",
            .entry_point = long_running_task,
            .arg = (void*)(uintptr_t)LONG_TASK_DURATION_SECONDS,
            .stack_size = 2048
        };

        TaskCreationResultDTO* result = runtime_application_service_create_task(runtime, &cmd);
        TEST_ASSERT(result->success);
        task_creation_result_dto_destroy(result);
    }

    // 监控任务进度
    time_t start_time = time(NULL);
    bool all_completed = false;

    while (!all_completed && (time(NULL) - start_time) < (LONG_TASK_DURATION_SECONDS + 60)) {
        all_completed = true;
        TaskStatisticsDTO* stats = runtime_application_service_get_task_statistics(runtime, NULL);

        if (stats->completed_tasks < NUM_LONG_TASKS) {
            all_completed = false;

            // 验证运行中的任务状态
            TEST_ASSERT(stats->running_tasks <= NUM_LONG_TASKS);
            TEST_ASSERT(stats->failed_tasks == 0);  // 不应该有失败的任务

            printf("Progress: %d/%d tasks completed\n", stats->completed_tasks, NUM_LONG_TASKS);
        }

        task_statistics_dto_destroy(stats);

        if (!all_completed) {
            sleep(5);  // 每5秒检查一次
        }
    }

    TEST_ASSERT(all_completed);  // 所有任务应该完成

    // 验证最终统计
    TaskStatisticsDTO* final_stats = runtime_application_service_get_task_statistics(runtime, NULL);
    TEST_ASSERT_EQUAL(NUM_LONG_TASKS, final_stats->completed_tasks);
    TEST_ASSERT(final_stats->average_execution_time_ms >= LONG_TASK_DURATION_SECONDS * 1000);

    printf("Long running test completed: avg execution time %.2f ms\n",
           final_stats->average_execution_time_ms);

    task_statistics_dto_destroy(final_stats);
    runtime_application_service_destroy(runtime);
}

// 资源耗尽恢复测试
void test_resource_exhaustion_recovery(void) {
    RuntimeApplicationService* runtime = runtime_application_service_create();
    TEST_ASSERT(runtime_application_service_initialize(runtime));

    // 首先创建一些正常任务
    for (int i = 0; i < 10; i++) {
        CreateTaskCommand cmd = {
            .name = "normal_task",
            .description = "正常任务",
            .entry_point = simple_task_function,
            .arg = NULL,
            .stack_size = 1024
        };

        TaskCreationResultDTO* result = runtime_application_service_create_task(runtime, &cmd);
        TEST_ASSERT(result->success);
        task_creation_result_dto_destroy(result);
    }

    // 尝试创建大量任务导致资源耗尽
    const int EXCESSIVE_TASKS = 10000;
    int successful_creations = 0;
    int failed_creations = 0;

    for (int i = 0; i < EXCESSIVE_TASKS; i++) {
        CreateTaskCommand cmd = {
            .name = "excessive_task",
            .description = "过度任务创建",
            .entry_point = simple_task_function,
            .arg = NULL,
            .stack_size = 1024
        };

        TaskCreationResultDTO* result = runtime_application_service_create_task(runtime, &cmd);
        if (result->success) {
            successful_creations++;
        } else {
            failed_creations++;
        }
        task_creation_result_dto_destroy(result);
    }

    // 验证资源控制：不应该创建过多任务
    TEST_ASSERT(successful_creations < EXCESSIVE_TASKS * 0.1);  // 成功率应该很低
    TEST_ASSERT(failed_creations > EXCESSIVE_TASKS * 0.9);     // 失败率应该很高

    // 等待现有任务完成
    TEST_ASSERT(runtime_application_service_wait_for_all_tasks(runtime, 30000));

    // 验证系统仍然稳定
    RuntimeStatusDTO* status = runtime_application_service_get_runtime_status(runtime, NULL);
    TEST_ASSERT_NOT_NULL(status);
    TEST_ASSERT(status->active_tasks >= 0);  // 系统应该仍然正常运行

    printf("Resource exhaustion test: %d successful, %d failed\n", successful_creations, failed_creations);

    runtime_status_dto_destroy(status);
    runtime_application_service_destroy(runtime);
}

// 网络故障模拟测试
void test_network_failure_simulation(void) {
    RuntimeApplicationService* runtime = runtime_application_service_create();
    TEST_ASSERT(runtime_application_service_initialize(runtime));

    // 启用网络故障模拟
    TEST_ASSERT(runtime_application_service_enable_network_failure_simulation(runtime, 0.1));  // 10%失败率

    const int NETWORK_TASKS = 100;
    int successful_network_calls = 0;
    int failed_network_calls = 0;

    // 创建涉及网络操作的任务
    for (int i = 0; i < NETWORK_TASKS; i++) {
        CreateTaskCommand cmd = {
            .name = "network_task",
            .description = "网络操作任务",
            .entry_point = network_operation_task,
            .arg = NULL,
            .stack_size = 2048
        };

        TaskCreationResultDTO* result = runtime_application_service_create_task(runtime, &cmd);
        TEST_ASSERT(result->success);
        task_creation_result_dto_destroy(result);
    }

    // 等待任务完成
    TEST_ASSERT(runtime_application_service_wait_for_all_tasks(runtime, 120000));  // 2分钟超时

    // 统计网络操作结果
    TaskListDTO* tasks = runtime_application_service_get_tasks(runtime, NULL);
    for (size_t i = 0; i < tasks->count; i++) {
        if (strcmp(tasks->tasks[i].name, "network_task") == 0) {
            if (strcmp(tasks->tasks[i].status, "COMPLETED") == 0) {
                successful_network_calls++;
            } else if (strcmp(tasks->tasks[i].status, "FAILED") == 0) {
                failed_network_calls++;
            }
        }
    }

    // 验证容错性：应该有一些成功和失败
    TEST_ASSERT(successful_network_calls > 0);
    TEST_ASSERT(failed_network_calls > 0);
    TEST_ASSERT(successful_network_calls + failed_network_calls >= NETWORK_TASKS * 0.8);  // 至少80%有明确结果

    // 验证系统稳定性
    RuntimeStatusDTO* final_status = runtime_application_service_get_runtime_status(runtime, NULL);
    TEST_ASSERT_NOT_NULL(final_status);

    printf("Network failure simulation: %d successful, %d failed\n",
           successful_network_calls, failed_network_calls);

    task_list_dto_destroy(tasks);
    runtime_status_dto_destroy(final_status);

    // 禁用网络故障模拟
    runtime_application_service_disable_network_failure_simulation(runtime);
    runtime_application_service_destroy(runtime);
}

// 测试辅助结构和函数
typedef struct StressTestThread {
    RuntimeApplicationService* runtime;
    int thread_id;
    int num_tasks;
    int successful_creations;
    int errors;
} StressTestThread;

void* concurrent_task_creation_thread(void* arg) {
    StressTestThread* thread_data = (StressTestThread*)arg;

    for (int i = 0; i < thread_data->num_tasks; i++) {
        CreateTaskCommand cmd = {
            .name = "concurrent_task",
            .description = "并发创建任务",
            .entry_point = simple_task_function,
            .arg = (void*)(uintptr_t)(thread_data->thread_id * 1000 + i),
            .stack_size = 1024
        };

        TaskCreationResultDTO* result = runtime_application_service_create_task(thread_data->runtime, &cmd);
        if (result && result->success) {
            thread_data->successful_creations++;
        } else {
            thread_data->errors++;
        }

        if (result) {
            task_creation_result_dto_destroy(result);
        }

        // 小延迟以增加并发竞争
        usleep(1000);
    }

    return NULL;
}

void memory_intensive_task(void* arg) {
    size_t memory_size = (size_t)arg;
    if (memory_size == 0) memory_size = 1024 * 1024;  // 默认1MB

    // 分配大量内存
    void* buffer = malloc(memory_size);
    if (buffer) {
        // 填充内存
        memset(buffer, 0xAA, memory_size);

        // 进行一些计算
        volatile char* ptr = (volatile char*)buffer;
        for (size_t i = 0; i < memory_size; i++) {
            ptr[i] = (char)(ptr[i] + 1);
        }

        free(buffer);
    }
}

void long_running_task(void* arg) {
    int duration_seconds = (int)(uintptr_t)arg;
    if (duration_seconds <= 0) duration_seconds = 10;

    time_t start_time = time(NULL);
    while ((time(NULL) - start_time) < duration_seconds) {
        // 模拟长时间运行
        volatile int sum = 0;
        for (int i = 0; i < 100000; i++) {
            sum += i;
        }
        usleep(10000);  // 10ms pause
    }
}

void network_operation_task(void* arg) {
    // 模拟网络操作
    for (int i = 0; i < 10; i++) {
        // 模拟网络延迟
        usleep(50000);  // 50ms

        // 随机模拟网络失败（由测试框架控制）
        if (rand() % 10 == 0) {  // 10%失败率
            // 模拟网络错误
            return;
        }
    }
}

// 运行时压力测试套件
TestSuite* runtime_stress_test_suite(void) {
    TestSuite* suite = TEST_SUITE("Runtime Stress Tests", "运行时压力测试");

    TEST_ADD(suite, "high_concurrency_task_creation", "高并发任务创建测试", test_high_concurrency_task_creation);
    TEST_ADD(suite, "memory_pressure_handling", "内存压力处理测试", test_memory_pressure_handling);
    TEST_ADD(suite, "long_running_operations", "长运行操作测试", test_long_running_operations);
    TEST_ADD(suite, "resource_exhaustion_recovery", "资源耗尽恢复测试", test_resource_exhaustion_recovery);
    TEST_ADD(suite, "network_failure_simulation", "网络故障模拟测试", test_network_failure_simulation);

    return suite;
}
