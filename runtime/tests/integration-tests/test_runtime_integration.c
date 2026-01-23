#include "../unit-tests/test_framework.h"
#include "../../application/services/runtime_application_service.h"
#include "../../application/commands/task_commands.h"
#include "../../application/commands/coroutine_commands.h"
#include "../../application/queries/status_queries.h"
#include "../../application/dtos/status_dtos.h"
#include <string.h>

// 测试套件
TestSuite* runtime_integration_test_suite(void);

// 测试用例
void test_task_lifecycle_integration(void);
void test_coroutine_task_interaction(void);
void test_runtime_status_monitoring(void);
void test_error_handling_integration(void);
void test_performance_under_load(void);

// 任务生命周期集成测试
void test_task_lifecycle_integration(void) {
    RuntimeApplicationService* runtime = runtime_application_service_create();
    TEST_ASSERT(runtime_application_service_initialize(runtime));

    // 创建任务命令
    CreateTaskCommand cmd = {
        .name = "integration_task",
        .description = "集成测试任务",
        .entry_point = task_lifecycle_test_function,
        .arg = NULL,
        .stack_size = 4096
    };

    // 创建任务
    TaskCreationResultDTO* result = runtime_application_service_create_task(runtime, &cmd);
    TEST_ASSERT_NOT_NULL(result);
    TEST_ASSERT(result->success);
    TEST_ASSERT(result->task_id > 0);

    // 查询任务状态
    TaskDTO* task_dto = runtime_application_service_get_task(runtime, result->task_id);
    TEST_ASSERT_NOT_NULL(task_dto);
    TEST_ASSERT_STR_EQUAL("integration_task", task_dto->name);
    TEST_ASSERT_STR_EQUAL("QUEUED", task_dto->status);

    // 等待任务完成
    task_dto = runtime_application_service_get_task(runtime, result->task_id);
    TEST_ASSERT_STR_EQUAL("COMPLETED", task_dto->status);

    // 验证任务执行结果
    TaskExecutionResultDTO* exec_result = runtime_application_service_get_task_execution_result(runtime, result->task_id);
    TEST_ASSERT_NOT_NULL(exec_result);
    TEST_ASSERT(exec_result->success);

    // 清理
    task_creation_result_dto_destroy(result);
    task_dto_destroy(task_dto);
    task_execution_result_dto_destroy(exec_result);
    runtime_application_service_destroy(runtime);
}

// 协程与任务交互集成测试
void test_coroutine_task_interaction(void) {
    RuntimeApplicationService* runtime = runtime_application_service_create();
    TEST_ASSERT(runtime_application_service_initialize(runtime));

    // 创建协程任务
    CreateCoroutineCommand coro_cmd = {
        .name = "coro_task_integration",
        .entry_point = coroutine_task_interaction_function,
        .arg = NULL,
        .stack_size = 8192,
        .priority = 5,
        .auto_start = true
    };

    // 创建协程
    OperationResultDTO* coro_result = runtime_application_service_create_coroutine(runtime, &coro_cmd);
    TEST_ASSERT_NOT_NULL(coro_result);
    TEST_ASSERT(coro_result->success);

    // 协程内部会创建和管理任务，这里验证协程状态
    CoroutineStatusDTO* coro_status = runtime_application_service_get_coroutine_status(runtime, coro_cmd.id);
    TEST_ASSERT_NOT_NULL(coro_status);
    TEST_ASSERT_STR_EQUAL("RUNNING", coro_status->status);

    // 等待协程完成
    TEST_ASSERT(runtime_application_service_wait_for_coroutine(runtime, coro_cmd.id, 10000));
    coro_status = runtime_application_service_get_coroutine_status(runtime, coro_cmd.id);
    TEST_ASSERT_STR_EQUAL("COMPLETED", coro_status->status);

    // 验证协程创建的任务都已完成
    TaskListDTO* tasks = runtime_application_service_get_tasks_by_coroutine(runtime, coro_cmd.id);
    TEST_ASSERT_NOT_NULL(tasks);
    for (size_t i = 0; i < tasks->count; i++) {
        TEST_ASSERT_STR_EQUAL("COMPLETED", tasks->tasks[i].status);
    }

    // 清理
    operation_result_dto_destroy(coro_result);
    coroutine_status_dto_destroy(coro_status);
    task_list_dto_destroy(tasks);
    runtime_application_service_destroy(runtime);
}

