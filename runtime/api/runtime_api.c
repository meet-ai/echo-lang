#include "runtime_api.h"
#include "../shared/types/common_types.h"
#include "../application/services/runtime_application_service.h"
#include "../application/commands/task_commands.h"
#include "../application/commands/coroutine_commands.h"
#include "../application/dtos/result_dtos.h"
#include "../application/dtos/status_dtos.h"
#include "../domain/coroutine/coroutine.h"
#include "../shared/result/result.h"
#include "../shared/handles/file_handle_manager.h"
#include "../shared/handles/socket_handle_manager.h"
#include <stdlib.h>
#include <string.h>
#include <stdio.h>
#include <ctype.h>
#include <stdarg.h>
#include <unistd.h>
#include <time.h>
#include <sys/time.h>
#ifdef __USE_XOPEN
#include <time.h>  // 确保包含 strptime（如果可用）
#endif
#include <sys/types.h>
#include <wchar.h>
#include <locale.h>
#include <fcntl.h>
#include <errno.h>
#include <pthread.h>
#include <limits.h>
#include <stdatomic.h>
#include <sys/socket.h>
#include <netinet/in.h>
#include <arpa/inet.h>
#include <netdb.h>
#include <math.h>
#include <stdint.h>
#include "../application/services/task_application_service.h"
#include "../application/services/coroutine_application_service.h"
#include "../application/services/channel_application_service.h"
#include "../application/services/async_application_service.h"
// EventBus初始化器（基础设施层）
#include "../infrastructure/event/event_bus_initializer.h"

// 内部结构体定义
struct RuntimeHandle {
    RuntimeApplicationService* app_service;
    TaskApplicationService* task_service;
    CoroutineApplicationService* coroutine_service;
    ChannelApplicationService* channel_service;
    AsyncApplicationService* async_service;
    void* config;
    char last_error[1024];
    int last_error_code;
    bool initialized;
};

// 错误代码定义
#define RUNTIME_ERROR_NONE              0
#define RUNTIME_ERROR_INVALID_ARGUMENT  1
#define RUNTIME_ERROR_OUT_OF_MEMORY     2
#define RUNTIME_ERROR_NOT_INITIALIZED   3
#define RUNTIME_ERROR_ALREADY_EXISTS    4
#define RUNTIME_ERROR_NOT_FOUND         5
#define RUNTIME_ERROR_TIMEOUT           6
#define RUNTIME_ERROR_PERMISSION_DENIED 7
#define RUNTIME_ERROR_SYSTEM_ERROR      8
#define RUNTIME_ERROR_INVALID_STATE     9
#define RUNTIME_ERROR_OPERATION_FAILED  10

// 内部辅助函数
static void set_error(RuntimeHandle* runtime, int error_code, const char* format, ...) {
    if (!runtime) return;

    runtime->last_error_code = error_code;

    va_list args;
    va_start(args, format);
    vsnprintf(runtime->last_error, sizeof(runtime->last_error), format, args);
    va_end(args);
}

static void clear_error(RuntimeHandle* runtime) {
    if (!runtime) return;

    runtime->last_error_code = RUNTIME_ERROR_NONE;
    runtime->last_error[0] = '\0';
}

// 验证配置参数
static bool validate_config(const RuntimeConfig* config) {
    if (!config) return false;

    // 检查基本配置
    if (config->heap_size == 0 || config->stack_size == 0) {
        return false;
    }

    if (config->max_tasks == 0 || config->max_coroutines == 0) {
        return false;
    }

    return true;
}

// 创建默认配置
static RuntimeConfig* create_default_config(void) {
    RuntimeConfig* config = (RuntimeConfig*)malloc(sizeof(RuntimeConfig));
    if (!config) return NULL;

    memset(config, 0, sizeof(RuntimeConfig));

    // 基本配置
    config->heap_size = 64 * 1024 * 1024;  // 64MB
    config->stack_size = 64 * 1024;        // 64KB
    config->max_tasks = 10000;
    config->max_coroutines = 1000;
    config->enable_gc = true;
    config->enable_profiling = false;
    config->log_level = strdup("info");

    return config;
}

static RuntimeConfig* convert_config(const RuntimeConfig* api_config) {
    if (!api_config) return NULL;

    RuntimeConfig* config = calloc(1, sizeof(RuntimeConfig));
    if (!config) return NULL;

    // 转换API配置到内部配置
    config->heap_size = api_config->heap_size;
    config->stack_size = api_config->stack_size;
    config->max_tasks = api_config->max_tasks;
    config->max_coroutines = api_config->max_coroutines;
    config->enable_gc = api_config->enable_gc;
    config->enable_profiling = api_config->enable_profiling;

    if (api_config->log_level) {
        // 这里应该复制字符串，但为了简化暂时不实现
        config->log_level = "info"; // 默认值
    }

    return config;
}

// API实现

RuntimeHandle* runtime_init(const RuntimeConfig* config) {
    if (!config) {
        return NULL;
    }

    RuntimeHandle* runtime = calloc(1, sizeof(RuntimeHandle));
    if (!runtime) {
        return NULL;
    }

    // 转换配置
    runtime->config = convert_config(config);
    if (!runtime->config) {
        free(runtime);
        return NULL;
    }

    // 初始化应用服务
    runtime->app_service = runtime_application_service_create();
    if (!runtime->app_service) {
        free(runtime->config);
        free(runtime);
        return NULL;
    }

    // 初始化EventBus并注入到适配层（阶段7：事件发布机制更新）
    EventBus* event_bus = event_bus_initializer_setup(100);  // 默认缓冲大小100
    if (!event_bus) {
        fprintf(stderr, "ERROR: Failed to initialize EventBus\n");
        runtime_application_service_destroy(runtime->app_service);
        free(runtime->config);
        free(runtime);
        return NULL;
    }
    
    // 初始化各个专门的应用服务
    runtime->task_service = task_application_service_create();
    runtime->coroutine_service = coroutine_application_service_create();
    runtime->channel_service = channel_application_service_create();
    runtime->async_service = async_application_service_create();
    
    // 初始化AsyncApplicationService并注入EventBus
    if (runtime->async_service) {
        // TODO: 需要传入async_runtime和future_manager，目前先传入NULL
        if (!async_application_service_initialize(runtime->async_service, NULL, NULL, event_bus)) {
            fprintf(stderr, "ERROR: Failed to initialize AsyncApplicationService\n");
            async_application_service_destroy(runtime->async_service);
            runtime->async_service = NULL;
        }
    }

    // 检查所有服务是否创建成功
    if (!runtime->task_service || !runtime->coroutine_service ||
        !runtime->channel_service || !runtime->async_service) {
        // 清理已创建的服务
        if (runtime->async_service) async_application_service_destroy(runtime->async_service);
        if (runtime->channel_service) channel_application_service_destroy(runtime->channel_service);
        if (runtime->coroutine_service) coroutine_application_service_destroy(runtime->coroutine_service);
        if (runtime->task_service) task_application_service_destroy(runtime->task_service);
        runtime_application_service_destroy(runtime->app_service);
        free(runtime->config);
        free(runtime);
        return NULL;
    }

    runtime->initialized = true;
    clear_error(runtime);

    return runtime;
}

void runtime_shutdown(RuntimeHandle* runtime) {
    if (!runtime || !runtime->initialized) {
        return;
    }

    // 清理EventBus（阶段7：事件发布机制更新）
    EventBus* event_bus = event_bus_initializer_get();
    if (event_bus) {
        event_bus_initializer_cleanup(event_bus);
    }

    // 关闭各个应用服务
    if (runtime->async_service) {
        async_application_service_destroy(runtime->async_service);
    }
    if (runtime->channel_service) {
        channel_application_service_destroy(runtime->channel_service);
    }
    if (runtime->coroutine_service) {
        coroutine_application_service_destroy(runtime->coroutine_service);
    }
    if (runtime->task_service) {
        task_application_service_destroy(runtime->task_service);
    }

    // 关闭主应用服务
    if (runtime->app_service) {
        OperationResultDTO* result = runtime_application_service_shutdown_runtime(runtime->app_service);
        if (result) {
            operation_result_dto_destroy(result);
        }
        runtime_application_service_destroy(runtime->app_service);
    }

    // 清理配置
    if (runtime->config) {
        free(runtime->config);
    }

    // 清理句柄
    free(runtime);
}

const char* runtime_get_version(void) {
    return "Echo Runtime v1.0.0";
}

bool runtime_get_status(RuntimeHandle* runtime, char* json_buffer, size_t buffer_size) {
    if (!runtime || !runtime->initialized) {
        return false;
    }

    RuntimeStatusDTO* status = runtime_application_service_get_runtime_status(runtime->app_service, NULL);
    if (!status) {
        set_error(runtime, RUNTIME_ERROR_SYSTEM_ERROR, "Failed to get runtime status");
        return false;
    }

    // 这里应该序列化status为JSON，但为了简化暂时返回固定字符串
    const char* status_json = "{\"status\":\"running\",\"version\":\"1.0.0\"}";
    size_t json_len = strlen(status_json);

    if (json_len >= buffer_size) {
        runtime_status_dto_destroy(status);
        set_error(runtime, RUNTIME_ERROR_INVALID_ARGUMENT, "Buffer too small");
        return false;
    }

    strcpy(json_buffer, status_json);
    runtime_status_dto_destroy(status);

    return true;
}

// 任务管理API实现
TaskHandle* runtime_create_task(RuntimeHandle* runtime,
                               void (*function)(void*),
                               void* arg,
                               size_t stack_size) {
    if (!runtime || !runtime->initialized || !function) {
        return NULL;
    }

    // 简化实现：直接创建任务句柄
    // TODO: 实现完整的任务创建逻辑
    TaskHandle* handle = calloc(1, sizeof(TaskHandle));
    if (!handle) {
        set_error(runtime, RUNTIME_ERROR_OUT_OF_MEMORY, "Failed to allocate task handle");
        return NULL;
    }

    // 分配一个简单的ID
    static uint64_t next_task_id = 1;
    handle->id = next_task_id++;

    return handle;
}

bool runtime_start_task(RuntimeHandle* runtime, TaskHandle* task) {
    if (!runtime || !runtime->initialized || !task) {
        set_error(runtime, RUNTIME_ERROR_INVALID_ARGUMENT, "Invalid arguments");
        return false;
    }

    // 简化实现：总是返回成功
    // TODO: 实现真正的任务启动逻辑
    return true;
}

bool runtime_wait_task(RuntimeHandle* runtime, TaskHandle* task, uint32_t timeout_ms) {
    if (!runtime || !runtime->initialized || !task) {
        set_error(runtime, RUNTIME_ERROR_INVALID_ARGUMENT, "Invalid arguments");
        return false;
    }

    // 简化实现：总是返回成功
    // TODO: 实现真正的任务等待逻辑
    return true;
}

// 前向声明：标准库版本的取消任务函数（在文件后面定义）
void* runtime_cancel_task(int32_t task_id);

// 旧版本的取消任务函数（需要 RuntimeHandle，保留用于向后兼容）
bool runtime_cancel_task_with_handle(RuntimeHandle* runtime, TaskHandle* task) {
    if (!runtime || !runtime->initialized || !task) {
        set_error(runtime, RUNTIME_ERROR_INVALID_ARGUMENT, "Invalid arguments");
        return false;
    }

    // 调用新的实现
    void* result = runtime_cancel_task((int32_t)task->id);
    if (result && result_is_ok(result)) {
        result_free(result);
        return true;
    }
    if (result) {
        result_free(result);
    }
    return false;
}

bool runtime_get_task_status(RuntimeHandle* runtime, TaskHandle* task,
                           char* status_buffer, size_t buffer_size) {
    if (!runtime || !runtime->initialized || !task || !status_buffer) {
        set_error(runtime, RUNTIME_ERROR_INVALID_ARGUMENT, "Invalid arguments");
        return false;
    }

    // 简化实现：返回假的状态信息
    // TODO: 实现真正的任务状态查询逻辑
    const char* status_json = "{\"id\":1,\"name\":\"api_task\",\"status\":\"RUNNING\"}";
    size_t json_len = strlen(status_json);

    if (json_len >= buffer_size) {
        set_error(runtime, RUNTIME_ERROR_INVALID_ARGUMENT, "Buffer too small");
        return false;
    }

    strcpy(status_buffer, status_json);
    return true;
}

void runtime_destroy_task(RuntimeHandle* runtime, TaskHandle* task) {
    if (!runtime || !task) {
        return;
    }

    // 这里可以添加清理逻辑
    free(task);
}

// 协程管理API实现
CoroutineHandle* runtime_create_coroutine(RuntimeHandle* runtime,
                                        void (*function)(void*),
                                        void* arg,
                                        size_t stack_size) {
    if (!runtime || !runtime->initialized || !function) {
        return NULL;
    }

    CreateCoroutineCommand cmd = {
        .name = "api_created_coroutine",
        .entry_point = function,
        .arg = arg,
        .stack_size = stack_size > 0 ? stack_size : 4096,
        .priority = 5,
        .auto_start = false
    };

    CoroutineCreationResultDTO* result = runtime->coroutine_service->vtable->create_coroutine(runtime->coroutine_service, &cmd);
    if (!result || !result->success) {
        if (result) {
            coroutine_creation_result_dto_destroy(result);
        }
        set_error(runtime, RUNTIME_ERROR_SYSTEM_ERROR, "Failed to create coroutine");
        return NULL;
    }

    CoroutineHandle* handle = calloc(1, sizeof(CoroutineHandle));
    if (!handle) {
        coroutine_creation_result_dto_destroy(result);
        set_error(runtime, RUNTIME_ERROR_OUT_OF_MEMORY, "Failed to allocate coroutine handle");
        return NULL;
    }

    // 这里需要从结果中获取协程ID，暂时使用一个假ID
    handle->id = result->coroutine_id; // 从结果中获取真实ID

    coroutine_creation_result_dto_destroy(result);

    return handle;
}

bool runtime_start_coroutine(RuntimeHandle* runtime, CoroutineHandle* coroutine) {
    if (!runtime || !runtime->initialized || !coroutine) {
        set_error(runtime, RUNTIME_ERROR_INVALID_ARGUMENT, "Invalid arguments");
        return false;
    }

    StartCoroutineCommand cmd = { .coroutine_id = coroutine->id };
    OperationResultDTO* result = runtime->coroutine_service->vtable->start_coroutine(runtime->coroutine_service, &cmd);
    if (!result || !result->success) {
        if (result) operation_result_dto_destroy(result);
        set_error(runtime, RUNTIME_ERROR_SYSTEM_ERROR, "Failed to start coroutine");
        return false;
    }
    operation_result_dto_destroy(result);
    return true;
}

bool runtime_resume_coroutine(RuntimeHandle* runtime, CoroutineHandle* coroutine) {
    if (!runtime || !runtime->initialized || !coroutine) {
        set_error(runtime, RUNTIME_ERROR_INVALID_ARGUMENT, "Invalid arguments");
        return false;
    }

    ResumeCoroutineCommand cmd = { .coroutine_id = coroutine->id };
    OperationResultDTO* result = runtime->coroutine_service->vtable->resume_coroutine(runtime->coroutine_service, &cmd);
    if (!result || !result->success) {
        if (result) operation_result_dto_destroy(result);
        set_error(runtime, RUNTIME_ERROR_SYSTEM_ERROR, "Failed to resume coroutine");
        return false;
    }
    operation_result_dto_destroy(result);
    return true;
}

bool runtime_yield_coroutine(RuntimeHandle* runtime) {
    if (!runtime || !runtime->initialized) {
        set_error(runtime, RUNTIME_ERROR_INVALID_ARGUMENT, "Invalid runtime");
        return false;
    }

    // 通过协程应用服务调用yield
    OperationResultDTO* result = runtime->coroutine_service->vtable->yield_current(runtime->coroutine_service);
    bool success = result && result->success;
    if (result) free(result);
    return success;
}

bool runtime_get_coroutine_status(RuntimeHandle* runtime, CoroutineHandle* coroutine,
                                char* status_buffer, size_t buffer_size) {
    if (!runtime || !runtime->initialized || !coroutine || !status_buffer) {
        set_error(runtime, RUNTIME_ERROR_INVALID_ARGUMENT, "Invalid arguments");
        return false;
    }

    CoroutineDTO* coro_status = runtime->coroutine_service->vtable->get_coroutine(runtime->coroutine_service, coroutine->id);
    if (!coro_status) {
        set_error(runtime, RUNTIME_ERROR_NOT_FOUND, "Coroutine not found");
        return false;
    }

    int written = snprintf(status_buffer, buffer_size,
                          "{\"id\":%llu,\"name\":\"%s\",\"status\":\"%s\"}",
                          coro_status->coroutine_id, coro_status->name, coro_status->status);

    coroutine_dto_destroy(coro_status);

    if (written < 0 || (size_t)written >= buffer_size) {
        set_error(runtime, RUNTIME_ERROR_INVALID_ARGUMENT, "Buffer too small");
        return false;
    }

    return true;
}

void runtime_destroy_coroutine(RuntimeHandle* runtime, CoroutineHandle* coroutine) {
    if (!runtime || !coroutine) {
        return;
    }

    free(coroutine);
}

// Future API实现
FutureHandle* runtime_create_future(RuntimeHandle* runtime) {
    if (!runtime || !runtime->initialized) {
        return NULL;
    }

    if (!runtime->async_service) {
        set_error(runtime, RUNTIME_ERROR_INVALID_STATE, "AsyncApplicationService not initialized");
        return NULL;
    }

    // 1. 创建命令对象
    CreateFutureCommand cmd = {0};
    // cmd.description 可以根据需要设置
    // cmd.initial_value 可以根据需要设置

    // 2. 调用应用服务创建Future
    FutureCreationResultDTO* result = async_application_service_create_future(
        runtime->async_service,
        &cmd
    );

    if (!result || !result->success) {
        if (result) {
            set_error(runtime, RUNTIME_ERROR_OPERATION_FAILED, result->message);
            // TODO: 需要释放result DTO
        } else {
            set_error(runtime, RUNTIME_ERROR_OPERATION_FAILED, "Failed to create future");
        }
        return NULL;
    }

    // 3. 创建FutureHandle并返回
    FutureHandle* handle = calloc(1, sizeof(FutureHandle));
    if (!handle) {
        set_error(runtime, RUNTIME_ERROR_OUT_OF_MEMORY, "Failed to allocate future handle");
        // TODO: 需要释放result DTO
        return NULL;
    }

    handle->id = result->future_id;
    handle->internal_data = NULL; // Future聚合根通过仓储管理，不直接存储指针

    // TODO: 需要释放result DTO

    return handle;
}

bool runtime_resolve_future(RuntimeHandle* runtime, FutureHandle* future,
                          const void* result, size_t result_size) {
    if (!runtime || !runtime->initialized || !future) {
        set_error(runtime, RUNTIME_ERROR_INVALID_ARGUMENT, "Invalid arguments");
        return false;
    }

    if (!runtime->async_service) {
        set_error(runtime, RUNTIME_ERROR_INVALID_STATE, "AsyncApplicationService not initialized");
        return false;
    }

    // 1. 创建命令对象
    ResolveFutureCommand cmd = {0};
    cmd.future_id = future->id;
    cmd.result = (void*)result; // 注意：这里传递的是const void*，需要转换

    // 2. 调用应用服务解决Future
    OperationResultDTO* result_dto = async_application_service_resolve_future(
        runtime->async_service,
        &cmd
    );

    if (!result_dto || !result_dto->success) {
        if (result_dto) {
            set_error(runtime, RUNTIME_ERROR_OPERATION_FAILED, result_dto->message);
            // TODO: 需要释放result_dto
        } else {
            set_error(runtime, RUNTIME_ERROR_OPERATION_FAILED, "Failed to resolve future");
        }
        return false;
    }

    // TODO: 需要释放result_dto

    return true;
}

bool runtime_reject_future(RuntimeHandle* runtime, FutureHandle* future,
                         const char* error_message) {
    if (!runtime || !runtime->initialized || !future || !error_message) {
        set_error(runtime, RUNTIME_ERROR_INVALID_ARGUMENT, "Invalid arguments");
        return false;
    }

    if (!runtime->async_service) {
        set_error(runtime, RUNTIME_ERROR_INVALID_STATE, "AsyncApplicationService not initialized");
        return false;
    }

    // 1. 创建命令对象
    RejectFutureCommand cmd = {0};
    cmd.future_id = future->id;
    strncpy(cmd.error_message, error_message, sizeof(cmd.error_message) - 1);
    cmd.error_message[sizeof(cmd.error_message) - 1] = '\0';

    // 2. 调用应用服务拒绝Future
    OperationResultDTO* result_dto = async_application_service_reject_future(
        runtime->async_service,
        &cmd
    );

    if (!result_dto || !result_dto->success) {
        if (result_dto) {
            set_error(runtime, RUNTIME_ERROR_OPERATION_FAILED, result_dto->message);
            // TODO: 需要释放result_dto
        } else {
            set_error(runtime, RUNTIME_ERROR_OPERATION_FAILED, "Failed to reject future");
        }
        return false;
    }

    // TODO: 需要释放result_dto

    return true;
}

bool runtime_await_future(RuntimeHandle* runtime, FutureHandle* future, uint32_t timeout_ms) {
    if (!runtime || !runtime->initialized || !future) {
        set_error(runtime, RUNTIME_ERROR_INVALID_ARGUMENT, "Invalid arguments");
        return false;
    }

    if (!runtime->async_service) {
        set_error(runtime, RUNTIME_ERROR_INVALID_STATE, "AsyncApplicationService not initialized");
        return false;
    }

    // 1. 创建命令对象
    AwaitFutureCommand cmd = {0};
    cmd.future_id = future->id;
    cmd.timeout_ms = timeout_ms;
    cmd.blocking = true; // API层的await通常是阻塞的

    // 2. 调用应用服务等待Future
    AsyncOperationResultDTO* result_dto = async_application_service_await_future(
        runtime->async_service,
        &cmd
    );

    if (!result_dto || !result_dto->success) {
        if (result_dto) {
            set_error(runtime, RUNTIME_ERROR_OPERATION_FAILED, result_dto->message);
            // TODO: 需要释放result_dto
        } else {
            set_error(runtime, RUNTIME_ERROR_OPERATION_FAILED, "Failed to await future");
        }
        return false;
    }

    // TODO: 需要释放result_dto

    return true;
}

bool runtime_get_future_result(RuntimeHandle* runtime, FutureHandle* future,
                             void* result_buffer, size_t buffer_size, size_t* result_size) {
    if (!runtime || !runtime->initialized || !future || !result_buffer || !result_size) {
        set_error(runtime, RUNTIME_ERROR_INVALID_ARGUMENT, "Invalid arguments");
        return false;
    }

    // 这里应该获取future结果
    *result_size = 0;
    return true;
}

bool runtime_get_future_status(RuntimeHandle* runtime, FutureHandle* future,
                             char* status_buffer, size_t buffer_size) {
    if (!runtime || !runtime->initialized || !future || !status_buffer) {
        set_error(runtime, RUNTIME_ERROR_INVALID_ARGUMENT, "Invalid arguments");
        return false;
    }

    if (!runtime->async_service) {
        set_error(runtime, RUNTIME_ERROR_INVALID_STATE, "AsyncApplicationService not initialized");
        return false;
    }

    // 1. 调用应用服务查询Future
    FutureDTO* future_dto = async_application_service_get_future(
        runtime->async_service,
        future->id
    );

    if (!future_dto) {
        set_error(runtime, RUNTIME_ERROR_OPERATION_FAILED, "Failed to get future status");
        return false;
    }

    // 2. 将FutureDTO转换为JSON格式
    // TODO: 需要实现DTO到JSON的转换，或者使用DTO的字段构建JSON
    // 临时实现：使用基本状态信息
    const char* status_json = "{\"status\":\"pending\"}";
    size_t json_len = strlen(status_json);

    if (json_len >= buffer_size) {
        set_error(runtime, RUNTIME_ERROR_INVALID_ARGUMENT, "Buffer too small");
        // TODO: 需要释放future_dto
        return false;
    }

    // TODO: 根据future_dto->state构建完整的JSON状态信息
    // 例如：snprintf(status_buffer, buffer_size, "{\"id\":%llu,\"state\":%d,...}", 
    //                future_dto->future_id, future_dto->state, ...);

    strcpy(status_buffer, status_json);

    // TODO: 需要释放future_dto

    return true;
}

void runtime_destroy_future(RuntimeHandle* runtime, FutureHandle* future) {
    if (!runtime || !future) {
        return;
    }

    free(future);
}

// 前向声明：标准库版本的互斥锁函数（在文件后面定义）
int32_t runtime_create_mutex(void);
void runtime_mutex_lock(int32_t handle);
bool runtime_mutex_try_lock(int32_t handle);
void runtime_mutex_unlock(int32_t handle);

// 并发原语API实现（旧版本，需要 RuntimeHandle，保留用于向后兼容）
// 注意：标准库使用新的 runtime_create_mutex() 函数（不需要 RuntimeHandle）
void* runtime_create_mutex_with_runtime(RuntimeHandle* runtime) {
    if (!runtime || !runtime->initialized) {
        set_error(runtime, RUNTIME_ERROR_INVALID_ARGUMENT, "Invalid runtime");
        return NULL;
    }

    // 调用新的实现
    int32_t handle = runtime_create_mutex();
    if (handle < 0) {
        set_error(runtime, RUNTIME_ERROR_SYSTEM_ERROR, "Failed to create mutex");
        return NULL;
    }
    
    // 返回句柄指针（为了兼容旧的 API）
    int32_t* handle_ptr = (int32_t*)malloc(sizeof(int32_t));
    if (handle_ptr) {
        *handle_ptr = handle;
    }
    return (void*)handle_ptr;
}

void runtime_destroy_mutex(RuntimeHandle* runtime, void* mutex) {
    if (!runtime || !mutex) {
        return;
    }

    // 销毁互斥锁
}

bool runtime_lock_mutex(RuntimeHandle* runtime, void* mutex) {
    if (!runtime || !runtime->initialized || !mutex) {
        set_error(runtime, RUNTIME_ERROR_INVALID_ARGUMENT, "Invalid arguments");
        return false;
    }

    // 锁定互斥锁
    return true;
}

bool runtime_try_lock_mutex(RuntimeHandle* runtime, void* mutex) {
    if (!runtime || !runtime->initialized || !mutex) {
        set_error(runtime, RUNTIME_ERROR_INVALID_ARGUMENT, "Invalid arguments");
        return false;
    }

    return true;
}

bool runtime_unlock_mutex(RuntimeHandle* runtime, void* mutex) {
    if (!runtime || !runtime->initialized || !mutex) {
        set_error(runtime, RUNTIME_ERROR_INVALID_ARGUMENT, "Invalid arguments");
        return false;
    }

    return true;
}

// 通道API实现
ChannelHandle* runtime_create_channel(RuntimeHandle* runtime, size_t element_size, size_t buffer_size) {
    if (!runtime || !runtime->initialized || element_size == 0) {
        return NULL;
    }

    ChannelHandle* handle = calloc(1, sizeof(ChannelHandle));
    if (!handle) {
        set_error(runtime, RUNTIME_ERROR_OUT_OF_MEMORY, "Failed to allocate channel handle");
        return NULL;
    }

    static uint64_t next_channel_id = 1;
    handle->id = next_channel_id++;

    return handle;
}

bool runtime_send_channel(RuntimeHandle* runtime, ChannelHandle* channel,
                        const void* data, uint32_t timeout_ms) {
    if (!runtime || !runtime->initialized || !channel || !data) {
        set_error(runtime, RUNTIME_ERROR_INVALID_ARGUMENT, "Invalid arguments");
        return false;
    }

    return true;
}

bool runtime_receive_channel(RuntimeHandle* runtime, ChannelHandle* channel,
                           void* buffer, size_t buffer_size, uint32_t timeout_ms) {
    if (!runtime || !runtime->initialized || !channel || !buffer) {
        set_error(runtime, RUNTIME_ERROR_INVALID_ARGUMENT, "Invalid arguments");
        return false;
    }

    return true;
}

void runtime_close_channel(RuntimeHandle* runtime, ChannelHandle* channel) {
    if (!runtime || !channel) {
        return;
    }
}

bool runtime_get_channel_status(RuntimeHandle* runtime, ChannelHandle* channel,
                              char* status_buffer, size_t buffer_size) {
    if (!runtime || !runtime->initialized || !channel || !status_buffer) {
        set_error(runtime, RUNTIME_ERROR_INVALID_ARGUMENT, "Invalid arguments");
        return false;
    }

    const char* status_json = "{\"status\":\"open\",\"buffer_size\":0}";
    size_t json_len = strlen(status_json);

    if (json_len >= buffer_size) {
        set_error(runtime, RUNTIME_ERROR_INVALID_ARGUMENT, "Buffer too small");
        return false;
    }

    strcpy(status_buffer, status_json);
    return true;
}

void runtime_destroy_channel(RuntimeHandle* runtime, ChannelHandle* channel) {
    if (!runtime || !channel) {
        return;
    }

    free(channel);
}

// 内存管理API实现
void* runtime_allocate_memory(RuntimeHandle* runtime, size_t size, const char* source) {
    if (!runtime || !runtime->initialized || size == 0) {
        set_error(runtime, RUNTIME_ERROR_INVALID_ARGUMENT, "Invalid arguments");
        return NULL;
    }

    void* ptr = malloc(size);
    if (!ptr) {
        set_error(runtime, RUNTIME_ERROR_OUT_OF_MEMORY, "Failed to allocate memory");
        return NULL;
    }

    return ptr;
}

void* runtime_reallocate_memory(RuntimeHandle* runtime, void* ptr, size_t new_size, const char* source) {
    if (!runtime || !runtime->initialized) {
        set_error(runtime, RUNTIME_ERROR_INVALID_ARGUMENT, "Invalid runtime");
        return NULL;
    }

    void* new_ptr = realloc(ptr, new_size);
    if (!new_ptr && new_size > 0) {
        set_error(runtime, RUNTIME_ERROR_OUT_OF_MEMORY, "Failed to reallocate memory");
        return NULL;
    }

    return new_ptr;
}

void runtime_free_memory(RuntimeHandle* runtime, void* ptr) {
    if (!runtime || !ptr) {
        return;
    }

    free(ptr);
}

bool runtime_get_memory_stats(RuntimeHandle* runtime, char* stats_buffer, size_t buffer_size) {
    if (!runtime || !runtime->initialized || !stats_buffer) {
        set_error(runtime, RUNTIME_ERROR_INVALID_ARGUMENT, "Invalid arguments");
        return false;
    }

    const char* stats_json = "{\"allocated\":0,\"used\":0,\"free\":0}";
    size_t json_len = strlen(stats_json);

    if (json_len >= buffer_size) {
        set_error(runtime, RUNTIME_ERROR_INVALID_ARGUMENT, "Buffer too small");
        return false;
    }

    strcpy(stats_buffer, stats_json);
    return true;
}

// 错误处理API实现
const char* runtime_get_last_error(RuntimeHandle* runtime) {
    if (!runtime) {
        return "Invalid runtime handle";
    }

    return runtime->last_error;
}

int runtime_get_last_error_code(RuntimeHandle* runtime) {
    if (!runtime) {
        return RUNTIME_ERROR_INVALID_ARGUMENT;
    }

    return runtime->last_error_code;
}

void runtime_clear_error(RuntimeHandle* runtime) {
    if (!runtime) {
        return;
    }

    clear_error(runtime);
}

// 工具函数实现
void runtime_sleep_ms(RuntimeHandle* runtime, uint32_t milliseconds) {
    if (!runtime || !runtime->initialized) {
        return;
    }

    usleep(milliseconds * 1000);
}

uint64_t runtime_current_time_ms(RuntimeHandle* runtime) {
    if (!runtime || !runtime->initialized) {
        return 0;
    }

    struct timespec ts;
    clock_gettime(CLOCK_MONOTONIC, &ts);
    return (uint64_t)ts.tv_sec * 1000 + (uint64_t)ts.tv_nsec / 1000000;
}

uint64_t runtime_random_uint64(RuntimeHandle* runtime, uint64_t min, uint64_t max) {
    if (!runtime || !runtime->initialized || min >= max) {
        return min;
    }

    // 简单的随机数生成（实际应该使用更好的随机数生成器）
    return min + (rand() % (max - min + 1));
}

// 信号量API实现（简化版本）
void* runtime_create_semaphore(RuntimeHandle* runtime, uint32_t initial_value) {
    if (!runtime || !runtime->initialized) {
        return NULL;
    }

    return (void*)1; // 临时实现
}

void runtime_destroy_semaphore(RuntimeHandle* runtime, void* semaphore) {
    if (!runtime || !semaphore) {
        return;
    }
}

bool runtime_wait_semaphore(RuntimeHandle* runtime, void* semaphore) {
    if (!runtime || !runtime->initialized || !semaphore) {
        return false;
    }

    return true;
}

bool runtime_try_wait_semaphore(RuntimeHandle* runtime, void* semaphore) {
    if (!runtime || !runtime->initialized || !semaphore) {
        return false;
    }

    return true;
}

bool runtime_post_semaphore(RuntimeHandle* runtime, void* semaphore) {
    if (!runtime || !runtime->initialized || !semaphore) {
        return false;
    }

    return true;
}

// ============================================================================
// P0 优先级运行时函数实现（核心功能）
// ============================================================================
// 这些函数是标准库的核心依赖，必须优先实现

/**
 * @brief 获取当前时间（Unix 时间戳，毫秒）
 * @return 时间戳（毫秒）
 * 
 * 编译器签名：i64 runtime_time_now_ms()
 * 使用位置：stdlib/time/time.eo - Time.now()
 */
int64_t runtime_time_now_ms(void) {
    struct timeval tv;
    if (gettimeofday(&tv, NULL) != 0) {
        return 0; // 获取时间失败，返回 0
    }
    
    // 转换为毫秒
    return (int64_t)tv.tv_sec * 1000LL + (int64_t)tv.tv_usec / 1000LL;
}

// ==================== 时间格式化与解析 ====================

