#ifndef RUNTIME_APPLICATION_SERVICE_H
#define RUNTIME_APPLICATION_SERVICE_H

#include "../queries/status_queries.h"
#include "../dtos/status_dtos.h"
#include "../dtos/result_dtos.h"
#include <stdint.h>
#include <stdbool.h>

// 前向声明
struct RuntimeApplicationService;
typedef struct RuntimeApplicationService RuntimeApplicationService;

// 运行时应用服务接口 - 负责运行时的整体管理
typedef struct {
    // 运行时生命周期管理
    OperationResultDTO* (*initialize_runtime)(RuntimeApplicationService* service);
    OperationResultDTO* (*shutdown_runtime)(RuntimeApplicationService* service);
    OperationResultDTO* (*restart_runtime)(RuntimeApplicationService* service);

    // 运行时状态查询
    RuntimeStatusDTO* (*get_runtime_status)(RuntimeApplicationService* service, const GetRuntimeStatusQuery* query);
    SystemStatusDTO* (*get_system_status)(RuntimeApplicationService* service);
    MemoryStatusDTO* (*get_memory_status)(RuntimeApplicationService* service);
    ConcurrencyStatusDTO* (*get_concurrency_status)(RuntimeApplicationService* service);
    GCStatusDTO* (*get_gc_status)(RuntimeApplicationService* service);

    // 运行时配置管理
    OperationResultDTO* (*update_configuration)(RuntimeApplicationService* service, const char* config_json);
    char* (*get_configuration)(RuntimeApplicationService* service);
    OperationResultDTO* (*validate_configuration)(RuntimeApplicationService* service, const char* config_json);

    // 运行时监控和诊断
    OperationResultDTO* (*enable_monitoring)(RuntimeApplicationService* service);
    OperationResultDTO* (*disable_monitoring)(RuntimeApplicationService* service);
    OperationResultDTO* (*generate_diagnostic_report)(RuntimeApplicationService* service);

    // 运行时资源管理
    OperationResultDTO* (*allocate_resources)(RuntimeApplicationService* service, const char* resource_request);
    OperationResultDTO* (*release_resources)(RuntimeApplicationService* service, uint64_t resource_id);
    OperationResultDTO* (*get_resource_usage)(RuntimeApplicationService* service);

    // 运行时扩展管理
    OperationResultDTO* (*load_extension)(RuntimeApplicationService* service, const char* extension_path);
    OperationResultDTO* (*unload_extension)(RuntimeApplicationService* service, const char* extension_name);
    char* (*list_extensions)(RuntimeApplicationService* service);
} RuntimeApplicationServiceInterface;

// 运行时应用服务实现
struct RuntimeApplicationService {
    RuntimeApplicationServiceInterface* vtable;  // 虚函数表

    // 依赖的领域服务和基础设施
    void* task_service;             // 任务应用服务
    void* coroutine_service;        // 协程应用服务
    void* channel_service;          // 通道应用服务
    void* async_service;            // 异步应用服务
    void* gc_controller;            // GC控制器
    void* config_manager;           // 配置管理器
    void* monitoring_system;        // 监控系统
    void* extension_manager;        // 扩展管理器

    // 应用服务状态
    bool initialized;
    char service_name[256];
    time_t started_at;
    char version[32];
    char status[32];
};

// 构造函数和析构函数
RuntimeApplicationService* runtime_application_service_create(void);
void runtime_application_service_destroy(RuntimeApplicationService* service);

// 初始化和清理
bool runtime_application_service_initialize(RuntimeApplicationService* service,
                                          void* task_service,
                                          void* coroutine_service,
                                          void* channel_service,
                                          void* async_service,
                                          void* gc_controller,
                                          void* config_manager,
                                          void* monitoring_system,
                                          void* extension_manager);
void runtime_application_service_cleanup(RuntimeApplicationService* service);

// 服务状态查询
bool runtime_application_service_is_healthy(const RuntimeApplicationService* service);
const char* runtime_application_service_get_name(const RuntimeApplicationService* service);
const char* runtime_application_service_get_version(const RuntimeApplicationService* service);
const char* runtime_application_service_get_status(const RuntimeApplicationService* service);
time_t runtime_application_service_get_started_at(const RuntimeApplicationService* service);

// 便捷函数（直接调用虚函数表）
static inline RuntimeStatusDTO* runtime_application_service_get_runtime_status(
    RuntimeApplicationService* service, const GetRuntimeStatusQuery* query) {
    return service->vtable->get_runtime_status(service, query);
}

static inline OperationResultDTO* runtime_application_service_initialize_runtime(
    RuntimeApplicationService* service) {
    return service->vtable->initialize_runtime(service);
}

static inline OperationResultDTO* runtime_application_service_shutdown_runtime(
    RuntimeApplicationService* service) {
    return service->vtable->shutdown_runtime(service);
}

static inline SystemStatusDTO* runtime_application_service_get_system_status(
    RuntimeApplicationService* service) {
    return service->vtable->get_system_status(service);
}

#endif // RUNTIME_APPLICATION_SERVICE_H
