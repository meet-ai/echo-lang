#ifndef RUNTIME_API_H
#define RUNTIME_API_H

#include <stdint.h>
#include <stdbool.h>
#include <stddef.h>

// 前向声明
struct RuntimeHandle;
typedef struct RuntimeHandle RuntimeHandle;

// 运行时配置结构体
typedef struct {
    size_t heap_size;              // 堆大小
    size_t stack_size;             // 默认栈大小
    uint32_t max_tasks;            // 最大任务数
    uint32_t max_coroutines;       // 最大协程数
    bool enable_gc;                // 启用GC
    bool enable_profiling;         // 启用性能分析
    char* log_level;               // 日志级别
} RuntimeConfig;

// 任务句柄
typedef struct TaskHandle {
    uint64_t id;
    void* internal_data;
} TaskHandle;

// 协程句柄
typedef struct CoroutineHandle {
    uint64_t id;
    void* internal_data;
} CoroutineHandle;

// Future句柄
typedef struct FutureHandle {
    uint64_t id;
    void* internal_data;
} FutureHandle;

// 通道句柄
typedef struct ChannelHandle {
    uint64_t id;
    void* internal_data;
} ChannelHandle;

// 运行时API函数

// ============================================================================
// 运行时生命周期管理
// ============================================================================

/**
 * @brief 初始化运行时
 * @param config 运行时配置
 * @return 运行时句柄，失败返回NULL
 */
RuntimeHandle* runtime_init(const RuntimeConfig* config);

/**
 * @brief 关闭运行时
 * @param runtime 运行时句柄
 */
void runtime_shutdown(RuntimeHandle* runtime);

/**
 * @brief 获取运行时版本
 * @return 版本字符串
 */
const char* runtime_get_version(void);

/**
 * @brief 获取运行时状态信息
 * @param runtime 运行时句柄
 * @param json_buffer 输出JSON缓冲区
 * @param buffer_size 缓冲区大小
 * @return 成功返回true
 */
bool runtime_get_status(RuntimeHandle* runtime, char* json_buffer, size_t buffer_size);

// ============================================================================
// 任务管理API
// ============================================================================

/**
 * @brief 创建任务
 * @param runtime 运行时句柄
 * @param function 任务函数
 * @param arg 任务参数
 * @param stack_size 栈大小（0表示使用默认）
 * @return 任务句柄，失败返回NULL
 */
TaskHandle* runtime_create_task(RuntimeHandle* runtime,
                               void (*function)(void*),
                               void* arg,
                               size_t stack_size);

/**
 * @brief 启动任务
 * @param runtime 运行时句柄
 * @param task 任务句柄
 * @return 成功返回true
 */
bool runtime_start_task(RuntimeHandle* runtime, TaskHandle* task);

/**
 * @brief 等待任务完成
 * @param runtime 运行时句柄
 * @param task 任务句柄
 * @param timeout_ms 超时时间（毫秒），0表示无限等待
 * @return 成功返回true，超时返回false
 */
bool runtime_wait_task(RuntimeHandle* runtime, TaskHandle* task, uint32_t timeout_ms);

/**
 * @brief 取消任务
 * @param runtime 运行时句柄
 * @param task 任务句柄
 * @return 成功返回true
 */
bool runtime_cancel_task(RuntimeHandle* runtime, TaskHandle* task);

/**
 * @brief 获取任务状态
 * @param runtime 运行时句柄
 * @param task 任务句柄
 * @param status_buffer 输出状态缓冲区（JSON格式）
 * @param buffer_size 缓冲区大小
 * @return 成功返回true
 */
bool runtime_get_task_status(RuntimeHandle* runtime, TaskHandle* task,
                           char* status_buffer, size_t buffer_size);

/**
 * @brief 销毁任务句柄
 * @param runtime 运行时句柄
 * @param task 任务句柄
 */
void runtime_destroy_task(RuntimeHandle* runtime, TaskHandle* task);

// ============================================================================
// 协程管理API
// ============================================================================

/**
 * @brief 创建协程
 * @param runtime 运行时句柄
 * @param function 协程函数
 * @param arg 协程参数
 * @param stack_size 栈大小（0表示使用默认）
 * @return 协程句柄，失败返回NULL
 */
CoroutineHandle* runtime_create_coroutine(RuntimeHandle* runtime,
                                        void (*function)(void*),
                                        void* arg,
                                        size_t stack_size);

/**
 * @brief 启动协程
 * @param runtime 运行时句柄
 * @param coroutine 协程句柄
 * @return 成功返回true
 */
bool runtime_start_coroutine(RuntimeHandle* runtime, CoroutineHandle* coroutine);

/**
 * @brief 恢复协程执行
 * @param runtime 运行时句柄
 * @param coroutine 协程句柄
 * @return 成功返回true
 */
bool runtime_resume_coroutine(RuntimeHandle* runtime, CoroutineHandle* coroutine);

/**
 * @brief 让出协程控制权
 * @param runtime 运行时句柄
 * @return 成功返回true
 */
bool runtime_yield_coroutine(RuntimeHandle* runtime);

