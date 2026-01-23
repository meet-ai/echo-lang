#include "../../api/runtime_api.h"
#include "../../application/services/coroutine_application_service.h"
#include "../../application/commands/coroutine_commands.h"
#include "../../application/dtos/result_dtos.h"
#include "../../application/dtos/status_dtos.h"
#include "../../application/queries/status_queries.h"
#include <stdio.h>
#include <stdlib.h>
#include <string.h>
#include <unistd.h>
#include <assert.h>

// 测试协程函数
void* test_coroutine_function(void* arg) {
    const char* message = (const char*)arg;
    printf("协程执行: %s\n", message);

    // 模拟一些工作
    for (int i = 0; i < 5; i++) {
        printf("协程工作 %d\n", i);
        sleep(1);  // 模拟耗时操作
    }

    printf("协程完成: %s\n", message);
    return (void*)"completed";
}

// 测试用例：完整的协程生命周期
void test_coroutine_lifecycle(void) {
    printf("=== 测试协程生命周期 ===\n");

    // 初始化运行时
    RuntimeConfig config = {0};
    RuntimeHandle* runtime = runtime_init(&config);
    assert(runtime != NULL);

    // 创建协程应用服务
    CoroutineApplicationService* service = coroutine_application_service_create();
    assert(service != NULL);

    // 初始化服务
    bool init_result = coroutine_application_service_initialize(service, NULL, NULL, NULL);
    assert(init_result == true);

    // 创建协程
    CreateCoroutineCommand create_cmd = {
        .name = "test_coroutine",
        .entry_point = test_coroutine_function,
        .arg = (void*)"Hello from coroutine",
        .stack_size = 64 * 1024
    };

    CoroutineCreationResultDTO* creation_result = coroutine_application_service_create_coroutine(service, &create_cmd);
    assert(creation_result != NULL);
    assert(creation_result->success == true);
    assert(creation_result->coroutine_id > 0);

    uint64_t coroutine_id = creation_result->coroutine_id;
    printf("协程创建成功，ID: %lu\n", coroutine_id);

    // 查询协程状态
    GetCoroutineStatusCommand status_cmd = {
        .coroutine_id = coroutine_id
    };

    CoroutineStatusDTO* status_result = coroutine_application_service_get_coroutine_status(service, &status_cmd);
    assert(status_result != NULL);
    assert(status_result->found == true);
    assert(status_result->coroutine_id == coroutine_id);
    assert(strcmp(status_result->name, "test_coroutine") == 0);

    printf("协程状态查询成功: %s\n", status_result->name);

    // 等待协程完成
    sleep(6);  // 等待协程执行完成

    // 再次查询状态
    status_result = coroutine_application_service_get_coroutine_status(service, &status_cmd);
    assert(status_result != NULL);
    assert(status_result->found == true);

    printf("协程执行状态: %s\n", status_result->status);

    // 列出所有协程
    ListCoroutinesQuery list_query = {0};
    CoroutineListDTO* list_result = coroutine_application_service_list_coroutines(service, &list_query);
    assert(list_result != NULL);
    assert(list_result->success == true);
    assert(list_result->count > 0);

    printf("协程列表查询成功，共 %zu 个协程\n", list_result->count);

    // 清理资源
    if (list_result) {
        coroutine_list_dto_destroy(list_result);
    }
    if (status_result) {
        coroutine_status_dto_destroy(status_result);
    }
    if (creation_result) {
        coroutine_creation_result_dto_destroy(creation_result);
    }

    // 销毁服务
    coroutine_application_service_destroy(service);

    // 关闭运行时
    runtime_shutdown(runtime);

    printf("协程生命周期测试通过\n");
}