/**
 * @brief 将Go风格的时间格式转换为strftime格式
 * @param go_layout Go风格的时间格式（如 "2006-01-02 15:04:05"）
 * @param strftime_format 输出的strftime格式字符串缓冲区
 * @param buffer_size 缓冲区大小
 * @return 转换后的格式字符串长度，如果失败返回-1
 * 
 * Go时间格式说明符映射：
 * - 2006 -> %Y (4位年份)
 * - 01 -> %m (月份)
 * - 02 -> %d (日期)
 * - 15 -> %H (24小时制小时)
 * - 04 -> %M (分钟)
 * - 05 -> %S (秒)
 * - 06 -> %m (月份，简化)
 * - 03 -> %I (12小时制小时)
 * - PM -> %p (AM/PM)
 */
static int convert_go_layout_to_strftime(const char* go_layout, char* strftime_format, size_t buffer_size) {
    if (!go_layout || !strftime_format || buffer_size == 0) {
        return -1;
    }
    
    size_t go_len = strlen(go_layout);
    size_t out_pos = 0;
    
    for (size_t i = 0; i < go_len && out_pos < buffer_size - 1; i++) {
        // 检查是否是Go格式说明符
        if (i + 3 < go_len && strncmp(go_layout + i, "2006", 4) == 0) {
            // 年份（4位）
            if (out_pos + 2 < buffer_size - 1) {
                strftime_format[out_pos++] = '%';
                strftime_format[out_pos++] = 'Y';
                i += 3; // 跳过 "2006"
            }
        } else if (i + 1 < go_len && strncmp(go_layout + i, "01", 2) == 0) {
            // 月份
            if (out_pos + 2 < buffer_size - 1) {
                strftime_format[out_pos++] = '%';
                strftime_format[out_pos++] = 'm';
                i += 1; // 跳过 "01"
            }
        } else if (i + 1 < go_len && strncmp(go_layout + i, "02", 2) == 0) {
            // 日期
            if (out_pos + 2 < buffer_size - 1) {
                strftime_format[out_pos++] = '%';
                strftime_format[out_pos++] = 'd';
                i += 1; // 跳过 "02"
            }
        } else if (i + 1 < go_len && strncmp(go_layout + i, "15", 2) == 0) {
            // 24小时制小时
            if (out_pos + 2 < buffer_size - 1) {
                strftime_format[out_pos++] = '%';
                strftime_format[out_pos++] = 'H';
                i += 1; // 跳过 "15"
            }
        } else if (i + 1 < go_len && strncmp(go_layout + i, "04", 2) == 0) {
            // 分钟
            if (out_pos + 2 < buffer_size - 1) {
                strftime_format[out_pos++] = '%';
                strftime_format[out_pos++] = 'M';
                i += 1; // 跳过 "04"
            }
        } else if (i + 1 < go_len && strncmp(go_layout + i, "05", 2) == 0) {
            // 秒
            if (out_pos + 2 < buffer_size - 1) {
                strftime_format[out_pos++] = '%';
                strftime_format[out_pos++] = 'S';
                i += 1; // 跳过 "05"
            }
        } else {
            // 普通字符，直接复制
            strftime_format[out_pos++] = go_layout[i];
        }
    }
    
    strftime_format[out_pos] = '\0';
    return (int)out_pos;
}

/**
 * @brief 格式化时间
 * @param sec Unix 时间戳（秒）
 * @param nsec 纳秒偏移（0-999999999）
 * @param layout 格式化模板字符串（i8*）
 * @return 格式化后的字符串指针（i8*），需要调用者释放内存
 * 
 * 编译器签名：i8* runtime_time_format(i64 sec, i32 nsec, i8* layout)
 * 使用位置：stdlib/time/time.eo - Time.format()
 * 
 * 实现：使用 strftime 格式化时间，支持标准格式说明符
 * 注意：返回的字符串是新分配的内存，需要由调用者或GC管理
 */
char* runtime_time_format(int64_t sec, int32_t nsec, const char* layout) {
    if (!layout) {
        char* result = (char*)malloc(1);
        if (result) {
            result[0] = '\0';
        }
        return result;
    }
    
    // 将Unix时间戳转换为struct tm
    time_t time_sec = (time_t)sec;
    struct tm* tm_info = localtime(&time_sec);
    if (!tm_info) {
        char* result = (char*)malloc(1);
        if (result) {
            result[0] = '\0';
        }
        return result;
    }
    
    // 转换Go风格格式为strftime格式
    char strftime_format[256];
    int format_len = convert_go_layout_to_strftime(layout, strftime_format, sizeof(strftime_format));
    if (format_len < 0) {
        // 转换失败，使用原始layout（假设已经是strftime格式）
        strncpy(strftime_format, layout, sizeof(strftime_format) - 1);
        strftime_format[sizeof(strftime_format) - 1] = '\0';
    }
    
    // 使用strftime格式化
    char buffer[512];
    size_t result_len = strftime(buffer, sizeof(buffer), strftime_format, tm_info);
    
    if (result_len == 0) {
        // 格式化失败，返回空字符串
        char* result = (char*)malloc(1);
        if (result) {
            result[0] = '\0';
        }
        return result;
    }
    
    // 如果需要纳秒精度，添加纳秒部分
    // 注意：strftime不支持纳秒，如果需要纳秒，需要在格式字符串中包含特殊标记
    // 这里简化处理，只格式化到秒
    
    // 分配内存并复制结果
    char* result = (char*)malloc(result_len + 1);
    if (!result) {
        return NULL;
    }
    
    strncpy(result, buffer, result_len);
    result[result_len] = '\0';
    
    return result;
}

/**
 * @brief 解析时间字符串
 * @param layout 格式化模板字符串（i8*）
 * @param value 时间字符串（i8*）
 * @return 时间解析结果结构体指针，需要调用者释放内存
 * 
 * 编译器签名：i8* runtime_time_parse(i8* layout, i8* value)
 * 使用位置：stdlib/time/time.eo - parse_time()
 * 
 * 实现：使用 strptime 解析时间字符串（如果可用），否则使用简化解析
 * 注意：返回的结构体包含解析结果，需要由调用者或GC管理
 */
TimeParseResult* runtime_time_parse(const char* layout, const char* value) {
    // 分配结果结构体
    TimeParseResult* result = (TimeParseResult*)malloc(sizeof(TimeParseResult));
    if (!result) {
        return NULL;
    }
    
    // 初始化失败状态
    result->success = false;
    result->sec = 0;
    result->nsec = 0;
    
    if (!layout || !value) {
        return result;
    }
    
    // 转换Go风格格式为strftime格式
    char strftime_format[256];
    int format_len = convert_go_layout_to_strftime(layout, strftime_format, sizeof(strftime_format));
    if (format_len < 0) {
        // 转换失败，使用原始layout（假设已经是strftime格式）
        strncpy(strftime_format, layout, sizeof(strftime_format) - 1);
        strftime_format[sizeof(strftime_format) - 1] = '\0';
    }
    
    // 使用strptime解析（如果可用）
    struct tm tm_info = {0};
    char* remaining = NULL;
    
#ifdef _GNU_SOURCE
    // GNU扩展支持strptime
    remaining = strptime(value, strftime_format, &tm_info);
    if (!remaining || *remaining != '\0') {
        // 解析不完全，可能失败
        // 检查是否至少解析了部分内容
        if (remaining == value) {
            return result; // 完全没有解析，返回失败状态
        }
    }
#else
    // 如果没有strptime，使用简化解析（仅支持ISO 8601格式）
    // 例如：2006-01-02 15:04:05
    if (strcmp(strftime_format, "%Y-%m-%d %H:%M:%S") == 0 ||
        strcmp(strftime_format, "%Y-%m-%d") == 0 ||
        strcmp(strftime_format, "%H:%M:%S") == 0) {
        // 简化解析：直接解析数字
        int year = 0, month = 0, day = 0, hour = 0, min = 0, sec = 0;
        int parsed = sscanf(value, "%d-%d-%d %d:%d:%d", &year, &month, &day, &hour, &min, &sec);
        if (parsed < 3) {
            // 尝试只解析日期
            parsed = sscanf(value, "%d-%d-%d", &year, &month, &day);
            if (parsed != 3) {
                return result; // 解析失败
            }
        } else if (parsed < 6) {
            // 尝试只解析时间
            parsed = sscanf(value, "%d:%d:%d", &hour, &min, &sec);
            if (parsed != 3) {
                return result; // 解析失败
            }
        }
        
        tm_info.tm_year = year - 1900; // tm_year是1900年以来的年数
        tm_info.tm_mon = month - 1;    // tm_mon是0-11
        tm_info.tm_mday = day;
        tm_info.tm_hour = hour;
        tm_info.tm_min = min;
        tm_info.tm_sec = sec;
        remaining = (char*)value + strlen(value); // 标记为完全解析
    } else {
        return result; // 不支持的格式
    }
#endif
    
    // 转换为Unix时间戳
    time_t time_sec = mktime(&tm_info);
    if (time_sec == -1) {
        return result; // 无效时间
    }
    
    // 设置成功结果
    result->success = true;
    result->sec = (int64_t)time_sec;
    result->nsec = 0; // 简化实现，不支持纳秒解析
    
    return result;
}

// ==================== 随机数生成 ====================

/**
 * @brief 获取随机种子（使用系统熵源）
 * @return u64 随机种子值
 * 
 * 编译器签名：u64 runtime_get_random_seed()
 * 使用位置：stdlib/math/random.eo - Rng.new_rng()
 * 
 * 实现：使用当前时间（纳秒级）和进程ID组合生成随机种子
 */
uint64_t runtime_get_random_seed(void) {
    // 获取高精度时间（纳秒级）
    struct timespec ts;
    uint64_t time_seed = 0;
    if (clock_gettime(CLOCK_REALTIME, &ts) == 0) {
        // 组合秒和纳秒部分
        time_seed = (uint64_t)ts.tv_sec * 1000000000ULL + (uint64_t)ts.tv_nsec;
    } else {
        // 后备方案：使用 time() 和微秒级时间
        struct timeval tv;
        if (gettimeofday(&tv, NULL) == 0) {
            time_seed = (uint64_t)tv.tv_sec * 1000000ULL + (uint64_t)tv.tv_usec;
        } else {
            time_seed = (uint64_t)time(NULL) * 1000000ULL;
        }
    }
    
    // 获取进程ID（增加随机性）
    pid_t pid = getpid();
    
    // 组合时间和进程ID生成种子
    // 使用简单的哈希组合，确保不同进程和时间点产生不同的种子
    uint64_t seed = time_seed ^ ((uint64_t)pid << 32) ^ ((uint64_t)pid << 16);
    
    // 如果种子为0，使用默认值
    if (seed == 0) {
        seed = 1234567890123456789ULL;
    }
    
    return seed;
}

/**
 * @brief 计算字符串长度（UTF-8 编码）
 * @param s 字符串指针（i8*）
 * @return 字符数（不是字节数）
 * 
 * 编译器签名：i32 runtime_string_len_utf8(i8* s)
 * 使用位置：stdlib/string/string.eo - string.len()
 */
int32_t runtime_string_len_utf8(const char* s) {
    if (!s) {
        return 0;
    }

    // 计算 UTF-8 字符数
    int32_t count = 0;
    const unsigned char* p = (const unsigned char*)s;
    
    while (*p) {
        // UTF-8 字符的第一个字节决定了字符的字节数
        if ((*p & 0x80) == 0) {
            // ASCII 字符（1 字节）
            count++;
            p++;
        } else if ((*p & 0xE0) == 0xC0) {
            // 2 字节 UTF-8 字符
            count++;
            p += 2;
        } else if ((*p & 0xF0) == 0xE0) {
            // 3 字节 UTF-8 字符
            count++;
            p += 3;
        } else if ((*p & 0xF8) == 0xF0) {
            // 4 字节 UTF-8 字符
            count++;
            p += 4;
        } else {
            // 无效的 UTF-8 序列，跳过这个字节
            p++;
        }
    }
    
    return count;
}

/**
 * @brief 计算字符串哈希值（FNV-1a 算法）
 * @param s 字符串指针（i8*）
 * @return 哈希值（u64）
 * 
 * 编译器签名：i64 runtime_string_hash(i8* s)
 * 使用位置：stdlib/core/primitives.eo - string.hash()
 * 
 * 使用 FNV-1a 哈希算法，这是一个快速且分布良好的哈希算法
 */
int64_t runtime_string_hash(const char* s) {
    if (!s) {
        return 0;
    }

    // FNV-1a 哈希算法的初始值和质数
    uint64_t hash = 2166136261ULL;  // FNV-1a offset basis (32位版本)
    const unsigned char* p = (const unsigned char*)s;
    
    // 遍历字符串的每个字节
    while (*p) {
        hash ^= (uint64_t)(*p);      // XOR 操作
        hash *= 16777619ULL;         // FNV-1a prime (32位版本)
        p++;
    }
    
    return (int64_t)hash;
}

/**
 * @brief 计算32位整数哈希值
 * @param key 整数值（i32）
 * @return 哈希值（i64）
 * 
 * 编译器签名：i64 runtime_int32_hash(i32 key)
 * 使用位置：map[int]string 等整数键Map的哈希计算
 * 
 * 使用乘法哈希算法，提供良好的分布特性
 */
int64_t runtime_int32_hash(int32_t key) {
    // 使用乘法哈希算法（更好的分布）
    // 0x9e3779b9 是黄金比例相关的常数，常用于哈希算法
    return (int64_t)(key * 0x9e3779b9);
}

/**
 * @brief 计算64位整数哈希值
 * @param key 整数值（i64）
 * @return 哈希值（i64）
 * 
 * 编译器签名：i64 runtime_int64_hash(i64 key)
 * 使用位置：map[i64]string 等64位整数键Map的哈希计算
 * 
 * 使用改进的整数哈希算法（Wang hash）
 */
int64_t runtime_int64_hash(int64_t key) {
    // 64位整数哈希（使用Wang hash算法）
    key = (~key) + (key << 21);
    key = key ^ (key >> 24);
    key = (key + (key << 3)) + (key << 8);
    key = key ^ (key >> 14);
    key = (key + (key << 2)) + (key << 4);
    key = key ^ (key >> 28);
    key = key + (key << 31);
    return key;
}

/**
 * @brief 计算浮点数哈希值（位表示转换）
 * @param f 浮点数值（f64）
 * @return 哈希值（u64）
 * 
 * 编译器签名：i64 runtime_float_hash(f64 f)
 * 使用位置：stdlib/core/primitives.eo - float.hash()
 * 
 * 实现：将浮点数的位表示转换为 u64，然后应用哈希算法改善分布
 * 注意：NaN 和 ±0.0 需要特殊处理
 */
int64_t runtime_float_hash(double f) {
    // 使用 union 进行类型转换（类型安全的位操作）
    union {
        double d;
        uint64_t u;
    } converter;
    
    converter.d = f;
    uint64_t bits = converter.u;
    
    // 特殊值处理：
    // - NaN: 所有 NaN 值应该产生相同的哈希值
    // - ±0.0: 正零和负零应该产生相同的哈希值
    if (isnan(f)) {
        // NaN 的位模式可能不同，统一处理为 0x7FF8000000000000
        bits = 0x7FF8000000000000ULL;
    } else if (f == 0.0 && bits != 0) {
        // 负零（-0.0）转换为正零（+0.0）的位模式
        bits = 0;
    }
    
    // 应用简单的哈希算法改善分布（避免直接使用位模式）
    // 使用 FNV-1a 风格的哈希处理
    uint64_t hash = 2166136261ULL;  // FNV-1a offset basis
    
    // 将 64 位分成 8 个字节，逐个处理
    for (int i = 0; i < 8; i++) {
        uint8_t byte = (bits >> (i * 8)) & 0xFF;
        hash ^= (uint64_t)byte;
        hash *= 16777619ULL;  // FNV-1a prime
    }
    
    return (int64_t)hash;
}

/**
 * @brief 检查字符串是否包含子字符串
 * @param s 主字符串指针（i8*）
 * @param sub 子字符串指针（i8*）
 * @return bool -> i1（true 表示包含，false 表示不包含）
 * 
 * 编译器签名：i1 runtime_string_contains(i8* s, i8* sub)
 * 使用位置：stdlib/string/string.eo - string.contains()
 */
bool runtime_string_contains(const char* s, const char* sub) {
    if (!s || !sub) {
        return false;
    }
    
    // 使用 strstr 查找子字符串
    return strstr(s, sub) != NULL;
}

/**
 * @brief 检查字符串是否以指定字符串开头
 * @param s 主字符串指针（i8*）
 * @param prefix 前缀字符串指针（i8*）
 * @return bool -> i1（true 表示以prefix开头，false 表示不是）
 * 
 * 编译器签名：i1 runtime_string_starts_with(i8* s, i8* prefix)
 * 使用位置：stdlib/string/string.eo - string.starts_with()
 */
bool runtime_string_starts_with(const char* s, const char* prefix) {
    if (!s || !prefix) {
        return false;
    }
    
    size_t prefix_len = strlen(prefix);
    if (prefix_len == 0) {
        return true; // 空字符串是任何字符串的前缀
    }
    
    // 使用 strncmp 比较前 prefix_len 个字符
    return strncmp(s, prefix, prefix_len) == 0;
}

/**
 * @brief 检查字符串是否以指定字符串结尾
 * @param s 主字符串指针（i8*）
 * @param suffix 后缀字符串指针（i8*）
 * @return bool -> i1（true 表示以suffix结尾，false 表示不是）
 * 
 * 编译器签名：i1 runtime_string_ends_with(i8* s, i8* suffix)
 * 使用位置：stdlib/string/string.eo - string.ends_with()
 */
bool runtime_string_ends_with(const char* s, const char* suffix) {
    if (!s || !suffix) {
        return false;
    }
    
    size_t s_len = strlen(s);
    size_t suffix_len = strlen(suffix);
    
    if (suffix_len == 0) {
        return true; // 空字符串是任何字符串的后缀
    }
    
    if (suffix_len > s_len) {
        return false; // 后缀长度大于字符串长度
    }
    
    return strcmp(s + (s_len - suffix_len), suffix) == 0;
}

/**
 * @brief 比较两个字符串是否相等
 * @param s1 第一个字符串指针（i8*）
 * @param s2 第二个字符串指针（i8*）
 * @return 布尔值（i1），true表示两个字符串相等
 * 
 * 编译器签名：i1 runtime_string_equals(i8* s1, i8* s2)
 * 使用位置：stdlib/net/http/client.eo - string_to_method()
 * 
 * 实现：使用 strcmp 比较字符串，返回是否相等
 */
bool runtime_string_equals(const char* s1, const char* s2) {
    if (!s1 && !s2) {
        return true; // 两个都是NULL，认为相等
    }
    if (!s1 || !s2) {
        return false; // 一个为NULL，另一个不为NULL，不相等
    }
    return strcmp(s1, s2) == 0;
}

bool runtime_string_ends_with_old(const char* s, const char* suffix) {
    if (!s || !suffix) {
        return false;
    }
    
    size_t s_len = strlen(s);
    size_t suffix_len = strlen(suffix);
    
    if (suffix_len == 0) {
        return true; // 空字符串是任何字符串的后缀
    }
    
    if (suffix_len > s_len) {
        return false; // 后缀长度大于主字符串长度
    }
    
    // 比较主字符串的最后 suffix_len 个字符
    return strncmp(s + (s_len - suffix_len), suffix, suffix_len) == 0;
}

/**
 * @brief 去除字符串首尾空白字符
 * @param s 字符串指针（i8*）
 * @return 新字符串指针（i8*），需要调用者释放内存
 * 
 * 编译器签名：i8* runtime_string_trim(i8* s)
 * 使用位置：stdlib/string/string.eo - string.trim()
 */
char* runtime_string_trim(const char* s) {
    if (!s) {
        return NULL;
    }
    
    // 跳过前导空白字符
    const char* start = s;
    while (*start && isspace((unsigned char)*start)) {
        start++;
    }
    
    // 如果字符串全是空白字符
    if (*start == '\0') {
        char* result = (char*)malloc(1);
        if (result) {
            result[0] = '\0';
        }
        return result;
    }
    
    // 找到末尾的非空白字符
    const char* end = start;
    const char* last_non_space = start;
    while (*end) {
        if (!isspace((unsigned char)*end)) {
            last_non_space = end;
        }
        end++;
    }
    
    // 计算新字符串长度
    size_t len = last_non_space - start + 1;
    char* result = (char*)malloc(len + 1);
    if (!result) {
        return NULL;
    }
    
    // 复制字符串
    strncpy(result, start, len);
    result[len] = '\0';
    
    return result;
}

/**
 * @brief 将字符串转换为大写
 * @param s 字符串指针（i8*）
 * @return 新字符串指针（i8*），需要调用者释放内存
 * 
 * 编译器签名：i8* runtime_string_to_upper(i8* s)
 * 使用位置：stdlib/string/string.eo - string.to_upper()
 */
char* runtime_string_to_upper(const char* s) {
    if (!s) {
        return NULL;
    }
    
    size_t len = strlen(s);
    char* result = (char*)malloc(len + 1);
    if (!result) {
        return NULL;
    }
    
    // 转换为大写（处理ASCII字符）
    for (size_t i = 0; i < len; i++) {
        if (s[i] >= 'a' && s[i] <= 'z') {
            result[i] = s[i] - ('a' - 'A');
        } else {
            result[i] = s[i];
        }
    }
    result[len] = '\0';
    
    return result;
}

/**
 * @brief 将字符串转换为小写
 * @param s 字符串指针（i8*）
 * @return 新字符串指针（i8*），需要调用者释放内存
 * 
 * 编译器签名：i8* runtime_string_to_lower(i8* s)
 * 使用位置：stdlib/string/string.eo - string.to_lower()
 */
char* runtime_string_to_lower(const char* s) {
    if (!s) {
        return NULL;
    }
    
    size_t len = strlen(s);
    char* result = (char*)malloc(len + 1);
    if (!result) {
        return NULL;
    }
    
    // 转换为小写（处理ASCII字符）
    for (size_t i = 0; i < len; i++) {
        if (s[i] >= 'A' && s[i] <= 'Z') {
            result[i] = s[i] + ('a' - 'A');
        } else {
            result[i] = s[i];
        }
    }
    result[len] = '\0';
    
    return result;
}

/**
 * @brief 将字符串转换为UTF-8字节数组
 * @param s 字符串指针（i8*）
 * @return 字符串到字节数组转换结果结构体指针，需要调用者释放内存
 * 
 * 编译器签名：i8* runtime_string_to_bytes(i8* s)
 * 使用位置：stdlib/encoding/base64.eo - encode_string()
 * 
 * 注意：返回的结构体包含字节数组指针和长度，需要由调用者或GC管理
 */
StringToBytesResult* runtime_string_to_bytes(const char* s) {
    if (!s) {
        StringToBytesResult* result = (StringToBytesResult*)malloc(sizeof(StringToBytesResult));
        if (result) {
            result->bytes = NULL;
            result->len = 0;
        }
        return result;
    }
    
    // 字符串已经是UTF-8编码，直接复制字节
    size_t len = strlen(s);
    uint8_t* bytes = (uint8_t*)malloc(len);
    if (!bytes) {
        StringToBytesResult* result = (StringToBytesResult*)malloc(sizeof(StringToBytesResult));
        if (result) {
            result->bytes = NULL;
            result->len = 0;
        }
        return result;
    }
    
    // 复制字节
    memcpy(bytes, s, len);
    
    // 创建结果结构体
    StringToBytesResult* result = (StringToBytesResult*)malloc(sizeof(StringToBytesResult));
    if (!result) {
        free(bytes);
        return NULL;
    }
    
    result->bytes = bytes;
    result->len = (int32_t)len;
    
    return result;
}

/**
 * @brief 将UTF-8字节数组转换为字符串
 * @param bytes 字节数组指针（i8*）
 * @param len 字节数组长度
 * @return 字符串指针（i8*），需要调用者释放内存
 * 
 * 编译器签名：i8* runtime_bytes_to_string(i8* bytes, i32 len)
 * 使用位置：stdlib/encoding/base64.eo - decode_string()
 * 
 * 注意：返回的字符串是新分配的内存，需要由调用者或GC管理
 */
char* runtime_bytes_to_string(const uint8_t* bytes, int32_t len) {
    if (!bytes || len < 0) {
        char* result = (char*)malloc(1);
        if (result) {
            result[0] = '\0';
        }
        return result;
    }
    
    // 分配内存（包括null终止符）
    char* result = (char*)malloc((size_t)len + 1);
    if (!result) {
        return NULL;
    }
    
    // 复制字节
    memcpy(result, bytes, (size_t)len);
    result[len] = '\0';
    
    return result;
}

/**
 * @brief 将字符串直接编码为Base64字符串
 * @param s 输入字符串指针（i8*）
 * @return Base64编码后的字符串指针（i8*），需要调用者释放内存
 * 
 * 编译器签名：i8* runtime_base64_encode_string(i8* s)
 * 使用位置：stdlib/encoding/base64.eo - encode_string()
 * 
 * 注意：这是一个临时实现，用于绕过切片类型转换的限制
 * 返回的字符串是新分配的内存，需要由调用者或GC管理
 * 实现：使用标准Base64编码算法
 */
char* runtime_base64_encode_string(const char* s) {
    if (!s) {
        char* result = (char*)malloc(1);
        if (result) {
            result[0] = '\0';
        }
        return result;
    }
    
    // Base64字符表
    const char base64_chars[] = "ABCDEFGHIJKLMNOPQRSTUVWXYZabcdefghijklmnopqrstuvwxyz0123456789+/";
    
    size_t input_len = strlen(s);
    // Base64编码后的长度：每3个字节编码为4个字符，加上填充
    size_t output_len = ((input_len + 2) / 3) * 4;
    
    char* result = (char*)malloc(output_len + 1); // +1 for null terminator
    if (!result) {
        return NULL;
    }
    
    size_t i = 0;
    size_t j = 0;
    
    // 处理每3个字节一组
    while (i < input_len) {
        uint8_t b1 = (uint8_t)s[i++];
        uint8_t b2 = (i < input_len) ? (uint8_t)s[i++] : 0;
        uint8_t b3 = (i < input_len) ? (uint8_t)s[i++] : 0;
        
        // 转换为4个Base64字符
        result[j++] = base64_chars[(b1 >> 2) & 0x3F];
        result[j++] = base64_chars[(((b1 & 0x03) << 4) | (b2 >> 4)) & 0x3F];
        
        if (i - 2 < input_len) {
            result[j++] = base64_chars[(((b2 & 0x0F) << 2) | (b3 >> 6)) & 0x3F];
        } else {
            result[j++] = '=';
        }
        
        if (i - 1 < input_len) {
            result[j++] = base64_chars[b3 & 0x3F];
        } else {
            result[j++] = '=';
        }
    }
    
    result[j] = '\0';
    return result;
}

/**
 * @brief 将浮点数转换为字符串
 * @param f 浮点数值（f64）
 * @return 字符串指针（i8*），需要调用者释放内存
 * 
 * 编译器签名：i8* runtime_f64_to_string(f64 f)
 * 使用位置：stdlib/encoding/json.eo - encode_number()
 * 
 * 注意：返回的字符串是新分配的内存，需要由调用者或GC管理
 * 实现：使用 snprintf 格式化浮点数，支持整数、浮点数、科学计数法
 */
char* runtime_f64_to_string(double f) {
    // 特殊值处理
    if (isnan(f)) {
        char* result = (char*)malloc(4);
        if (result) {
            strcpy(result, "NaN");
        }
        return result;
    }
    if (isinf(f)) {
        char* result = (char*)malloc(9);
        if (result) {
            if (f > 0) {
                strcpy(result, "Infinity");
            } else {
                strcpy(result, "-Infinity");
            }
        }
        return result;
    }
    
    // 使用 snprintf 格式化浮点数
    // 先尝试整数格式（如果确实是整数）
    if (f == (double)(int64_t)f && f >= INT64_MIN && f <= INT64_MAX) {
        // 是整数，使用整数格式（避免小数点）
        char* result = (char*)malloc(32);
        if (result) {
            snprintf(result, 32, "%lld", (long long)(int64_t)f);
        }
        return result;
    }
    
    // 使用浮点数格式，自动选择最紧凑的表示
    // 尝试普通格式（最多15位小数）
    char buffer[64];
    int len = snprintf(buffer, sizeof(buffer), "%.15g", f);
    if (len < 0 || len >= (int)sizeof(buffer)) {
        // 如果失败，使用默认格式
        len = snprintf(buffer, sizeof(buffer), "%f", f);
    }
    
    char* result = (char*)malloc((size_t)len + 1);
    if (!result) {
        return NULL;
    }
    
    strcpy(result, buffer);
    return result;
}

/**
 * @brief 将字符串解析为浮点数
 * @param s 字符串指针（i8*）
 * @return 浮点数值（f64）
 * 
 * 编译器签名：f64 runtime_string_to_f64(i8* s)
 * 使用位置：stdlib/encoding/json.eo - decode_number()
 * 
 * 实现：使用 strtod 解析字符串，支持整数、浮点数、科学计数法
 * 注意：如果解析失败，返回 NaN
 */
double runtime_string_to_f64(const char* s) {
    if (!s) {
        return NAN;
    }
    
    // 跳过前导空白字符
    while (*s && isspace((unsigned char)*s)) {
        s++;
    }
    
    // 空字符串
    if (*s == '\0') {
        return NAN;
    }
    
    // 使用 strtod 解析
    char* endptr;
    double result = strtod(s, &endptr);
    
    // 检查是否完全解析（跳过尾随空白字符）
    while (*endptr && isspace((unsigned char)*endptr)) {
        endptr++;
    }
    
    // 如果还有未解析的字符，返回 NaN
    if (*endptr != '\0') {
        return NAN;
    }
    
    // 检查是否发生错误（errno 被设置）
    if (errno == ERANGE) {
        // 溢出或下溢，返回相应的值
        if (result == HUGE_VAL || result == -HUGE_VAL) {
            return result;  // 返回 Infinity 或 -Infinity
        }
        return NAN;
    }
    
    return result;
}

/**
 * @brief 分割字符串
 * @param s 字符串指针（i8*）
 * @param delimiter 分隔符字符串指针（i8*）
 * @return 字符串分割结果结构体指针（需要调用者释放内存）
 * 
 * 编译器签名：i8* runtime_string_split(i8* s, i8* delimiter)
 * 使用位置：stdlib/string/string.eo - string.split()
 */
// ==================== 类型转换支持 ====================

/**
 * @brief 将 C 字符串指针转换为 Echo string_t 结构体
 * @param ptr C 字符串指针（char*，即 i8*）
 * @return string_t 结构体（包含 data, length, capacity）
 * 
 * 编译器签名：i8* runtime_char_ptr_to_string(i8* ptr)
 * 使用位置：类型转换 char* → string（如 string(ptr) 或 ptr as string）
 * 
 * 实现：
 * - 检查指针有效性（非 NULL）
 * - 计算字符串长度（strlen）
 * - 创建 string_t 结构体并返回
 * 
 * 注意：
 * - 如果 ptr 为 NULL，返回空字符串（length=0, data=NULL）
 * - 返回的 string_t 结构体包含原始指针，不复制数据（零成本）
 * - 调用者需要确保 ptr 指向的字符串在 string_t 使用期间有效
 */
string_t runtime_char_ptr_to_string(char* ptr) {
    string_t result;
    
    if (ptr == NULL) {
        // 返回空字符串
        result.data = NULL;
        result.length = 0;
        result.capacity = 0;
        return result;
    }
    
    // 计算字符串长度
    size_t len = strlen(ptr);
    
    // 创建 string_t 结构体（不复制数据，只保存指针）
    result.data = ptr;
    result.length = len;
    result.capacity = len + 1;  // 包括 null terminator
    
    return result;
}

/**
 * @brief 将 C 字符串指针数组转换为 Echo []string 切片
 * @param ptrs 字符串指针数组（char**，即 i8**）
 * @param count 字符串数量（int32_t）
 * @return 切片数据指针（*i8），失败返回 NULL
 * 
 * 编译器签名：i8* runtime_char_ptr_array_to_string_slice(i8** ptrs, i32 count)
 * 使用位置：类型转换 char** + int32_t → []string（如从 StringSplitResult 提取字段）
 * 
 * 实现：
 * - 检查指针和数量有效性
 * - 在 LLVM IR 中，[]string 切片是 *i8（指向字符串指针数组的指针）
 * - 直接返回 ptrs（因为 char** 和 []string 在内存布局上相同）
 * 
 * 注意：
 * - 如果 ptrs 为 NULL 或 count < 0，返回 NULL
 * - 返回的指针指向原始数据，不复制（零成本）
 * - 调用者需要确保 ptrs 指向的数据在切片使用期间有效
 */
void* runtime_char_ptr_array_to_string_slice(char** ptrs, int32_t count) {
    if (ptrs == NULL || count < 0) {
        return NULL;
    }
    
    // 在 LLVM IR 中，[]string 切片是 *i8（指向字符串指针数组的指针）
    // char** 和 []string 在内存布局上相同，直接返回
    return (void*)ptrs;
}