/**
 * @brief 获取协程状态
 * @param runtime 运行时句柄
 * @param coroutine 协程句柄
 * @param status_buffer 输出状态缓冲区（JSON格式）
 * @param buffer_size 缓冲区大小
 * @return 成功返回true
 */
bool runtime_get_coroutine_status(RuntimeHandle* runtime, CoroutineHandle* coroutine,
                                char* status_buffer, size_t buffer_size);

/**
 * @brief 销毁协程句柄
 * @param runtime 运行时句柄
 * @param coroutine 协程句柄
 */
void runtime_destroy_coroutine(RuntimeHandle* runtime, CoroutineHandle* coroutine);

// ============================================================================
// 异步编程API (Future/Promise)
// ============================================================================

/**
 * @brief 创建Future
 * @param runtime 运行时句柄
 * @return Future句柄，失败返回NULL
 */
FutureHandle* runtime_create_future(RuntimeHandle* runtime);

/**
 * @brief 解析Future（设置结果）
 * @param runtime 运行时句柄
 * @param future Future句柄
 * @param result 结果数据
 * @param result_size 结果大小
 * @return 成功返回true
 */
bool runtime_resolve_future(RuntimeHandle* runtime, FutureHandle* future,
                          const void* result, size_t result_size);

/**
 * @brief 拒绝Future（设置错误）
 * @param runtime 运行时句柄
 * @param future Future句柄
 * @param error_message 错误信息
 * @return 成功返回true
 */
bool runtime_reject_future(RuntimeHandle* runtime, FutureHandle* future,
                         const char* error_message);

/**
 * @brief 等待Future完成
 * @param runtime 运行时句柄
 * @param future Future句柄
 * @param timeout_ms 超时时间（毫秒），0表示无限等待
 * @return 成功返回true，超时返回false
 */
bool runtime_await_future(RuntimeHandle* runtime, FutureHandle* future, uint32_t timeout_ms);

/**
 * @brief 获取Future结果
 * @param runtime 运行时句柄
 * @param future Future句柄
 * @param result_buffer 输出结果缓冲区
 * @param buffer_size 缓冲区大小
 * @param result_size 输出实际结果大小
 * @return 成功返回true
 */
bool runtime_get_future_result(RuntimeHandle* runtime, FutureHandle* future,
                             void* result_buffer, size_t buffer_size, size_t* result_size);

/**
 * @brief 获取Future状态
 * @param runtime 运行时句柄
 * @param future Future句柄
 * @param status_buffer 输出状态缓冲区（JSON格式）
 * @param buffer_size 缓冲区大小
 * @return 成功返回true
 */
bool runtime_get_future_status(RuntimeHandle* runtime, FutureHandle* future,
                             char* status_buffer, size_t buffer_size);

/**
 * @brief 销毁Future句柄
 * @param runtime 运行时句柄
 * @param future Future句柄
 */
void runtime_destroy_future(RuntimeHandle* runtime, FutureHandle* future);

// ============================================================================
// 并发原语API
// ============================================================================

/**
 * @brief 创建互斥锁
 * @param runtime 运行时句柄
 * @return 互斥锁句柄，失败返回NULL
 */
void* runtime_create_mutex(RuntimeHandle* runtime);

/**
 * @brief 销毁互斥锁
 * @param runtime 运行时句柄
 * @param mutex 互斥锁句柄
 */
void runtime_destroy_mutex(RuntimeHandle* runtime, void* mutex);

/**
 * @brief 锁定互斥锁
 * @param runtime 运行时句柄
 * @param mutex 互斥锁句柄
 * @return 成功返回true
 */
bool runtime_lock_mutex(RuntimeHandle* runtime, void* mutex);

/**
 * @brief 尝试锁定互斥锁
 * @param runtime 运行时句柄
 * @param mutex 互斥锁句柄
 * @return 成功返回true
 */
bool runtime_try_lock_mutex(RuntimeHandle* runtime, void* mutex);

/**
 * @brief 解锁互斥锁
 * @param runtime 运行时句柄
 * @param mutex 互斥锁句柄
 * @return 成功返回true
 */
bool runtime_unlock_mutex(RuntimeHandle* runtime, void* mutex);

/**
 * @brief 创建信号量
 * @param runtime 运行时句柄
 * @param initial_value 初始值
 * @return 信号量句柄，失败返回NULL
 */
void* runtime_create_semaphore(RuntimeHandle* runtime, uint32_t initial_value);

/**
 * @brief 销毁信号量
 * @param runtime 运行时句柄
 * @param semaphore 信号量句柄
 */
void runtime_destroy_semaphore(RuntimeHandle* runtime, void* semaphore);

/**
 * @brief 等待信号量
 * @param runtime 运行时句柄
 * @param semaphore 信号量句柄
 * @return 成功返回true
 */
bool runtime_wait_semaphore(RuntimeHandle* runtime, void* semaphore);

/**
 * @brief 尝试等待信号量
 * @param runtime 运行时句柄
 * @param semaphore 信号量句柄
 * @return 成功返回true
 */
