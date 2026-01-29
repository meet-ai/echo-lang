#include "task_application_service.h"
#include "../../shared/types/common_types.h"
#include "../../domain/task/entity/task.h"
#include "../../domain/task/service/task_scheduler.h"
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

// 辅助函数：计算任务执行时间（毫秒）
static uint64_t calculate_execution_time_ms(const Task* task) {
    uint64_t time_us = task_get_execution_time_us(task);
    return time_us / 1000; // 转换为毫秒
}

// 内部实现结构体
typedef struct TaskApplicationServiceImpl {
    TaskApplicationService base;
    // 私有成员
    pthread_mutex_t mutex;           // 保护并发访问
    char last_error[1024];          // 最后错误信息
    time_t last_operation_time;     // 最后操作时间
    uint64_t operation_count;       // 操作计数
} TaskApplicationServiceImpl;

// 虚函数表实现
static TaskCreationResultDTO* task_application_service_create_task_impl(TaskApplicationService* service, const CreateTaskCommand* cmd);
static TaskCancellationResultDTO* task_application_service_cancel_task_impl(TaskApplicationService* service, const CancelTaskCommand* cmd);
static OperationResultDTO* task_application_service_pause_task_impl(TaskApplicationService* service, const PauseTaskCommand* cmd);
static OperationResultDTO* task_application_service_resume_task_impl(TaskApplicationService* service, const ResumeTaskCommand* cmd);
static OperationResultDTO* task_application_service_update_task_priority_impl(TaskApplicationService* service, const UpdateTaskPriorityCommand* cmd);
static TaskDTO* task_application_service_get_task_impl(TaskApplicationService* service, uint64_t task_id);
static TaskListDTO* task_application_service_get_tasks_impl(TaskApplicationService* service, const GetTaskStatusQuery* query);
static TaskStatisticsDTO* task_application_service_get_task_statistics_impl(TaskApplicationService* service, const GetTaskStatusQuery* query);
static OperationResultDTO* task_application_service_start_scheduler_impl(TaskApplicationService* service);
static OperationResultDTO* task_application_service_stop_scheduler_impl(TaskApplicationService* service);
static OperationResultDTO* task_application_service_pause_scheduler_impl(TaskApplicationService* service);
static OperationResultDTO* task_application_service_resume_scheduler_impl(TaskApplicationService* service);
static uint32_t task_application_service_get_running_task_count_impl(TaskApplicationService* service);
static uint32_t task_application_service_get_queued_task_count_impl(TaskApplicationService* service);
static bool task_application_service_is_scheduler_running_impl(TaskApplicationService* service);

// 虚函数表定义
static TaskApplicationServiceInterface vtable = {
    .create_task = task_application_service_create_task_impl,
    .cancel_task = task_application_service_cancel_task_impl,
    .pause_task = task_application_service_pause_task_impl,
    .resume_task = task_application_service_resume_task_impl,
    .update_task_priority = task_application_service_update_task_priority_impl,
    .get_task = task_application_service_get_task_impl,
    .get_tasks = task_application_service_get_tasks_impl,
    .get_task_statistics = task_application_service_get_task_statistics_impl,
    .start_scheduler = task_application_service_start_scheduler_impl,
    .stop_scheduler = task_application_service_stop_scheduler_impl,
    .pause_scheduler = task_application_service_pause_scheduler_impl,
    .resume_scheduler = task_application_service_resume_scheduler_impl,
    .get_running_task_count = task_application_service_get_running_task_count_impl,
    .get_queued_task_count = task_application_service_get_queued_task_count_impl,
    .is_scheduler_running = task_application_service_is_scheduler_running_impl,
};