StringSplitResult* runtime_string_split(const char* s, const char* delimiter) {
    if (!s || !delimiter) {
        return NULL;
    }
    
    size_t s_len = strlen(s);
    size_t delim_len = strlen(delimiter);
    
    // 如果分隔符为空，返回只包含原字符串的数组
    if (delim_len == 0) {
        StringSplitResult* result = (StringSplitResult*)malloc(sizeof(StringSplitResult));
        if (!result) {
            return NULL;
        }
        
        result->strings = (char**)malloc(sizeof(char*));
        if (!result->strings) {
            free(result);
            return NULL;
        }
        
        result->strings[0] = (char*)malloc(s_len + 1);
        if (!result->strings[0]) {
            free(result->strings);
            free(result);
            return NULL;
        }
        
        strcpy(result->strings[0], s);
        result->count = 1;
        return result;
    }
    
    // 计算分割后的字符串数量（先遍历一次）
    int count = 1; // 至少有一个字符串
    const char* pos = s;
    while ((pos = strstr(pos, delimiter)) != NULL) {
        count++;
        pos += delim_len;
    }
    
    // 分配结果结构体
    StringSplitResult* result = (StringSplitResult*)malloc(sizeof(StringSplitResult));
    if (!result) {
        return NULL;
    }
    
    // 分配字符串指针数组
    result->strings = (char**)malloc(sizeof(char*) * count);
    if (!result->strings) {
        free(result);
        return NULL;
    }
    
    result->count = count;
    
    // 分割字符串
    const char* start = s;
    int index = 0;
    pos = s;
    
    while ((pos = strstr(start, delimiter)) != NULL) {
        // 计算当前子字符串的长度
        size_t substr_len = pos - start;
        
        // 分配子字符串内存
        result->strings[index] = (char*)malloc(substr_len + 1);
        if (!result->strings[index]) {
            // 释放已分配的内存
            for (int i = 0; i < index; i++) {
                free(result->strings[i]);
            }
            free(result->strings);
            free(result);
            return NULL;
        }
        
        // 复制子字符串
        strncpy(result->strings[index], start, substr_len);
        result->strings[index][substr_len] = '\0';
        
        index++;
        start = pos + delim_len;
    }
    
    // 处理最后一个子字符串（从最后一个分隔符到字符串末尾）
    size_t last_len = strlen(start);
    result->strings[index] = (char*)malloc(last_len + 1);
    if (!result->strings[index]) {
        // 释放已分配的内存
        for (int i = 0; i < index; i++) {
            free(result->strings[i]);
        }
        free(result->strings);
        free(result);
        return NULL;
    }
    
    strcpy(result->strings[index], start);
    
    return result;
}

/**
 * @brief 重新分配切片内存
 * @param ptr 原始切片数据指针（i8*）
 * @param old_cap 当前容量（i32）
 * @param new_cap 新容量（i32）
 * @param elem_size 元素大小（字节）（i32）
 * @return 新的切片数据指针（i8*），失败返回 NULL
 * 
 * 编译器签名：i8* runtime_realloc_slice(i8* ptr, i32 old_cap, i32 new_cap, i32 elem_size)
 * 使用位置：stdlib/collections/vec.eo - grow(), reserve(), shrink_to_fit()
 * 
 * 注意：这是泛型函数，通过类型擦除方式实现
 */
void* runtime_realloc_slice(void* ptr, int32_t old_cap, int32_t new_cap, int32_t elem_size) {
    if (new_cap <= 0 || elem_size <= 0) {
        return NULL; // 无效参数
    }

    if (old_cap < 0) {
        old_cap = 0; // 修正负数容量
    }

    // 计算新的大小
    size_t new_size = (size_t)new_cap * (size_t)elem_size;
    
    if (new_size == 0) {
        // 新容量为 0，释放内存
        if (ptr) {
            free(ptr);
        }
        return NULL;
    }

    // 重新分配内存
    void* new_ptr = realloc(ptr, new_size);
    if (!new_ptr) {
        return NULL; // 内存分配失败
    }

    // 如果新容量大于旧容量，初始化新增的内存区域为 0
    if (new_cap > old_cap && old_cap > 0) {
        size_t old_size = (size_t)old_cap * (size_t)elem_size;
        size_t additional_size = new_size - old_size;
        memset((char*)new_ptr + old_size, 0, additional_size);
    }

    return new_ptr;
}

/**
 * @brief 分配新切片内存
 * @param cap 初始容量（i32）
 * @param elem_size 元素大小（字节）（i32）
 * @return 新分配的切片数据指针（i8*），失败返回 NULL
 * 
 * 编译器签名：i8* runtime_alloc_slice(i32 cap, i32 elem_size)
 * 使用位置：stdlib/collections/vec.eo - 初始化
 * 
 * 注意：这是泛型函数，通过类型擦除方式实现
 */
void* runtime_alloc_slice(int32_t cap, int32_t elem_size) {
    if (cap <= 0 || elem_size <= 0) {
        return NULL; // 无效参数
    }

    size_t size = (size_t)cap * (size_t)elem_size;
    void* ptr = calloc(1, size); // 使用 calloc 初始化为 0
    
    return ptr;
}

/**
 * @brief 释放切片内存
 * @param ptr 切片数据指针（i8*）
 * 
 * 编译器签名：void runtime_free_slice(i8* ptr)
 * 使用位置：stdlib/collections/vec.eo - 清理
 * 
 * 注意：这是泛型函数，通过类型擦除方式实现
 */
void runtime_free_slice(void* ptr) {
    if (ptr) {
        free(ptr);
    }
}

/**
 * @brief 从指针和长度构造切片（辅助函数）
 * @param ptr 数据指针（i8*）
 * @param len 长度（i32）
 * @return 数据指针（i8*），直接返回输入指针
 * 
 * 编译器签名：i8* runtime_slice_from_ptr_len(i8* ptr, i32 len)
 * 使用位置：stdlib/net/http/server.eo - parse_request(), build_response()
 *          stdlib/string/string.eo - split()
 * 
 * 注意：
 * - 这个函数主要用于类型转换，不分配新内存
 * - 返回的指针与输入指针相同
 * - 长度信息需要在符号表中存储，或通过其他方式传递
 * - 实际实现中，切片在 LLVM IR 中只是一个指针，长度信息需要单独存储
 */
void* runtime_slice_from_ptr_len(void* ptr, int32_t len) {
    // 这个函数主要用于类型转换，不分配新内存
    // 返回的指针与输入指针相同
    // 长度信息 len 参数用于类型检查和验证，但不影响返回值
    (void)len; // 标记为已使用，避免编译警告
    return ptr;
}

/**
 * @brief 获取切片长度
 * @param slice_ptr 切片数据指针（i8*）
 * @return 切片长度（i32）
 * 
 * 编译器签名：i32 runtime_slice_len(i8* slice_ptr)
 * 
 * 注意：
 * - 在当前的简化实现中，切片只是一个指针，长度信息需要单独存储
 * - 这个函数需要从运行时切片结构中提取长度信息
 * - 目前返回0，表示长度信息需要从符号表或其他方式获取
 */
int32_t runtime_slice_len(void* slice_ptr) {
    // 简化实现：在当前架构中，切片长度信息存储在符号表中
    // 运行时无法直接从指针获取长度，需要编译器在调用时传递长度信息
    // 或者使用更复杂的切片结构（包含长度字段）
    (void)slice_ptr; // 标记为已使用，避免编译警告
    return 0; // 返回0表示长度未知，实际应该从符号表或切片结构中获取
}

// ============================================================================
// P1 优先级运行时函数实现（重要功能）
// ============================================================================
// 这些函数是标准库的重要依赖，包括文件 I/O、同步原语、异步运行时

// ==================== 文件 I/O ====================

/**
 * @brief 打开文件
 * @param path 文件路径（i8*）
 * @return Result[FileHandle, string] -> i8*
 * 
 * 编译器签名：i8* runtime_open_file(i8* path)
 * 使用位置：stdlib/io/file.eo - File.open()
 */
void* runtime_open_file(const char* path) {
    if (!path) {
        return result_err("Path is null");
    }

    // 初始化全局文件句柄表
    FileHandleTable* table = get_global_file_handle_table();
    if (!table) {
        return result_err("Failed to initialize file handle table");
    }

    // 打开文件（只读模式）
    FILE* file = fopen(path, "rb");
    if (!file) {
        char error_msg[256];
        snprintf(error_msg, sizeof(error_msg), "Failed to open file: %s (errno: %d)", path, errno);
        return result_err(error_msg);
    }

    // 分配文件句柄
    int32_t handle = file_handle_allocate(table, file);
    if (handle < 0) {
        fclose(file);
        return result_err("Failed to allocate file handle (table full)");
    }

    // 返回 Ok(FileHandle)
    return result_ok(&handle, sizeof(int32_t));
}

/**
 * @brief 创建文件
 * @param path 文件路径（i8*）
 * @return Result[FileHandle, string] -> i8*
 * 
 * 编译器签名：i8* runtime_create_file(i8* path)
 * 使用位置：stdlib/io/file.eo - File.create()
 */
void* runtime_create_file(const char* path) {
    if (!path) {
        return result_err("Path is null");
    }

    FileHandleTable* table = get_global_file_handle_table();
    if (!table) {
        return result_err("Failed to initialize file handle table");
    }

    // 创建文件（写入模式，如果文件存在则截断）
    FILE* file = fopen(path, "wb");
    if (!file) {
        char error_msg[256];
        snprintf(error_msg, sizeof(error_msg), "Failed to create file: %s (errno: %d)", path, errno);
        return result_err(error_msg);
    }

    int32_t handle = file_handle_allocate(table, file);
    if (handle < 0) {
        fclose(file);
        return result_err("Failed to allocate file handle (table full)");
    }

    return result_ok(&handle, sizeof(int32_t));
}

/**
 * @brief 关闭文件
 * @param handle 文件句柄（i32）
 * @return Result[void, string] -> i8*
 * 
 * 编译器签名：i8* runtime_close_file(i32 handle)
 * 使用位置：stdlib/io/file.eo - File.close()
 */
void* runtime_close_file(int32_t handle) {
    FileHandleTable* table = get_global_file_handle_table();
    if (!table) {
        return result_err("File handle table not initialized");
    }

    if (!file_handle_is_valid(table, handle)) {
        return result_err("Invalid file handle");
    }

    if (file_handle_release(table, handle)) {
        // 返回 Ok(void) - 使用空值
        return result_ok(NULL, 0);
    } else {
        return result_err("Failed to close file");
    }
}

/**
 * @brief 读取文件
 * @param handle 文件句柄（i32）
 * @param buf_ptr 缓冲区指针（i8*）
 * @param buf_len 缓冲区长度（i32）
 * @return Result[int, string] -> i8*
 * 
 * 编译器签名：i8* runtime_read_file(i32 handle, i8* buf_ptr, i32 buf_len)
 * 使用位置：stdlib/io/file.eo - File.read()
 */
void* runtime_read_file(int32_t handle, void* buf_ptr, int32_t buf_len) {
    if (!buf_ptr || buf_len < 0) {
        return result_err("Invalid buffer parameters");
    }

    FileHandleTable* table = get_global_file_handle_table();
    if (!table) {
        return result_err("File handle table not initialized");
    }

    FILE* file = file_handle_get(table, handle);
    if (!file) {
        return result_err("Invalid file handle");
    }

    // 读取数据
    size_t bytes_read = fread(buf_ptr, 1, (size_t)buf_len, file);
    if (ferror(file)) {
        return result_err("File read error");
    }

    // 返回 Ok(读取的字节数)
    int32_t bytes_read_i32 = (int32_t)bytes_read;
    return result_ok(&bytes_read_i32, sizeof(int32_t));
}

/**
 * @brief 写入文件
 * @param handle 文件句柄（i32）
 * @param buf_ptr 缓冲区指针（i8*）
 * @param buf_len 缓冲区长度（i32）
 * @return Result[int, string] -> i8*
 * 
 * 编译器签名：i8* runtime_write_file(i32 handle, i8* buf_ptr, i32 buf_len)
 * 使用位置：stdlib/io/file.eo - File.write()
 */
void* runtime_write_file(int32_t handle, const void* buf_ptr, int32_t buf_len) {
    if (!buf_ptr || buf_len < 0) {
        return result_err("Invalid buffer parameters");
    }

    FileHandleTable* table = get_global_file_handle_table();
    if (!table) {
        return result_err("File handle table not initialized");
    }

    FILE* file = file_handle_get(table, handle);
    if (!file) {
        return result_err("Invalid file handle");
    }

    // 写入数据
    size_t bytes_written = fwrite(buf_ptr, 1, (size_t)buf_len, file);
    if (ferror(file)) {
        return result_err("File write error");
    }

    // 返回 Ok(写入的字节数)
    int32_t bytes_written_i32 = (int32_t)bytes_written;
    return result_ok(&bytes_written_i32, sizeof(int32_t));
}

/**
 * @brief 刷新文件缓冲区
 * @param handle 文件句柄（i32）
 * @return Result[void, string] -> i8*
 * 
 * 编译器签名：i8* runtime_flush_file(i32 handle)
 * 使用位置：stdlib/io/file.eo - File.flush()
 */
void* runtime_flush_file(int32_t handle) {
    FileHandleTable* table = get_global_file_handle_table();
    if (!table) {
        return result_err("File handle table not initialized");
    }

    FILE* file = file_handle_get(table, handle);
    if (!file) {
        return result_err("Invalid file handle");
    }

    if (fflush(file) != 0) {
        return result_err("File flush error");
    }

    // 返回 Ok(void)
    return result_ok(NULL, 0);
}

// ==================== 同步原语 ====================

// 互斥锁句柄表（简化实现：使用数组存储 pthread_mutex_t）
#define MAX_MUTEX_HANDLES 1024
static pthread_mutex_t g_mutexes[MAX_MUTEX_HANDLES];
static bool g_mutex_used[MAX_MUTEX_HANDLES];
static int32_t g_next_mutex_handle = 1;

// 读写锁句柄表
static pthread_rwlock_t g_rwlocks[MAX_MUTEX_HANDLES];
static bool g_rwlock_used[MAX_MUTEX_HANDLES];
static int32_t g_next_rwlock_handle = 1;

/**
 * @brief 创建互斥锁（标准库版本，不需要 RuntimeHandle）
 * @return MutexHandle -> i32
 * 
 * 编译器签名：i32 runtime_create_mutex()
 * 使用位置：stdlib/sync/mutex.eo - Mutex.new()
 * 
 * 注意：这个函数与旧的 runtime_create_mutex(RuntimeHandle*) 不同
 * 旧版本用于运行时系统内部，新版本用于标准库调用
 * 由于编译器声明的函数名是 runtime_create_mutex()，我们需要重命名旧的函数
 */
int32_t runtime_create_mutex(void) {
    // 查找空闲槽位
    for (int i = 0; i < MAX_MUTEX_HANDLES; i++) {
        if (!g_mutex_used[i]) {
            if (pthread_mutex_init(&g_mutexes[i], NULL) != 0) {
                return -1; // 初始化失败
            }
            g_mutex_used[i] = true;
            int32_t handle = i + 1; // 句柄从 1 开始
            return handle;
        }
    }
    return -1; // 表已满
}

/**
 * @brief 锁定互斥锁（阻塞）
 * @param handle 互斥锁句柄（i32）
 * 
 * 编译器签名：void runtime_mutex_lock(i32 handle)
 * 使用位置：stdlib/sync/mutex.eo - Mutex.lock()
 */
void runtime_mutex_lock(int32_t handle) {
    if (handle < 1 || handle > MAX_MUTEX_HANDLES) {
        return; // 无效句柄
    }
    int index = handle - 1;
    if (g_mutex_used[index]) {
        pthread_mutex_lock(&g_mutexes[index]);
    }
}

/**
 * @brief 尝试锁定互斥锁（非阻塞）
 * @param handle 互斥锁句柄（i32）
 * @return bool -> i1（true 表示成功，false 表示失败）
 * 
 * 编译器签名：i1 runtime_mutex_try_lock(i32 handle)
 * 使用位置：stdlib/sync/mutex.eo - Mutex.try_lock()
 */
bool runtime_mutex_try_lock(int32_t handle) {
    if (handle < 1 || handle > MAX_MUTEX_HANDLES) {
        return false; // 无效句柄
    }
    int index = handle - 1;
    if (g_mutex_used[index]) {
        return (pthread_mutex_trylock(&g_mutexes[index]) == 0);
    }
    return false;
}

/**
 * @brief 解锁互斥锁
 * @param handle 互斥锁句柄（i32）
 * 
 * 编译器签名：void runtime_mutex_unlock(i32 handle)
 * 使用位置：stdlib/sync/mutex.eo - Mutex.unlock()
 */
void runtime_mutex_unlock(int32_t handle) {
    if (handle < 1 || handle > MAX_MUTEX_HANDLES) {
        return; // 无效句柄
    }
    int index = handle - 1;
    if (g_mutex_used[index]) {
        pthread_mutex_unlock(&g_mutexes[index]);
    }
}

/**
 * @brief 创建读写锁
 * @return RwLockHandle -> i32
 * 
 * 编译器签名：i32 runtime_create_rwlock()
 * 使用位置：stdlib/sync/rwlock.eo - RwLock.new()
 */
int32_t runtime_create_rwlock(void) {
    // 查找空闲槽位
    for (int i = 0; i < MAX_MUTEX_HANDLES; i++) {
        if (!g_rwlock_used[i]) {
            if (pthread_rwlock_init(&g_rwlocks[i], NULL) != 0) {
                return -1; // 初始化失败
            }
            g_rwlock_used[i] = true;
            int32_t handle = i + 1; // 句柄从 1 开始
            return handle;
        }
    }
    return -1; // 表已满
}

/**
 * @brief 获取读锁（阻塞）
 * @param handle 读写锁句柄（i32）
 * 
 * 编译器签名：void runtime_rwlock_read_lock(i32 handle)
 * 使用位置：stdlib/sync/rwlock.eo - RwLock.read_lock()
 */
void runtime_rwlock_read_lock(int32_t handle) {
    if (handle < 1 || handle > MAX_MUTEX_HANDLES) {
        return; // 无效句柄
    }
    int index = handle - 1;
    if (g_rwlock_used[index]) {
        pthread_rwlock_rdlock(&g_rwlocks[index]);
    }
}

/**
 * @brief 尝试获取读锁（非阻塞）
 * @param handle 读写锁句柄（i32）
 * @return bool -> i1（true 表示成功，false 表示失败）
 * 
 * 编译器签名：i1 runtime_rwlock_try_read_lock(i32 handle)
 * 使用位置：stdlib/sync/rwlock.eo - RwLock.try_read_lock()
 */
bool runtime_rwlock_try_read_lock(int32_t handle) {
    if (handle < 1 || handle > MAX_MUTEX_HANDLES) {
        return false; // 无效句柄
    }
    int index = handle - 1;
    if (g_rwlock_used[index]) {
        return (pthread_rwlock_tryrdlock(&g_rwlocks[index]) == 0);
    }
    return false;
}

/**
 * @brief 获取写锁（阻塞）
 * @param handle 读写锁句柄（i32）
 * 
 * 编译器签名：void runtime_rwlock_write_lock(i32 handle)
 * 使用位置：stdlib/sync/rwlock.eo - RwLock.write_lock()
 */
void runtime_rwlock_write_lock(int32_t handle) {
    if (handle < 1 || handle > MAX_MUTEX_HANDLES) {
        return; // 无效句柄
    }
    int index = handle - 1;
    if (g_rwlock_used[index]) {
        pthread_rwlock_wrlock(&g_rwlocks[index]);
    }
}

/**
 * @brief 尝试获取写锁（非阻塞）
 * @param handle 读写锁句柄（i32）
 * @return bool -> i1（true 表示成功，false 表示失败）
 * 
 * 编译器签名：i1 runtime_rwlock_try_write_lock(i32 handle)
 * 使用位置：stdlib/sync/rwlock.eo - RwLock.try_write_lock()
 */
bool runtime_rwlock_try_write_lock(int32_t handle) {
    if (handle < 1 || handle > MAX_MUTEX_HANDLES) {
        return false; // 无效句柄
    }
    int index = handle - 1;
    if (g_rwlock_used[index]) {
        return (pthread_rwlock_trywrlock(&g_rwlocks[index]) == 0);
    }
    return false;
}

/**
 * @brief 释放读锁
 * @param handle 读写锁句柄（i32）
 * 
 * 编译器签名：void runtime_rwlock_read_unlock(i32 handle)
 * 使用位置：stdlib/sync/rwlock.eo - RwLock.read_unlock()
 */
void runtime_rwlock_read_unlock(int32_t handle) {
    if (handle < 1 || handle > MAX_MUTEX_HANDLES) {
        return; // 无效句柄
    }
    int index = handle - 1;
    if (g_rwlock_used[index]) {
        pthread_rwlock_unlock(&g_rwlocks[index]);
    }
}

/**
 * @brief 释放写锁
 * @param handle 读写锁句柄（i32）
 * 
 * 编译器签名：void runtime_rwlock_write_unlock(i32 handle)
 * 使用位置：stdlib/sync/rwlock.eo - RwLock.write_unlock()
 */
void runtime_rwlock_write_unlock(int32_t handle) {
    if (handle < 1 || handle > MAX_MUTEX_HANDLES) {
        return; // 无效句柄
    }
    int index = handle - 1;
    if (g_rwlock_used[index]) {
        pthread_rwlock_unlock(&g_rwlocks[index]);
    }
}

// ==================== 原子操作 ====================

/**
 * @brief 原子加载 i32
 * @param ptr 指针（*i32）
 * @return i32
 * 
 * 编译器签名：i32 runtime_atomic_load_i32(*i32 ptr)
 * 使用位置：stdlib/sync/atomic.eo - Atomic.load_i32()
 */
int32_t runtime_atomic_load_i32(int32_t* ptr) {
    if (!ptr) {
        return 0;
    }
    return atomic_load((atomic_int_least32_t*)ptr);
}

/**
 * @brief 原子存储 i32
 * @param ptr 指针（*i32）
 * @param value 值（i32）
 * 
 * 编译器签名：void runtime_atomic_store_i32(*i32 ptr, i32 value)
 * 使用位置：stdlib/sync/atomic.eo - Atomic.store_i32()
 */
void runtime_atomic_store_i32(int32_t* ptr, int32_t value) {
    if (ptr) {
        atomic_store((atomic_int_least32_t*)ptr, value);
    }
}

/**
 * @brief 原子交换 i32
 * @param ptr 指针（*i32）
 * @param value 新值（i32）
 * @return i32（旧值）
 * 
 * 编译器签名：i32 runtime_atomic_swap_i32(*i32 ptr, i32 value)
 * 使用位置：stdlib/sync/atomic.eo - Atomic.swap_i32()
 */
int32_t runtime_atomic_swap_i32(int32_t* ptr, int32_t value) {
    if (!ptr) {
        return 0;
    }
    return atomic_exchange((atomic_int_least32_t*)ptr, value);
}

/**
 * @brief 原子比较并交换 i32
 * @param ptr 指针（*i32）
 * @param expected 期望值（i32）
 * @param desired 新值（i32）
 * @return bool -> i1（true 表示成功，false 表示失败）
 * 
 * 编译器签名：i1 runtime_atomic_cas_i32(*i32 ptr, i32 expected, i32 desired)
 * 使用位置：stdlib/sync/atomic.eo - Atomic.cas_i32()
 */
bool runtime_atomic_cas_i32(int32_t* ptr, int32_t expected, int32_t desired) {
    if (!ptr) {
        return false;
    }
    int32_t expected_val = expected;
    return atomic_compare_exchange_strong((atomic_int_least32_t*)ptr, &expected_val, desired);
}

/**
 * @brief 原子加法 i32
 * @param ptr 指针（*i32）
 * @param delta 增量（i32）
 * @return i32（旧值）
 * 
 * 编译器签名：i32 runtime_atomic_add_i32(*i32 ptr, i32 delta)
 * 使用位置：stdlib/sync/atomic.eo - Atomic.add_i32()
 */
int32_t runtime_atomic_add_i32(int32_t* ptr, int32_t delta) {
    if (!ptr) {
        return 0;
    }
    return atomic_fetch_add((atomic_int_least32_t*)ptr, delta);
}

/**
 * @brief 原子加载 i64
 * @param ptr 指针（*i64）
 * @return i64
 * 
 * 编译器签名：i64 runtime_atomic_load_i64(*i64 ptr)
 * 使用位置：stdlib/sync/atomic.eo - Atomic.load_i64()
 */
int64_t runtime_atomic_load_i64(int64_t* ptr) {
    if (!ptr) {
        return 0;
    }
    return atomic_load((atomic_int_least64_t*)ptr);
}

/**
 * @brief 原子存储 i64
 * @param ptr 指针（*i64）
 * @param value 值（i64）
 * 
 * 编译器签名：void runtime_atomic_store_i64(*i64 ptr, i64 value)
 * 使用位置：stdlib/sync/atomic.eo - Atomic.store_i64()
 */
void runtime_atomic_store_i64(int64_t* ptr, int64_t value) {
    if (ptr) {
        atomic_store((atomic_int_least64_t*)ptr, value);
    }
}

/**
 * @brief 原子交换 i64
 * @param ptr 指针（*i64）
 * @param value 新值（i64）
 * @return i64（旧值）
 * 
 * 编译器签名：i64 runtime_atomic_swap_i64(*i64 ptr, i64 value)
 * 使用位置：stdlib/sync/atomic.eo - Atomic.swap_i64()
 */
int64_t runtime_atomic_swap_i64(int64_t* ptr, int64_t value) {
    if (!ptr) {
        return 0;
    }
    return atomic_exchange((atomic_int_least64_t*)ptr, value);
}

/**
 * @brief 原子比较并交换 i64
 * @param ptr 指针（*i64）
 * @param expected 期望值（i64）
 * @param desired 新值（i64）
 * @return bool -> i1（true 表示成功，false 表示失败）
 * 
 * 编译器签名：i1 runtime_atomic_cas_i64(*i64 ptr, i64 expected, i64 desired)
 * 使用位置：stdlib/sync/atomic.eo - Atomic.cas_i64()
 */
bool runtime_atomic_cas_i64(int64_t* ptr, int64_t expected, int64_t desired) {
    if (!ptr) {
        return false;
    }
    int64_t expected_val = expected;
    return atomic_compare_exchange_strong((atomic_int_least64_t*)ptr, &expected_val, desired);
}

/**
 * @brief 原子加法 i64
 * @param ptr 指针（*i64）
 * @param delta 增量（i64）
 * @return i64（旧值）
 * 
 * 编译器签名：i64 runtime_atomic_add_i64(*i64 ptr, i64 delta)
 * 使用位置：stdlib/sync/atomic.eo - Atomic.add_i64()
 */
int64_t runtime_atomic_add_i64(int64_t* ptr, int64_t delta) {
    if (!ptr) {
        return 0;
    }
    return atomic_fetch_add((atomic_int_least64_t*)ptr, delta);
}

/**
 * @brief 原子加载 bool（使用 i8 实现）
 * @param ptr 指针（*bool，实际为 *i8）
 * @return bool -> i1（true 或 false）
 * 
 * 编译器签名：i1 runtime_atomic_load_bool(*i8 ptr)
 * 使用位置：stdlib/sync/atomic.eo - AtomicBool.load()
 * 
 * 注意：bool 在 C 中通常使用 uint8_t（i8）表示，所以使用 i8 的原子操作
 */
bool runtime_atomic_load_bool(uint8_t* ptr) {
    if (!ptr) {
        return false;
    }
    // 使用 atomic_uint_least8_t 来实现 bool 的原子操作
    return (bool)atomic_load((atomic_uint_least8_t*)ptr);
}

/**
 * @brief 原子存储 bool（使用 i8 实现）
 * @param ptr 指针（*bool，实际为 *i8）
 * @param value 值（bool -> i1，实际为 i8）
 * 
 * 编译器签名：void runtime_atomic_store_bool(*i8 ptr, i8 value)
 * 使用位置：stdlib/sync/atomic.eo - AtomicBool.store()
 * 
 * 注意：bool 在 C 中通常使用 uint8_t（i8）表示，所以使用 i8 的原子操作
 */
void runtime_atomic_store_bool(uint8_t* ptr, uint8_t value) {
    if (ptr) {
        // 使用 atomic_uint_least8_t 来实现 bool 的原子操作
        atomic_store((atomic_uint_least8_t*)ptr, value);
    }
}

/**
 * @brief 原子交换 bool（使用 i8 实现）
 * @param ptr 指针（*bool，实际为 *i8）
 * @param value 新值（bool -> i1，实际为 i8）
 * @return bool -> i1（旧值）
 * 
 * 编译器签名：i1 runtime_atomic_swap_bool(*i8 ptr, i8 value)
 * 使用位置：stdlib/sync/atomic.eo - AtomicBool.swap()
 * 
 * 注意：bool 在 C 中通常使用 uint8_t（i8）表示，所以使用 i8 的原子操作
 */
bool runtime_atomic_swap_bool(uint8_t* ptr, uint8_t value) {
    if (!ptr) {
        return false;
    }
    // 使用 atomic_uint_least8_t 来实现 bool 的原子操作
    return (bool)atomic_exchange((atomic_uint_least8_t*)ptr, value);
}

/**
 * @brief 原子比较并交换 bool（使用 i8 实现）
 * @param ptr 指针（*bool，实际为 *i8）
 * @param expected 期望值（bool -> i1，实际为 i8）
 * @param desired 新值（bool -> i1，实际为 i8）
 * @return bool -> i1（true 表示成功，false 表示失败）
 * 
 * 编译器签名：i1 runtime_atomic_cas_bool(*i8 ptr, i8 expected, i8 desired)
 * 使用位置：stdlib/sync/atomic.eo - AtomicBool.compare_and_swap()
 * 
 * 注意：bool 在 C 中通常使用 uint8_t（i8）表示，所以使用 i8 的原子操作
 */
bool runtime_atomic_cas_bool(uint8_t* ptr, uint8_t expected, uint8_t desired) {
    if (!ptr) {
        return false;
    }
    // 使用 atomic_uint_least8_t 来实现 bool 的原子操作
    uint8_t expected_val = expected;
    return atomic_compare_exchange_strong((atomic_uint_least8_t*)ptr, &expected_val, desired);
}

// ==================== 异步运行时 ====================

// 任务句柄表（简化实现）
#define MAX_TASK_HANDLES 1024
static uint64_t g_task_ids[MAX_TASK_HANDLES];
static bool g_task_used[MAX_TASK_HANDLES];
static int32_t g_next_task_id = 1;

// 通道句柄表（简化实现）
static uint64_t g_channel_ids[MAX_TASK_HANDLES];
static bool g_channel_used[MAX_TASK_HANDLES];
static int32_t g_next_channel_id = 1;

/**
 * @brief 创建任务
 * @param func_ptr 函数指针（i8*，泛型处理）
 * @param arg_ptr 参数指针（i8*）
 * @return TaskId -> i32
 * 
 * 编译器签名：i32 runtime_spawn_task(i8* func_ptr, i8* arg_ptr)
 * 使用位置：stdlib/async/task.eo - Task.spawn()
 * 
 * 注意：这是简化实现，实际应该调用任务调度器
 */
int32_t runtime_spawn_task(void* func_ptr, void* arg_ptr) {
    // 简化实现：分配任务 ID
    // TODO: 实际应该调用任务调度器创建任务
    for (int i = 0; i < MAX_TASK_HANDLES; i++) {
        if (!g_task_used[i]) {
            g_task_ids[i] = g_next_task_id++;
            g_task_used[i] = true;
            return (int32_t)g_task_ids[i];
        }
    }
    return -1; // 表已满
}

/**
 * @brief 查询任务状态
 * @param task_id 任务 ID（i32）
 * @return string -> i8*（状态字符串）
 * 
 * 编译器签名：i8* runtime_task_status(i32 task_id)
 * 使用位置：stdlib/async/task.eo - Task.status()
 * 
 * 注意：返回的字符串需要由调用者释放
 */
char* runtime_task_status(int32_t task_id) {
    // 简化实现：返回固定状态
    // TODO: 实际应该查询任务调度器
    char* status = (char*)malloc(32);
    if (status) {
        strcpy(status, "running"); // 简化：总是返回 "running"
    }
    return status;
}

/**
 * @brief 取消任务（标准库版本，不需要 RuntimeHandle）
 * @param task_id 任务 ID（i32）
 * @return Result[void, string] -> i8*
 * 
 * 编译器签名：i8* runtime_cancel_task(i32 task_id)
 * 使用位置：stdlib/async/task.eo - Task.cancel()
 * 
 * 注意：这个函数与旧的 runtime_cancel_task(RuntimeHandle*, TaskHandle*) 不同
 * 旧版本用于运行时系统内部，新版本用于标准库调用
 * 由于编译器声明的函数名是 runtime_cancel_task，我们使用这个名称
 */
void* runtime_cancel_task(int32_t task_id) {
    // 简化实现：检查任务是否存在
    // TODO: 实际应该调用任务调度器取消任务
    for (int i = 0; i < MAX_TASK_HANDLES; i++) {
        if (g_task_used[i] && g_task_ids[i] == (uint64_t)task_id) {
            g_task_used[i] = false;
            return result_ok(NULL, 0); // 返回 Ok(void)
        }
    }
    return result_err("Task not found");
}

/**
 * @brief 创建无缓冲通道
 * @return ChannelHandle -> i32
 * 
 * 编译器签名：i32 runtime_create_channel_unbuffered()
 * 使用位置：stdlib/async/channel.eo - Channel.unbuffered()
 */
int32_t runtime_create_channel_unbuffered(void) {
    // 简化实现：分配通道 ID
    // TODO: 实际应该调用通道管理器创建通道
    for (int i = 0; i < MAX_TASK_HANDLES; i++) {
        if (!g_channel_used[i]) {
            g_channel_ids[i] = g_next_channel_id++;
            g_channel_used[i] = true;
            return (int32_t)g_channel_ids[i];
        }
    }
    return -1; // 表已满
}

/**
 * @brief 创建有缓冲通道
 * @param cap 容量（i32）
 * @return ChannelHandle -> i32
 * 
 * 编译器签名：i32 runtime_create_channel_buffered(i32 cap)
 * 使用位置：stdlib/async/channel.eo - Channel.buffered()
 */
int32_t runtime_create_channel_buffered(int32_t cap) {
    if (cap < 0) {
        return -1; // 无效容量
    }
    
    // 简化实现：分配通道 ID（忽略容量参数）
    // TODO: 实际应该调用通道管理器创建有缓冲通道
    return runtime_create_channel_unbuffered();
}

/**
 * @brief 发送数据到通道（阻塞）
 * @param handle 通道句柄（i32）
 * @param item_ptr 数据指针（i8*）
 * @param item_size 数据大小（i32）
 * @return Result[void, string] -> i8*
 * 
 * 编译器签名：i8* runtime_channel_send(i32 handle, i8* item_ptr, i32 item_size)
 * 使用位置：stdlib/async/channel.eo - Channel.send()
 */
