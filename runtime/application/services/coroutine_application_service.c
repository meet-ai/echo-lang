#include "coroutine_application_service.h"
#include "../../shared/types/common_types.h"
#include "../../domain/coroutine/coroutine.h"
#include "../queries/status_queries.h"
#include <stdlib.h>
#include <stdio.h>
#include <string.h>
#include <time.h>
#include <pthread.h>
#include <stdatomic.h>

// 辅助函数：生成唯一ID
static uint64_t generate_unique_id(void) {
    static atomic_uint_fast64_t counter = 0;
    return atomic_fetch_add(&counter, 1) + 1;
}

// 内部实现结构体
typedef struct CoroutineApplicationServiceImpl {
    CoroutineApplicationService base;
    // 私有成员
    pthread_mutex_t mutex;           // 保护并发访问
    char last_error[1024];          // 最后错误信息
    time_t last_operation_time;     // 最后操作时间
    uint64_t operation_count;       // 操作计数
} CoroutineApplicationServiceImpl;

// 虚函数表实现
static CoroutineCreationResultDTO* coroutine_application_service_create_coroutine_impl(CoroutineApplicationService* service, const CreateCoroutineCommand* cmd);
static OperationResultDTO* coroutine_application_service_start_coroutine_impl(CoroutineApplicationService* service, const StartCoroutineCommand* cmd);
static OperationResultDTO* coroutine_application_service_pause_coroutine_impl(CoroutineApplicationService* service, const PauseCoroutineCommand* cmd);
static OperationResultDTO* coroutine_application_service_resume_coroutine_impl(CoroutineApplicationService* service, const ResumeCoroutineCommand* cmd);
static OperationResultDTO* coroutine_application_service_cancel_coroutine_impl(CoroutineApplicationService* service, const CancelCoroutineCommand* cmd);
static CoroutineDTO* coroutine_application_service_get_coroutine_impl(CoroutineApplicationService* service, uint64_t coroutine_id);
static CoroutineListDTO* coroutine_application_service_get_coroutines_impl(CoroutineApplicationService* service, const GetCoroutineStatusQuery* query);
static CoroutineStatisticsDTO* coroutine_application_service_get_coroutine_statistics_impl(CoroutineApplicationService* service, const GetCoroutineStatusQuery* query);
static OperationResultDTO* coroutine_application_service_yield_current_impl(CoroutineApplicationService* service);
static OperationResultDTO* coroutine_application_service_set_coroutine_priority_impl(CoroutineApplicationService* service, const SetCoroutinePriorityCommand* cmd);
static OperationResultDTO* coroutine_application_service_wait_coroutine_impl(CoroutineApplicationService* service, const WaitCoroutineCommand* cmd);
static OperationResultDTO* coroutine_application_service_join_coroutine_impl(CoroutineApplicationService* service, const JoinCoroutineCommand* cmd);

// 虚函数表定义
static CoroutineApplicationServiceInterface vtable = {
    .create_coroutine = coroutine_application_service_create_coroutine_impl,
    .start_coroutine = coroutine_application_service_start_coroutine_impl,
    .pause_coroutine = coroutine_application_service_pause_coroutine_impl,
    .resume_coroutine = coroutine_application_service_resume_coroutine_impl,
    .cancel_coroutine = coroutine_application_service_cancel_coroutine_impl,
    .get_coroutine = coroutine_application_service_get_coroutine_impl,
    .get_coroutines = coroutine_application_service_get_coroutines_impl,
    .get_coroutine_statistics = coroutine_application_service_get_coroutine_statistics_impl,
    .yield_current = coroutine_application_service_yield_current_impl,
    .set_coroutine_priority = coroutine_application_service_set_coroutine_priority_impl,
    .wait_coroutine = coroutine_application_service_wait_coroutine_impl,
    .join_coroutine = coroutine_application_service_join_coroutine_impl,
};