bool runtime_try_wait_semaphore(RuntimeHandle* runtime, void* semaphore);

/**
 * @brief 发布信号量
 * @param runtime 运行时句柄
 * @param semaphore 信号量句柄
 * @return 成功返回true
 */
bool runtime_post_semaphore(RuntimeHandle* runtime, void* semaphore);

// ============================================================================
// 通道API (CSP风格通信)
// ============================================================================

/**
 * @brief 创建通道
 * @param runtime 运行时句柄
 * @param element_size 元素大小
 * @param buffer_size 缓冲区大小（0表示无缓冲）
 * @return 通道句柄，失败返回NULL
 */
ChannelHandle* runtime_create_channel(RuntimeHandle* runtime, size_t element_size, size_t buffer_size);

/**
 * @brief 发送数据到通道
 * @param runtime 运行时句柄
 * @param channel 通道句柄
 * @param data 数据指针
 * @param timeout_ms 超时时间（毫秒），0表示无限等待
 * @return 成功返回true，超时返回false
 */
bool runtime_send_channel(RuntimeHandle* runtime, ChannelHandle* channel,
                        const void* data, uint32_t timeout_ms);

/**
 * @brief 从通道接收数据
 * @param runtime 运行时句柄
 * @param channel 通道句柄
 * @param buffer 接收缓冲区
 * @param buffer_size 缓冲区大小
 * @param timeout_ms 超时时间（毫秒），0表示无限等待
 * @return 成功返回true，超时返回false
 */
bool runtime_receive_channel(RuntimeHandle* runtime, ChannelHandle* channel,
                           void* buffer, size_t buffer_size, uint32_t timeout_ms);

/**
 * @brief 关闭通道
 * @param runtime 运行时句柄
 * @param channel 通道句柄
 */
void runtime_close_channel(RuntimeHandle* runtime, ChannelHandle* channel);

/**
 * @brief 获取通道状态
 * @param runtime 运行时句柄
 * @param channel 通道句柄
 * @param status_buffer 输出状态缓冲区（JSON格式）
 * @param buffer_size 缓冲区大小
 * @return 成功返回true
 */
bool runtime_get_channel_status(RuntimeHandle* runtime, ChannelHandle* channel,
                              char* status_buffer, size_t buffer_size);

/**
 * @brief 销毁通道句柄
 * @param runtime 运行时句柄
 * @param channel 通道句柄
 */
void runtime_destroy_channel(RuntimeHandle* runtime, ChannelHandle* channel);

// ============================================================================
// 内存管理API
// ============================================================================

/**
 * @brief 分配内存
 * @param runtime 运行时句柄
 * @param size 分配大小
 * @param source 分配源（用于调试）
 * @return 内存指针，失败返回NULL
 */
void* runtime_allocate_memory(RuntimeHandle* runtime, size_t size, const char* source);

/**
 * @brief 重新分配内存
 * @param runtime 运行时句柄
 * @param ptr 原内存指针
 * @param new_size 新大小
 * @param source 分配源
 * @return 新内存指针，失败返回NULL
 */
void* runtime_reallocate_memory(RuntimeHandle* runtime, void* ptr, size_t new_size, const char* source);

/**
 * @brief 释放内存
 * @param runtime 运行时句柄
 * @param ptr 内存指针
 */
void runtime_free_memory(RuntimeHandle* runtime, void* ptr);

/**
 * @brief 获取内存统计信息
 * @param runtime 运行时句柄
 * @param stats_buffer 输出统计缓冲区（JSON格式）
 * @param buffer_size 缓冲区大小
 * @return 成功返回true
 */
bool runtime_get_memory_stats(RuntimeHandle* runtime, char* stats_buffer, size_t buffer_size);

// ============================================================================
// 错误处理API
// ============================================================================

/**
 * @brief 获取最后错误信息
 * @param runtime 运行时句柄
 * @return 错误信息字符串
 */
const char* runtime_get_last_error(RuntimeHandle* runtime);

/**
 * @brief 获取错误代码
 * @param runtime 运行时句柄
 * @return 错误代码
 */
int runtime_get_last_error_code(RuntimeHandle* runtime);

/**
 * @brief 清除错误状态
 * @param runtime 运行时句柄
 */
void runtime_clear_error(RuntimeHandle* runtime);

// ============================================================================
// 工具函数
// ============================================================================

/**
 * @brief 睡眠指定毫秒数
 * @param runtime 运行时句柄
 * @param milliseconds 毫秒数
 */
void runtime_sleep_ms(RuntimeHandle* runtime, uint32_t milliseconds);

/**
 * @brief 获取当前时间戳（毫秒）
 * @param runtime 运行时句柄
 * @return 时间戳
 */
uint64_t runtime_current_time_ms(RuntimeHandle* runtime);

/**
 * @brief 生成随机数
 * @param runtime 运行时句柄
 * @param min 最小值
 * @param max 最大值
 * @return 随机数
 */
uint64_t runtime_random_uint64(RuntimeHandle* runtime, uint64_t min, uint64_t max);

#endif // RUNTIME_API_H