void* runtime_channel_send(int32_t handle, void* item_ptr, int32_t item_size) {
    if (!item_ptr || item_size < 0) {
        return result_err("Invalid parameters");
    }
    
    // 简化实现：检查通道是否存在
    // TODO: 实际应该调用通道管理器发送数据
    for (int i = 0; i < MAX_TASK_HANDLES; i++) {
        if (g_channel_used[i] && g_channel_ids[i] == (uint64_t)handle) {
            return result_ok(NULL, 0); // 返回 Ok(void)
        }
    }
    return result_err("Channel not found");
}

/**
 * @brief 从通道接收数据（阻塞）
 * @param handle 通道句柄（i32）
 * @param item_ptr 数据指针（i8*，输出）
 * @param item_size 数据大小（i32）
 * @return Result[int, string] -> i8*（返回接收到的数据大小或错误）
 * 
 * 编译器签名：i8* runtime_channel_recv(i32 handle, i8* item_ptr, i32 item_size)
 * 使用位置：stdlib/async/channel.eo - Channel.recv()
 */
void* runtime_channel_recv(int32_t handle, void* item_ptr, int32_t item_size) {
    if (!item_ptr || item_size < 0) {
        return result_err("Invalid parameters");
    }
    
    // 简化实现：检查通道是否存在
    // TODO: 实际应该调用通道管理器接收数据
    for (int i = 0; i < MAX_TASK_HANDLES; i++) {
        if (g_channel_used[i] && g_channel_ids[i] == (uint64_t)handle) {
            // 简化：返回 0 字节（表示没有数据）
            int32_t bytes_received = 0;
            return result_ok(&bytes_received, sizeof(int32_t));
        }
    }
    return result_err("Channel not found");
}

/**
 * @brief 尝试发送数据到通道（非阻塞）
 * @param handle 通道句柄（i32）
 * @param item_ptr 数据指针（i8*）
 * @param item_size 数据大小（i32）
 * @return Result[void, string] -> i8*
 * 
 * 编译器签名：i8* runtime_channel_try_send(i32 handle, i8* item_ptr, i32 item_size)
 * 使用位置：stdlib/async/channel.eo - Channel.try_send()
 */
void* runtime_channel_try_send(int32_t handle, void* item_ptr, int32_t item_size) {
    // 简化实现：与阻塞版本相同
    // TODO: 实际应该实现非阻塞逻辑
    return runtime_channel_send(handle, item_ptr, item_size);
}

/**
 * @brief 尝试从通道接收数据（非阻塞）
 * @param handle 通道句柄（i32）
 * @param item_ptr 数据指针（i8*，输出）
 * @param item_size 数据大小（i32）
 * @return Result[int, string] -> i8*
 * 
 * 编译器签名：i8* runtime_channel_try_recv(i32 handle, i8* item_ptr, i32 item_size)
 * 使用位置：stdlib/async/channel.eo - Channel.try_recv()
 */
void* runtime_channel_try_recv(int32_t handle, void* item_ptr, int32_t item_size) {
    // 简化实现：与阻塞版本相同
    // TODO: 实际应该实现非阻塞逻辑
    return runtime_channel_recv(handle, item_ptr, item_size);
}

/**
 * @brief 关闭通道
 * @param handle 通道句柄（i32）
 * 
 * 编译器签名：void runtime_channel_close(i32 handle)
 * 使用位置：stdlib/async/channel.eo - Channel.close()
 */
void runtime_channel_close(int32_t handle) {
    // 简化实现：标记通道为未使用
    // TODO: 实际应该调用通道管理器关闭通道
    for (int i = 0; i < MAX_TASK_HANDLES; i++) {
        if (g_channel_used[i] && g_channel_ids[i] == (uint64_t)handle) {
            g_channel_used[i] = false;
            break;
        }
    }
}

/**
 * @brief 查询通道状态
 * @param handle 通道句柄（i32）
 * @return string -> i8*（状态字符串）
 * 
 * 编译器签名：i8* runtime_channel_status(i32 handle)
 * 使用位置：stdlib/async/channel.eo - Channel.status()
 * 
 * 注意：返回的字符串需要由调用者释放
 */
char* runtime_channel_status(int32_t handle) {
    // 简化实现：返回固定状态
    // TODO: 实际应该查询通道管理器
    char* status = (char*)malloc(32);
    if (status) {
        strcpy(status, "open"); // 简化：总是返回 "open"
    }
    return status;
}

// ============================================================================
// P2 优先级运行时函数实现（增强功能）
// ============================================================================
// 这些函数是标准库的增强功能，包括网络 I/O 和数学函数

// ==================== 网络 I/O ====================

/**
 * @brief 解析地址字符串（格式：host:port）
 * @param addr 地址字符串
 * @param host 输出主机名（缓冲区）
 * @param host_len 主机名缓冲区大小
 * @param port 输出端口号
 * @return true 表示成功，false 表示失败
 */
static bool parse_address(const char* addr, char* host, size_t host_len, uint16_t* port) {
    if (!addr || !host || !port) {
        return false;
    }

    // 查找冒号
    const char* colon = strchr(addr, ':');
    if (!colon) {
        return false;
    }

    // 提取主机名
    size_t host_name_len = colon - addr;
    if (host_name_len >= host_len) {
        return false;
    }
    strncpy(host, addr, host_name_len);
    host[host_name_len] = '\0';

    // 提取端口号
    char* endptr;
    long port_long = strtol(colon + 1, &endptr, 10);
    if (*endptr != '\0' || port_long < 0 || port_long > 65535) {
        return false;
    }
    *port = (uint16_t)port_long;

    return true;
}

/**
 * @brief 建立 TCP 连接
 * @param addr 地址字符串（格式：host:port）
 * @return Result[SocketHandle, string] -> i8*
 * 
 * 编译器签名：i8* runtime_tcp_connect(i8* addr)
 * 使用位置：stdlib/net/tcp.eo - TcpStream.connect()
 */
void* runtime_tcp_connect(const char* addr) {
    if (!addr) {
        return result_err("Address is null");
    }

    // 解析地址
    char host[256];
    uint16_t port;
    if (!parse_address(addr, host, sizeof(host), &port)) {
        return result_err("Invalid address format (expected host:port)");
    }

    // 创建套接字
    int sockfd = socket(AF_INET, SOCK_STREAM, 0);
    if (sockfd < 0) {
        char error_msg[256];
        snprintf(error_msg, sizeof(error_msg), "Failed to create socket (errno: %d)", errno);
        return result_err(error_msg);
    }

    // 解析主机名
    struct hostent* host_entry = gethostbyname(host);
    if (!host_entry) {
        close(sockfd);
        char error_msg[256];
        snprintf(error_msg, sizeof(error_msg), "Failed to resolve hostname: %s", host);
        return result_err(error_msg);
    }

    // 设置服务器地址
    struct sockaddr_in server_addr;
    memset(&server_addr, 0, sizeof(server_addr));
    server_addr.sin_family = AF_INET;
    server_addr.sin_port = htons(port);
    memcpy(&server_addr.sin_addr, host_entry->h_addr_list[0], host_entry->h_length);

    // 连接服务器
    if (connect(sockfd, (struct sockaddr*)&server_addr, sizeof(server_addr)) < 0) {
        close(sockfd);
        char error_msg[256];
        snprintf(error_msg, sizeof(error_msg), "Failed to connect to %s:%d (errno: %d)", host, port, errno);
        return result_err(error_msg);
    }

    // 分配 Socket 句柄
    SocketHandleTable* table = get_global_socket_handle_table();
    if (!table) {
        close(sockfd);
        return result_err("Failed to initialize socket handle table");
    }

    int32_t handle = socket_handle_allocate(table, sockfd, false);
    if (handle < 0) {
        close(sockfd);
        return result_err("Failed to allocate socket handle (table full)");
    }

    // 返回 Ok(SocketHandle)
    return result_ok(&handle, sizeof(int32_t));
}

/**
 * @brief 绑定地址并监听
 * @param addr 地址字符串（格式：host:port）
 * @return Result[SocketHandle, string] -> i8*
 * 
 * 编译器签名：i8* runtime_tcp_bind(i8* addr)
 * 使用位置：stdlib/net/tcp.eo - TcpListener.bind()
 */
void* runtime_tcp_bind(const char* addr) {
    if (!addr) {
        return result_err("Address is null");
    }

    // 解析地址
    char host[256];
    uint16_t port;
    if (!parse_address(addr, host, sizeof(host), &port)) {
        return result_err("Invalid address format (expected host:port)");
    }

    // 创建套接字
    int sockfd = socket(AF_INET, SOCK_STREAM, 0);
    if (sockfd < 0) {
        char error_msg[256];
        snprintf(error_msg, sizeof(error_msg), "Failed to create socket (errno: %d)", errno);
        return result_err(error_msg);
    }

    // 设置 SO_REUSEADDR 选项
    int opt = 1;
    if (setsockopt(sockfd, SOL_SOCKET, SO_REUSEADDR, &opt, sizeof(opt)) < 0) {
        close(sockfd);
        char error_msg[256];
        snprintf(error_msg, sizeof(error_msg), "Failed to set socket options (errno: %d)", errno);
        return result_err(error_msg);
    }

    // 设置服务器地址
    struct sockaddr_in server_addr;
    memset(&server_addr, 0, sizeof(server_addr));
    server_addr.sin_family = AF_INET;
    server_addr.sin_port = htons(port);
    
    // 如果 host 是 "*" 或 "0.0.0.0"，绑定到所有接口
    if (strcmp(host, "*") == 0 || strcmp(host, "0.0.0.0") == 0) {
        server_addr.sin_addr.s_addr = INADDR_ANY;
    } else {
        if (inet_aton(host, &server_addr.sin_addr) == 0) {
            close(sockfd);
            char error_msg[256];
            snprintf(error_msg, sizeof(error_msg), "Invalid IP address: %s", host);
            return result_err(error_msg);
        }
    }

    // 绑定地址
    if (bind(sockfd, (struct sockaddr*)&server_addr, sizeof(server_addr)) < 0) {
        close(sockfd);
        char error_msg[256];
        snprintf(error_msg, sizeof(error_msg), "Failed to bind to %s:%d (errno: %d)", host, port, errno);
        return result_err(error_msg);
    }

    // 开始监听
    if (listen(sockfd, 128) < 0) {
        close(sockfd);
        char error_msg[256];
        snprintf(error_msg, sizeof(error_msg), "Failed to listen (errno: %d)", errno);
        return result_err(error_msg);
    }

    // 分配 Socket 句柄
    SocketHandleTable* table = get_global_socket_handle_table();
    if (!table) {
        close(sockfd);
        return result_err("Failed to initialize socket handle table");
    }

    int32_t handle = socket_handle_allocate(table, sockfd, true); // 标记为监听套接字
    if (handle < 0) {
        close(sockfd);
        return result_err("Failed to allocate socket handle (table full)");
    }

    // 返回 Ok(SocketHandle)
    return result_ok(&handle, sizeof(int32_t));
}

/**
 * @brief 接受连接
 * @param listener 监听套接字句柄（i32）
 * @return Result[SocketHandle, string] -> i8*
 * 
 * 编译器签名：i8* runtime_tcp_accept(i32 listener)
 * 使用位置：stdlib/net/tcp.eo - TcpListener.accept()
 */
void* runtime_tcp_accept(int32_t listener) {
    SocketHandleTable* table = get_global_socket_handle_table();
    if (!table) {
        return result_err("Socket handle table not initialized");
    }

    if (!socket_handle_is_listener(table, listener)) {
        return result_err("Invalid listener socket handle");
    }

    int listener_fd = socket_handle_get(table, listener);
    if (listener_fd < 0) {
        return result_err("Invalid listener socket");
    }

    // 接受连接
    struct sockaddr_in client_addr;
    socklen_t client_addr_len = sizeof(client_addr);
    int client_fd = accept(listener_fd, (struct sockaddr*)&client_addr, &client_addr_len);
    if (client_fd < 0) {
        char error_msg[256];
        snprintf(error_msg, sizeof(error_msg), "Failed to accept connection (errno: %d)", errno);
        return result_err(error_msg);
    }

    // 分配 Socket 句柄
    int32_t handle = socket_handle_allocate(table, client_fd, false);
    if (handle < 0) {
        close(client_fd);
        return result_err("Failed to allocate socket handle (table full)");
    }

    // 返回 Ok(SocketHandle)
    return result_ok(&handle, sizeof(int32_t));
}

/**
 * @brief 读取 TCP 数据
 * @param socket 套接字句柄（i32）
 * @param buf_ptr 缓冲区指针（i8*）
 * @param buf_len 缓冲区长度（i32）
 * @return Result[int, string] -> i8*
 * 
 * 编译器签名：i8* runtime_tcp_read(i32 socket, i8* buf_ptr, i32 buf_len)
 * 使用位置：stdlib/net/tcp.eo - TcpStream.read()
 */
void* runtime_tcp_read(int32_t socket, void* buf_ptr, int32_t buf_len) {
    if (!buf_ptr || buf_len < 0) {
        return result_err("Invalid buffer parameters");
    }

    SocketHandleTable* table = get_global_socket_handle_table();
    if (!table) {
        return result_err("Socket handle table not initialized");
    }

    int sockfd = socket_handle_get(table, socket);
    if (sockfd < 0) {
        return result_err("Invalid socket handle");
    }

    // 读取数据
    ssize_t bytes_read = recv(sockfd, buf_ptr, (size_t)buf_len, 0);
    if (bytes_read < 0) {
        char error_msg[256];
        snprintf(error_msg, sizeof(error_msg), "Failed to read from socket (errno: %d)", errno);
        return result_err(error_msg);
    }

    // 返回 Ok(读取的字节数)
    int32_t bytes_read_i32 = (int32_t)bytes_read;
    return result_ok(&bytes_read_i32, sizeof(int32_t));
}

/**
 * @brief 写入 TCP 数据
 * @param socket 套接字句柄（i32）
 * @param buf_ptr 缓冲区指针（i8*）
 * @param buf_len 缓冲区长度（i32）
 * @return Result[int, string] -> i8*
 * 
 * 编译器签名：i8* runtime_tcp_write(i32 socket, i8* buf_ptr, i32 buf_len)
 * 使用位置：stdlib/net/tcp.eo - TcpStream.write()
 */
void* runtime_tcp_write(int32_t socket, const void* buf_ptr, int32_t buf_len) {
    if (!buf_ptr || buf_len < 0) {
        return result_err("Invalid buffer parameters");
    }

    SocketHandleTable* table = get_global_socket_handle_table();
    if (!table) {
        return result_err("Socket handle table not initialized");
    }

    int sockfd = socket_handle_get(table, socket);
    if (sockfd < 0) {
        return result_err("Invalid socket handle");
    }

    // 写入数据
    ssize_t bytes_written = send(sockfd, buf_ptr, (size_t)buf_len, 0);
    if (bytes_written < 0) {
        char error_msg[256];
        snprintf(error_msg, sizeof(error_msg), "Failed to write to socket (errno: %d)", errno);
        return result_err(error_msg);
    }

    // 返回 Ok(写入的字节数)
    int32_t bytes_written_i32 = (int32_t)bytes_written;
    return result_ok(&bytes_written_i32, sizeof(int32_t));
}

/**
 * @brief 关闭 TCP 连接
 * @param socket 套接字句柄（i32）
 * @return Result[void, string] -> i8*
 * 
 * 编译器签名：i8* runtime_tcp_close(i32 socket)
 * 使用位置：stdlib/net/tcp.eo - TcpStream.close(), TcpListener.close()
 */
void* runtime_tcp_close(int32_t socket) {
    SocketHandleTable* table = get_global_socket_handle_table();
    if (!table) {
        return result_err("Socket handle table not initialized");
    }

    if (!socket_handle_is_valid(table, socket)) {
        return result_err("Invalid socket handle");
    }

    if (socket_handle_release(table, socket)) {
        // 返回 Ok(void)
        return result_ok(NULL, 0);
    } else {
        return result_err("Failed to close socket");
    }
}

/**
 * @brief 刷新 TCP 缓冲区
 * @param socket 套接字句柄（i32）
 * @return Result[void, string] -> i8*
 * 
 * 编译器签名：i8* runtime_tcp_flush(i32 socket)
 * 使用位置：stdlib/net/tcp.eo - TcpStream.flush()
 * 
 * 注意：TCP 是流式协议，没有显式的 flush 操作。此函数为了 API 一致性而提供，
 * 实际上 TCP 数据会自动发送，无需手动 flush。此函数总是返回成功。
 */
void* runtime_tcp_flush(int32_t socket) {
    SocketHandleTable* table = get_global_socket_handle_table();
    if (!table) {
        return result_err("Socket handle table not initialized");
    }

    if (!socket_handle_is_valid(table, socket)) {
        return result_err("Invalid socket handle");
    }

    // TCP 是流式协议，数据会自动发送，无需手动 flush
    // 为了 API 一致性，此函数总是返回成功
    // 返回 Ok(void)
    return result_ok(NULL, 0);
}

/**
 * @brief 获取 TCP 连接的本地地址
 * @param socket 套接字句柄（i32）
 * @return Result[string, string] -> i8*
 * 
 * 编译器签名：i8* runtime_tcp_local_addr(i32 socket)
 * 使用位置：stdlib/net/tcp.eo - TcpStream.local_addr()
 */
void* runtime_tcp_local_addr(int32_t socket) {
    SocketHandleTable* table = get_global_socket_handle_table();
    if (!table) {
        return result_err("Socket handle table not initialized");
    }

    if (!socket_handle_is_valid(table, socket)) {
        return result_err("Invalid socket handle");
    }

    int sockfd = socket_handle_get(table, socket);
    if (sockfd < 0) {
        return result_err("Invalid socket handle");
    }

    // 获取本地地址
    struct sockaddr_in addr;
    socklen_t addr_len = sizeof(addr);
    if (getsockname(sockfd, (struct sockaddr*)&addr, &addr_len) < 0) {
        char error_msg[256];
        snprintf(error_msg, sizeof(error_msg), "Failed to get local address (errno: %d)", errno);
        return result_err(error_msg);
    }

    // 格式化地址字符串（host:port）
    char addr_str[256];
    char* host_str = inet_ntoa(addr.sin_addr);
    uint16_t port = ntohs(addr.sin_port);
    snprintf(addr_str, sizeof(addr_str), "%s:%d", host_str, port);

    // 分配字符串内存并返回
    size_t len = strlen(addr_str);
    char* result_str = (char*)malloc(len + 1);
    if (!result_str) {
        return result_err("Failed to allocate memory for address string");
    }
    strcpy(result_str, addr_str);

    // 返回 Ok(string)
    return result_ok(result_str, (int32_t)(len + 1));
}

/**
 * @brief 获取 TCP 连接的远程地址（对端地址）
 * @param socket 套接字句柄（i32）
 * @return Result[string, string] -> i8*
 * 
 * 编译器签名：i8* runtime_tcp_peer_addr(i32 socket)
 * 使用位置：stdlib/net/tcp.eo - TcpStream.peer_addr()
 */
void* runtime_tcp_peer_addr(int32_t socket) {
    SocketHandleTable* table = get_global_socket_handle_table();
    if (!table) {
        return result_err("Socket handle table not initialized");
    }

    if (!socket_handle_is_valid(table, socket)) {
        return result_err("Invalid socket handle");
    }

    int sockfd = socket_handle_get(table, socket);
    if (sockfd < 0) {
        return result_err("Invalid socket handle");
    }

    // 获取远程地址
    struct sockaddr_in addr;
    socklen_t addr_len = sizeof(addr);
    if (getpeername(sockfd, (struct sockaddr*)&addr, &addr_len) < 0) {
        char error_msg[256];
        snprintf(error_msg, sizeof(error_msg), "Failed to get peer address (errno: %d)", errno);
        return result_err(error_msg);
    }

    // 格式化地址字符串（host:port）
    char addr_str[256];
    char* host_str = inet_ntoa(addr.sin_addr);
    uint16_t port = ntohs(addr.sin_port);
    snprintf(addr_str, sizeof(addr_str), "%s:%d", host_str, port);

    // 分配字符串内存并返回
    size_t len = strlen(addr_str);
    char* result_str = (char*)malloc(len + 1);
    if (!result_str) {
        return result_err("Failed to allocate memory for address string");
    }
    strcpy(result_str, addr_str);

    // 返回 Ok(string)
    return result_ok(result_str, (int32_t)(len + 1));
}

// ==================== 数学函数 ====================

/**
 * @brief 平方根
 * @param x 输入值（f64）
 * @return f64
 * 
 * 编译器签名：f64 runtime_math_sqrt(f64 x)
 * 使用位置：stdlib/math/math.eo - Math.sqrt()
 */
double runtime_math_sqrt(double x) {
    return sqrt(x);
}

/**
 * @brief 立方根
 * @param x 输入值（f64）
 * @return f64
 * 
 * 编译器签名：f64 runtime_math_cbrt(f64 x)
 * 使用位置：stdlib/math/math.eo - Math.cbrt()
 */
double runtime_math_cbrt(double x) {
    return cbrt(x);
}

/**
 * @brief 幂运算
 * @param base 底数（f64）
 * @param exp 指数（f64）
 * @return f64
 * 
 * 编译器签名：f64 runtime_math_pow(f64 base, f64 exp)
 * 使用位置：stdlib/math/math.eo - Math.pow()
 */
double runtime_math_pow(double base, double exp) {
    return pow(base, exp);
}

/**
 * @brief 自然指数（e^x）
 * @param x 输入值（f64）
 * @return f64
 * 
 * 编译器签名：f64 runtime_math_exp(f64 x)
 * 使用位置：stdlib/math/math.eo - Math.exp()
 */
double runtime_math_exp(double x) {
    return exp(x);
}

/**
 * @brief 自然对数（ln(x)）
 * @param x 输入值（f64）
 * @return f64
 * 
 * 编译器签名：f64 runtime_math_ln(f64 x)
 * 使用位置：stdlib/math/math.eo - Math.ln()
 */
double runtime_math_ln(double x) {
    return log(x);
}

/**
 * @brief 常用对数（log10(x)）
 * @param x 输入值（f64）
 * @return f64
 * 
 * 编译器签名：f64 runtime_math_log10(f64 x)
 * 使用位置：stdlib/math/math.eo - Math.log10()
 */
double runtime_math_log10(double x) {
    return log10(x);
}

/**
 * @brief 二进制对数（log2(x)）
 * @param x 输入值（f64）
 * @return f64
 * 
 * 编译器签名：f64 runtime_math_log2(f64 x)
 * 使用位置：stdlib/math/math.eo - Math.log2()
 */
double runtime_math_log2(double x) {
    return log2(x);
}

/**
 * @brief 正弦函数
 * @param x 输入值（f64，弧度）
 * @return f64
 * 
 * 编译器签名：f64 runtime_math_sin(f64 x)
 * 使用位置：stdlib/math/math.eo - Math.sin()
 */
double runtime_math_sin(double x) {
    return sin(x);
}

/**
 * @brief 余弦函数
 * @param x 输入值（f64，弧度）
 * @return f64
 * 
 * 编译器签名：f64 runtime_math_cos(f64 x)
 * 使用位置：stdlib/math/math.eo - Math.cos()
 */
double runtime_math_cos(double x) {
    return cos(x);
}

/**
 * @brief 正切函数
 * @param x 输入值（f64，弧度）
 * @return f64
 * 
 * 编译器签名：f64 runtime_math_tan(f64 x)
 * 使用位置：stdlib/math/math.eo - Math.tan()
 */
double runtime_math_tan(double x) {
    return tan(x);
}

/**
 * @brief 反正弦函数
 * @param x 输入值（f64，范围：-1 到 1）
 * @return f64（弧度）
 * 
 * 编译器签名：f64 runtime_math_asin(f64 x)
 * 使用位置：stdlib/math/math.eo - Math.asin()
 */
double runtime_math_asin(double x) {
    return asin(x);
}

/**
 * @brief 反余弦函数
 * @param x 输入值（f64，范围：-1 到 1）
 * @return f64（弧度）
 * 
 * 编译器签名：f64 runtime_math_acos(f64 x)
 * 使用位置：stdlib/math/math.eo - Math.acos()
 */
double runtime_math_acos(double x) {
    return acos(x);
}

/**
 * @brief 反正切函数
 * @param x 输入值（f64）
 * @return f64（弧度）
 * 
 * 编译器签名：f64 runtime_math_atan(f64 x)
 * 使用位置：stdlib/math/math.eo - Math.atan()
 */
double runtime_math_atan(double x) {
    return atan(x);
}

/**
 * @brief 向上取整
 * @param x 输入值（f64）
 * @return f64（向上取整后的值）
 * 
 * 编译器签名：f64 runtime_math_ceil(f64 x)
 * 使用位置：stdlib/math/math.eo - Math.ceil()
 */
double runtime_math_ceil(double x) {
    return ceil(x);
}

/**
 * @brief 向下取整
 * @param x 输入值（f64）
 * @return f64（向下取整后的值）
 * 
 * 编译器签名：f64 runtime_math_floor(f64 x)
 * 使用位置：stdlib/math/math.eo - Math.floor()
 */
double runtime_math_floor(double x) {
    return floor(x);
}

/**
 * @brief 四舍五入
 * @param x 输入值（f64）
 * @return f64（四舍五入后的值）
 * 
 * 编译器签名：f64 runtime_math_round(f64 x)
 * 使用位置：stdlib/math/math.eo - Math.round()
 */
double runtime_math_round(double x) {
    return round(x);
}

/**
 * @brief 截断（向零取整）
 * @param x 输入值（f64）
 * @return f64（截断后的值）
 * 
 * 编译器签名：f64 runtime_math_trunc(f64 x)
 * 使用位置：stdlib/math/math.eo - Math.trunc()
 */
double runtime_math_trunc(double x) {
    return trunc(x);
}

/**
 * @brief 反正切函数（两个参数版本）
 * @param y y坐标（f64）
 * @param x x坐标（f64）
 * @return f64（弧度，范围 [-π, π]）
 * 
 * 编译器签名：f64 runtime_math_atan2(f64 y, f64 x)
 * 使用位置：stdlib/math/math.eo - Math.atan2()
 */
double runtime_math_atan2(double y, double x) {
    return atan2(y, x);
}

/**
 * @brief 双曲正弦函数
 * @param x 输入值（f64）
 * @return f64（双曲正弦值）
 * 
 * 编译器签名：f64 runtime_math_sinh(f64 x)
 * 使用位置：stdlib/math/math.eo - Math.sinh()
 */
double runtime_math_sinh(double x) {
    return sinh(x);
}

/**
 * @brief 双曲余弦函数
 * @param x 输入值（f64）
 * @return f64（双曲余弦值）
 * 
 * 编译器签名：f64 runtime_math_cosh(f64 x)
 * 使用位置：stdlib/math/math.eo - Math.cosh()
 */
double runtime_math_cosh(double x) {
    return cosh(x);
}

/**
 * @brief 双曲正切函数
 * @param x 输入值（f64）
 * @return f64（双曲正切值）
 * 
 * 编译器签名：f64 runtime_math_tanh(f64 x)
 * 使用位置：stdlib/math/math.eo - Math.tanh()
 */
double runtime_math_tanh(double x) {
    return tanh(x);
}

// ==================== HTTP 和 URL 处理 ====================

/**
 * @brief 解析URL
 * @param url URL字符串（i8*）
 * @return URL解析结果结构体指针，需要调用者释放内存
 * 
 * 编译器签名：i8* runtime_url_parse(i8* url)
 * 使用位置：stdlib/net/http/client.eo - parse_url()
 * 
 * 实现：解析URL字符串，提取scheme、host、port、path等组件
 * 注意：返回的结构体包含解析结果，需要由调用者或GC管理
 */
UrlParseResult* runtime_url_parse(const char* url) {
    // 分配结果结构体
    UrlParseResult* result = (UrlParseResult*)malloc(sizeof(UrlParseResult));
    if (!result) {
        return NULL;
    }
    
    // 初始化失败状态
    result->success = false;
    result->scheme = NULL;
    result->host = NULL;
    result->port = -1;
    result->path = NULL;
    result->query = NULL;
    result->fragment = NULL;
    
    if (!url || strlen(url) == 0) {
        return result;
    }
    
    const char* url_ptr = url;
    size_t url_len = strlen(url);
    
    // 1. 解析scheme（协议）
    const char* scheme_end = strstr(url_ptr, "://");
    if (scheme_end) {
        size_t scheme_len = scheme_end - url_ptr;
        result->scheme = (char*)malloc(scheme_len + 1);
        if (result->scheme) {
            strncpy(result->scheme, url_ptr, scheme_len);
            result->scheme[scheme_len] = '\0';
        }
        url_ptr = scheme_end + 3; // 跳过 "://"
    } else {
        // 没有scheme，默认为空
        result->scheme = (char*)malloc(1);
        if (result->scheme) {
            result->scheme[0] = '\0';
        }
    }
    
    // 2. 解析host和port
    const char* path_start = strchr(url_ptr, '/');
    const char* query_start = strchr(url_ptr, '?');
    const char* fragment_start = strchr(url_ptr, '#');
    
    // 找到host:port的结束位置
    const char* host_end = url_ptr;
    if (path_start) {
        host_end = path_start;
    } else if (query_start) {
        host_end = query_start;
    } else if (fragment_start) {
        host_end = fragment_start;
    } else {
        host_end = url_ptr + strlen(url_ptr);
    }
    
    // 检查是否有端口号
    const char* colon = strchr(url_ptr, ':');
    if (colon && colon < host_end) {
        // 有端口号
        size_t host_len = colon - url_ptr;
        result->host = (char*)malloc(host_len + 1);
        if (result->host) {
            strncpy(result->host, url_ptr, host_len);
            result->host[host_len] = '\0';
        }
        
        // 解析端口号
        const char* port_start = colon + 1;
        size_t port_len = host_end - port_start;
        if (port_len > 0) {
            char port_str[32];
            strncpy(port_str, port_start, port_len < sizeof(port_str) - 1 ? port_len : sizeof(port_str) - 1);
            port_str[port_len < sizeof(port_str) - 1 ? port_len : sizeof(port_str) - 1] = '\0';
            result->port = (int32_t)atoi(port_str);
        }
    } else {
        // 没有端口号
        size_t host_len = host_end - url_ptr;
        result->host = (char*)malloc(host_len + 1);
        if (result->host) {
            strncpy(result->host, url_ptr, host_len);
            result->host[host_len] = '\0';
        }
        result->port = -1; // 使用默认端口
    }
    
    // 3. 解析path
    if (path_start) {
        const char* path_end = query_start ? query_start : (fragment_start ? fragment_start : (url_ptr + strlen(url_ptr)));
        size_t path_len = path_end - path_start;
        result->path = (char*)malloc(path_len + 1);
        if (result->path) {
            strncpy(result->path, path_start, path_len);
            result->path[path_len] = '\0';
        }
    } else {
        // 没有path，默认为 "/"
        result->path = (char*)malloc(2);
        if (result->path) {
            strcpy(result->path, "/");
        }
    }
    
    // 4. 解析query
    if (query_start) {
        const char* query_end = fragment_start ? fragment_start : (url_ptr + strlen(url_ptr));
        size_t query_len = query_end - query_start - 1; // 跳过 '?'
        result->query = (char*)malloc(query_len + 1);
        if (result->query) {
            strncpy(result->query, query_start + 1, query_len);
            result->query[query_len] = '\0';
        }
    }
    
    // 5. 解析fragment
    if (fragment_start) {
        size_t fragment_len = strlen(fragment_start + 1); // 跳过 '#'
        result->fragment = (char*)malloc(fragment_len + 1);
        if (result->fragment) {
            strcpy(result->fragment, fragment_start + 1);
        }
    }
    
    // 设置成功状态
    result->success = true;
    
    return result;
}

/**
 * @brief 解析HTTP请求
 * @param data HTTP请求字节数组（i8*）
 * @param data_len 字节数组长度（i32）
 * @return HTTP请求解析结果结构体指针，需要调用者释放内存
 * 
 * 编译器签名：i8* runtime_http_parse_request(i8* data, i32 data_len)
 * 使用位置：stdlib/net/http/server.eo - parse_request()
 * 
 * 实现：解析HTTP请求字节数组，提取请求行、头部、正文
 */