// 运行时状态监控集成测试
void test_runtime_status_monitoring(void) {
    RuntimeApplicationService* runtime = runtime_application_service_create();
    TEST_ASSERT(runtime_application_service_initialize(runtime));

    // 启用监控
    TEST_ASSERT(runtime_application_service_enable_monitoring(runtime));

    // 创建一些任务来产生状态变化
    for (int i = 0; i < 5; i++) {
        CreateTaskCommand cmd = {
            .name = "monitoring_task",
            .description = "监控测试任务",
            .entry_point = simple_task_function,
            .arg = (void*)(uintptr_t)i,
            .stack_size = 2048
        };

        TaskCreationResultDTO* result = runtime_application_service_create_task(runtime, &cmd);
        TEST_ASSERT(result->success);
        task_creation_result_dto_destroy(result);
    }

    // 等待任务执行
    sleep(2);

    // 查询运行时状态
    GetRuntimeStatusQuery status_query = {
        .include_tasks = true,
        .include_coroutines = true,
        .include_memory = true,
        .detailed = true
    };

    RuntimeStatusDTO* status = runtime_application_service_get_runtime_status(runtime, &status_query);
    TEST_ASSERT_NOT_NULL(status);
    TEST_ASSERT(status->active_tasks >= 0);
    TEST_ASSERT(status->total_memory_used > 0);

    // 查询内存状态
    MemoryStatusDTO* memory = runtime_application_service_get_memory_status(runtime);
    TEST_ASSERT_NOT_NULL(memory);
    TEST_ASSERT(memory->total_allocated >= 0);

    // 查询GC状态
    GCStatusDTO* gc = runtime_application_service_get_gc_status(runtime);
    TEST_ASSERT_NOT_NULL(gc);

    // 验证统计数据合理性
    TEST_ASSERT(status->uptime_seconds > 0);
    TEST_ASSERT(memory->currently_used >= 0);

    // 清理
    runtime_status_dto_destroy(status);
    memory_status_dto_destroy(memory);
    gc_status_dto_destroy(gc);
    runtime_application_service_destroy(runtime);
}

// 错误处理集成测试
void test_error_handling_integration(void) {
    RuntimeApplicationService* runtime = runtime_application_service_create();
    TEST_ASSERT(runtime_application_service_initialize(runtime));

    // 测试无效任务创建
    CreateTaskCommand invalid_cmd = {
        .name = "",  // 无效名称
        .description = "无效任务",
        .entry_point = NULL,  // 无效入口点
        .arg = NULL,
        .stack_size = 0  // 无效栈大小
    };

    TaskCreationResultDTO* result = runtime_application_service_create_task(runtime, &invalid_cmd);
    TEST_ASSERT_NOT_NULL(result);
    TEST_ASSERT(!result->success);
    TEST_ASSERT(strlen(result->message) > 0);

    // 测试取消不存在的任务
    CancelTaskCommand cancel_cmd = {
        .task_id = 99999,  // 不存在的任务ID
        .reason = "测试取消"
    };

    TaskCancellationResultDTO* cancel_result = runtime_application_service_cancel_task(runtime, &cancel_cmd);
    TEST_ASSERT_NOT_NULL(cancel_result);
    TEST_ASSERT(!cancel_result->success);

    // 测试查询不存在的任务
    TaskDTO* task = runtime_application_service_get_task(runtime, 99999);
    TEST_ASSERT_NULL(task);  // 应该返回NULL

    // 验证错误日志记录
    ErrorResultDTO* errors = runtime_application_service_get_recent_errors(runtime);
    TEST_ASSERT_NOT_NULL(errors);
    TEST_ASSERT(errors->error_count >= 2);  // 至少有2个错误

    // 清理
    task_creation_result_dto_destroy(result);
    task_cancellation_result_dto_destroy(cancel_result);
    error_result_dto_destroy(errors);
    runtime_application_service_destroy(runtime);
}