// 创建DTO对象的辅助函数
static TaskCreationResultDTO* create_task_creation_result(uint64_t task_id, bool success, const char* message) {
    TaskCreationResultDTO* result = (TaskCreationResultDTO*)malloc(sizeof(TaskCreationResultDTO));
    if (!result) return NULL;

    result->task_id = task_id;
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

static TaskCancellationResultDTO* create_task_cancellation_result(uint64_t task_id, bool success, const char* reason) {
    TaskCancellationResultDTO* result = (TaskCancellationResultDTO*)malloc(sizeof(TaskCancellationResultDTO));
    if (!result) return NULL;

    result->task_id = task_id;
    result->success = success;
    result->cancelled_at = time(NULL);

    if (reason) {
        strncpy(result->reason, reason, sizeof(result->reason) - 1);
        result->reason[sizeof(result->reason) - 1] = '\0';
    } else {
        result->reason[0] = '\0';
    }

    return result;
}

static TaskDTO* create_task_dto(const Task* task) {
    if (!task) return NULL;

    TaskDTO* dto = (TaskDTO*)malloc(sizeof(TaskDTO));
    if (!dto) return NULL;

    dto->id = task->id;
    strncpy(dto->priority, task_priority_string(task->priority), sizeof(dto->priority) - 1);
    dto->priority[sizeof(dto->priority) - 1] = '\0';
    strncpy(dto->status, task_status_string(task->status), sizeof(dto->status) - 1);
    dto->status[sizeof(dto->status) - 1] = '\0';
    dto->created_at = task->created_at;
    dto->started_at = task->started_at;
    dto->completed_at = task->completed_at;
    dto->exit_code = task->exit_code;
    dto->execution_time_ms = calculate_execution_time_ms(task);

    if (task->name[0]) {
        strncpy(dto->name, task->name, sizeof(dto->name) - 1);
        dto->name[sizeof(dto->name) - 1] = '\0';
    } else {
        dto->name[0] = '\0';
    }

    if (task->description[0]) {
        strncpy(dto->description, task->description, sizeof(dto->description) - 1);
        dto->description[sizeof(dto->description) - 1] = '\0';
    } else {
        dto->description[0] = '\0';
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
TaskApplicationService* task_application_service_create(void) {
    TaskApplicationServiceImpl* impl = (TaskApplicationServiceImpl*)malloc(sizeof(TaskApplicationServiceImpl));
    if (!impl) return NULL;

    memset(impl, 0, sizeof(TaskApplicationServiceImpl));

    // 初始化基类
    impl->base.vtable = &vtable;
    impl->base.initialized = false;
    strncpy(impl->base.service_name, "TaskApplicationService", sizeof(impl->base.service_name) - 1);
    impl->base.started_at = time(NULL);

    // 初始化私有成员
    pthread_mutex_init(&impl->mutex, NULL);
    impl->last_operation_time = time(NULL);
    impl->operation_count = 0;

    return (TaskApplicationService*)impl;
}

void task_application_service_destroy(TaskApplicationService* service) {
    if (!service) return;

    TaskApplicationServiceImpl* impl = (TaskApplicationServiceImpl*)service;

    // 清理资源
    task_application_service_cleanup(service);

    // 销毁互斥锁
    pthread_mutex_destroy(&impl->mutex);

    free(impl);
}

// 初始化和清理
bool task_application_service_initialize(TaskApplicationService* service,
                                       void* task_scheduler,
                                       void* task_repository,
                                       void* event_publisher) {
    if (!service) return false;

    TaskApplicationServiceImpl* impl = (TaskApplicationServiceImpl*)service;

    pthread_mutex_lock(&impl->mutex);

    // 设置依赖
    impl->base.task_scheduler = task_scheduler;
    impl->base.task_repository = task_repository;
    impl->base.event_publisher = event_publisher;

    impl->base.initialized = true;

    pthread_mutex_unlock(&impl->mutex);

    return true;
}

void task_application_service_cleanup(TaskApplicationService* service) {
    if (!service) return;

    TaskApplicationServiceImpl* impl = (TaskApplicationServiceImpl*)service;

    pthread_mutex_lock(&impl->mutex);

    // 清理依赖引用
    impl->base.task_scheduler = NULL;
    impl->base.task_repository = NULL;
    impl->base.event_publisher = NULL;

    impl->base.initialized = false;

    pthread_mutex_unlock(&impl->mutex);
}

// 服务状态查询
bool task_application_service_is_healthy(const TaskApplicationService* service) {
    if (!service) return false;
    return service->initialized;
}

const char* task_application_service_get_name(const TaskApplicationService* service) {
    return service ? service->service_name : NULL;
}

time_t task_application_service_get_started_at(const TaskApplicationService* service) {
    return service ? service->started_at : 0;
}

// 虚函数表实现

static TaskCreationResultDTO* task_application_service_create_task_impl(TaskApplicationService* service, const CreateTaskCommand* cmd) {
    if (!service || !cmd || !service->initialized) {
        return create_task_creation_result(0, false, "Service not initialized or invalid command");
    }

    TaskApplicationServiceImpl* impl = (TaskApplicationServiceImpl*)service;

    // 用例编排：创建任务
    // 1. 验证命令参数
    // 2. 创建领域实体
    // 3. 保存到仓储
    // 4. 提交到调度器
    // 5. 发布领域事件

    pthread_mutex_lock(&impl->mutex);

    // 1. 验证参数
    if (strlen(cmd->name) == 0) {
        pthread_mutex_unlock(&impl->mutex);
        return create_task_creation_result(0, false, "Task name is required");
    }

    if (!cmd->entry_point) {
        pthread_mutex_unlock(&impl->mutex);
        return create_task_creation_result(0, false, "Task entry point is required");
    }

    // 2. 创建任务实体
    Task* task = task_create(cmd->name, cmd->entry_point, cmd->arg, cmd->stack_size);
    if (!task) {
        pthread_mutex_unlock(&impl->mutex);
        return create_task_creation_result(0, false, "Failed to create task entity");
    }

    // 3. 设置任务属性
    // TODO: task_set_description 函数不存在，需要添加或使用其他方式设置描述
    // 临时实现：直接设置 task->description 字段（如果存在）
    // if (strlen(cmd->description) > 0) {
    //     task_set_description(task, cmd->description);
    // }
    // 注意：CreateTaskCommand 中没有 priority 字段，使用默认优先级

    // 4. 保存到仓储（如果有仓储接口）
    // 这里应该调用仓储接口保存任务
    // 暂时跳过

    // 5. 提交到调度器
    if (impl->base.task_scheduler) {
        TaskScheduler* scheduler = (TaskScheduler*)impl->base.task_scheduler;
        if (!task_scheduler_submit(scheduler, task)) {
            task_entity_destroy(task);  // 使用task_entity_destroy，因为task_create来自task/entity/task.h
            pthread_mutex_unlock(&impl->mutex);
            return create_task_creation_result(0, false, "Failed to submit task to scheduler");
        }
    }

    // 6. 发布事件（如果有事件发布器）
    // 这里应该发布TaskCreated事件
    // 暂时跳过

    impl->operation_count++;
    impl->last_operation_time = time(NULL);

    pthread_mutex_unlock(&impl->mutex);

    return create_task_creation_result(task->id, true, "Task created successfully");
}

static TaskCancellationResultDTO* task_application_service_cancel_task_impl(TaskApplicationService* service, const CancelTaskCommand* cmd) {
    if (!service || !cmd || !service->initialized) {
        return create_task_cancellation_result(cmd ? cmd->task_id : 0, false, "Service not initialized or invalid command");
    }

    TaskApplicationServiceImpl* impl = (TaskApplicationServiceImpl*)service;

    // 用例编排：取消任务
    // 1. 从调度器取消任务
    // 2. 更新任务状态
    // 3. 发布取消事件

    pthread_mutex_lock(&impl->mutex);

    bool success = false;
    const char* message = "Task cancellation failed";

    if (impl->base.task_scheduler) {
        TaskScheduler* scheduler = (TaskScheduler*)impl->base.task_scheduler;
        success = task_scheduler_cancel(scheduler, cmd->task_id);
        if (success) {
            message = "Task cancelled successfully";
        } else {
            message = "Task not found or cannot be cancelled";
        }
    }

    impl->operation_count++;
    impl->last_operation_time = time(NULL);

    pthread_mutex_unlock(&impl->mutex);

    return create_task_cancellation_result(cmd->task_id, success, message);
}

static OperationResultDTO* task_application_service_pause_task_impl(TaskApplicationService* service, const PauseTaskCommand* cmd) {
    // 实现任务暂停用例
    return create_operation_result(false, "Not implemented");
}

static OperationResultDTO* task_application_service_resume_task_impl(TaskApplicationService* service, const ResumeTaskCommand* cmd) {
    // 实现任务恢复用例
    return create_operation_result(false, "Not implemented");
}

static OperationResultDTO* task_application_service_update_task_priority_impl(TaskApplicationService* service, const UpdateTaskPriorityCommand* cmd) {
    // 实现任务优先级更新用例
    return create_operation_result(false, "Not implemented");
}

static TaskDTO* task_application_service_get_task_impl(TaskApplicationService* service, uint64_t task_id) {
    if (!service || !service->initialized) {
        return NULL;
    }

    // 这里应该从仓储中查询任务
    // 暂时返回NULL
    return NULL;
}

static TaskListDTO* task_application_service_get_tasks_impl(TaskApplicationService* service, const GetTaskStatusQuery* query) {
    // 实现任务列表查询用例
    return NULL; // 暂时未实现
}

static TaskStatisticsDTO* task_application_service_get_task_statistics_impl(TaskApplicationService* service, const GetTaskStatusQuery* query) {
    // 实现任务统计查询用例
    return NULL; // 暂时未实现
}

static OperationResultDTO* task_application_service_start_scheduler_impl(TaskApplicationService* service) {
    if (!service || !service->initialized) {
        return create_operation_result(false, "Service not initialized");
    }

    TaskApplicationServiceImpl* impl = (TaskApplicationServiceImpl*)service;

    pthread_mutex_lock(&impl->mutex);

    if (impl->base.task_scheduler) {
        TaskScheduler* scheduler = (TaskScheduler*)impl->base.task_scheduler;
        task_scheduler_start(scheduler);
    }

    impl->operation_count++;
    impl->last_operation_time = time(NULL);

    pthread_mutex_unlock(&impl->mutex);

    return create_operation_result(true, "Scheduler started");
}

static OperationResultDTO* task_application_service_stop_scheduler_impl(TaskApplicationService* service) {
    if (!service || !service->initialized) {
        return create_operation_result(false, "Service not initialized");
    }

    TaskApplicationServiceImpl* impl = (TaskApplicationServiceImpl*)service;

    pthread_mutex_lock(&impl->mutex);

    if (impl->base.task_scheduler) {
        TaskScheduler* scheduler = (TaskScheduler*)impl->base.task_scheduler;
        task_scheduler_stop(scheduler);
    }

    impl->operation_count++;
    impl->last_operation_time = time(NULL);

    pthread_mutex_unlock(&impl->mutex);

    return create_operation_result(true, "Scheduler stopped");
}

static OperationResultDTO* task_application_service_pause_scheduler_impl(TaskApplicationService* service) {
    if (!service || !service->initialized) {
        return create_operation_result(false, "Service not initialized");
    }

    TaskApplicationServiceImpl* impl = (TaskApplicationServiceImpl*)service;

    pthread_mutex_lock(&impl->mutex);

    if (impl->base.task_scheduler) {
        TaskScheduler* scheduler = (TaskScheduler*)impl->base.task_scheduler;
        task_scheduler_pause(scheduler);
    }

    impl->operation_count++;
    impl->last_operation_time = time(NULL);

    pthread_mutex_unlock(&impl->mutex);

    return create_operation_result(true, "Scheduler paused");
}

static OperationResultDTO* task_application_service_resume_scheduler_impl(TaskApplicationService* service) {
    if (!service || !service->initialized) {
        return create_operation_result(false, "Service not initialized");
    }

    TaskApplicationServiceImpl* impl = (TaskApplicationServiceImpl*)service;

    pthread_mutex_lock(&impl->mutex);

    if (impl->base.task_scheduler) {
        TaskScheduler* scheduler = (TaskScheduler*)impl->base.task_scheduler;
        task_scheduler_resume(scheduler);
    }

    impl->operation_count++;
    impl->last_operation_time = time(NULL);

    pthread_mutex_unlock(&impl->mutex);

    return create_operation_result(true, "Scheduler resumed");
}

static uint32_t task_application_service_get_running_task_count_impl(TaskApplicationService* service) {
    if (!service || !service->initialized || !service->task_scheduler) {
        return 0;
    }

    TaskScheduler* scheduler = (TaskScheduler*)service->task_scheduler;
    return task_scheduler_get_running_count(scheduler);
}

static uint32_t task_application_service_get_queued_task_count_impl(TaskApplicationService* service) {
    if (!service || !service->initialized || !service->task_scheduler) {
        return 0;
    }

    TaskScheduler* scheduler = (TaskScheduler*)service->task_scheduler;
    return task_scheduler_get_queued_count(scheduler);
}

static bool task_application_service_is_scheduler_running_impl(TaskApplicationService* service) {
    if (!service || !service->initialized || !service->task_scheduler) {
        return false;
    }

    TaskScheduler* scheduler = (TaskScheduler*)service->task_scheduler;
    return task_scheduler_is_running(scheduler);
}