HttpParseRequestResult* runtime_http_parse_request(const uint8_t* data, int32_t data_len) {
    // 分配结果结构体
    HttpParseRequestResult* result = (HttpParseRequestResult*)malloc(sizeof(HttpParseRequestResult));
    if (!result) {
        return NULL;
    }
    
    // 初始化失败状态
    result->success = false;
    result->method = NULL;
    result->path = NULL;
    result->version = NULL;
    result->header_keys = NULL;
    result->header_values = NULL;
    result->header_count = 0;
    result->body = NULL;
    result->body_len = 0;
    
    if (!data || data_len <= 0) {
        return result;
    }
    
    // 将字节数组转换为字符串（用于解析）
    char* request_str = (char*)malloc(data_len + 1);
    if (!request_str) {
        return result;
    }
    memcpy(request_str, data, data_len);
    request_str[data_len] = '\0';
    
    // 1. 解析请求行（第一行）
    char* line_end = strstr(request_str, "\r\n");
    if (!line_end) {
        free(request_str);
        return result;
    }
    
    // 提取请求行
    size_t request_line_len = line_end - request_str;
    char* request_line = (char*)malloc(request_line_len + 1);
    if (!request_line) {
        free(request_str);
        return result;
    }
    strncpy(request_line, request_str, request_line_len);
    request_line[request_line_len] = '\0';
    
    // 解析请求行：METHOD PATH VERSION
    char* method_end = strchr(request_line, ' ');
    if (!method_end) {
        free(request_line);
        free(request_str);
        return result;
    }
    
    size_t method_len = method_end - request_line;
    result->method = (char*)malloc(method_len + 1);
    if (result->method) {
        strncpy(result->method, request_line, method_len);
        result->method[method_len] = '\0';
    }
    
    char* path_start = method_end + 1;
    char* path_end = strchr(path_start, ' ');
    if (!path_end) {
        free(request_line);
        free(request_str);
        return result;
    }
    
    size_t path_len = path_end - path_start;
    result->path = (char*)malloc(path_len + 1);
    if (result->path) {
        strncpy(result->path, path_start, path_len);
        result->path[path_len] = '\0';
    }
    
    char* version_start = path_end + 1;
    result->version = (char*)malloc(strlen(version_start) + 1);
    if (result->version) {
        strcpy(result->version, version_start);
    }
    
    free(request_line);
    
    // 2. 解析头部（从第二行开始，直到空行）
    char* headers_start = line_end + 2; // 跳过 "\r\n"
    char* headers_end = strstr(headers_start, "\r\n\r\n");
    if (!headers_end) {
        // 没有头部，直接跳到正文
        headers_end = headers_start;
    }
    
    // 分配头部数组（最多64个头部）
    const int max_headers = 64;
    result->header_keys = (char**)malloc(max_headers * sizeof(char*));
    result->header_values = (char**)malloc(max_headers * sizeof(char*));
    if (!result->header_keys || !result->header_values) {
        free(request_str);
        return result;
    }
    
    // 解析每个头部行
    char* header_line_start = headers_start;
    int header_count = 0;
    
    while (header_line_start < headers_end && header_count < max_headers) {
        char* header_line_end = strstr(header_line_start, "\r\n");
        if (!header_line_end || header_line_end >= headers_end) {
            break;
        }
        
        size_t header_line_len = header_line_end - header_line_start;
        char* header_line = (char*)malloc(header_line_len + 1);
        if (!header_line) {
            break;
        }
        strncpy(header_line, header_line_start, header_line_len);
        header_line[header_line_len] = '\0';
        
        // 解析 "Key: Value"
        char* colon = strchr(header_line, ':');
        if (colon) {
            size_t key_len = colon - header_line;
            result->header_keys[header_count] = (char*)malloc(key_len + 1);
            if (result->header_keys[header_count]) {
                strncpy(result->header_keys[header_count], header_line, key_len);
                result->header_keys[header_count][key_len] = '\0';
            }
            
            // 跳过 ": "（冒号和空格）
            char* value_start = colon + 1;
            while (*value_start == ' ') {
                value_start++;
            }
            
            result->header_values[header_count] = (char*)malloc(strlen(value_start) + 1);
            if (result->header_values[header_count]) {
                strcpy(result->header_values[header_count], value_start);
            }
            
            header_count++;
        }
        
        free(header_line);
        header_line_start = header_line_end + 2; // 跳过 "\r\n"
    }
    
    result->header_count = header_count;
    
    // 3. 解析正文（空行之后的内容）
    char* body_start = headers_end;
    if (strstr(body_start, "\r\n\r\n")) {
        body_start = strstr(body_start, "\r\n\r\n") + 4; // 跳过 "\r\n\r\n"
    } else if (strstr(body_start, "\n\n")) {
        body_start = strstr(body_start, "\n\n") + 2; // 跳过 "\n\n"
    }
    
    if (body_start < request_str + data_len) {
        size_t body_len = (request_str + data_len) - body_start;
        result->body = (uint8_t*)malloc(body_len);
        if (result->body) {
            memcpy(result->body, body_start, body_len);
            result->body_len = (int32_t)body_len;
        }
    }
    
    free(request_str);
    
    // 设置成功状态
    result->success = true;
    
    return result;
}

/**
 * @brief 构建HTTP响应
 * @param status_code 状态码（i32）
 * @param status_text 状态文本（i8*）
 * @param header_keys 头部键数组（i8**）
 * @param header_values 头部值数组（i8**）
 * @param header_count 头部数量（i32）
 * @param body 响应正文（i8*）
 * @param body_len 正文长度（i32）
 * @return HTTP响应构建结果结构体指针，需要调用者释放内存
 * 
 * 编译器签名：i8* runtime_http_build_response(i32 status_code, i8* status_text, i8** header_keys, i8** header_values, i32 header_count, i8* body, i32 body_len)
 * 使用位置：stdlib/net/http/server.eo - build_response()
 * 
 * 实现：构建HTTP响应字节数组，包含状态行、头部、正文
 */
HttpBuildResponseResult* runtime_http_build_response(
    int32_t status_code,
    const char* status_text,
    const char** header_keys,
    const char** header_values,
    int32_t header_count,
    const uint8_t* body,
    int32_t body_len
) {
    // 分配结果结构体
    HttpBuildResponseResult* result = (HttpBuildResponseResult*)malloc(sizeof(HttpBuildResponseResult));
    if (!result) {
        return NULL;
    }
    
    // 初始化失败状态
    result->success = false;
    result->data = NULL;
    result->data_len = 0;
    
    // 估算响应大小（状态行 + 头部 + 正文 + 一些额外空间）
    size_t estimated_size = 256; // 状态行和基本头部
    if (header_keys && header_values) {
        for (int32_t i = 0; i < header_count; i++) {
            if (header_keys[i] && header_values[i]) {
                estimated_size += strlen(header_keys[i]) + strlen(header_values[i]) + 4; // "Key: Value\r\n"
            }
        }
    }
    if (body && body_len > 0) {
        estimated_size += (size_t)body_len;
    }
    
    // 分配响应缓冲区
    result->data = (uint8_t*)malloc(estimated_size);
    if (!result->data) {
        return result;
    }
    
    size_t pos = 0;
    
    // 1. 构建状态行：HTTP/1.1 STATUS_CODE STATUS_TEXT\r\n
    int written = snprintf((char*)(result->data + pos), estimated_size - pos,
                           "HTTP/1.1 %d %s\r\n", status_code, status_text ? status_text : "OK");
    if (written < 0 || (size_t)written >= estimated_size - pos) {
        free(result->data);
        result->data = NULL;
        return result;
    }
    pos += (size_t)written;
    
    // 2. 构建头部
    if (header_keys && header_values) {
        for (int32_t i = 0; i < header_count; i++) {
            if (header_keys[i] && header_values[i]) {
                written = snprintf((char*)(result->data + pos), estimated_size - pos,
                                   "%s: %s\r\n", header_keys[i], header_values[i]);
                if (written < 0 || (size_t)written >= estimated_size - pos) {
                    free(result->data);
                    result->data = NULL;
                    return result;
                }
                pos += (size_t)written;
            }
        }
    }
    
    // 3. 添加空行（分隔头部和正文）
    if (pos + 2 < estimated_size) {
        result->data[pos++] = '\r';
        result->data[pos++] = '\n';
    }
    
    // 4. 添加正文
    if (body && body_len > 0) {
        if (pos + (size_t)body_len <= estimated_size) {
            memcpy(result->data + pos, body, (size_t)body_len);
            pos += (size_t)body_len;
        } else {
            // 缓冲区不够，重新分配
            uint8_t* new_data = (uint8_t*)realloc(result->data, pos + (size_t)body_len);
            if (new_data) {
                result->data = new_data;
                memcpy(result->data + pos, body, (size_t)body_len);
                pos += (size_t)body_len;
            } else {
                free(result->data);
                result->data = NULL;
                return result;
            }
        }
    }
    
    result->data_len = (int32_t)pos;
    result->success = true;
    
    return result;
}

/**
 * @brief 返回 NULL 指针
 * 用于在Echo语言中表示空指针（因为语言暂不支持 nil 字面量）
 */
void* runtime_null_ptr(void) {
    return NULL;
}

/**
 * @brief 获取map的键值对数组
 * 从map中提取所有键值对，返回键数组和值数组
 * 
 * 注意：map的内部表示使用hash_table_t结构
 * map_ptr指向hash_table_t结构（从common_types.h定义）
 */
MapIterResult* runtime_map_get_keys(void* map_ptr) {
    if (!map_ptr) {
        return NULL;
    }
    
    // 分配结果结构体
    MapIterResult* result = (MapIterResult*)malloc(sizeof(MapIterResult));
    if (!result) {
        return NULL;
    }
    
    // 初始化
    result->keys = NULL;
    result->values = NULL;
    result->count = 0;
    
    // 将map_ptr转换为hash_table_t指针
    // 注意：hash_table_t定义在runtime/shared/types/common_types.h中
    // 由于runtime_api.c可能不直接包含该头文件，我们需要使用前向声明
    // 或者直接使用void*指针进行类型转换
    
    // 前向声明hash_table_t和hash_entry_t结构
    // 注意：这些结构定义在runtime/shared/types/common_types.h中
    // string_t是结构体：{ char* data; size_t length; size_t capacity; }
    typedef struct {
        char* data;
        size_t length;
        size_t capacity;
    } string_t;
    
    typedef struct hash_entry_internal {
        string_t key;          // string_t key
        void* value;          // void* value（对于map[string]string，value是string_t*）
        struct hash_entry_internal* next;
    } hash_entry_internal_t;
    
    typedef struct {
        hash_entry_internal_t** buckets;  // hash_entry_t** buckets
        size_t bucket_count;              // size_t bucket_count
        size_t entry_count;               // size_t entry_count
        void* hash_function;              // uint32_t (*hash_function)(const char* key)
    } hash_table_internal_t;
    
    hash_table_internal_t* map = (hash_table_internal_t*)map_ptr;
    
    // 如果map为空，返回空结果
    if (!map || map->entry_count == 0) {
        return result;
    }
    
    // 分配键数组和值数组
    // 注意：键和值都是char*（string类型）
    result->keys = (char**)malloc(sizeof(char*) * map->entry_count);
    result->values = (char**)malloc(sizeof(char*) * map->entry_count);
    if (!result->keys || !result->values) {
        if (result->keys) free(result->keys);
        if (result->values) free(result->values);
        free(result);
        return NULL;
    }
    
    // 遍历所有bucket和entry，收集键值对
    size_t index = 0;
    for (size_t i = 0; i < map->bucket_count; i++) {
        if (!map->buckets) break;
        hash_entry_internal_t* entry = map->buckets[i];
        while (entry) {
            if (index >= map->entry_count) {
                // 防止数组越界
                break;
            }
            
            // 复制键（string_t.data是char*）
            if (entry->key.data && entry->key.length > 0) {
                size_t key_len = entry->key.length + 1;
                result->keys[index] = (char*)malloc(key_len);
                if (result->keys[index]) {
                    memcpy(result->keys[index], entry->key.data, entry->key.length);
                    result->keys[index][entry->key.length] = '\0';
                }
            } else {
                result->keys[index] = NULL;
            }
            
            // 复制值（对于map[string]string，value是string_t*）
            if (entry->value) {
                string_t* value_str = (string_t*)entry->value;
                if (value_str->data && value_str->length > 0) {
                    size_t value_len = value_str->length + 1;
                    result->values[index] = (char*)malloc(value_len);
                    if (result->values[index]) {
                        memcpy(result->values[index], value_str->data, value_str->length);
                        result->values[index][value_str->length] = '\0';
                    }
                } else {
                    result->values[index] = NULL;
                }
            } else {
                result->values[index] = NULL;
            }
            
            index++;
            entry = entry->next;
        }
    }
    
    result->count = (int32_t)index;
    
    return result;
}

// ==================== Map索引赋值支持 ====================

// 前向声明hash_table_t和hash_entry_t结构（用于map操作）
// 注意：这些结构定义在runtime/shared/types/common_types.h中
typedef struct {
    char* data;
    size_t length;
    size_t capacity;
} string_t_internal;

typedef struct hash_entry_internal {
    string_t_internal key;          // string_t key
    void* value;                    // void* value（对于map[string]string，value是string_t*）
    struct hash_entry_internal* next;
} hash_entry_internal_t;

typedef struct {
    hash_entry_internal_t** buckets;  // hash_entry_t** buckets
    size_t bucket_count;              // size_t bucket_count
    size_t entry_count;               // size_t entry_count
    void* hash_function;              // uint32_t (*hash_function)(const char* key)
} hash_table_internal_t;

// 辅助函数：从C字符串创建string_t_internal
static string_t_internal* create_string_from_cstr(const char* str) {
    if (!str) return NULL;
    
    size_t len = strlen(str);
    string_t_internal* s = (string_t_internal*)malloc(sizeof(string_t_internal));
    if (!s) return NULL;
    
    s->data = (char*)malloc(len + 1);
    if (!s->data) {
        free(s);
        return NULL;
    }
    
    memcpy(s->data, str, len);
    s->data[len] = '\0';
    s->length = len;
    s->capacity = len + 1;
    
    return s;
}

// 辅助函数：比较string_t_internal和C字符串
static int string_equals_internal(const string_t_internal* a, const char* b) {
    if (!a || !b) return 0;
    if (!a->data) return 0;
    size_t b_len = strlen(b);
    if (a->length != b_len) return 0;
    return memcmp(a->data, b, a->length) == 0;
}

// 辅助函数：比较32位整数
static inline int int32_equals_internal(int32_t a, int32_t b) {
    return a == b;
}

// 辅助函数：比较64位整数
static inline int int64_equals_internal(int64_t a, int64_t b) {
    return a == b;
}

/**
 * @brief 浮点数比较函数（内部使用）
 * @param a 第一个浮点数
 * @param b 第二个浮点数
 * @return 1 如果相等，0 如果不相等
 * 
 * 注意：处理特殊值（NaN、±0.0）
 */
static inline int float_equals_internal(double a, double b) {
    // 处理NaN：NaN != NaN，但为了Map键比较，我们让所有NaN相等
    if (isnan(a) && isnan(b)) {
        return 1; // 所有NaN视为相等
    }
    if (isnan(a) || isnan(b)) {
        return 0; // 一个NaN，一个非NaN，不相等
    }
    
    // 处理±0.0：正零和负零应该相等
    if (a == 0.0 && b == 0.0) {
        return 1; // ±0.0视为相等
    }
    
    // 普通浮点数比较（使用精确比较，因为哈希值已经处理了±0.0）
    return a == b;
}

/**
 * @brief 释放string_t_internal结构体（内部使用）
 * @param s 字符串结构体指针
 */
static inline void free_string_internal(string_t_internal* s) {
    if (!s) return;
    if (s->data) {
        free(s->data);
    }
    free(s);
}

/**
 * @brief 扩容Map（重新哈希所有条目）
 * @param map Map指针
 * 
 * 实现：将Map的bucket数量扩容为原来的2倍，并重新哈希所有条目
 * 注意：
 * - 负载因子阈值：0.75
 * - 扩容策略：新容量 = 旧容量 × 2
 * - 重新哈希：遍历所有条目，重新计算哈希值并插入到新的buckets
 */
static void runtime_map_resize_internal(hash_table_internal_t* map) {
    if (!map || !map->buckets || map->bucket_count == 0) {
        return;
    }
    
    // 1. 保存所有旧的条目（收集所有entry到一个临时数组）
    size_t old_entry_count = map->entry_count;
    if (old_entry_count == 0) {
        return; // 没有条目，无需扩容
    }
    
    // 分配临时数组保存所有条目
    hash_entry_internal_t** old_entries = (hash_entry_internal_t**)malloc(
        sizeof(hash_entry_internal_t*) * old_entry_count);
    if (!old_entries) {
        return; // 内存分配失败，无法扩容
    }
    
    // 收集所有条目
    size_t entry_index = 0;
    for (size_t i = 0; i < map->bucket_count; i++) {
        hash_entry_internal_t* entry = map->buckets[i];
        while (entry) {
            old_entries[entry_index++] = entry;
            entry = entry->next;
        }
    }
    
    // 2. 计算新的bucket数量（扩容为原来的2倍）
    size_t new_bucket_count = map->bucket_count * 2;
    if (new_bucket_count == 0) {
        new_bucket_count = 16; // 防止溢出
    }
    
    // 3. 分配新的buckets数组
    hash_entry_internal_t** new_buckets = (hash_entry_internal_t**)calloc(
        new_bucket_count, sizeof(hash_entry_internal_t*));
    if (!new_buckets) {
        free(old_entries);
        return; // 内存分配失败，无法扩容
    }
    
    // 4. 保存旧的buckets数组（稍后释放）
    hash_entry_internal_t** old_buckets = map->buckets;
    size_t old_bucket_count = map->bucket_count;
    
    // 5. 更新map的buckets和bucket_count
    map->buckets = new_buckets;
    map->bucket_count = new_bucket_count;
    // 注意：entry_count保持不变，因为我们只是重新哈希，没有添加或删除条目
    
    // 6. 重新哈希所有条目到新的buckets
    for (size_t i = 0; i < old_entry_count; i++) {
        hash_entry_internal_t* entry = old_entries[i];
        
        // 重新计算哈希值（使用entry->key.data作为C字符串）
        int64_t hash64 = runtime_string_hash(entry->key.data);
        uint32_t hash = (uint32_t)(hash64 & 0xFFFFFFFF); // 取低32位
        size_t new_bucket_index = hash % new_bucket_count;
        
        // 断开旧链表连接
        entry->next = NULL;
        
        // 插入到新bucket的链表头部
        if (new_buckets[new_bucket_index]) {
            entry->next = new_buckets[new_bucket_index];
        }
        new_buckets[new_bucket_index] = entry;
    }
    
    // 7. 释放临时数组和旧的buckets数组
    free(old_entries);
    free(old_buckets);
}

/**
 * @brief 设置 map 中的键值对
 * @param map_ptr map指针（i8*，不透明指针）
 * @param key 键（char*，字符串）
 * @param value 值（char*，字符串）
 * @return void
 * 
 * 编译器签名：void runtime_map_set(i8* map_ptr, i8* key, i8* value)
 * 使用位置：编译器生成的map索引赋值代码（headers[key] = value）
 * 
 * 实现：在map中插入或更新键值对
 * 注意：
 * - 如果key已存在，更新值
 * - 如果key不存在，插入新条目
 * - 对于map[string]string，key和value都是char*（string类型）
 * - 需要处理哈希冲突（使用链表）
 */
void runtime_map_set(void* map_ptr, void* key, void* value) {
    // NULL检查
    if (!map_ptr || !key) {
        return;
    }
    
    // 类型转换：map_ptr -> hash_table_internal_t*
    hash_table_internal_t* map = (hash_table_internal_t*)map_ptr;
    
    // 如果map未初始化，初始化（分配buckets）
    if (!map->buckets || map->bucket_count == 0) {
        // 初始化默认容量（16个bucket）
        map->bucket_count = 16;
        map->buckets = (hash_entry_internal_t**)calloc(16, sizeof(hash_entry_internal_t*));
        if (!map->buckets) {
            return; // 内存分配失败
        }
        map->entry_count = 0;
        // 注意：hash_function字段类型是uint32_t(*)(const char*)，但runtime_string_hash返回int64_t
        // 这里存储NULL，我们直接使用runtime_string_hash函数
        map->hash_function = NULL;
    }
    
    // 计算key的哈希值
    // 注意：runtime_string_hash返回int64_t，但hash_table_t的hash_function是uint32_t
    // 我们需要使用runtime_string_hash并转换为uint32_t
    int64_t hash64 = runtime_string_hash((const char*)key);
    uint32_t hash = (uint32_t)(hash64 & 0xFFFFFFFF); // 取低32位
    size_t bucket_index = hash % map->bucket_count;
    
    // 在bucket中查找是否已存在该key
    hash_entry_internal_t* entry = map->buckets[bucket_index];
    hash_entry_internal_t* prev = NULL;
    
    while (entry) {
        // 比较key（string_t比较）
        if (string_equals_internal(&entry->key, (const char*)key)) {
            // key已存在，更新value
            // 释放旧value（如果是string_t*）
            if (entry->value) {
                string_t_internal* old_value = (string_t_internal*)entry->value;
                if (old_value->data) {
                    free(old_value->data);
                }
                free(old_value);
            }
            // 创建新的value（string_t*）
            string_t_internal* new_value = create_string_from_cstr((const char*)value);
            if (!new_value) {
                return; // 内存分配失败
            }
            entry->value = new_value;
            return;
        }
        prev = entry;
        entry = entry->next;
    }
    
    // key不存在，创建新条目
    hash_entry_internal_t* new_entry = (hash_entry_internal_t*)malloc(sizeof(hash_entry_internal_t));
    if (!new_entry) {
        return; // 内存分配失败
    }
    
    string_t_internal* key_str = create_string_from_cstr((const char*)key);
    if (!key_str) {
        free(new_entry);
        return; // 内存分配失败
    }
    new_entry->key = *key_str;
    free(key_str); // 只需要结构体内容，不需要指针
    
    new_entry->value = create_string_from_cstr((const char*)value);
    if (!new_entry->value) {
        free(new_entry->key.data);
        free(new_entry);
        return; // 内存分配失败
    }
    
    new_entry->next = NULL;
    
    // 插入到bucket链表头部
    if (prev) {
        prev->next = new_entry;
    } else {
        map->buckets[bucket_index] = new_entry;
    }
    
    map->entry_count++;
    
    // 负载因子检查和自动扩容
    // 负载因子 = 条目数 / bucket数量
    // 当负载因子超过0.75时，触发扩容（扩容为原来的2倍）
    if (map->bucket_count > 0) {
        float load_factor = (float)map->entry_count / (float)map->bucket_count;
        if (load_factor > 0.75f) {
            runtime_map_resize_internal(map);
        }
    }
}

/**
 * @brief 删除 map 中的键值对
 * @param map_ptr map指针（i8*，不透明指针）
 * @param key 键（char*，字符串）
 * @return void
 * 
 * 编译器签名：void runtime_map_delete(i8* map_ptr, i8* key)
 * 使用位置：编译器生成的delete语句代码（delete(map, key)）
 * 
 * 实现：从map中删除指定的键值对
 * 注意：
 * - 如果key存在，删除对应的条目并释放内存
 * - 如果key不存在，静默忽略（不报错）
 * - 对于map[string]string，key是char*（string类型）
 * - 需要处理哈希冲突（使用链表）
 */
void runtime_map_delete(void* map_ptr, void* key) {
    // NULL检查
    if (!map_ptr || !key) {
        return;
    }
    
    // 类型转换：map_ptr -> hash_table_internal_t*
    hash_table_internal_t* map = (hash_table_internal_t*)map_ptr;
    
    // 如果map未初始化或为空，直接返回（静默忽略）
    if (!map->buckets || map->bucket_count == 0 || map->entry_count == 0) {
        return;
    }
    
    // 计算key的哈希值
    int64_t hash64 = runtime_string_hash((const char*)key);
    uint32_t hash = (uint32_t)(hash64 & 0xFFFFFFFF); // 取低32位
    size_t bucket_index = hash % map->bucket_count;
    
    // 在bucket的链表中查找并删除匹配的条目
    hash_entry_internal_t* entry = map->buckets[bucket_index];
    hash_entry_internal_t* prev = NULL;
    
    while (entry) {
        // 比较key（string_t比较）
        if (string_equals_internal(&entry->key, (const char*)key)) {
            // 找到匹配的条目，删除它
            if (prev == NULL) {
                // 是链表的第一个节点
                map->buckets[bucket_index] = entry->next;
            } else {
                // 是链表的中间或末尾节点
                prev->next = entry->next;
            }
            
            // 释放内存
            // 释放key的string_t_internal（key是string_t结构体，不是指针，但data字段需要释放）
            if (entry->key.data) {
                free(entry->key.data);
            }
            
            // 释放value的string_t_internal（value是string_t*指针）
            if (entry->value) {
                string_t_internal* value_str = (string_t_internal*)entry->value;
                if (value_str->data) {
                    free(value_str->data);
                }
                free(value_str);
            }
            
            // 释放entry本身
            free(entry);
            
            // 更新map大小
            map->entry_count--;
            
            // 删除完成，返回
            return;
        }
        
        prev = entry;
        entry = entry->next;
    }
    
    // 如果没有找到匹配的条目，静默返回（不报错）
    // 这是符合Go语言delete(map, key)的行为
}

/**
 * @brief 获取 map[string]string 中的值
 * 编译器签名：i8* runtime_map_get_string_string(i8* map_ptr, i8* key)
 */
void* runtime_map_get_string_string(void* map_ptr, void* key) {
    if (!map_ptr || !key) {
        return NULL;
    }
    hash_table_internal_t* map = (hash_table_internal_t*)map_ptr;
    if (!map->buckets || map->bucket_count == 0) {
        return NULL;
    }
    int64_t hash64 = runtime_string_hash((const char*)key);
    uint32_t hash = (uint32_t)(hash64 & 0xFFFFFFFF);
    size_t bucket_index = hash % map->bucket_count;
    hash_entry_internal_t* entry = map->buckets[bucket_index];
    while (entry) {
        if (string_equals_internal(&entry->key, (const char*)key)) {
            return entry->value;
        }
        entry = entry->next;
    }
    return NULL;
}

/**
 * @brief 设置 map[string]string 中的键值对（与 runtime_map_set 同义）
 * 编译器签名：void runtime_map_set_string_string(i8* map_ptr, i8* key, i8* value)
 */
void runtime_map_set_string_string(void* map_ptr, void* key, void* value) {
    runtime_map_set(map_ptr, key, value);
}

// ==================== Map多类型支持：map[int]string ====================

// 前向声明通用哈希表结构（用于map[int]string等）
typedef struct {
    hash_entry_generic_t** buckets;
    size_t bucket_count;
    size_t entry_count;
    map_key_type_t key_type;
    map_value_type_t value_type;
    
    // ✅ 新增：表级别的函数指针（所有条目共享，用于结构体键）
    map_key_hash_func_t key_hash_func;      // 哈希函数指针（结构体键使用）
    map_key_equals_func_t key_equals_func;  // 比较函数指针（结构体键使用）
} hash_table_generic_internal_t;

// 前向声明：扩容函数
static void runtime_map_resize_internal_int_string(hash_table_generic_internal_t* map);

/**
 * @brief 设置 map[int]string 中的键值对
 * @param map_ptr map指针（i8*，不透明指针）
 * @param key 键（i32，整数）
 * @param value 值（i8*，字符串）
 * @return void
 * 
 * 编译器签名：void runtime_map_set_int_string(i8* map_ptr, i32 key, i8* value)
 * 使用位置：编译器生成的map索引赋值代码（map[key] = value，其中map是map[int]string类型）
 * 
 * 实现：在map中插入或更新键值对
 * 注意：
 * - 如果key已存在，更新值
 * - 如果key不存在，插入新条目
 * - 对于map[int]string，key是i32，value是char*（string类型）
 * - 需要处理哈希冲突（使用链表）
 */
void runtime_map_set_int_string(void* map_ptr, int32_t key, void* value) {
    // NULL检查
    if (!map_ptr) {
        return;
    }
    
    // 类型转换：map_ptr -> hash_table_generic_internal_t*
    hash_table_generic_internal_t* map = (hash_table_generic_internal_t*)map_ptr;
    
    // 如果map未初始化，初始化（分配buckets）
    if (!map->buckets || map->bucket_count == 0) {
        // 初始化默认容量（16个bucket）
        map->bucket_count = 16;
        map->buckets = (hash_entry_generic_t**)calloc(16, sizeof(hash_entry_generic_t*));
        if (!map->buckets) {
            return; // 内存分配失败
        }
        map->entry_count = 0;
        map->key_type = MAP_KEY_TYPE_INT32;
        map->value_type = MAP_VALUE_TYPE_STRING;
    }
    
    // 计算key的哈希值（使用整数哈希函数）
    int64_t hash64 = runtime_int32_hash(key);
    uint32_t hash = (uint32_t)(hash64 & 0xFFFFFFFF); // 取低32位
    size_t bucket_index = hash % map->bucket_count;
    
    // 在bucket中查找是否已存在该key
    hash_entry_generic_t* entry = map->buckets[bucket_index];
    hash_entry_generic_t* prev = NULL;
    
    while (entry) {
        // 比较key（整数比较）
        if (entry->key_type == MAP_KEY_TYPE_INT32 && entry->key) {
            int32_t* entry_key = (int32_t*)entry->key;
            if (int32_equals_internal(*entry_key, key)) {
                // key已存在，更新value
                // 释放旧value（如果是string_t*）
                if (entry->value) {
                    string_t_internal* old_value = (string_t_internal*)entry->value;
                    if (old_value && old_value->data) {
                        free(old_value->data);
                    }
                    if (old_value) {
                        free(old_value);
                    }
                }
                // 创建新的value（string_t*）
                string_t_internal* new_value = create_string_from_cstr((const char*)value);
                if (!new_value) {
                    return; // 内存分配失败
                }
                entry->value = new_value;
                entry->value_type = MAP_VALUE_TYPE_STRING;
                entry->value_size = sizeof(string_t_internal);
                return;
            }
        }
        prev = entry;
        entry = entry->next;
    }
    
    // key不存在，创建新条目
    hash_entry_generic_t* new_entry = (hash_entry_generic_t*)malloc(sizeof(hash_entry_generic_t));
    if (!new_entry) {
        return; // 内存分配失败
    }
    
    // 存储整数键（分配内存存储int32_t）
    int32_t* key_ptr = (int32_t*)malloc(sizeof(int32_t));
    if (!key_ptr) {
        free(new_entry);
        return; // 内存分配失败
    }
    *key_ptr = key;
    new_entry->key = (void*)key_ptr;
    new_entry->key_type = MAP_KEY_TYPE_INT32;
    new_entry->key_size = sizeof(int32_t);
    
    // 创建字符串值
    new_entry->value = create_string_from_cstr((const char*)value);
    if (!new_entry->value) {
        free(key_ptr);
        free(new_entry);
        return; // 内存分配失败
    }
    new_entry->value_type = MAP_VALUE_TYPE_STRING;
    new_entry->value_size = sizeof(string_t_internal);
    
    new_entry->next = NULL;
    
    // 插入到bucket链表头部
    if (prev) {
        prev->next = new_entry;
    } else {
        map->buckets[bucket_index] = new_entry;
    }
    
    map->entry_count++;
    
    // 负载因子检查和自动扩容
    if (map->bucket_count > 0) {
        float load_factor = (float)map->entry_count / (float)map->bucket_count;
        if (load_factor > 0.75f) {
            runtime_map_resize_internal_int_string(map);
        }
    }
}

/**
 * @brief 获取 map[int]string 中的值
 * @param map_ptr map指针（i8*，不透明指针）
 * @param key 键（i32，整数）
 * @return 值（i8*，字符串指针），如果不存在返回NULL
 * 
 * 编译器签名：i8* runtime_map_get_int_string(i8* map_ptr, i32 key)
 * 使用位置：编译器生成的map索引访问代码（map[key]，其中map是map[int]string类型）
 */
void* runtime_map_get_int_string(void* map_ptr, int32_t key) {
    // NULL检查
    if (!map_ptr) {
        return NULL;
    }
    
    // 类型转换：map_ptr -> hash_table_generic_internal_t*
    hash_table_generic_internal_t* map = (hash_table_generic_internal_t*)map_ptr;
    
    // 如果map未初始化或为空，返回NULL
    if (!map->buckets || map->bucket_count == 0 || map->entry_count == 0) {
        return NULL;
    }
    
    // 计算key的哈希值
    int64_t hash64 = runtime_int32_hash(key);
    uint32_t hash = (uint32_t)(hash64 & 0xFFFFFFFF); // 取低32位
    size_t bucket_index = hash % map->bucket_count;
    
    // 在bucket的链表中查找匹配的条目
    hash_entry_generic_t* entry = map->buckets[bucket_index];
    while (entry) {
        // 比较key（整数比较）
        if (entry->key_type == MAP_KEY_TYPE_INT32 && entry->key) {
            int32_t* entry_key = (int32_t*)entry->key;
            if (int32_equals_internal(*entry_key, key)) {
                // 找到匹配的条目，返回value（string_t*）
                if (entry->value && entry->value_type == MAP_VALUE_TYPE_STRING) {
                    string_t_internal* value_str = (string_t_internal*)entry->value;
                    // 返回字符串的data字段（char*）
                    return value_str ? value_str->data : NULL;
                }
                return NULL;
            }
        }
        entry = entry->next;
    }
    
    // 没有找到匹配的条目，返回NULL
    return NULL;
}

/**
 * @brief 删除 map[int]string 中的键值对
 * @param map_ptr map指针（i8*，不透明指针）
 * @param key 键（i32，整数）
 * @return void
 * 
 * 编译器签名：void runtime_map_delete_int_string(i8* map_ptr, i32 key)
 * 使用位置：编译器生成的delete语句代码（delete(map, key)，其中map是map[int]string类型）
 */
void runtime_map_delete_int_string(void* map_ptr, int32_t key) {
    // NULL检查
    if (!map_ptr) {
        return;
    }
    
    // 类型转换：map_ptr -> hash_table_generic_internal_t*
    hash_table_generic_internal_t* map = (hash_table_generic_internal_t*)map_ptr;
    
    // 如果map未初始化或为空，直接返回（静默忽略）
    if (!map->buckets || map->bucket_count == 0 || map->entry_count == 0) {
        return;
    }
    
    // 计算key的哈希值
    int64_t hash64 = runtime_int32_hash(key);
    uint32_t hash = (uint32_t)(hash64 & 0xFFFFFFFF); // 取低32位
    size_t bucket_index = hash % map->bucket_count;
    
    // 在bucket的链表中查找并删除匹配的条目
    hash_entry_generic_t* entry = map->buckets[bucket_index];
    hash_entry_generic_t* prev = NULL;
    
    while (entry) {
        // 比较key（整数比较）
        if (entry->key_type == MAP_KEY_TYPE_INT32 && entry->key) {
            int32_t* entry_key = (int32_t*)entry->key;
            if (int32_equals_internal(*entry_key, key)) {
                // 找到匹配的条目，删除它
                if (prev == NULL) {
                    // 是链表的第一个节点
                    map->buckets[bucket_index] = entry->next;
                } else {
                    // 是链表的中间或末尾节点
                    prev->next = entry->next;
                }
                
                // 释放内存
                // 释放key（整数键直接释放）
                if (entry->key) {
                    free(entry->key);
                }
                
                // 释放value（字符串值需要释放string_t结构体和data字段）
                if (entry->value && entry->value_type == MAP_VALUE_TYPE_STRING) {
                    string_t_internal* value_str = (string_t_internal*)entry->value;
                    if (value_str && value_str->data) {
                        free(value_str->data);
                    }
                    if (value_str) {
                        free(value_str);
                    }
                }
                
                // 释放entry本身
                free(entry);
                
                // 更新map大小
                map->entry_count--;
                
                // 删除完成，返回
                return;
            }
        }
        
        prev = entry;
        entry = entry->next;
    }
    
    // 如果没有找到匹配的条目，静默返回（不报错）
}