// 负载下性能测试
void test_performance_under_load(void) {
    RuntimeApplicationService* runtime = runtime_application_service_create();
    TEST_ASSERT(runtime_application_service_initialize(runtime));

    const int NUM_TASKS = 100;
    uint64_t task_ids[NUM_TASKS];

    // 创建大量任务
    for (int i = 0; i < NUM_TASKS; i++) {
        CreateTaskCommand cmd = {
            .name = "load_test_task",
            .description = "负载测试任务",
            .entry_point = load_test_task_function,
            .arg = (void*)(uintptr_t)i,
            .stack_size = 2048
        };

        TaskCreationResultDTO* result = runtime_application_service_create_task(runtime, &cmd);
        TEST_ASSERT(result->success);
        task_ids[i] = result->task_id;
        task_creation_result_dto_destroy(result);
    }

    // 等待所有任务完成
    bool all_completed = false;
    time_t start_time = time(NULL);
    while (!all_completed && (time(NULL) - start_time) < 30) {  // 最多等待30秒
        all_completed = true;
        for (int i = 0; i < NUM_TASKS; i++) {
            TaskDTO* task = runtime_application_service_get_task(runtime, task_ids[i]);
            if (strcmp(task->status, "COMPLETED") != 0) {
                all_completed = false;
            }
            task_dto_destroy(task);
        }
        if (!all_completed) {
            sleep(1);  // 等待1秒后再检查
        }
    }

    TEST_ASSERT(all_completed);

    // 验证性能指标
    TaskStatisticsDTO* stats = runtime_application_service_get_task_statistics(runtime, NULL);
    TEST_ASSERT_NOT_NULL(stats);
    TEST_ASSERT_EQUAL(NUM_TASKS, stats->total_tasks);
    TEST_ASSERT_EQUAL(NUM_TASKS, stats->completed_tasks);
    TEST_ASSERT(stats->average_execution_time_ms > 0);

    // 验证内存使用没有泄漏
    MemoryStatusDTO* memory = runtime_application_service_get_memory_status(runtime);
    TEST_ASSERT_NOT_NULL(memory);
    TEST_ASSERT(memory->currently_used < memory->total_allocated * 2);  // 内存使用不应超过总分配的2倍

    // 清理
    task_statistics_dto_destroy(stats);
    memory_status_dto_destroy(memory);
    runtime_application_service_destroy(runtime);
}

// 测试辅助函数
void task_lifecycle_test_function(void* arg) {
    // 模拟任务执行
    volatile int sum = 0;
    for (int i = 0; i < 10000; i++) {
        sum += i;
    }
}

void coroutine_task_interaction_function(void* arg) {
    // 协程内部创建和管理任务
    for (int i = 0; i < 3; i++) {
        // 创建子任务
        Task* sub_task = task_create("sub_task", simple_task_function, (void*)(uintptr_t)i, 1024);
        task_start(sub_task);
        task_wait(sub_task, 5000);
        task_destroy(sub_task);
    }
}

void simple_task_function(void* arg) {
    int id = (int)(uintptr_t)arg;
    volatile int iterations = 1000 + (id * 100);

    for (int i = 0; i < iterations; i++) {
        // 简单的计算
    }
}

void load_test_task_function(void* arg) {
    int id = (int)(uintptr_t)arg;

    // 不同任务有不同的计算量
    volatile int iterations = 5000 + (id * 50);
    volatile int sum = 0;

    for (int i = 0; i < iterations; i++) {
        sum += i;
        if (i % 1000 == 0) {
            // 模拟I/O等待
            usleep(1000);
        }
    }
}

// 运行时集成测试套件
TestSuite* runtime_integration_test_suite(void) {
    TestSuite* suite = TEST_SUITE("Runtime Integration", "运行时集成测试");

    TEST_ADD(suite, "task_lifecycle_integration", "任务生命周期集成测试", test_task_lifecycle_integration);
    TEST_ADD(suite, "coroutine_task_interaction", "协程与任务交互集成测试", test_coroutine_task_interaction);
    TEST_ADD(suite, "runtime_status_monitoring", "运行时状态监控集成测试", test_runtime_status_monitoring);
    TEST_ADD(suite, "error_handling_integration", "错误处理集成测试", test_error_handling_integration);
    TEST_ADD(suite, "performance_under_load", "负载下性能测试", test_performance_under_load);

    return suite;
}