// 创建DTO对象的辅助函数
static CoroutineCreationResultDTO* create_coroutine_creation_result(uint64_t coroutine_id, bool success, const char* message) {
    CoroutineCreationResultDTO* result = (CoroutineCreationResultDTO*)malloc(sizeof(CoroutineCreationResultDTO));
    if (!result) return NULL;

    result->coroutine_id = coroutine_id;
    result->success = success;
    result->created_at = time(NULL);

    if (message) {
        strncpy(result->message, message, sizeof(result->message) - 1);
        result->message[sizeof(result->message) - 1] = '\0';
    } else {
        result->message[0] = '\0';
    }

    return result;
}

static CoroutineDTO* create_coroutine_dto(const Coroutine* coroutine) {
    if (!coroutine) return NULL;

    CoroutineDTO* dto = (CoroutineDTO*)malloc(sizeof(CoroutineDTO));
    if (!dto) return NULL;

    dto->coroutine_id = coroutine->id;
    // 将枚举转换为字符串
    const char* priority_str = "normal";
    switch (coroutine->priority) {
        case COROUTINE_PRIORITY_LOW: priority_str = "low"; break;
        case COROUTINE_PRIORITY_NORMAL: priority_str = "normal"; break;
        case COROUTINE_PRIORITY_HIGH: priority_str = "high"; break;
        case COROUTINE_PRIORITY_CRITICAL: priority_str = "critical"; break;
    }
    strncpy(dto->priority, priority_str, sizeof(dto->priority) - 1);
    dto->priority[sizeof(dto->priority) - 1] = '\0';
    
    const char* status_str = "new";
    switch (coroutine->state) {
        case COROUTINE_NEW: status_str = "new"; break;
        case COROUTINE_READY: status_str = "ready"; break;
        case COROUTINE_RUNNING: status_str = "running"; break;
        case COROUTINE_SUSPENDED: status_str = "suspended"; break;
        case COROUTINE_COMPLETED: status_str = "completed"; break;
        case COROUTINE_FAILED: status_str = "failed"; break;
        case COROUTINE_CANCELLED: status_str = "cancelled"; break;
    }
    strncpy(dto->status, status_str, sizeof(dto->status) - 1);
    dto->status[sizeof(dto->status) - 1] = '\0';
    dto->created_at = coroutine->created_at;
    dto->started_at = coroutine->started_at;
    dto->completed_at = coroutine->completed_at;
    dto->stack_usage = coroutine_get_stack_usage(coroutine);
    uint64_t time_us = coroutine_get_execution_time_us(coroutine);
    dto->execution_time_ms = time_us / 1000; // 转换为毫秒

    if (coroutine->name[0]) {
        strncpy(dto->name, coroutine->name, sizeof(dto->name) - 1);
        dto->name[sizeof(dto->name) - 1] = '\0';
    } else {
        dto->name[0] = '\0';
    }

    return dto;
}

static OperationResultDTO* create_operation_result(bool success, const char* message) {
    OperationResultDTO* result = (OperationResultDTO*)malloc(sizeof(OperationResultDTO));
    if (!result) return NULL;

    result->success = success;
    result->timestamp = time(NULL);
    result->operation_id = generate_unique_id();
    result->error_code = success ? 0 : -1;

    if (message) {
        strncpy(result->message, message, sizeof(result->message) - 1);
        result->message[sizeof(result->message) - 1] = '\0';
    } else {
        result->message[0] = '\0';
    }
    
    result->error_details[0] = '\0';

    return result;
}

// 构造函数和析构函数
CoroutineApplicationService* coroutine_application_service_create(void) {
    CoroutineApplicationServiceImpl* impl = (CoroutineApplicationServiceImpl*)malloc(sizeof(CoroutineApplicationServiceImpl));
    if (!impl) return NULL;

    memset(impl, 0, sizeof(CoroutineApplicationServiceImpl));

    // 初始化基类
    impl->base.vtable = &vtable;
    impl->base.initialized = false;
    strncpy(impl->base.service_name, "CoroutineApplicationService", sizeof(impl->base.service_name) - 1);
    impl->base.started_at = time(NULL);

    // 初始化私有成员
    pthread_mutex_init(&impl->mutex, NULL);
    impl->last_operation_time = time(NULL);
    impl->operation_count = 0;

    return (CoroutineApplicationService*)impl;
}