/**
 * @brief 扩容 map[int]string（重新哈希所有条目）
 * @param map Map指针
 * 
 * 实现：将Map的bucket数量扩容为原来的2倍，并重新哈希所有条目
 * 注意：这是map[int]string专用的扩容函数
 */
static void runtime_map_resize_internal_int_string(hash_table_generic_internal_t* map) {
    if (!map || !map->buckets || map->bucket_count == 0) {
        return;
    }
    
    // 1. 保存所有旧的条目
    size_t old_entry_count = map->entry_count;
    if (old_entry_count == 0) {
        return; // 没有条目，无需扩容
    }
    
    // 分配临时数组保存所有条目
    hash_entry_generic_t** old_entries = (hash_entry_generic_t**)malloc(
        sizeof(hash_entry_generic_t*) * old_entry_count);
    if (!old_entries) {
        return; // 内存分配失败，无法扩容
    }
    
    // 收集所有条目
    size_t entry_index = 0;
    for (size_t i = 0; i < map->bucket_count; i++) {
        hash_entry_generic_t* entry = map->buckets[i];
        while (entry) {
            old_entries[entry_index++] = entry;
            entry = entry->next;
        }
    }
    
    // 2. 计算新的bucket数量（扩容为原来的2倍）
    size_t new_bucket_count = map->bucket_count * 2;
    if (new_bucket_count == 0) {
        new_bucket_count = 16; // 防止溢出
    }
    
    // 3. 分配新的buckets数组
    hash_entry_generic_t** new_buckets = (hash_entry_generic_t**)calloc(
        new_bucket_count, sizeof(hash_entry_generic_t*));
    if (!new_buckets) {
        free(old_entries);
        return; // 内存分配失败，无法扩容
    }
    
    // 4. 保存旧的buckets数组（稍后释放）
    hash_entry_generic_t** old_buckets = map->buckets;
    size_t old_bucket_count = map->bucket_count;
    
    // 5. 更新map的buckets和bucket_count
    map->buckets = new_buckets;
    map->bucket_count = new_bucket_count;
    // 注意：entry_count保持不变，因为我们只是重新哈希，没有添加或删除条目
    
    // 6. 重新哈希所有条目到新的buckets
    for (size_t i = 0; i < old_entry_count; i++) {
        hash_entry_generic_t* entry = old_entries[i];
        
        // 重新计算哈希值（使用整数哈希函数）
        if (entry->key_type == MAP_KEY_TYPE_INT32 && entry->key) {
            int32_t* entry_key = (int32_t*)entry->key;
            int64_t hash64 = runtime_int32_hash(*entry_key);
            uint32_t hash = (uint32_t)(hash64 & 0xFFFFFFFF); // 取低32位
            size_t new_bucket_index = hash % new_bucket_count;
            
            // 断开旧链表连接
            entry->next = NULL;
            
            // 插入到新bucket的链表头部
            if (new_buckets[new_bucket_index]) {
                entry->next = new_buckets[new_bucket_index];
            }
            new_buckets[new_bucket_index] = entry;
        }
    }
    
    // 7. 释放临时数组和旧的buckets数组
    free(old_entries);
    free(old_buckets);
}

// ============================================================================
// Map多类型支持：map[string]int
// ============================================================================

// 前向声明
static void runtime_map_resize_internal_string_int(hash_table_generic_internal_t* map);

/**
 * @brief 设置 map[string]int 中的键值对
 * @param map_ptr map指针（i8*，不透明指针）
 * @param key 键（i8*，字符串）
 * @param value 值（i32，整数）
 * @return void
 * 
 * 编译器签名：void runtime_map_set_string_int(i8* map_ptr, i8* key, i32 value)
 * 使用位置：编译器生成的map索引赋值代码（map[key] = value，其中map是map[string]int类型）
 * 
 * 实现：在map中插入或更新键值对
 * 注意：
 * - 如果key已存在，更新值
 * - 如果key不存在，插入新条目
 * - 对于map[string]int，key是char*（string类型），value是i32（整数类型）
 * - 需要处理哈希冲突（使用链表）
 */
void runtime_map_set_string_int(void* map_ptr, void* key, int32_t value) {
    // NULL检查
    if (!map_ptr || !key) {
        return;
    }
    
    // 类型转换：map_ptr -> hash_table_generic_internal_t*
    hash_table_generic_internal_t* map = (hash_table_generic_internal_t*)map_ptr;
    
    // 如果map未初始化，初始化（分配buckets）
    if (!map->buckets || map->bucket_count == 0) {
        // 初始化默认容量（16个bucket）
        map->bucket_count = 16;
        map->buckets = (hash_entry_generic_t**)calloc(16, sizeof(hash_entry_generic_t*));
        if (!map->buckets) {
            return; // 内存分配失败
        }
        map->entry_count = 0;
        map->key_type = MAP_KEY_TYPE_STRING;
        map->value_type = MAP_VALUE_TYPE_INT32;
    }
    
    // 计算key的哈希值（使用字符串哈希函数）
    int64_t hash64 = runtime_string_hash((const char*)key);
    uint32_t hash = (uint32_t)(hash64 & 0xFFFFFFFF); // 取低32位
    size_t bucket_index = hash % map->bucket_count;
    
    // 在bucket中查找是否已存在该key
    hash_entry_generic_t* entry = map->buckets[bucket_index];
    hash_entry_generic_t* prev = NULL;
    
    while (entry) {
        // 比较key（字符串比较）
        if (entry->key_type == MAP_KEY_TYPE_STRING && entry->key) {
            string_t_internal* entry_key = (string_t_internal*)entry->key;
            if (string_equals_internal(entry_key, (const char*)key)) {
                // key已存在，更新value
                // 释放旧value（如果是int32_t*）
                if (entry->value) {
                    free(entry->value);
                }
                // 创建新的value（int32_t*）
                int32_t* new_value = (int32_t*)malloc(sizeof(int32_t));
                if (!new_value) {
                    return; // 内存分配失败
                }
                *new_value = value;
                entry->value = new_value;
                entry->value_type = MAP_VALUE_TYPE_INT32;
                entry->value_size = sizeof(int32_t);
                return;
            }
        }
        prev = entry;
        entry = entry->next;
    }
    
    // key不存在，创建新条目
    hash_entry_generic_t* new_entry = (hash_entry_generic_t*)malloc(sizeof(hash_entry_generic_t));
    if (!new_entry) {
        return; // 内存分配失败
    }
    
    // 存储字符串键（创建string_t结构体）
    string_t_internal* key_str = create_string_from_cstr((const char*)key);
    if (!key_str) {
        free(new_entry);
        return; // 内存分配失败
    }
    new_entry->key = (void*)key_str;
    new_entry->key_type = MAP_KEY_TYPE_STRING;
    new_entry->key_size = sizeof(string_t_internal);
    
    // 存储整数值（分配内存存储int32_t）
    int32_t* value_ptr = (int32_t*)malloc(sizeof(int32_t));
    if (!value_ptr) {
        // 释放已分配的key
        if (key_str && key_str->data) {
            free(key_str->data);
        }
        if (key_str) {
            free(key_str);
        }
        free(new_entry);
        return; // 内存分配失败
    }
    *value_ptr = value;
    new_entry->value = (void*)value_ptr;
    new_entry->value_type = MAP_VALUE_TYPE_INT32;
    new_entry->value_size = sizeof(int32_t);
    
    new_entry->next = NULL;
    
    // 插入到bucket链表头部
    if (prev) {
        prev->next = new_entry;
    } else {
        map->buckets[bucket_index] = new_entry;
    }
    
    map->entry_count++;
    
    // 负载因子检查和自动扩容
    if (map->bucket_count > 0) {
        float load_factor = (float)map->entry_count / (float)map->bucket_count;
        if (load_factor > 0.75f) {
            runtime_map_resize_internal_string_int(map);
        }
    }
}

/**
 * @brief 获取 map[string]int 中的值
 * @param map_ptr map指针（i8*，不透明指针）
 * @param key 键（i8*，字符串）
 * @return 值（i32*，整数指针），如果不存在返回NULL
 * 
 * 编译器签名：i32* runtime_map_get_string_int(i8* map_ptr, i8* key)
 * 使用位置：编译器生成的map索引访问代码（map[key]，其中map是map[string]int类型）
 * 
 * 注意：返回int32_t*指针，调用者需要解引用获取值。如果key不存在，返回NULL。
 */
int32_t* runtime_map_get_string_int(void* map_ptr, void* key) {
    // NULL检查
    if (!map_ptr || !key) {
        return NULL;
    }
    
    // 类型转换：map_ptr -> hash_table_generic_internal_t*
    hash_table_generic_internal_t* map = (hash_table_generic_internal_t*)map_ptr;
    
    // 如果map未初始化或为空，返回NULL
    if (!map->buckets || map->bucket_count == 0 || map->entry_count == 0) {
        return NULL;
    }
    
    // 计算key的哈希值
    int64_t hash64 = runtime_string_hash((const char*)key);
    uint32_t hash = (uint32_t)(hash64 & 0xFFFFFFFF); // 取低32位
    size_t bucket_index = hash % map->bucket_count;
    
    // 在bucket的链表中查找匹配的条目
    hash_entry_generic_t* entry = map->buckets[bucket_index];
    while (entry) {
        // 比较key（字符串比较）
        if (entry->key_type == MAP_KEY_TYPE_STRING && entry->key) {
            string_t_internal* entry_key = (string_t_internal*)entry->key;
            if (string_equals_internal(entry_key, (const char*)key)) {
                // 找到匹配的条目，返回value（int32_t*）
                if (entry->value && entry->value_type == MAP_VALUE_TYPE_INT32) {
                    return (int32_t*)entry->value;
                }
                return NULL;
            }
        }
        entry = entry->next;
    }
    
    // 没有找到匹配的条目，返回NULL
    return NULL;
}

/**
 * @brief 删除 map[string]int 中的键值对
 * @param map_ptr map指针（i8*，不透明指针）
 * @param key 键（i8*，字符串）
 * @return void
 * 
 * 编译器签名：void runtime_map_delete_string_int(i8* map_ptr, i8* key)
 * 使用位置：编译器生成的delete语句代码（delete(map, key)，其中map是map[string]int类型）
 */
void runtime_map_delete_string_int(void* map_ptr, void* key) {
    // NULL检查
    if (!map_ptr || !key) {
        return;
    }
    
    // 类型转换：map_ptr -> hash_table_generic_internal_t*
    hash_table_generic_internal_t* map = (hash_table_generic_internal_t*)map_ptr;
    
    // 如果map未初始化或为空，直接返回（静默忽略）
    if (!map->buckets || map->bucket_count == 0 || map->entry_count == 0) {
        return;
    }
    
    // 计算key的哈希值
    int64_t hash64 = runtime_string_hash((const char*)key);
    uint32_t hash = (uint32_t)(hash64 & 0xFFFFFFFF); // 取低32位
    size_t bucket_index = hash % map->bucket_count;
    
    // 在bucket的链表中查找并删除匹配的条目
    hash_entry_generic_t* entry = map->buckets[bucket_index];
    hash_entry_generic_t* prev = NULL;
    
    while (entry) {
        // 比较key（字符串比较）
        if (entry->key_type == MAP_KEY_TYPE_STRING && entry->key) {
            string_t_internal* entry_key = (string_t_internal*)entry->key;
            if (string_equals_internal(entry_key, (const char*)key)) {
                // 找到匹配的条目，删除它
                if (prev == NULL) {
                    // 是链表的第一个节点
                    map->buckets[bucket_index] = entry->next;
                } else {
                    // 是链表的中间或末尾节点
                    prev->next = entry->next;
                }
                
                // 释放内存
                // 释放key（字符串键需要释放string_t结构体和data字段）
                if (entry->key) {
                    string_t_internal* key_str = (string_t_internal*)entry->key;
                    if (key_str && key_str->data) {
                        free(key_str->data);
                    }
                    if (key_str) {
                        free(key_str);
                    }
                }
                
                // 释放value（整数值直接释放）
                if (entry->value) {
                    free(entry->value);
                }
                
                // 释放entry本身
                free(entry);
                
                // 更新map大小
                map->entry_count--;
                
                // 删除完成，返回
                return;
            }
        }
        
        prev = entry;
        entry = entry->next;
    }
    
    // 如果没有找到匹配的条目，静默返回（不报错）
}

/**
 * @brief 扩容 map[string]int（重新哈希所有条目）
 * @param map Map指针
 * 
 * 实现：将Map的bucket数量扩容为原来的2倍，并重新哈希所有条目
 * 注意：这是map[string]int专用的扩容函数
 */
static void runtime_map_resize_internal_string_int(hash_table_generic_internal_t* map) {
    if (!map || !map->buckets || map->bucket_count == 0) {
        return;
    }
    
    // 1. 保存所有旧的条目
    size_t old_entry_count = map->entry_count;
    if (old_entry_count == 0) {
        return; // 没有条目，无需扩容
    }
    
    // 分配临时数组保存所有条目
    hash_entry_generic_t** old_entries = (hash_entry_generic_t**)malloc(
        sizeof(hash_entry_generic_t*) * old_entry_count);
    if (!old_entries) {
        return; // 内存分配失败，无法扩容
    }
    
    // 收集所有条目
    size_t entry_index = 0;
    for (size_t i = 0; i < map->bucket_count; i++) {
        hash_entry_generic_t* entry = map->buckets[i];
        while (entry) {
            old_entries[entry_index++] = entry;
            entry = entry->next;
        }
    }
    
    // 2. 计算新的bucket数量（扩容为原来的2倍）
    size_t new_bucket_count = map->bucket_count * 2;
    if (new_bucket_count == 0) {
        new_bucket_count = 16; // 防止溢出
    }
    
    // 3. 分配新的buckets数组
    hash_entry_generic_t** new_buckets = (hash_entry_generic_t**)calloc(
        new_bucket_count, sizeof(hash_entry_generic_t*));
    if (!new_buckets) {
        free(old_entries);
        return; // 内存分配失败，无法扩容
    }
    
    // 4. 保存旧的buckets数组（稍后释放）
    hash_entry_generic_t** old_buckets = map->buckets;
    size_t old_bucket_count = map->bucket_count;
    
    // 5. 更新map的buckets和bucket_count
    map->buckets = new_buckets;
    map->bucket_count = new_bucket_count;
    // 注意：entry_count保持不变，因为我们只是重新哈希，没有添加或删除条目
    
    // 6. 重新哈希所有条目到新的buckets
    for (size_t i = 0; i < old_entry_count; i++) {
        hash_entry_generic_t* entry = old_entries[i];
        
        // 重新计算哈希值（使用字符串哈希函数）
        if (entry->key_type == MAP_KEY_TYPE_STRING && entry->key) {
            string_t_internal* entry_key = (string_t_internal*)entry->key;
            int64_t hash64 = runtime_string_hash(entry_key->data);
            uint32_t hash = (uint32_t)(hash64 & 0xFFFFFFFF); // 取低32位
            size_t new_bucket_index = hash % new_bucket_count;
            
            // 断开旧链表连接
            entry->next = NULL;
            
            // 插入到新bucket的链表头部
            if (new_buckets[new_bucket_index]) {
                entry->next = new_buckets[new_bucket_index];
            }
            new_buckets[new_bucket_index] = entry;
        }
    }
    
    // 7. 释放临时数组和旧的buckets数组
    free(old_entries);
    free(old_buckets);
}

// ============================================================================
// Map多类型支持：map[int]int
// ============================================================================

// 前向声明
static void runtime_map_resize_internal_int_int(hash_table_generic_internal_t* map);

/**
 * @brief 设置 map[int]int 中的键值对
 * @param map_ptr map指针（i8*，不透明指针）
 * @param key 键（i32，整数）
 * @param value 值（i32，整数）
 * @return void
 * 
 * 编译器签名：void runtime_map_set_int_int(i8* map_ptr, i32 key, i32 value)
 * 使用位置：编译器生成的map索引赋值代码（map[key] = value，其中map是map[int]int类型）
 */
void runtime_map_set_int_int(void* map_ptr, int32_t key, int32_t value) {
    if (!map_ptr) return;
    
    hash_table_generic_internal_t* map = (hash_table_generic_internal_t*)map_ptr;
    
    if (!map->buckets || map->bucket_count == 0) {
        map->bucket_count = 16;
        map->buckets = (hash_entry_generic_t**)calloc(16, sizeof(hash_entry_generic_t*));
        if (!map->buckets) return;
        map->entry_count = 0;
        map->key_type = MAP_KEY_TYPE_INT32;
        map->value_type = MAP_VALUE_TYPE_INT32;
    }
    
    int64_t hash64 = runtime_int32_hash(key);
    uint32_t hash = (uint32_t)(hash64 & 0xFFFFFFFF);
    size_t bucket_index = hash % map->bucket_count;
    
    hash_entry_generic_t* entry = map->buckets[bucket_index];
    hash_entry_generic_t* prev = NULL;
    
    while (entry) {
        if (entry->key_type == MAP_KEY_TYPE_INT32 && entry->key) {
            int32_t* entry_key = (int32_t*)entry->key;
            if (int32_equals_internal(*entry_key, key)) {
                if (entry->value) free(entry->value);
                int32_t* new_value = (int32_t*)malloc(sizeof(int32_t));
                if (!new_value) return;
                *new_value = value;
                entry->value = new_value;
                entry->value_type = MAP_VALUE_TYPE_INT32;
                entry->value_size = sizeof(int32_t);
                return;
            }
        }
        prev = entry;
        entry = entry->next;
    }
    
    hash_entry_generic_t* new_entry = (hash_entry_generic_t*)malloc(sizeof(hash_entry_generic_t));
    if (!new_entry) return;
    
    int32_t* key_ptr = (int32_t*)malloc(sizeof(int32_t));
    if (!key_ptr) { free(new_entry); return; }
    *key_ptr = key;
    new_entry->key = (void*)key_ptr;
    new_entry->key_type = MAP_KEY_TYPE_INT32;
    new_entry->key_size = sizeof(int32_t);
    
    int32_t* value_ptr = (int32_t*)malloc(sizeof(int32_t));
    if (!value_ptr) { free(key_ptr); free(new_entry); return; }
    *value_ptr = value;
    new_entry->value = (void*)value_ptr;
    new_entry->value_type = MAP_VALUE_TYPE_INT32;
    new_entry->value_size = sizeof(int32_t);
    
    new_entry->next = NULL;
    
    if (prev) prev->next = new_entry;
    else map->buckets[bucket_index] = new_entry;
    
    map->entry_count++;
    
    if (map->bucket_count > 0) {
        float load_factor = (float)map->entry_count / (float)map->bucket_count;
        if (load_factor > 0.75f) {
            runtime_map_resize_internal_int_int(map);
        }
    }
}

/**
 * @brief 获取 map[int]int 中的值
 */
int32_t* runtime_map_get_int_int(void* map_ptr, int32_t key) {
    if (!map_ptr) return NULL;
    
    hash_table_generic_internal_t* map = (hash_table_generic_internal_t*)map_ptr;
    
    if (!map->buckets || map->bucket_count == 0 || map->entry_count == 0) {
        return NULL;
    }
    
    int64_t hash64 = runtime_int32_hash(key);
    uint32_t hash = (uint32_t)(hash64 & 0xFFFFFFFF);
    size_t bucket_index = hash % map->bucket_count;
    
    hash_entry_generic_t* entry = map->buckets[bucket_index];
    while (entry) {
        if (entry->key_type == MAP_KEY_TYPE_INT32 && entry->key) {
            int32_t* entry_key = (int32_t*)entry->key;
            if (int32_equals_internal(*entry_key, key)) {
                if (entry->value && entry->value_type == MAP_VALUE_TYPE_INT32) {
                    return (int32_t*)entry->value;
                }
                return NULL;
            }
        }
        entry = entry->next;
    }
    
    return NULL;
}

/**
 * @brief 删除 map[int]int 中的键值对
 */
void runtime_map_delete_int_int(void* map_ptr, int32_t key) {
    if (!map_ptr) return;
    
    hash_table_generic_internal_t* map = (hash_table_generic_internal_t*)map_ptr;
    
    if (!map->buckets || map->bucket_count == 0 || map->entry_count == 0) {
        return;
    }
    
    int64_t hash64 = runtime_int32_hash(key);
    uint32_t hash = (uint32_t)(hash64 & 0xFFFFFFFF);
    size_t bucket_index = hash % map->bucket_count;
    
    hash_entry_generic_t* entry = map->buckets[bucket_index];
    hash_entry_generic_t* prev = NULL;
    
    while (entry) {
        if (entry->key_type == MAP_KEY_TYPE_INT32 && entry->key) {
            int32_t* entry_key = (int32_t*)entry->key;
            if (int32_equals_internal(*entry_key, key)) {
                if (prev == NULL) {
                    map->buckets[bucket_index] = entry->next;
                } else {
                    prev->next = entry->next;
                }
                
                if (entry->key) free(entry->key);
                if (entry->value) free(entry->value);
                free(entry);
                
                map->entry_count--;
                return;
            }
        }
        prev = entry;
        entry = entry->next;
    }
}

/**
 * @brief 扩容 map[int]int（重新哈希所有条目）
 */
static void runtime_map_resize_internal_int_int(hash_table_generic_internal_t* map) {
    if (!map || !map->buckets || map->bucket_count == 0) return;
    
    size_t old_entry_count = map->entry_count;
    if (old_entry_count == 0) return;
    
    hash_entry_generic_t** old_entries = (hash_entry_generic_t**)malloc(
        sizeof(hash_entry_generic_t*) * old_entry_count);
    if (!old_entries) return;
    
    size_t entry_index = 0;
    for (size_t i = 0; i < map->bucket_count; i++) {
        hash_entry_generic_t* entry = map->buckets[i];
        while (entry) {
            old_entries[entry_index++] = entry;
            entry = entry->next;
        }
    }
    
    size_t new_bucket_count = map->bucket_count * 2;
    if (new_bucket_count == 0) new_bucket_count = 16;
    
    hash_entry_generic_t** new_buckets = (hash_entry_generic_t**)calloc(
        new_bucket_count, sizeof(hash_entry_generic_t*));
    if (!new_buckets) { free(old_entries); return; }
    
    hash_entry_generic_t** old_buckets = map->buckets;
    size_t old_bucket_count = map->bucket_count;
    
    map->buckets = new_buckets;
    map->bucket_count = new_bucket_count;
    
    for (size_t i = 0; i < old_entry_count; i++) {
        hash_entry_generic_t* entry = old_entries[i];
        
        if (entry->key_type == MAP_KEY_TYPE_INT32 && entry->key) {
            int32_t* entry_key = (int32_t*)entry->key;
            int64_t hash64 = runtime_int32_hash(*entry_key);
            uint32_t hash = (uint32_t)(hash64 & 0xFFFFFFFF);
            size_t new_bucket_index = hash % new_bucket_count;
            
            entry->next = NULL;
            
            if (new_buckets[new_bucket_index]) {
                entry->next = new_buckets[new_bucket_index];
            }
            new_buckets[new_bucket_index] = entry;
        }
    }
    
    free(old_entries);
    free(old_buckets);
}

// ============================================================================
// Map多类型支持：map[float]string
// ============================================================================

// 前向声明
static void runtime_map_resize_internal_float_string(hash_table_generic_internal_t* map);

/**
 * @brief 设置 map[float]string 中的键值对
 * @param map_ptr map指针（i8*，不透明指针）
 * @param key 键（f64，浮点数）
 * @param value 值（i8*，字符串）
 * @return void
 * 
 * 编译器签名：void runtime_map_set_float_string(i8* map_ptr, f64 key, i8* value)
 */
void runtime_map_set_float_string(void* map_ptr, double key, void* value) {
    if (!map_ptr) return;
    
    hash_table_generic_internal_t* map = (hash_table_generic_internal_t*)map_ptr;
    
    if (!map->buckets || map->bucket_count == 0) {
        map->bucket_count = 16;
        map->buckets = (hash_entry_generic_t**)calloc(16, sizeof(hash_entry_generic_t*));
        if (!map->buckets) return;
        map->entry_count = 0;
        map->key_type = MAP_KEY_TYPE_FLOAT;
        map->value_type = MAP_VALUE_TYPE_STRING;
    }
    
    int64_t hash64 = runtime_float_hash(key);
    uint32_t hash = (uint32_t)(hash64 & 0xFFFFFFFF);
    size_t bucket_index = hash % map->bucket_count;
    
    hash_entry_generic_t* entry = map->buckets[bucket_index];
    hash_entry_generic_t* prev = NULL;
    
    while (entry) {
        if (entry->key_type == MAP_KEY_TYPE_FLOAT && entry->key) {
            double* entry_key = (double*)entry->key;
            if (float_equals_internal(*entry_key, key)) {
                if (entry->value) {
                    string_t_internal* old_value = (string_t_internal*)entry->value;
                    free_string_internal(old_value);
                }
                string_t_internal* new_value = create_string_from_cstr((const char*)value);
                if (!new_value) return;
                entry->value = new_value;
                entry->value_type = MAP_VALUE_TYPE_STRING;
                entry->value_size = sizeof(string_t_internal);
                return;
            }
        }
        prev = entry;
        entry = entry->next;
    }
    
    hash_entry_generic_t* new_entry = (hash_entry_generic_t*)malloc(sizeof(hash_entry_generic_t));
    if (!new_entry) return;
    
    double* key_ptr = (double*)malloc(sizeof(double));
    if (!key_ptr) { free(new_entry); return; }
    *key_ptr = key;
    new_entry->key = (void*)key_ptr;
    new_entry->key_type = MAP_KEY_TYPE_FLOAT;
    new_entry->key_size = sizeof(double);
    
    string_t_internal* value_str = create_string_from_cstr((const char*)value);
    if (!value_str) { free(key_ptr); free(new_entry); return; }
    new_entry->value = (void*)value_str;
    new_entry->value_type = MAP_VALUE_TYPE_STRING;
    new_entry->value_size = sizeof(string_t_internal);
    
    new_entry->next = NULL;
    
    if (prev) prev->next = new_entry;
    else map->buckets[bucket_index] = new_entry;
    
    map->entry_count++;
    
    if (map->bucket_count > 0) {
        float load_factor = (float)map->entry_count / (float)map->bucket_count;
        if (load_factor > 0.75f) {
            runtime_map_resize_internal_float_string(map);
        }
    }
}

/**
 * @brief 获取 map[float]string 中的值
 */
void* runtime_map_get_float_string(void* map_ptr, double key) {
    if (!map_ptr) return NULL;
    
    hash_table_generic_internal_t* map = (hash_table_generic_internal_t*)map_ptr;
    
    if (!map->buckets || map->bucket_count == 0 || map->entry_count == 0) {
        return NULL;
    }
    
    int64_t hash64 = runtime_float_hash(key);
    uint32_t hash = (uint32_t)(hash64 & 0xFFFFFFFF);
    size_t bucket_index = hash % map->bucket_count;
    
    hash_entry_generic_t* entry = map->buckets[bucket_index];
    while (entry) {
        if (entry->key_type == MAP_KEY_TYPE_FLOAT && entry->key) {
            double* entry_key = (double*)entry->key;
            if (float_equals_internal(*entry_key, key)) {
                if (entry->value && entry->value_type == MAP_VALUE_TYPE_STRING) {
                    return (void*)entry->value; // 返回string_t_internal*（char*）
                }
                return NULL;
            }
        }
        entry = entry->next;
    }
    
    return NULL;
}

/**
 * @brief 删除 map[float]string 中的键值对
 */
void runtime_map_delete_float_string(void* map_ptr, double key) {
    if (!map_ptr) return;
    
    hash_table_generic_internal_t* map = (hash_table_generic_internal_t*)map_ptr;
    
    if (!map->buckets || map->bucket_count == 0 || map->entry_count == 0) {
        return;
    }
    
    int64_t hash64 = runtime_float_hash(key);
    uint32_t hash = (uint32_t)(hash64 & 0xFFFFFFFF);
    size_t bucket_index = hash % map->bucket_count;
    
    hash_entry_generic_t* entry = map->buckets[bucket_index];
    hash_entry_generic_t* prev = NULL;
    
    while (entry) {
        if (entry->key_type == MAP_KEY_TYPE_FLOAT && entry->key) {
            double* entry_key = (double*)entry->key;
            if (float_equals_internal(*entry_key, key)) {
                if (prev == NULL) {
                    map->buckets[bucket_index] = entry->next;
                } else {
                    prev->next = entry->next;
                }
                
                if (entry->key) free(entry->key);
                if (entry->value) {
                    string_t_internal* value_str = (string_t_internal*)entry->value;
                    free_string_internal(value_str);
                }
                free(entry);
                
                map->entry_count--;
                return;
            }
        }
        prev = entry;
        entry = entry->next;
    }
}

/**
 * @brief 扩容 map[float]string（重新哈希所有条目）
 */
static void runtime_map_resize_internal_float_string(hash_table_generic_internal_t* map) {
    if (!map || !map->buckets || map->bucket_count == 0) return;
    
    size_t old_entry_count = map->entry_count;
    if (old_entry_count == 0) return;
    
    hash_entry_generic_t** old_entries = (hash_entry_generic_t**)malloc(
        sizeof(hash_entry_generic_t*) * old_entry_count);
    if (!old_entries) return;
    
    size_t entry_index = 0;
    for (size_t i = 0; i < map->bucket_count; i++) {
        hash_entry_generic_t* entry = map->buckets[i];
        while (entry) {
            old_entries[entry_index++] = entry;
            entry = entry->next;
        }
    }
    
    size_t new_bucket_count = map->bucket_count * 2;
    if (new_bucket_count == 0) new_bucket_count = 16;
    
    hash_entry_generic_t** new_buckets = (hash_entry_generic_t**)calloc(
        new_bucket_count, sizeof(hash_entry_generic_t*));
    if (!new_buckets) { free(old_entries); return; }
    
    hash_entry_generic_t** old_buckets = map->buckets;
    size_t old_bucket_count = map->bucket_count;
    
    map->buckets = new_buckets;
    map->bucket_count = new_bucket_count;
    
    for (size_t i = 0; i < old_entry_count; i++) {
        hash_entry_generic_t* entry = old_entries[i];
        
        if (entry->key_type == MAP_KEY_TYPE_FLOAT && entry->key) {
            double* entry_key = (double*)entry->key;
            int64_t hash64 = runtime_float_hash(*entry_key);
            uint32_t hash = (uint32_t)(hash64 & 0xFFFFFFFF);
            size_t new_bucket_index = hash % new_bucket_count;
            
            entry->next = NULL;
            
            if (new_buckets[new_bucket_index]) {
                entry->next = new_buckets[new_bucket_index];
            }
            new_buckets[new_bucket_index] = entry;
        }
    }
    
    free(old_entries);
    free(old_buckets);
}

// ============================================================================
// Map多类型支持：map[string]float
// ============================================================================

// 前向声明
static void runtime_map_resize_internal_string_float(hash_table_generic_internal_t* map);

/**
 * @brief 设置 map[string]float 中的键值对
 * @param map_ptr map指针（i8*，不透明指针）
 * @param key 键（i8*，字符串）
 * @param value 值（f64，浮点数）
 * @return void
 * 
 * 编译器签名：void runtime_map_set_string_float(i8* map_ptr, i8* key, f64 value)
 */
void runtime_map_set_string_float(void* map_ptr, void* key, double value) {
    if (!map_ptr) return;
    
    hash_table_generic_internal_t* map = (hash_table_generic_internal_t*)map_ptr;
    
    if (!map->buckets || map->bucket_count == 0) {
        map->bucket_count = 16;
        map->buckets = (hash_entry_generic_t**)calloc(16, sizeof(hash_entry_generic_t*));
        if (!map->buckets) return;
        map->entry_count = 0;
        map->key_type = MAP_KEY_TYPE_STRING;
        map->value_type = MAP_VALUE_TYPE_FLOAT;
    }
    
    int64_t hash64 = runtime_string_hash((const char*)key);
    uint32_t hash = (uint32_t)(hash64 & 0xFFFFFFFF);
    size_t bucket_index = hash % map->bucket_count;
    
    hash_entry_generic_t* entry = map->buckets[bucket_index];
    hash_entry_generic_t* prev = NULL;
    
    while (entry) {
        if (entry->key_type == MAP_KEY_TYPE_STRING && entry->key) {
            string_t_internal* entry_key = (string_t_internal*)entry->key;
            if (string_equals_internal(entry_key, (const char*)key)) {
                if (entry->value) {
                    free(entry->value);
                }
                double* new_value = (double*)malloc(sizeof(double));
                if (!new_value) return;
                *new_value = value;
                entry->value = new_value;
                entry->value_type = MAP_VALUE_TYPE_FLOAT;
                entry->value_size = sizeof(double);
                return;
            }
        }
        prev = entry;
        entry = entry->next;
    }
    
    hash_entry_generic_t* new_entry = (hash_entry_generic_t*)malloc(sizeof(hash_entry_generic_t));
    if (!new_entry) return;
    
    string_t_internal* key_str = create_string_from_cstr((const char*)key);
    if (!key_str) { free(new_entry); return; }
    new_entry->key = (void*)key_str;
    new_entry->key_type = MAP_KEY_TYPE_STRING;
    new_entry->key_size = sizeof(string_t_internal);
    
    double* value_ptr = (double*)malloc(sizeof(double));
    if (!value_ptr) { free_string_internal(key_str); free(new_entry); return; }
    *value_ptr = value;
    new_entry->value = (void*)value_ptr;
    new_entry->value_type = MAP_VALUE_TYPE_FLOAT;
    new_entry->value_size = sizeof(double);
    
    new_entry->next = NULL;
    
    if (prev) prev->next = new_entry;
    else map->buckets[bucket_index] = new_entry;
    
    map->entry_count++;
    
    if (map->bucket_count > 0) {
        float load_factor = (float)map->entry_count / (float)map->bucket_count;
        if (load_factor > 0.75f) {
            runtime_map_resize_internal_string_float(map);
        }
    }
}