// 测试用例：协程取消
void test_coroutine_cancellation(void) {
    printf("=== 测试协程取消 ===\n");

    // 初始化运行时
    RuntimeConfig config = {0};
    RuntimeHandle* runtime = runtime_init(&config);
    assert(runtime != NULL);

    // 创建协程应用服务
    CoroutineApplicationService* service = coroutine_application_service_create();
    assert(service != NULL);

    coroutine_application_service_initialize(service, NULL, NULL, NULL);

    // 创建长时间运行的协程
    void* long_running_coroutine(void* arg) {
        printf("长时间协程开始执行\n");
        for (int i = 0; i < 100; i++) {
            printf("长时间协程工作 %d\n", i);
            sleep(1);
        }
        printf("长时间协程执行完毕\n");
        return NULL;
    }

    CreateCoroutineCommand create_cmd = {
        .name = "long_running_coroutine",
        .entry_point = long_running_coroutine,
        .arg = NULL,
        .stack_size = 64 * 1024
    };

    CoroutineCreationResultDTO* creation_result = coroutine_application_service_create_coroutine(service, &create_cmd);
    assert(creation_result != NULL);
    assert(creation_result->success == true);

    uint64_t coroutine_id = creation_result->coroutine_id;
    printf("长时间协程创建成功，ID: %lu\n", coroutine_id);

    // 等待一段时间让协程开始执行
    sleep(3);

    // 取消协程
    CancelCoroutineCommand cancel_cmd = {
        .coroutine_id = coroutine_id
    };

    OperationResultDTO* cancel_result = coroutine_application_service_cancel_coroutine(service, &cancel_cmd);
    assert(cancel_result != NULL);
    assert(cancel_result->success == true);

    printf("协程取消成功\n");

    // 清理资源
    if (cancel_result) {
        operation_result_dto_destroy(cancel_result);
    }
    if (creation_result) {
        coroutine_creation_result_dto_destroy(creation_result);
    }

    // 销毁服务
    coroutine_application_service_destroy(service);

    // 关闭运行时
    runtime_shutdown(runtime);

    printf("协程取消测试通过\n");
}

// 测试用例：协程错误处理
void test_coroutine_error_handling(void) {
    printf("=== 测试协程错误处理 ===\n");

    // 初始化运行时
    RuntimeConfig config = {0};
    RuntimeHandle* runtime = runtime_init(&config);
    assert(runtime != NULL);

    // 创建协程应用服务
    CoroutineApplicationService* service = coroutine_application_service_create();
    assert(service != NULL);

    coroutine_application_service_initialize(service, NULL, NULL, NULL);

    // 创建会出错的协程
    void* error_coroutine(void* arg) {
        printf("错误协程开始执行\n");
        sleep(1);
        printf("错误协程抛出异常\n");
        // 模拟错误
        return NULL;  // 在实际实现中，这里应该返回错误
    }

    CreateCoroutineCommand create_cmd = {
        .name = "error_coroutine",
        .entry_point = error_coroutine,
        .arg = NULL,
        .stack_size = 64 * 1024
    };

    CoroutineCreationResultDTO* creation_result = coroutine_application_service_create_coroutine(service, &create_cmd);
    assert(creation_result != NULL);
    assert(creation_result->success == true);

    uint64_t coroutine_id = creation_result->coroutine_id;
    printf("错误协程创建成功，ID: %lu\n", coroutine_id);

    // 等待协程完成
    sleep(3);

    // 查询最终状态
    GetCoroutineStatusCommand status_cmd = {
        .coroutine_id = coroutine_id
    };

    CoroutineStatusDTO* status_result = coroutine_application_service_get_coroutine_status(service, &status_cmd);
    assert(status_result != NULL);
    assert(status_result->found == true);

    printf("错误协程最终状态: %s\n", status_result->status);

    // 清理资源
    if (status_result) {
        coroutine_status_dto_destroy(status_result);
    }
    if (creation_result) {
        coroutine_creation_result_dto_destroy(creation_result);
    }

    // 销毁服务
    coroutine_application_service_destroy(service);

    // 关闭运行时
    runtime_shutdown(runtime);

    printf("协程错误处理测试通过\n");
}

// 主测试函数
int main(int argc, char* argv[]) {
    printf("开始协程集成测试\n");

    // 运行测试用例
    test_coroutine_lifecycle();
    test_coroutine_cancellation();
    test_coroutine_error_handling();

    printf("所有协程集成测试通过！\n");
    return 0;
}