void coroutine_application_service_destroy(CoroutineApplicationService* service) {
    if (!service) return;

    CoroutineApplicationServiceImpl* impl = (CoroutineApplicationServiceImpl*)service;

    // 清理资源
    coroutine_application_service_cleanup(service);

    // 销毁互斥锁
    pthread_mutex_destroy(&impl->mutex);

    free(impl);
}

// 初始化和清理
bool coroutine_application_service_initialize(CoroutineApplicationService* service,
                                           void* coroutine_runtime,
                                           void* coroutine_scheduler,
                                           void* event_publisher) {
    if (!service) return false;

    CoroutineApplicationServiceImpl* impl = (CoroutineApplicationServiceImpl*)service;

    pthread_mutex_lock(&impl->mutex);

    // 设置依赖
    impl->base.coroutine_runtime = coroutine_runtime;
    impl->base.coroutine_scheduler = coroutine_scheduler;
    impl->base.event_publisher = event_publisher;

    impl->base.initialized = true;

    pthread_mutex_unlock(&impl->mutex);

    return true;
}

void coroutine_application_service_cleanup(CoroutineApplicationService* service) {
    if (!service) return;

    CoroutineApplicationServiceImpl* impl = (CoroutineApplicationServiceImpl*)service;

    pthread_mutex_lock(&impl->mutex);

    // 清理依赖引用
    impl->base.coroutine_runtime = NULL;
    impl->base.coroutine_scheduler = NULL;
    impl->base.event_publisher = NULL;

    impl->base.initialized = false;

    pthread_mutex_unlock(&impl->mutex);
}

// 服务状态查询
bool coroutine_application_service_is_healthy(const CoroutineApplicationService* service) {
    if (!service) return false;
    return service->initialized;
}

const char* coroutine_application_service_get_name(const CoroutineApplicationService* service) {
    return service ? service->service_name : NULL;
}

time_t coroutine_application_service_get_started_at(const CoroutineApplicationService* service) {
    return service ? service->started_at : 0;
}

// 虚函数表实现

static CoroutineCreationResultDTO* coroutine_application_service_create_coroutine_impl(CoroutineApplicationService* service, const CreateCoroutineCommand* cmd) {
    if (!service || !cmd || !service->initialized) {
        return create_coroutine_creation_result(0, false, "Service not initialized or invalid command");
    }

    CoroutineApplicationServiceImpl* impl = (CoroutineApplicationServiceImpl*)service;

    // 用例编排：创建协程
    // 1. 验证命令参数
    // 2. 创建领域实体
    // 3. 设置协程属性
    // 4. 注册到调度器
    // 5. 发布领域事件

    pthread_mutex_lock(&impl->mutex);

    // 1. 验证参数
    if (strlen(cmd->name) == 0) {
        pthread_mutex_unlock(&impl->mutex);
        return create_coroutine_creation_result(0, false, "Coroutine name is required");
    }

    if (strlen(cmd->name) == 0) {
        pthread_mutex_unlock(&impl->mutex);
        return create_coroutine_creation_result(0, false, "Coroutine name is required");
    }
    
    if (!cmd->entry_point) {
        pthread_mutex_unlock(&impl->mutex);
        return create_coroutine_creation_result(0, false, "Coroutine entry point is required");
    }

    // 2. 创建协程实体
    Coroutine* coroutine = coroutine_create(cmd->name, cmd->entry_point, cmd->arg, cmd->stack_size);
    if (!coroutine) {
        pthread_mutex_unlock(&impl->mutex);
        return create_coroutine_creation_result(0, false, "Failed to create coroutine entity");
    }

    // 3. 设置协程属性
    // 将 uint32_t 转换为 CoroutinePriority 枚举
    CoroutinePriority priority = (CoroutinePriority)cmd->priority;
    if (priority > COROUTINE_PRIORITY_CRITICAL) {
        priority = COROUTINE_PRIORITY_NORMAL; // 默认优先级
    }
    coroutine_set_priority(coroutine, priority);

    // 4. 注册到调度器（如果有调度器）
    // 这里应该调用调度器接口注册协程
    // 暂时跳过

    // 5. 发布事件（如果有事件发布器）
    // 这里应该发布CoroutineCreated事件
    // 暂时跳过

    impl->operation_count++;
    impl->last_operation_time = time(NULL);

    pthread_mutex_unlock(&impl->mutex);

    return create_coroutine_creation_result(coroutine->id, true, "Coroutine created successfully");
}

