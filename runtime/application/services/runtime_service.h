/**
 * @file runtime_service.h
 * @brief Runtime应用服务接口
 *
 * 定义Runtime应用服务的主要接口，协调各个上下文的用例执行
 */

#ifndef RUNTIME_SERVICE_H
#define RUNTIME_SERVICE_H

#include <stdint.h>
#include <stdbool.h>

// 前向声明
struct runtime_config_t;
struct coroutine_handle_t;
struct task_handle_t;
struct network_connection_t;
struct file_handle_t;
struct timer_handle_t;

// Runtime配置
typedef struct runtime_config_t {
    uint32_t max_coroutines;      // 最大协程数量
    uint32_t stack_size_kb;       // 默认栈大小（KB）
    uint32_t gc_threshold_mb;     // GC触发阈值（MB）
    bool enable_network;          // 是否启用网络
    bool enable_filesystem;       // 是否启用文件系统
    const char* log_level;        // 日志级别
} runtime_config_t;

// Runtime应用服务接口
typedef struct runtime_service_t {
    // 初始化和销毁
    bool (*init)(struct runtime_service_t* self, const runtime_config_t* config);
    void (*shutdown)(struct runtime_service_t* self);

    // 协程管理
    struct coroutine_handle_t* (*create_coroutine)(struct runtime_service_t* self,
                                                   void (*entry)(void*), void* arg);
    bool (*resume_coroutine)(struct runtime_service_t* self,
                            struct coroutine_handle_t* coroutine);
    bool (*suspend_coroutine)(struct runtime_service_t* self,
                             struct coroutine_handle_t* coroutine);
    void (*destroy_coroutine)(struct runtime_service_t* self,
                             struct coroutine_handle_t* coroutine);

    // 任务调度
    struct task_handle_t* (*schedule_task)(struct runtime_service_t* self,
                                          void (*task_func)(void*), void* arg);
    bool (*cancel_task)(struct runtime_service_t* self,
                       struct task_handle_t* task);
    bool (*wait_task)(struct runtime_service_t* self,
                     struct task_handle_t* task, uint64_t timeout_ms);

    // 网络I/O
    struct network_connection_t* (*connect_network)(struct runtime_service_t* self,
                                                   const char* host, uint16_t port);
    bool (*send_data)(struct runtime_service_t* self,
                     struct network_connection_t* conn,
                     const void* data, size_t size);
    bool (*receive_data)(struct runtime_service_t* self,
                        struct network_connection_t* conn,
                        void* buffer, size_t* size, uint64_t timeout_ms);
    void (*close_connection)(struct runtime_service_t* self,
                            struct network_connection_t* conn);

    // 文件I/O
    struct file_handle_t* (*open_file)(struct runtime_service_t* self,
                                      const char* path, const char* mode);
    bool (*read_file)(struct runtime_service_t* self,
                     struct file_handle_t* file,
                     void* buffer, size_t size, size_t* read_size);
    bool (*write_file)(struct runtime_service_t* self,
                      struct file_handle_t* file,
                      const void* data, size_t size);
    void (*close_file)(struct runtime_service_t* self,
                      struct file_handle_t* file);

    // 定时器
    struct timer_handle_t* (*set_timer)(struct runtime_service_t* self,
                                       uint64_t delay_ms,
                                       void (*callback)(void*), void* arg);
    bool (*cancel_timer)(struct runtime_service_t* self,
                        struct timer_handle_t* timer);

    // 内存管理
    void* (*allocate_memory)(struct runtime_service_t* self, size_t size);
    void (*free_memory)(struct runtime_service_t* self, void* ptr);
    bool (*trigger_gc)(struct runtime_service_t* self);

    // 调试和监控
    bool (*get_stats)(struct runtime_service_t* self, void* stats_buffer, size_t* size);
    bool (*set_log_level)(struct runtime_service_t* self, const char* level);

    // 私有数据
    void* private_data;
} runtime_service_t;

// 全局Runtime服务实例
extern runtime_service_t* global_runtime_service;

// 便利函数
runtime_service_t* runtime_service_create(void);
void runtime_service_destroy(runtime_service_t* service);

// 快速启动函数
bool runtime_quick_start(const runtime_config_t* config);
void runtime_quick_shutdown(void);

#endif // RUNTIME_SERVICE_H