/**
 * @brief 获取 map[string]float 中的值
 */
double* runtime_map_get_string_float(void* map_ptr, void* key) {
    if (!map_ptr) return NULL;
    
    hash_table_generic_internal_t* map = (hash_table_generic_internal_t*)map_ptr;
    
    if (!map->buckets || map->bucket_count == 0 || map->entry_count == 0) {
        return NULL;
    }
    
    int64_t hash64 = runtime_string_hash((const char*)key);
    uint32_t hash = (uint32_t)(hash64 & 0xFFFFFFFF);
    size_t bucket_index = hash % map->bucket_count;
    
    hash_entry_generic_t* entry = map->buckets[bucket_index];
    while (entry) {
        if (entry->key_type == MAP_KEY_TYPE_STRING && entry->key) {
            string_t_internal* entry_key = (string_t_internal*)entry->key;
            if (string_equals_internal(entry_key, (const char*)key)) {
                if (entry->value && entry->value_type == MAP_VALUE_TYPE_FLOAT) {
                    return (double*)entry->value; // 返回double*（浮点数指针）
                }
                return NULL;
            }
        }
        entry = entry->next;
    }
    
    return NULL;
}

/**
 * @brief 删除 map[string]float 中的键值对
 */
void runtime_map_delete_string_float(void* map_ptr, void* key) {
    if (!map_ptr) return;
    
    hash_table_generic_internal_t* map = (hash_table_generic_internal_t*)map_ptr;
    
    if (!map->buckets || map->bucket_count == 0 || map->entry_count == 0) {
        return;
    }
    
    int64_t hash64 = runtime_string_hash((const char*)key);
    uint32_t hash = (uint32_t)(hash64 & 0xFFFFFFFF);
    size_t bucket_index = hash % map->bucket_count;
    
    hash_entry_generic_t* entry = map->buckets[bucket_index];
    hash_entry_generic_t* prev = NULL;
    
    while (entry) {
        if (entry->key_type == MAP_KEY_TYPE_STRING && entry->key) {
            string_t_internal* entry_key = (string_t_internal*)entry->key;
            if (string_equals_internal(entry_key, (const char*)key)) {
                if (prev == NULL) {
                    map->buckets[bucket_index] = entry->next;
                } else {
                    prev->next = entry->next;
                }
                
                if (entry->key) {
                    free_string_internal((string_t_internal*)entry->key);
                }
                if (entry->value) {
                    free(entry->value); // 释放double*内存
                }
                free(entry);
                
                map->entry_count--;
                return;
            }
        }
        prev = entry;
        entry = entry->next;
    }
}

/**
 * @brief 扩容 map[string]float（重新哈希所有条目）
 */
static void runtime_map_resize_internal_string_float(hash_table_generic_internal_t* map) {
    if (!map || !map->buckets || map->bucket_count == 0) return;
    
    size_t old_entry_count = map->entry_count;
    if (old_entry_count == 0) return;
    
    hash_entry_generic_t** old_entries = (hash_entry_generic_t**)malloc(
        sizeof(hash_entry_generic_t*) * old_entry_count);
    if (!old_entries) return;
    
    size_t entry_index = 0;
    for (size_t i = 0; i < map->bucket_count; i++) {
        hash_entry_generic_t* entry = map->buckets[i];
        while (entry) {
            old_entries[entry_index++] = entry;
            entry = entry->next;
        }
    }
    
    size_t new_bucket_count = map->bucket_count * 2;
    if (new_bucket_count == 0) new_bucket_count = 16;
    
    hash_entry_generic_t** new_buckets = (hash_entry_generic_t**)calloc(
        new_bucket_count, sizeof(hash_entry_generic_t*));
    if (!new_buckets) { free(old_entries); return; }
    
    hash_entry_generic_t** old_buckets = map->buckets;
    size_t old_bucket_count = map->bucket_count;
    
    map->buckets = new_buckets;
    map->bucket_count = new_bucket_count;
    
    for (size_t i = 0; i < old_entry_count; i++) {
        hash_entry_generic_t* entry = old_entries[i];
        
        if (entry->key_type == MAP_KEY_TYPE_STRING && entry->key) {
            string_t_internal* entry_key = (string_t_internal*)entry->key;
            int64_t hash64 = runtime_string_hash(entry_key->data);
            uint32_t hash = (uint32_t)(hash64 & 0xFFFFFFFF);
            size_t new_bucket_index = hash % new_bucket_count;
            
            entry->next = NULL;
            
            if (new_buckets[new_bucket_index]) {
                entry->next = new_buckets[new_bucket_index];
            }
            new_buckets[new_bucket_index] = entry;
        }
    }
    
    free(old_entries);
    free(old_buckets);
}

// ============================================================================
// Map多类型支持：map[bool]string
// ============================================================================

// 前向声明
static void runtime_map_resize_internal_bool_string(hash_table_generic_internal_t* map);

/**
 * @brief 布尔值哈希函数（用于map[bool]string）
 * @param b 布尔值（true=1, false=0）
 * @return 哈希值（int64_t）
 * 
 * 注意：布尔值只有两个可能值，哈希函数简单返回0或1
 */
static inline int64_t runtime_bool_hash(bool b) {
    return b ? 1 : 0;
}

/**
 * @brief 布尔值比较函数（内部使用）
 * @param a 第一个布尔值
 * @param b 第二个布尔值
 * @return 1 如果相等，0 如果不相等
 */
static inline int bool_equals_internal(bool a, bool b) {
    return a == b ? 1 : 0;
}

/**
 * @brief 设置 map[bool]string 中的键值对
 * @param map_ptr map指针（i8*，不透明指针）
 * @param key 键（i1，布尔值）
 * @param value 值（i8*，字符串）
 * @return void
 * 
 * 编译器签名：void runtime_map_set_bool_string(i8* map_ptr, i1 key, i8* value)
 */
void runtime_map_set_bool_string(void* map_ptr, bool key, void* value) {
    if (!map_ptr) return;
    
    hash_table_generic_internal_t* map = (hash_table_generic_internal_t*)map_ptr;
    
    if (!map->buckets || map->bucket_count == 0) {
        map->bucket_count = 16;
        map->buckets = (hash_entry_generic_t**)calloc(16, sizeof(hash_entry_generic_t*));
        if (!map->buckets) return;
        map->entry_count = 0;
        map->key_type = MAP_KEY_TYPE_BOOL;
        map->value_type = MAP_VALUE_TYPE_STRING;
    }
    
    int64_t hash64 = runtime_bool_hash(key);
    uint32_t hash = (uint32_t)(hash64 & 0xFFFFFFFF);
    size_t bucket_index = hash % map->bucket_count;
    
    hash_entry_generic_t* entry = map->buckets[bucket_index];
    hash_entry_generic_t* prev = NULL;
    
    while (entry) {
        if (entry->key_type == MAP_KEY_TYPE_BOOL && entry->key) {
            bool* entry_key = (bool*)entry->key;
            if (bool_equals_internal(*entry_key, key)) {
                if (entry->value) {
                    free_string_internal((string_t_internal*)entry->value);
                }
                string_t_internal* new_value = create_string_from_cstr((const char*)value);
                if (!new_value) return;
                entry->value = (void*)new_value;
                entry->value_type = MAP_VALUE_TYPE_STRING;
                entry->value_size = sizeof(string_t_internal);
                return;
            }
        }
        prev = entry;
        entry = entry->next;
    }
    
    hash_entry_generic_t* new_entry = (hash_entry_generic_t*)malloc(sizeof(hash_entry_generic_t));
    if (!new_entry) return;
    
    bool* key_ptr = (bool*)malloc(sizeof(bool));
    if (!key_ptr) { free(new_entry); return; }
    *key_ptr = key;
    new_entry->key = (void*)key_ptr;
    new_entry->key_type = MAP_KEY_TYPE_BOOL;
    new_entry->key_size = sizeof(bool);
    
    string_t_internal* value_str = create_string_from_cstr((const char*)value);
    if (!value_str) { free(key_ptr); free(new_entry); return; }
    new_entry->value = (void*)value_str;
    new_entry->value_type = MAP_VALUE_TYPE_STRING;
    new_entry->value_size = sizeof(string_t_internal);
    
    new_entry->next = NULL;
    
    if (prev) prev->next = new_entry;
    else map->buckets[bucket_index] = new_entry;
    
    map->entry_count++;
    
    if (map->bucket_count > 0) {
        float load_factor = (float)map->entry_count / (float)map->bucket_count;
        if (load_factor > 0.75f) {
            runtime_map_resize_internal_bool_string(map);
        }
    }
}

/**
 * @brief 获取 map[bool]string 中的值
 * @param map_ptr map指针（i8*，不透明指针）
 * @param key 键（i1，布尔值）
 * @return 值（i8*，字符串指针），如果不存在返回NULL
 * 
 * 编译器签名：i8* runtime_map_get_bool_string(i8* map_ptr, i1 key)
 */
void* runtime_map_get_bool_string(void* map_ptr, bool key) {
    if (!map_ptr) return NULL;
    
    hash_table_generic_internal_t* map = (hash_table_generic_internal_t*)map_ptr;
    
    if (!map->buckets || map->bucket_count == 0 || map->entry_count == 0) {
        return NULL;
    }
    
    int64_t hash64 = runtime_bool_hash(key);
    uint32_t hash = (uint32_t)(hash64 & 0xFFFFFFFF);
    size_t bucket_index = hash % map->bucket_count;
    
    hash_entry_generic_t* entry = map->buckets[bucket_index];
    while (entry) {
        if (entry->key_type == MAP_KEY_TYPE_BOOL && entry->key) {
            bool* entry_key = (bool*)entry->key;
            if (bool_equals_internal(*entry_key, key)) {
                if (entry->value && entry->value_type == MAP_VALUE_TYPE_STRING) {
                    string_t_internal* str = (string_t_internal*)entry->value;
                    return (void*)str->data; // 返回char*（字符串数据指针）
                }
                return NULL;
            }
        }
        entry = entry->next;
    }
    
    return NULL;
}

/**
 * @brief 删除 map[bool]string 中的键值对
 * @param map_ptr map指针（i8*，不透明指针）
 * @param key 键（i1，布尔值）
 * @return void
 * 
 * 编译器签名：void runtime_map_delete_bool_string(i8* map_ptr, i1 key)
 */
void runtime_map_delete_bool_string(void* map_ptr, bool key) {
    if (!map_ptr) return;
    
    hash_table_generic_internal_t* map = (hash_table_generic_internal_t*)map_ptr;
    
    if (!map->buckets || map->bucket_count == 0 || map->entry_count == 0) {
        return;
    }
    
    int64_t hash64 = runtime_bool_hash(key);
    uint32_t hash = (uint32_t)(hash64 & 0xFFFFFFFF);
    size_t bucket_index = hash % map->bucket_count;
    
    hash_entry_generic_t* entry = map->buckets[bucket_index];
    hash_entry_generic_t* prev = NULL;
    
    while (entry) {
        if (entry->key_type == MAP_KEY_TYPE_BOOL && entry->key) {
            bool* entry_key = (bool*)entry->key;
            if (bool_equals_internal(*entry_key, key)) {
                if (prev == NULL) {
                    map->buckets[bucket_index] = entry->next;
                } else {
                    prev->next = entry->next;
                }
                
                if (entry->key) {
                    free(entry->key); // 释放bool*内存
                }
                if (entry->value) {
                    free_string_internal((string_t_internal*)entry->value);
                }
                free(entry);
                
                map->entry_count--;
                return;
            }
        }
        prev = entry;
        entry = entry->next;
    }
}

/**
 * @brief 扩容 map[bool]string（重新哈希所有条目）
 */
static void runtime_map_resize_internal_bool_string(hash_table_generic_internal_t* map) {
    if (!map || !map->buckets || map->bucket_count == 0) return;
    
    size_t old_entry_count = map->entry_count;
    if (old_entry_count == 0) return;
    
    hash_entry_generic_t** old_entries = (hash_entry_generic_t**)malloc(
        sizeof(hash_entry_generic_t*) * old_entry_count);
    if (!old_entries) return;
    
    size_t entry_index = 0;
    for (size_t i = 0; i < map->bucket_count; i++) {
        hash_entry_generic_t* entry = map->buckets[i];
        while (entry) {
            old_entries[entry_index++] = entry;
            entry = entry->next;
        }
    }
    
    size_t new_bucket_count = map->bucket_count * 2;
    if (new_bucket_count == 0) new_bucket_count = 16;
    
    hash_entry_generic_t** new_buckets = (hash_entry_generic_t**)calloc(
        new_bucket_count, sizeof(hash_entry_generic_t*));
    if (!new_buckets) { free(old_entries); return; }
    
    hash_entry_generic_t** old_buckets = map->buckets;
    size_t old_bucket_count = map->bucket_count;
    
    map->buckets = new_buckets;
    map->bucket_count = new_bucket_count;
    
    for (size_t i = 0; i < old_entry_count; i++) {
        hash_entry_generic_t* entry = old_entries[i];
        
        if (entry->key_type == MAP_KEY_TYPE_BOOL && entry->key) {
            bool* entry_key = (bool*)entry->key;
            int64_t hash64 = runtime_bool_hash(*entry_key);
            uint32_t hash = (uint32_t)(hash64 & 0xFFFFFFFF);
            size_t new_bucket_index = hash % new_bucket_count;
            
            entry->next = NULL;
            
            if (new_buckets[new_bucket_index]) {
                entry->next = new_buckets[new_bucket_index];
            }
            new_buckets[new_bucket_index] = entry;
        }
    }
    
    free(old_entries);
    free(old_buckets);
}

// ============================================================================
// Map操作功能：通用函数（适用于所有Map类型）
// ============================================================================

/**
 * @brief 获取Map的长度（条目数量）
 * @param map_ptr map指针（i8*，不透明指针）
 * @return 条目数量（i32）
 * 
 * 编译器签名：i32 runtime_map_len(i8* map_ptr)
 * 
 * 注意：这是类型无关的函数，适用于所有Map类型
 */
int32_t runtime_map_len(void* map_ptr) {
    if (!map_ptr) return 0;
    
    hash_table_generic_internal_t* map = (hash_table_generic_internal_t*)map_ptr;
    
    if (!map->buckets || map->bucket_count == 0) {
        return 0;
    }
    
    return (int32_t)map->entry_count;
}

/**
 * @brief 清空Map（删除所有条目）
 * @param map_ptr map指针（i8*，不透明指针）
 * @return void
 * 
 * 编译器签名：void runtime_map_clear(i8* map_ptr)
 * 
 * 注意：这是类型无关的函数，适用于所有Map类型
 * 需要根据键值类型正确释放内存
 */
void runtime_map_clear(void* map_ptr) {
    if (!map_ptr) return;
    
    hash_table_generic_internal_t* map = (hash_table_generic_internal_t*)map_ptr;
    
    if (!map->buckets || map->bucket_count == 0 || map->entry_count == 0) {
        return;
    }
    
    // 遍历所有桶，释放所有条目
    for (size_t i = 0; i < map->bucket_count; i++) {
        hash_entry_generic_t* entry = map->buckets[i];
        while (entry) {
            hash_entry_generic_t* next = entry->next;
            
            // 释放键内存
            if (entry->key) {
                if (entry->key_type == MAP_KEY_TYPE_STRING) {
                    // 字符串键：释放string_t_internal
                    free_string_internal((string_t_internal*)entry->key);
                } else {
                    // 其他类型键：直接释放
                    free(entry->key);
                }
            }
            
            // 释放值内存
            if (entry->value) {
                if (entry->value_type == MAP_VALUE_TYPE_STRING) {
                    // 字符串值：释放string_t_internal
                    free_string_internal((string_t_internal*)entry->value);
                } else {
                    // 其他类型值：直接释放
                    free(entry->value);
                }
            }
            
            // 释放条目本身
            free(entry);
            
            entry = next;
        }
        map->buckets[i] = NULL;
    }
    
    // 重置计数
    map->entry_count = 0;
}

/**
 * @brief 检查Map中是否包含指定的键
 * @param map_ptr map指针（i8*，不透明指针）
 * @param key 键（根据Map类型，可能是i8*, i32, i64, f64, i1等）
 * @return 1 如果键存在，0 如果不存在
 * 
 * 编译器签名：i32 runtime_map_contains(i8* map_ptr, ... key)
 * 
 * 注意：这是类型相关的函数，需要根据键类型调用相应的比较函数
 * 由于键类型不同，这个函数需要多个重载版本
 * 或者使用类型参数来区分
 * 
 * 当前实现：提供一个通用接口，内部根据map的key_type进行判断
 * 但需要知道key的具体类型才能正确比较
 * 
 * 方案：为每种键类型提供专门的contains函数
 * runtime_map_contains_string_string, runtime_map_contains_int_string等
 * 
 * 但为了简化，我们先实现一个通用版本，接受void* key和key_type参数
 */
int32_t runtime_map_contains_generic(void* map_ptr, void* key, map_key_type_t key_type) {
    if (!map_ptr || !key) return 0;
    
    hash_table_generic_internal_t* map = (hash_table_generic_internal_t*)map_ptr;
    
    if (!map->buckets || map->bucket_count == 0 || map->entry_count == 0) {
        return 0;
    }
    
    // 根据键类型计算哈希值
    int64_t hash64 = 0;
    uint32_t hash = 0;
    
    switch (key_type) {
        case MAP_KEY_TYPE_STRING: {
            const char* str_key = (const char*)key;
            hash64 = runtime_string_hash(str_key);
            hash = (uint32_t)(hash64 & 0xFFFFFFFF);
            break;
        }
        case MAP_KEY_TYPE_INT32: {
            int32_t int_key = *(int32_t*)key;
            hash64 = runtime_int32_hash(int_key);
            hash = (uint32_t)(hash64 & 0xFFFFFFFF);
            break;
        }
        case MAP_KEY_TYPE_INT64: {
            int64_t int_key = *(int64_t*)key;
            hash64 = runtime_int64_hash(int_key);
            hash = (uint32_t)(hash64 & 0xFFFFFFFF);
            break;
        }
        case MAP_KEY_TYPE_FLOAT: {
            double float_key = *(double*)key;
            hash64 = runtime_float_hash(float_key);
            hash = (uint32_t)(hash64 & 0xFFFFFFFF);
            break;
        }
        case MAP_KEY_TYPE_BOOL: {
            bool bool_key = *(bool*)key;
            hash64 = runtime_bool_hash(bool_key);
            hash = (uint32_t)(hash64 & 0xFFFFFFFF);
            break;
        }
        default:
            return 0;
    }
    
    size_t bucket_index = hash % map->bucket_count;
    
    hash_entry_generic_t* entry = map->buckets[bucket_index];
    while (entry) {
        if (entry->key_type == key_type && entry->key) {
            // 根据键类型进行比较
            int match = 0;
            switch (key_type) {
                case MAP_KEY_TYPE_STRING: {
                    const char* str_key = (const char*)key;
                    string_t_internal* entry_key = (string_t_internal*)entry->key;
                    match = string_equals_internal(entry_key, str_key);
                    break;
                }
                case MAP_KEY_TYPE_INT32: {
                    int32_t int_key = *(int32_t*)key;
                    int32_t* entry_key = (int32_t*)entry->key;
                    match = int32_equals_internal(*entry_key, int_key);
                    break;
                }
                case MAP_KEY_TYPE_INT64: {
                    int64_t int_key = *(int64_t*)key;
                    int64_t* entry_key = (int64_t*)entry->key;
                    match = int64_equals_internal(*entry_key, int_key);
                    break;
                }
                case MAP_KEY_TYPE_FLOAT: {
                    double float_key = *(double*)key;
                    double* entry_key = (double*)entry->key;
                    match = float_equals_internal(*entry_key, float_key);
                    break;
                }
                case MAP_KEY_TYPE_BOOL: {
                    bool bool_key = *(bool*)key;
                    bool* entry_key = (bool*)entry->key;
                    match = bool_equals_internal(*entry_key, bool_key);
                    break;
                }
                default:
                    match = 0;
            }
            
            if (match) {
                return 1; // 找到匹配的键
            }
        }
        entry = entry->next;
    }
    
    return 0; // 未找到
}

// ============================================================================
// Map操作功能：类型特定的contains函数（简化编译器调用）
// ============================================================================

/**
 * @brief 检查 map[string]string 中是否包含指定的键
 * @param map_ptr map指针（i8*，不透明指针）
 * @param key 键（i8*，字符串）
 * @return 1 如果键存在，0 如果不存在
 * 
 * 编译器签名：i32 runtime_map_contains_string_string(i8* map_ptr, i8* key)
 */
int32_t runtime_map_contains_string_string(void* map_ptr, void* key) {
    return runtime_map_contains_generic(map_ptr, key, MAP_KEY_TYPE_STRING);
}

/**
 * @brief 检查 map[int]string 中是否包含指定的键
 * @param map_ptr map指针（i8*，不透明指针）
 * @param key 键（i32，整数）
 * @return 1 如果键存在，0 如果不存在
 * 
 * 编译器签名：i32 runtime_map_contains_int_string(i8* map_ptr, i32 key)
 */
int32_t runtime_map_contains_int_string(void* map_ptr, int32_t key) {
    return runtime_map_contains_generic(map_ptr, &key, MAP_KEY_TYPE_INT32);
}

/**
 * @brief 检查 map[string]int 中是否包含指定的键
 * @param map_ptr map指针（i8*，不透明指针）
 * @param key 键（i8*，字符串）
 * @return 1 如果键存在，0 如果不存在
 * 
 * 编译器签名：i32 runtime_map_contains_string_int(i8* map_ptr, i8* key)
 */
int32_t runtime_map_contains_string_int(void* map_ptr, void* key) {
    return runtime_map_contains_generic(map_ptr, key, MAP_KEY_TYPE_STRING);
}

/**
 * @brief 检查 map[int]int 中是否包含指定的键
 * @param map_ptr map指针（i8*，不透明指针）
 * @param key 键（i32，整数）
 * @return 1 如果键存在，0 如果不存在
 * 
 * 编译器签名：i32 runtime_map_contains_int_int(i8* map_ptr, i32 key)
 */
int32_t runtime_map_contains_int_int(void* map_ptr, int32_t key) {
    return runtime_map_contains_generic(map_ptr, &key, MAP_KEY_TYPE_INT32);
}

/**
 * @brief 检查 map[float]string 中是否包含指定的键
 * @param map_ptr map指针（i8*，不透明指针）
 * @param key 键（f64，浮点数）
 * @return 1 如果键存在，0 如果不存在
 * 
 * 编译器签名：i32 runtime_map_contains_float_string(i8* map_ptr, f64 key)
 */
int32_t runtime_map_contains_float_string(void* map_ptr, double key) {
    return runtime_map_contains_generic(map_ptr, &key, MAP_KEY_TYPE_FLOAT);
}

/**
 * @brief 检查 map[string]float 中是否包含指定的键
 * @param map_ptr map指针（i8*，不透明指针）
 * @param key 键（i8*，字符串）
 * @return 1 如果键存在，0 如果不存在
 * 
 * 编译器签名：i32 runtime_map_contains_string_float(i8* map_ptr, i8* key)
 */
int32_t runtime_map_contains_string_float(void* map_ptr, void* key) {
    return runtime_map_contains_generic(map_ptr, key, MAP_KEY_TYPE_STRING);
}

/**
 * @brief 检查 map[bool]string 中是否包含指定的键
 * @param map_ptr map指针（i8*，不透明指针）
 * @param key 键（i1，布尔值）
 * @return 1 如果键存在，0 如果不存在
 * 
 * 编译器签名：i32 runtime_map_contains_bool_string(i8* map_ptr, i1 key)
 */
int32_t runtime_map_contains_bool_string(void* map_ptr, bool key) {
    return runtime_map_contains_generic(map_ptr, &key, MAP_KEY_TYPE_BOOL);
}

// ============================================================================
// Map操作功能：keys() 和 values() 函数（类型特定）
// ============================================================================

/**
 * @brief 获取 map[string]string 的键数组
 * @param map_ptr map指针（i8*，不透明指针）
 * @return 键数组指针（i8**，字符串数组）
 * 
 * 编译器签名：i8** runtime_map_keys_string_string(i8* map_ptr)
 * 
 * 注意：返回的数组需要调用者释放内存
 */
char** runtime_map_keys_string_string(void* map_ptr) {
    if (!map_ptr) return NULL;
    
    hash_table_generic_internal_t* map = (hash_table_generic_internal_t*)map_ptr;
    
    if (!map->buckets || map->bucket_count == 0 || map->entry_count == 0) {
        // 返回空数组指针
        char** keys = (char**)malloc(sizeof(char*));
        if (keys) keys[0] = NULL;
        return keys;
    }
    
    // 分配键数组
    char** keys = (char**)malloc(sizeof(char*) * (map->entry_count + 1)); // +1 for NULL terminator
    if (!keys) return NULL;
    
    // 遍历所有条目，收集键
    size_t index = 0;
    for (size_t i = 0; i < map->bucket_count; i++) {
        hash_entry_generic_t* entry = map->buckets[i];
        while (entry && index < map->entry_count) {
            if (entry->key_type == MAP_KEY_TYPE_STRING && entry->key) {
                string_t_internal* key_str = (string_t_internal*)entry->key;
                if (key_str->data && key_str->length > 0) {
                    // 复制字符串键
                    size_t key_len = key_str->length + 1;
                    keys[index] = (char*)malloc(key_len);
                    if (keys[index]) {
                        memcpy(keys[index], key_str->data, key_str->length);
                        keys[index][key_str->length] = '\0';
                    }
                } else {
                    keys[index] = NULL;
                }
            }
            index++;
            entry = entry->next;
        }
    }
    
    // NULL终止符
    keys[index] = NULL;
    
    return keys;
}

/**
 * @brief 获取 map[int]string 的键数组
 * @param map_ptr map指针（i8*，不透明指针）
 * @return 键数组指针（i32*，整数数组）
 * 
 * 编译器签名：i32* runtime_map_keys_int_string(i8* map_ptr)
 * 
 * 注意：返回的数组需要调用者释放内存
 */
int32_t* runtime_map_keys_int_string(void* map_ptr) {
    if (!map_ptr) return NULL;
    
    hash_table_generic_internal_t* map = (hash_table_generic_internal_t*)map_ptr;
    
    if (!map->buckets || map->bucket_count == 0 || map->entry_count == 0) {
        return NULL;
    }
    
    // 分配键数组
    int32_t* keys = (int32_t*)malloc(sizeof(int32_t) * map->entry_count);
    if (!keys) return NULL;
    
    // 遍历所有条目，收集键
    size_t index = 0;
    for (size_t i = 0; i < map->bucket_count; i++) {
        hash_entry_generic_t* entry = map->buckets[i];
        while (entry && index < map->entry_count) {
            if (entry->key_type == MAP_KEY_TYPE_INT32 && entry->key) {
                int32_t* key_int = (int32_t*)entry->key;
                keys[index] = *key_int;
            }
            index++;
            entry = entry->next;
        }
    }
    
    return keys;
}

/**
 * @brief 获取 map[string]int 的键数组
 * @param map_ptr map指针（i8*，不透明指针）
 * @return 键数组指针（i8**，字符串数组）
 * 
 * 编译器签名：i8** runtime_map_keys_string_int(i8* map_ptr)
 */
char** runtime_map_keys_string_int(void* map_ptr) {
    // 与 map[string]string 的键数组实现相同
    return runtime_map_keys_string_string(map_ptr);
}

/**
 * @brief 获取 map[int]int 的键数组
 * @param map_ptr map指针（i8*，不透明指针）
 * @return 键数组指针（i32*，整数数组）
 * 
 * 编译器签名：i32* runtime_map_keys_int_int(i8* map_ptr)
 */
int32_t* runtime_map_keys_int_int(void* map_ptr) {
    // 与 map[int]string 的键数组实现相同
    return runtime_map_keys_int_string(map_ptr);
}

/**
 * @brief 获取 map[float]string 的键数组
 * @param map_ptr map指针（i8*，不透明指针）
 * @return 键数组指针（f64*，浮点数数组）
 * 
 * 编译器签名：f64* runtime_map_keys_float_string(i8* map_ptr)
 */
double* runtime_map_keys_float_string(void* map_ptr) {
    if (!map_ptr) return NULL;
    
    hash_table_generic_internal_t* map = (hash_table_generic_internal_t*)map_ptr;
    
    if (!map->buckets || map->bucket_count == 0 || map->entry_count == 0) {
        return NULL;
    }
    
    // 分配键数组
    double* keys = (double*)malloc(sizeof(double) * map->entry_count);
    if (!keys) return NULL;
    
    // 遍历所有条目，收集键
    size_t index = 0;
    for (size_t i = 0; i < map->bucket_count; i++) {
        hash_entry_generic_t* entry = map->buckets[i];
        while (entry && index < map->entry_count) {
            if (entry->key_type == MAP_KEY_TYPE_FLOAT && entry->key) {
                double* key_float = (double*)entry->key;
                keys[index] = *key_float;
            }
            index++;
            entry = entry->next;
        }
    }
    
    return keys;
}

/**
 * @brief 获取 map[string]float 的键数组
 * @param map_ptr map指针（i8*，不透明指针）
 * @return 键数组指针（i8**，字符串数组）
 * 
 * 编译器签名：i8** runtime_map_keys_string_float(i8* map_ptr)
 */
char** runtime_map_keys_string_float(void* map_ptr) {
    // 与 map[string]string 的键数组实现相同
    return runtime_map_keys_string_string(map_ptr);
}

/**
 * @brief 获取 map[bool]string 的键数组
 * @param map_ptr map指针（i8*，不透明指针）
 * @return 键数组指针（i1*，布尔数组）
 * 
 * 编译器签名：i1* runtime_map_keys_bool_string(i8* map_ptr)
 */
bool* runtime_map_keys_bool_string(void* map_ptr) {
    if (!map_ptr) return NULL;
    
    hash_table_generic_internal_t* map = (hash_table_generic_internal_t*)map_ptr;
    
    if (!map->buckets || map->bucket_count == 0 || map->entry_count == 0) {
        return NULL;
    }
    
    // 分配键数组
    bool* keys = (bool*)malloc(sizeof(bool) * map->entry_count);
    if (!keys) return NULL;
    
    // 遍历所有条目，收集键
    size_t index = 0;
    for (size_t i = 0; i < map->bucket_count; i++) {
        hash_entry_generic_t* entry = map->buckets[i];
        while (entry && index < map->entry_count) {
            if (entry->key_type == MAP_KEY_TYPE_BOOL && entry->key) {
                bool* key_bool = (bool*)entry->key;
                keys[index] = *key_bool;
            }
            index++;
            entry = entry->next;
        }
    }
    
    return keys;
}

// ============================================================================
// Map操作功能：values() 函数（类型特定）
// ============================================================================

/**
 * @brief 获取 map[string]string 的值数组
 * @param map_ptr map指针（i8*，不透明指针）
 * @return 值数组指针（i8**，字符串数组）
 * 
 * 编译器签名：i8** runtime_map_values_string_string(i8* map_ptr)
 */
char** runtime_map_values_string_string(void* map_ptr) {
    if (!map_ptr) return NULL;
    
    hash_table_generic_internal_t* map = (hash_table_generic_internal_t*)map_ptr;
    
    if (!map->buckets || map->bucket_count == 0 || map->entry_count == 0) {
        char** values = (char**)malloc(sizeof(char*));
        if (values) values[0] = NULL;
        return values;
    }
    
    // 分配值数组
    char** values = (char**)malloc(sizeof(char*) * (map->entry_count + 1));
    if (!values) return NULL;
    
    // 遍历所有条目，收集值
    size_t index = 0;
    for (size_t i = 0; i < map->bucket_count; i++) {
        hash_entry_generic_t* entry = map->buckets[i];
        while (entry && index < map->entry_count) {
            if (entry->value_type == MAP_VALUE_TYPE_STRING && entry->value) {
                string_t_internal* value_str = (string_t_internal*)entry->value;
                if (value_str->data && value_str->length > 0) {
                    // 复制字符串值
                    size_t value_len = value_str->length + 1;
                    values[index] = (char*)malloc(value_len);
                    if (values[index]) {
                        memcpy(values[index], value_str->data, value_str->length);
                        values[index][value_str->length] = '\0';
                    }
                } else {
                    values[index] = NULL;
                }
            }
            index++;
            entry = entry->next;
        }
    }
    
    // NULL终止符
    values[index] = NULL;
    
    return values;
}

/**
 * @brief 获取 map[int]string 的值数组
 * @param map_ptr map指针（i8*，不透明指针）
 * @return 值数组指针（i8**，字符串数组）
 * 
 * 编译器签名：i8** runtime_map_values_int_string(i8* map_ptr)
 */
char** runtime_map_values_int_string(void* map_ptr) {
    // 与 map[string]string 的值数组实现相同（值类型都是string）
    return runtime_map_values_string_string(map_ptr);
}

/**
 * @brief 获取 map[string]int 的值数组
 * @param map_ptr map指针（i8*，不透明指针）
 * @return 值数组指针（i32*，整数数组）
 * 
 * 编译器签名：i32* runtime_map_values_string_int(i8* map_ptr)
 */
int32_t* runtime_map_values_string_int(void* map_ptr) {
    if (!map_ptr) return NULL;
    
    hash_table_generic_internal_t* map = (hash_table_generic_internal_t*)map_ptr;
    
    if (!map->buckets || map->bucket_count == 0 || map->entry_count == 0) {
        return NULL;
    }
    
    // 分配值数组
    int32_t* values = (int32_t*)malloc(sizeof(int32_t) * map->entry_count);
    if (!values) return NULL;
    
    // 遍历所有条目，收集值
    size_t index = 0;
    for (size_t i = 0; i < map->bucket_count; i++) {
        hash_entry_generic_t* entry = map->buckets[i];
        while (entry && index < map->entry_count) {
            if (entry->value_type == MAP_VALUE_TYPE_INT32 && entry->value) {
                int32_t* value_int = (int32_t*)entry->value;
                values[index] = *value_int;
            }
            index++;
            entry = entry->next;
        }
    }
    
    return values;
}