static OperationResultDTO* coroutine_application_service_start_coroutine_impl(CoroutineApplicationService* service, const StartCoroutineCommand* cmd) {
    if (!service || !cmd || !service->initialized) {
        return create_operation_result(false, "Service not initialized or invalid command");
    }

    CoroutineApplicationServiceImpl* impl = (CoroutineApplicationServiceImpl*)service;

    // 用例编排：启动协程
    // 1. 查找协程
    // 2. 启动协程
    // 3. 发布启动事件

    pthread_mutex_lock(&impl->mutex);

    // TODO: 需要通过协程仓储查找协程，然后调用 coroutine_start
    // 暂时返回失败，因为缺少协程仓储接口
    bool success = false;
    const char* message = "Coroutine start not implemented (requires coroutine repository)";

    impl->operation_count++;
    impl->last_operation_time = time(NULL);

    pthread_mutex_unlock(&impl->mutex);

    return create_operation_result(success, message);
}

static OperationResultDTO* coroutine_application_service_pause_coroutine_impl(CoroutineApplicationService* service, const PauseCoroutineCommand* cmd) {
    // 实现协程暂停用例
    return create_operation_result(false, "Not implemented");
}

static OperationResultDTO* coroutine_application_service_resume_coroutine_impl(CoroutineApplicationService* service, const ResumeCoroutineCommand* cmd) {
    // 实现协程恢复用例
    return create_operation_result(false, "Not implemented");
}

static OperationResultDTO* coroutine_application_service_cancel_coroutine_impl(CoroutineApplicationService* service, const CancelCoroutineCommand* cmd) {
    // 实现协程取消用例
    return create_operation_result(false, "Not implemented");
}

static CoroutineDTO* coroutine_application_service_get_coroutine_impl(CoroutineApplicationService* service, uint64_t coroutine_id) {
    // 实现协程查询用例
    return NULL; // 暂时未实现
}

static CoroutineListDTO* coroutine_application_service_get_coroutines_impl(CoroutineApplicationService* service, const GetCoroutineStatusQuery* query) {
    // 实现协程列表查询用例
    return NULL; // 暂时未实现
}

static CoroutineStatisticsDTO* coroutine_application_service_get_coroutine_statistics_impl(CoroutineApplicationService* service, const GetCoroutineStatusQuery* query) {
    // 实现协程统计查询用例
    return NULL; // 暂时未实现
}

static OperationResultDTO* coroutine_application_service_yield_current_impl(CoroutineApplicationService* service) {
    // 实现当前协程让步用例
    return create_operation_result(false, "Not implemented");
}

static OperationResultDTO* coroutine_application_service_set_coroutine_priority_impl(CoroutineApplicationService* service, const SetCoroutinePriorityCommand* cmd) {
    // 实现协程优先级设置用例
    return create_operation_result(false, "Not implemented");
}

static OperationResultDTO* coroutine_application_service_wait_coroutine_impl(CoroutineApplicationService* service, const WaitCoroutineCommand* cmd) {
    // 实现协程等待用例
    return create_operation_result(false, "Not implemented");
}

static OperationResultDTO* coroutine_application_service_join_coroutine_impl(CoroutineApplicationService* service, const JoinCoroutineCommand* cmd) {
    // 实现协程join用例
    return create_operation_result(false, "Not implemented");
}