/**
 * @brief 获取 map[int]int 的值数组
 * @param map_ptr map指针（i8*，不透明指针）
 * @return 值数组指针（i32*，整数数组）
 * 
 * 编译器签名：i32* runtime_map_values_int_int(i8* map_ptr)
 */
int32_t* runtime_map_values_int_int(void* map_ptr) {
    // 与 map[string]int 的值数组实现相同（值类型都是int）
    return runtime_map_values_string_int(map_ptr);
}

/**
 * @brief 获取 map[float]string 的值数组
 * @param map_ptr map指针（i8*，不透明指针）
 * @return 值数组指针（i8**，字符串数组）
 * 
 * 编译器签名：i8** runtime_map_values_float_string(i8* map_ptr)
 */
char** runtime_map_values_float_string(void* map_ptr) {
    // 与 map[string]string 的值数组实现相同（值类型都是string）
    return runtime_map_values_string_string(map_ptr);
}

/**
 * @brief 获取 map[string]float 的值数组
 * @param map_ptr map指针（i8*，不透明指针）
 * @return 值数组指针（f64*，浮点数数组）
 * 
 * 编译器签名：f64* runtime_map_values_string_float(i8* map_ptr)
 */
double* runtime_map_values_string_float(void* map_ptr) {
    if (!map_ptr) return NULL;
    
    hash_table_generic_internal_t* map = (hash_table_generic_internal_t*)map_ptr;
    
    if (!map->buckets || map->bucket_count == 0 || map->entry_count == 0) {
        return NULL;
    }
    
    // 分配值数组
    double* values = (double*)malloc(sizeof(double) * map->entry_count);
    if (!values) return NULL;
    
    // 遍历所有条目，收集值
    size_t index = 0;
    for (size_t i = 0; i < map->bucket_count; i++) {
        hash_entry_generic_t* entry = map->buckets[i];
        while (entry && index < map->entry_count) {
            if (entry->value_type == MAP_VALUE_TYPE_FLOAT && entry->value) {
                double* value_float = (double*)entry->value;
                values[index] = *value_float;
            }
            index++;
            entry = entry->next;
        }
    }
    
    return values;
}

/**
 * @brief 获取 map[bool]string 的值数组
 * @param map_ptr map指针（i8*，不透明指针）
 * @return 值数组指针（i8**，字符串数组）
 * 
 * 编译器签名：i8** runtime_map_values_bool_string(i8* map_ptr)
 */
char** runtime_map_values_bool_string(void* map_ptr) {
    // 与 map[string]string 的值数组实现相同（值类型都是string）
    return runtime_map_values_string_string(map_ptr);
}

// ============================================================================
// Map操作功能：数组/切片类型支持（map[string][]int等）
// ============================================================================

/**
 * @brief 设置 map[string][]int 中的键值对
 * @param map_ptr map指针（i8*，不透明指针）
 * @param key 键（i8*，字符串）
 * @param value 值（i8*，切片指针）
 * @return void
 * 
 * 编译器签名：void runtime_map_set_string_slice_int(i8* map_ptr, i8* key, i8* slice_ptr)
 * 
 * 注意：切片值存储为指针（void*），需要调用者管理内存
 */
void runtime_map_set_string_slice_int(void* map_ptr, void* key, void* slice_ptr) {
    if (!map_ptr || !key) return;
    
    hash_table_generic_internal_t* map = (hash_table_generic_internal_t*)map_ptr;
    
    // 计算哈希值
    const char* str_key = (const char*)key;
    int64_t hash64 = runtime_string_hash(str_key);
    uint32_t hash = (uint32_t)(hash64 & 0xFFFFFFFF);
    size_t bucket_index = hash % map->bucket_count;
    
    // 查找是否已存在
    hash_entry_generic_t* entry = map->buckets[bucket_index];
    while (entry) {
        if (entry->key_type == MAP_KEY_TYPE_STRING && entry->key) {
            string_t_internal* key_str = (string_t_internal*)entry->key;
            const char* existing_key = key_str->data;
            if (existing_key && strcmp(existing_key, str_key) == 0) {
                // 更新值
                if (entry->value) {
                    // 释放旧值（切片指针，需要调用切片释放函数）
                    // 注意：这里简化处理，假设切片由GC管理
                    // 实际实现中可能需要调用 runtime_free_slice()
                }
                entry->value = slice_ptr; // 存储切片指针
                entry->value_type = MAP_VALUE_TYPE_SLICE;
                entry->value_size = sizeof(void*); // 指针大小
                return;
            }
        }
        entry = entry->next;
    }
    
    // 创建新条目
    hash_entry_generic_t* new_entry = (hash_entry_generic_t*)malloc(sizeof(hash_entry_generic_t));
    if (!new_entry) return;
    
    // 复制键（字符串）
    string_t_internal* key_str = (string_t_internal*)malloc(sizeof(string_t_internal));
    if (!key_str) {
        free(new_entry);
        return;
    }
    size_t key_len = strlen(str_key);
    key_str->data = (char*)malloc(key_len + 1);
    if (!key_str->data) {
        free(key_str);
        free(new_entry);
        return;
    }
    memcpy(key_str->data, str_key, key_len);
    key_str->data[key_len] = '\0';
    key_str->length = key_len;
    key_str->capacity = key_len + 1;
    
    new_entry->key = key_str;
    new_entry->key_type = MAP_KEY_TYPE_STRING;
    new_entry->key_size = sizeof(string_t_internal);
    
    new_entry->value = slice_ptr; // 存储切片指针
    new_entry->value_type = MAP_VALUE_TYPE_SLICE;
    new_entry->value_size = sizeof(void*);
    new_entry->next = NULL;
    
    // 插入到桶中
    if (!map->buckets[bucket_index]) {
        map->buckets[bucket_index] = new_entry;
    } else {
        new_entry->next = map->buckets[bucket_index];
        map->buckets[bucket_index] = new_entry;
    }
    
    map->entry_count++;
}

/**
 * @brief 获取 map[string][]int 中的值
 * @param map_ptr map指针（i8*，不透明指针）
 * @param key 键（i8*，字符串）
 * @return 值（i8*，切片指针），如果不存在返回NULL
 * 
 * 编译器签名：i8* runtime_map_get_string_slice_int(i8* map_ptr, i8* key)
 */
void* runtime_map_get_string_slice_int(void* map_ptr, void* key) {
    if (!map_ptr || !key) return NULL;
    
    hash_table_generic_internal_t* map = (hash_table_generic_internal_t*)map_ptr;
    
    if (!map->buckets || map->bucket_count == 0) {
        return NULL;
    }
    
    // 计算哈希值
    const char* str_key = (const char*)key;
    int64_t hash64 = runtime_string_hash(str_key);
    uint32_t hash = (uint32_t)(hash64 & 0xFFFFFFFF);
    size_t bucket_index = hash % map->bucket_count;
    
    // 查找条目
    hash_entry_generic_t* entry = map->buckets[bucket_index];
    while (entry) {
        if (entry->key_type == MAP_KEY_TYPE_STRING && entry->key) {
            string_t_internal* key_str = (string_t_internal*)entry->key;
            const char* existing_key = key_str->data;
            if (existing_key && strcmp(existing_key, str_key) == 0) {
                // 返回值（切片指针）
                return entry->value;
            }
        }
        entry = entry->next;
    }
    
    return NULL;
}

/**
 * @brief 设置 map[int][]string 中的键值对
 * @param map_ptr map指针（i8*，不透明指针）
 * @param key 键（i32，整数）
 * @param value 值（i8*，切片指针）
 * @return void
 * 
 * 编译器签名：void runtime_map_set_int_slice_string(i8* map_ptr, i32 key, i8* slice_ptr)
 */
void runtime_map_set_int_slice_string(void* map_ptr, int32_t key, void* slice_ptr) {
    if (!map_ptr) return;
    
    hash_table_generic_internal_t* map = (hash_table_generic_internal_t*)map_ptr;
    
    // 计算哈希值
    int64_t hash64 = runtime_int32_hash(key);
    uint32_t hash = (uint32_t)(hash64 & 0xFFFFFFFF);
    size_t bucket_index = hash % map->bucket_count;
    
    // 查找是否已存在
    hash_entry_generic_t* entry = map->buckets[bucket_index];
    while (entry) {
        if (entry->key_type == MAP_KEY_TYPE_INT32 && entry->key) {
            int32_t* existing_key = (int32_t*)entry->key;
            if (*existing_key == key) {
                // 更新值
                if (entry->value) {
                    // 释放旧值（切片指针）
                }
                entry->value = slice_ptr;
                entry->value_type = MAP_VALUE_TYPE_SLICE;
                entry->value_size = sizeof(void*);
                return;
            }
        }
        entry = entry->next;
    }
    
    // 创建新条目
    hash_entry_generic_t* new_entry = (hash_entry_generic_t*)malloc(sizeof(hash_entry_generic_t));
    if (!new_entry) return;
    
    // 复制键（整数）
    int32_t* key_int = (int32_t*)malloc(sizeof(int32_t));
    if (!key_int) {
        free(new_entry);
        return;
    }
    *key_int = key;
    
    new_entry->key = key_int;
    new_entry->key_type = MAP_KEY_TYPE_INT32;
    new_entry->key_size = sizeof(int32_t);
    
    new_entry->value = slice_ptr;
    new_entry->value_type = MAP_VALUE_TYPE_SLICE;
    new_entry->value_size = sizeof(void*);
    new_entry->next = NULL;
    
    // 插入到桶中
    if (!map->buckets[bucket_index]) {
        map->buckets[bucket_index] = new_entry;
    } else {
        new_entry->next = map->buckets[bucket_index];
        map->buckets[bucket_index] = new_entry;
    }
    
    map->entry_count++;
}

/**
 * @brief 获取 map[int][]string 中的值
 * @param map_ptr map指针（i8*，不透明指针）
 * @param key 键（i32，整数）
 * @return 值（i8*，切片指针），如果不存在返回NULL
 * 
 * 编译器签名：i8* runtime_map_get_int_slice_string(i8* map_ptr, i32 key)
 */
void* runtime_map_get_int_slice_string(void* map_ptr, int32_t key) {
    if (!map_ptr) return NULL;
    
    hash_table_generic_internal_t* map = (hash_table_generic_internal_t*)map_ptr;
    
    if (!map->buckets || map->bucket_count == 0) {
        return NULL;
    }
    
    // 计算哈希值
    int64_t hash64 = runtime_int32_hash(key);
    uint32_t hash = (uint32_t)(hash64 & 0xFFFFFFFF);
    size_t bucket_index = hash % map->bucket_count;
    
    // 查找条目
    hash_entry_generic_t* entry = map->buckets[bucket_index];
    while (entry) {
        if (entry->key_type == MAP_KEY_TYPE_INT32 && entry->key) {
            int32_t* existing_key = (int32_t*)entry->key;
            if (*existing_key == key) {
                // 返回值（切片指针）
                return entry->value;
            }
        }
        entry = entry->next;
    }
    
    return NULL;
}

// ============================================================================
// Map操作功能：嵌套Map类型支持（map[string]map[int]string等）
// ============================================================================

/**
 * @brief 设置 map[string]map[int]string 中的键值对
 * @param map_ptr map指针（i8*，不透明指针）
 * @param key 键（i8*，字符串）
 * @param value 值（i8*，内层Map指针）
 * @return void
 * 
 * 编译器签名：void runtime_map_set_string_map_int_string(i8* map_ptr, i8* key, i8* inner_map_ptr)
 * 
 * 注意：内层Map值存储为指针（void*），需要调用者管理内存
 */
void runtime_map_set_string_map_int_string(void* map_ptr, void* key, void* inner_map_ptr) {
    if (!map_ptr || !key) return;
    
    hash_table_generic_internal_t* map = (hash_table_generic_internal_t*)map_ptr;
    
    // 计算哈希值
    const char* str_key = (const char*)key;
    int64_t hash64 = runtime_string_hash(str_key);
    uint32_t hash = (uint32_t)(hash64 & 0xFFFFFFFF);
    size_t bucket_index = hash % map->bucket_count;
    
    // 查找是否已存在
    hash_entry_generic_t* entry = map->buckets[bucket_index];
    while (entry) {
        if (entry->key_type == MAP_KEY_TYPE_STRING && entry->key) {
            string_t_internal* key_str = (string_t_internal*)entry->key;
            const char* existing_key = key_str->data;
            if (existing_key && strcmp(existing_key, str_key) == 0) {
                // 更新值
                if (entry->value) {
                    // 释放旧值（内层Map指针，需要调用Map释放函数）
                    // 注意：这里简化处理，假设Map由GC管理
                    // 实际实现中可能需要调用 runtime_map_clear() 和 free()
                }
                entry->value = inner_map_ptr; // 存储内层Map指针
                entry->value_type = MAP_VALUE_TYPE_MAP;
                entry->value_size = sizeof(void*); // 指针大小
                return;
            }
        }
        entry = entry->next;
    }
    
    // 创建新条目
    hash_entry_generic_t* new_entry = (hash_entry_generic_t*)malloc(sizeof(hash_entry_generic_t));
    if (!new_entry) return;
    
    // 复制键（字符串）
    string_t_internal* key_str = (string_t_internal*)malloc(sizeof(string_t_internal));
    if (!key_str) {
        free(new_entry);
        return;
    }
    size_t key_len = strlen(str_key);
    key_str->data = (char*)malloc(key_len + 1);
    if (!key_str->data) {
        free(key_str);
        free(new_entry);
        return;
    }
    memcpy(key_str->data, str_key, key_len);
    key_str->data[key_len] = '\0';
    key_str->length = key_len;
    key_str->capacity = key_len + 1;
    
    new_entry->key = key_str;
    new_entry->key_type = MAP_KEY_TYPE_STRING;
    new_entry->key_size = sizeof(string_t_internal);
    
    new_entry->value = inner_map_ptr; // 存储内层Map指针
    new_entry->value_type = MAP_VALUE_TYPE_MAP;
    new_entry->value_size = sizeof(void*);
    new_entry->next = NULL;
    
    // 插入到桶中
    if (!map->buckets[bucket_index]) {
        map->buckets[bucket_index] = new_entry;
    } else {
        new_entry->next = map->buckets[bucket_index];
        map->buckets[bucket_index] = new_entry;
    }
    
    map->entry_count++;
}

/**
 * @brief 获取 map[string]map[int]string 中的值
 * @param map_ptr map指针（i8*，不透明指针）
 * @param key 键（i8*，字符串）
 * @return 值（i8*，内层Map指针），如果不存在返回NULL
 * 
 * 编译器签名：i8* runtime_map_get_string_map_int_string(i8* map_ptr, i8* key)
 */
void* runtime_map_get_string_map_int_string(void* map_ptr, void* key) {
    if (!map_ptr || !key) return NULL;
    
    hash_table_generic_internal_t* map = (hash_table_generic_internal_t*)map_ptr;
    
    if (!map->buckets || map->bucket_count == 0) {
        return NULL;
    }
    
    // 计算哈希值
    const char* str_key = (const char*)key;
    int64_t hash64 = runtime_string_hash(str_key);
    uint32_t hash = (uint32_t)(hash64 & 0xFFFFFFFF);
    size_t bucket_index = hash % map->bucket_count;
    
    // 查找条目
    hash_entry_generic_t* entry = map->buckets[bucket_index];
    while (entry) {
        if (entry->key_type == MAP_KEY_TYPE_STRING && entry->key) {
            string_t_internal* key_str = (string_t_internal*)entry->key;
            const char* existing_key = key_str->data;
            if (existing_key && strcmp(existing_key, str_key) == 0) {
                // 返回值（内层Map指针）
                return entry->value;
            }
        }
        entry = entry->next;
    }
    
    return NULL;
}

/**
 * @brief 设置 map[int]map[string]int 中的键值对
 * @param map_ptr map指针（i8*，不透明指针）
 * @param key 键（i32，整数）
 * @param value 值（i8*，内层Map指针）
 * @return void
 * 
 * 编译器签名：void runtime_map_set_int_map_string_int(i8* map_ptr, i32 key, i8* inner_map_ptr)
 */
void runtime_map_set_int_map_string_int(void* map_ptr, int32_t key, void* inner_map_ptr) {
    if (!map_ptr) return;
    
    hash_table_generic_internal_t* map = (hash_table_generic_internal_t*)map_ptr;
    
    // 计算哈希值
    int64_t hash64 = runtime_int32_hash(key);
    uint32_t hash = (uint32_t)(hash64 & 0xFFFFFFFF);
    size_t bucket_index = hash % map->bucket_count;
    
    // 查找是否已存在
    hash_entry_generic_t* entry = map->buckets[bucket_index];
    while (entry) {
        if (entry->key_type == MAP_KEY_TYPE_INT32 && entry->key) {
            int32_t* existing_key = (int32_t*)entry->key;
            if (*existing_key == key) {
                // 更新值
                if (entry->value) {
                    // 释放旧值（内层Map指针）
                }
                entry->value = inner_map_ptr;
                entry->value_type = MAP_VALUE_TYPE_MAP;
                entry->value_size = sizeof(void*);
                return;
            }
        }
        entry = entry->next;
    }
    
    // 创建新条目
    hash_entry_generic_t* new_entry = (hash_entry_generic_t*)malloc(sizeof(hash_entry_generic_t));
    if (!new_entry) return;
    
    // 复制键（整数）
    int32_t* key_int = (int32_t*)malloc(sizeof(int32_t));
    if (!key_int) {
        free(new_entry);
        return;
    }
    *key_int = key;
    
    new_entry->key = key_int;
    new_entry->key_type = MAP_KEY_TYPE_INT32;
    new_entry->key_size = sizeof(int32_t);
    
    new_entry->value = inner_map_ptr;
    new_entry->value_type = MAP_VALUE_TYPE_MAP;
    new_entry->value_size = sizeof(void*);
    new_entry->next = NULL;
    
    // 插入到桶中
    if (!map->buckets[bucket_index]) {
        map->buckets[bucket_index] = new_entry;
    } else {
        new_entry->next = map->buckets[bucket_index];
        map->buckets[bucket_index] = new_entry;
    }
    
    map->entry_count++;
}

/**
 * @brief 获取 map[int]map[string]int 中的值
 * @param map_ptr map指针（i8*，不透明指针）
 * @param key 键（i32，整数）
 * @return 值（i8*，内层Map指针），如果不存在返回NULL
 * 
 * 编译器签名：i8* runtime_map_get_int_map_string_int(i8* map_ptr, i32 key)
 */
void* runtime_map_get_int_map_string_int(void* map_ptr, int32_t key) {
    if (!map_ptr) return NULL;
    
    hash_table_generic_internal_t* map = (hash_table_generic_internal_t*)map_ptr;
    
    if (!map->buckets || map->bucket_count == 0) {
        return NULL;
    }
    
    // 计算哈希值
    int64_t hash64 = runtime_int32_hash(key);
    uint32_t hash = (uint32_t)(hash64 & 0xFFFFFFFF);
    size_t bucket_index = hash % map->bucket_count;
    
    // 查找条目
    hash_entry_generic_t* entry = map->buckets[bucket_index];
    while (entry) {
        if (entry->key_type == MAP_KEY_TYPE_INT32 && entry->key) {
            int32_t* existing_key = (int32_t*)entry->key;
            if (*existing_key == key) {
                // 返回值（内层Map指针）
                return entry->value;
            }
        }
        entry = entry->next;
    }
    
    return NULL;
}

// ============================================================================
// Map操作功能：结构体作为值支持（map[string]User等）
// ============================================================================

/**
 * @brief 设置 map[string]struct 中的键值对（通用结构体值）
 * @param map_ptr map指针（i8*，不透明指针）
 * @param key 键（i8*，字符串）
 * @param struct_ptr 值（i8*，结构体指针）
 * @return void
 * 
 * 编译器签名：void runtime_map_set_string_struct(i8* map_ptr, i8* key, i8* struct_ptr)
 * 
 * 注意：结构体值存储为指针（void*），需要调用者管理内存
 * 支持所有结构体类型（User, Person等），通过指针传递
 */
void runtime_map_set_string_struct(void* map_ptr, void* key, void* struct_ptr) {
    if (!map_ptr || !key) return;
    
    hash_table_generic_internal_t* map = (hash_table_generic_internal_t*)map_ptr;
    
    // 计算哈希值
    const char* str_key = (const char*)key;
    int64_t hash64 = runtime_string_hash(str_key);
    uint32_t hash = (uint32_t)(hash64 & 0xFFFFFFFF);
    size_t bucket_index = hash % map->bucket_count;
    
    // 查找是否已存在
    hash_entry_generic_t* entry = map->buckets[bucket_index];
    while (entry) {
        if (entry->key_type == MAP_KEY_TYPE_STRING && entry->key) {
            string_t_internal* key_str = (string_t_internal*)entry->key;
            const char* existing_key = key_str->data;
            if (existing_key && strcmp(existing_key, str_key) == 0) {
                // 更新值
                if (entry->value) {
                    // 释放旧值（结构体指针，需要调用结构体释放函数）
                    // 注意：这里简化处理，假设结构体由GC管理
                    // 实际实现中可能需要调用结构体析构函数和free()
                }
                entry->value = struct_ptr; // 存储结构体指针
                entry->value_type = MAP_VALUE_TYPE_STRUCT;
                entry->value_size = sizeof(void*); // 指针大小
                return;
            }
        }
        entry = entry->next;
    }
    
    // 创建新条目
    hash_entry_generic_t* new_entry = (hash_entry_generic_t*)malloc(sizeof(hash_entry_generic_t));
    if (!new_entry) return;
    
    // 复制键（字符串）
    string_t_internal* key_str = (string_t_internal*)malloc(sizeof(string_t_internal));
    if (!key_str) {
        free(new_entry);
        return;
    }
    size_t key_len = strlen(str_key);
    key_str->data = (char*)malloc(key_len + 1);
    if (!key_str->data) {
        free(key_str);
        free(new_entry);
        return;
    }
    memcpy(key_str->data, str_key, key_len);
    key_str->data[key_len] = '\0';
    key_str->length = key_len;
    key_str->capacity = key_len + 1;
    
    new_entry->key = key_str;
    new_entry->key_type = MAP_KEY_TYPE_STRING;
    new_entry->key_size = sizeof(string_t_internal);
    
    new_entry->value = struct_ptr; // 存储结构体指针
    new_entry->value_type = MAP_VALUE_TYPE_STRUCT;
    new_entry->value_size = sizeof(void*);
    new_entry->next = NULL;
    
    // 插入到桶中
    if (!map->buckets[bucket_index]) {
        map->buckets[bucket_index] = new_entry;
    } else {
        new_entry->next = map->buckets[bucket_index];
        map->buckets[bucket_index] = new_entry;
    }
    
    map->entry_count++;
}

/**
 * @brief 获取 map[string]struct 中的值（通用结构体值）
 * @param map_ptr map指针（i8*，不透明指针）
 * @param key 键（i8*，字符串）
 * @return 值（i8*，结构体指针），如果不存在返回NULL
 * 
 * 编译器签名：i8* runtime_map_get_string_struct(i8* map_ptr, i8* key)
 */
void* runtime_map_get_string_struct(void* map_ptr, void* key) {
    if (!map_ptr || !key) return NULL;
    
    hash_table_generic_internal_t* map = (hash_table_generic_internal_t*)map_ptr;
    
    if (!map->buckets || map->bucket_count == 0) {
        return NULL;
    }
    
    // 计算哈希值
    const char* str_key = (const char*)key;
    int64_t hash64 = runtime_string_hash(str_key);
    uint32_t hash = (uint32_t)(hash64 & 0xFFFFFFFF);
    size_t bucket_index = hash % map->bucket_count;
    
    // 查找条目
    hash_entry_generic_t* entry = map->buckets[bucket_index];
    while (entry) {
        if (entry->key_type == MAP_KEY_TYPE_STRING && entry->key) {
            string_t_internal* key_str = (string_t_internal*)entry->key;
            const char* existing_key = key_str->data;
            if (existing_key && strcmp(existing_key, str_key) == 0) {
                // 返回值（结构体指针）
                return entry->value;
            }
        }
        entry = entry->next;
    }
    
    return NULL;
}

/**
 * @brief 设置 map[int]struct 中的键值对（通用结构体值）
 * @param map_ptr map指针（i8*，不透明指针）
 * @param key 键（i32，整数）
 * @param struct_ptr 值（i8*，结构体指针）
 * @return void
 * 
 * 编译器签名：void runtime_map_set_int_struct(i8* map_ptr, i32 key, i8* struct_ptr)
 */
void runtime_map_set_int_struct(void* map_ptr, int32_t key, void* struct_ptr) {
    if (!map_ptr) return;
    
    hash_table_generic_internal_t* map = (hash_table_generic_internal_t*)map_ptr;
    
    // 计算哈希值
    int64_t hash64 = runtime_int32_hash(key);
    uint32_t hash = (uint32_t)(hash64 & 0xFFFFFFFF);
    size_t bucket_index = hash % map->bucket_count;
    
    // 查找是否已存在
    hash_entry_generic_t* entry = map->buckets[bucket_index];
    while (entry) {
        if (entry->key_type == MAP_KEY_TYPE_INT32 && entry->key) {
            int32_t* existing_key = (int32_t*)entry->key;
            if (*existing_key == key) {
                // 更新值
                if (entry->value) {
                    // 释放旧值（结构体指针）
                }
                entry->value = struct_ptr;
                entry->value_type = MAP_VALUE_TYPE_STRUCT;
                entry->value_size = sizeof(void*);
                return;
            }
        }
        entry = entry->next;
    }
    
    // 创建新条目
    hash_entry_generic_t* new_entry = (hash_entry_generic_t*)malloc(sizeof(hash_entry_generic_t));
    if (!new_entry) return;
    
    // 复制键（整数）
    int32_t* key_int = (int32_t*)malloc(sizeof(int32_t));
    if (!key_int) {
        free(new_entry);
        return;
    }
    *key_int = key;
    
    new_entry->key = key_int;
    new_entry->key_type = MAP_KEY_TYPE_INT32;
    new_entry->key_size = sizeof(int32_t);
    
    new_entry->value = struct_ptr;
    new_entry->value_type = MAP_VALUE_TYPE_STRUCT;
    new_entry->value_size = sizeof(void*);
    new_entry->next = NULL;
    
    // 插入到桶中
    if (!map->buckets[bucket_index]) {
        map->buckets[bucket_index] = new_entry;
    } else {
        new_entry->next = map->buckets[bucket_index];
        map->buckets[bucket_index] = new_entry;
    }
    
    map->entry_count++;
}

/**
 * @brief 获取 map[int]struct 中的值（通用结构体值）
 * @param map_ptr map指针（i8*，不透明指针）
 * @param key 键（i32，整数）
 * @return 值（i8*，结构体指针），如果不存在返回NULL
 * 
 * 编译器签名：i8* runtime_map_get_int_struct(i8* map_ptr, i32 key)
 */
void* runtime_map_get_int_struct(void* map_ptr, int32_t key) {
    if (!map_ptr) return NULL;
    
    hash_table_generic_internal_t* map = (hash_table_generic_internal_t*)map_ptr;
    
    if (!map->buckets || map->bucket_count == 0) {
        return NULL;
    }
    
    // 计算哈希值
    int64_t hash64 = runtime_int32_hash(key);
    uint32_t hash = (uint32_t)(hash64 & 0xFFFFFFFF);
    size_t bucket_index = hash % map->bucket_count;
    
    // 查找条目
    hash_entry_generic_t* entry = map->buckets[bucket_index];
    while (entry) {
        if (entry->key_type == MAP_KEY_TYPE_INT32 && entry->key) {
            int32_t* existing_key = (int32_t*)entry->key;
            if (*existing_key == key) {
                // 返回值（结构体指针）
                return entry->value;
            }
        }
        entry = entry->next;
    }
    
    return NULL;
}

// ============================================================================
// Map操作功能：结构体作为键支持（map[User]string等）
// ============================================================================

/**
 * @brief 设置 map[struct]string 中的键值对（通用结构体键版本）
 * @param map_ptr map指针（i8*，不透明指针）
 * @param key_ptr 键（i8*，结构体指针）
 * @param value 值（i8*，字符串）
 * @param hash_func 哈希函数指针（i64 (*)(i8*)）
 * @param equals_func 比较函数指针（i1 (*)(i8*, i8*)）
 * @return void
 * 
 * 编译器签名：void runtime_map_set_struct_string(i8* map_ptr, i8* key_ptr, i8* value, i64 (*hash_func)(i8*), i1 (*equals_func)(i8*, i8*))
 * 
 * 注意：结构体键存储为指针（void*），需要调用者管理内存
 * 哈希和比较通过函数指针调用，函数指针由编译器生成
 */
void runtime_map_set_struct_string(void* map_ptr, void* key_ptr, void* value, map_key_hash_func_t hash_func, map_key_equals_func_t equals_func) {
    if (!map_ptr || !key_ptr || !hash_func || !equals_func) return;
    
    hash_table_generic_internal_t* map = (hash_table_generic_internal_t*)map_ptr;
    
    // 使用函数指针计算哈希值
    int64_t hash64 = hash_func(key_ptr);
    uint32_t hash = (uint32_t)(hash64 & 0xFFFFFFFF);
    size_t bucket_index = hash % map->bucket_count;
    
    // 查找是否已存在
    hash_entry_generic_t* entry = map->buckets[bucket_index];
    while (entry) {
        if (entry->key_type == MAP_KEY_TYPE_STRUCT && entry->key) {
            // 使用函数指针比较键
            if (equals_func(entry->key, key_ptr)) {
                // 更新值
                if (entry->value) {
                    // 释放旧值（字符串）
                    string_t_internal* old_value = (string_t_internal*)entry->value;
                    if (old_value && old_value->data) {
                        free(old_value->data);
                    }
                    free(old_value);
                }
                
                // 复制新值（字符串）
                const char* str_value = (const char*)value;
                string_t_internal* value_str = (string_t_internal*)malloc(sizeof(string_t_internal));
                if (value_str) {
                    size_t value_len = strlen(str_value);
                    value_str->data = (char*)malloc(value_len + 1);
                    if (value_str->data) {
                        memcpy(value_str->data, str_value, value_len);
                        value_str->data[value_len] = '\0';
                        value_str->length = value_len;
                        value_str->capacity = value_len + 1;
                    }
                    entry->value = value_str;
                }
                entry->value_type = MAP_VALUE_TYPE_STRING;
                entry->value_size = sizeof(string_t_internal);
                return;
            }
        }
        entry = entry->next;
    }
    
    // 创建新条目
    hash_entry_generic_t* new_entry = (hash_entry_generic_t*)malloc(sizeof(hash_entry_generic_t));
    if (!new_entry) return;
    
    // 复制键（结构体指针，需要深拷贝结构体）
    // 注意：这里简化处理，假设结构体大小已知，需要调用者提供
    // 实际实现中，可能需要通过函数参数传递结构体大小
    // 当前实现：直接存储指针（假设结构体由调用者管理）
    new_entry->key = key_ptr; // 直接存储指针（需要调用者保证生命周期）
    new_entry->key_type = MAP_KEY_TYPE_STRUCT;
    new_entry->key_size = sizeof(void*); // 指针大小
    new_entry->key_hash_func = hash_func; // 存储函数指针
    new_entry->key_equals_func = equals_func; // 存储函数指针
    
    // 复制值（字符串）
    const char* str_value = (const char*)value;
    string_t_internal* value_str = (string_t_internal*)malloc(sizeof(string_t_internal));
    if (!value_str) {
        free(new_entry);
        return;
    }
    size_t value_len = strlen(str_value);
    value_str->data = (char*)malloc(value_len + 1);
    if (!value_str->data) {
        free(value_str);
        free(new_entry);
        return;
    }
    memcpy(value_str->data, str_value, value_len);
    value_str->data[value_len] = '\0';
    value_str->length = value_len;
    value_str->capacity = value_len + 1;
    
    new_entry->value = value_str;
    new_entry->value_type = MAP_VALUE_TYPE_STRING;
    new_entry->value_size = sizeof(string_t_internal);
    new_entry->next = NULL;
    
    // 插入到桶中
    if (!map->buckets[bucket_index]) {
        map->buckets[bucket_index] = new_entry;
    } else {
        new_entry->next = map->buckets[bucket_index];
        map->buckets[bucket_index] = new_entry;
    }
    
    map->entry_count++;
}

/**
 * @brief 获取 map[struct]string 中的值（通用结构体键版本）
 * @param map_ptr map指针（i8*，不透明指针）
 * @param key_ptr 键（i8*，结构体指针）
 * @param hash_func 哈希函数指针（i64 (*)(i8*)）
 * @param equals_func 比较函数指针（i1 (*)(i8*, i8*)）
 * @return 值（i8*，字符串指针），如果不存在返回NULL
 * 
 * 编译器签名：i8* runtime_map_get_struct_string(i8* map_ptr, i8* key_ptr, i64 (*hash_func)(i8*), i1 (*equals_func)(i8*, i8*))
 */
void* runtime_map_get_struct_string(void* map_ptr, void* key_ptr, map_key_hash_func_t hash_func, map_key_equals_func_t equals_func) {
    if (!map_ptr || !key_ptr || !hash_func || !equals_func) return NULL;
    
    hash_table_generic_internal_t* map = (hash_table_generic_internal_t*)map_ptr;
    
    if (!map->buckets || map->bucket_count == 0) {
        return NULL;
    }
    
    // 使用函数指针计算哈希值
    int64_t hash64 = hash_func(key_ptr);
    uint32_t hash = (uint32_t)(hash64 & 0xFFFFFFFF);
    size_t bucket_index = hash % map->bucket_count;
    
    // 查找条目
    hash_entry_generic_t* entry = map->buckets[bucket_index];
    while (entry) {
        if (entry->key_type == MAP_KEY_TYPE_STRUCT && entry->key) {
            // 使用函数指针比较键
            if (equals_func(entry->key, key_ptr)) {
                // 返回值（字符串指针）
                return entry->value;
            }
        }
        entry = entry->next;
    }
    
    return NULL;
}
