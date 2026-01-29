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
 * @brief 取消任务（旧版本，需要 RuntimeHandle，保留用于向后兼容）
 * @param runtime 运行时句柄
 * @param task 任务句柄
 * @return 成功返回true
 * 
 * 注意：标准库应使用新的 runtime_cancel_task(int32_t) 函数（不需要 RuntimeHandle）
 */
bool runtime_cancel_task_with_handle(RuntimeHandle* runtime, TaskHandle* task);

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
 * @brief 创建互斥锁（旧版本，需要 RuntimeHandle，保留用于向后兼容）
 * @param runtime 运行时句柄
 * @return 互斥锁句柄，失败返回NULL
 * 
 * 注意：标准库应使用新的 runtime_create_mutex() 函数（不需要 RuntimeHandle）
 */
void* runtime_create_mutex_with_runtime(RuntimeHandle* runtime);

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

// ============================================================================
// P0 优先级运行时函数（标准库核心依赖）
// ============================================================================
// 这些函数是标准库的核心依赖，必须优先实现
// 注意：这些函数不需要 RuntimeHandle，可以直接调用

/**
 * @brief 获取当前时间（Unix 时间戳，毫秒）
 * @return 时间戳（毫秒）
 * 
 * 编译器签名：i64 runtime_time_now_ms()
 * 使用位置：stdlib/time/time.eo - Time.now()
 */
int64_t runtime_time_now_ms(void);

// ==================== 时间格式化与解析 ====================

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
 * 
 * 支持的格式说明符（类似Go的time包）：
 * - 2006-01-02 15:04:05 -> %Y-%m-%d %H:%M:%S
 * - 2006/01/02 -> %Y/%m/%d
 * - 15:04:05 -> %H:%M:%S
 * 
 * 注意：layout参数使用Go风格的时间格式（2006-01-02 15:04:05），
 * 需要转换为C的strftime格式说明符
 */
char* runtime_time_format(int64_t sec, int32_t nsec, const char* layout);

/**
 * @brief 时间解析结果结构体
 * 用于返回时间解析结果
 */
typedef struct {
    bool success;   // 解析是否成功
    int64_t sec;    // Unix时间戳（秒）
    int32_t nsec;   // 纳秒偏移（0-999999999）
} TimeParseResult;

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
 * 
 * 支持的格式（类似Go的time包）：
 * - 2006-01-02 15:04:05
 * - 2006/01/02
 * - 15:04:05
 */
TimeParseResult* runtime_time_parse(const char* layout, const char* value);

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
uint64_t runtime_get_random_seed(void);

/**
 * @brief 计算字符串长度（UTF-8 编码）
 * @param s 字符串指针（i8*）
 * @return 字符数（不是字节数）
 * 
 * 编译器签名：i32 runtime_string_len_utf8(i8* s)
 * 使用位置：stdlib/string/string.eo - string.len()
 */
int32_t runtime_string_len_utf8(const char* s);

/**
 * @brief 计算字符串哈希值（FNV-1a 算法）
 * @param s 字符串指针（i8*）
 * @return 哈希值（u64）
 * 
 * 编译器签名：i64 runtime_string_hash(i8* s)
 * 使用位置：stdlib/core/primitives.eo - string.hash()
 */
int64_t runtime_string_hash(const char* s);

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
int64_t runtime_float_hash(double f);

/**
 * @brief 检查字符串是否包含子字符串
 * @param s 主字符串指针（i8*）
 * @param sub 子字符串指针（i8*）
 * @return bool -> i1（true 表示包含，false 表示不包含）
 * 
 * 编译器签名：i1 runtime_string_contains(i8* s, i8* sub)
 * 使用位置：stdlib/string/string.eo - string.contains()
 */
bool runtime_string_contains(const char* s, const char* sub);

/**
 * @brief 检查字符串是否以指定字符串开头
 * @param s 主字符串指针（i8*）
 * @param prefix 前缀字符串指针（i8*）
 * @return bool -> i1（true 表示以prefix开头，false 表示不是）
 * 
 * 编译器签名：i1 runtime_string_starts_with(i8* s, i8* prefix)
 * 使用位置：stdlib/string/string.eo - string.starts_with()
 */
bool runtime_string_starts_with(const char* s, const char* prefix);

/**
 * @brief 检查字符串是否以指定字符串结尾
 * @param s 主字符串指针（i8*）
 * @param suffix 后缀字符串指针（i8*）
 * @return bool -> i1（true 表示以suffix结尾，false 表示不是）
 * 
 * 编译器签名：i1 runtime_string_ends_with(i8* s, i8* suffix)
 * 使用位置：stdlib/string/string.eo - string.ends_with()
 */
bool runtime_string_ends_with(const char* s, const char* suffix);

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
bool runtime_string_equals(const char* s1, const char* s2);

/**
 * @brief 去除字符串首尾空白字符
 * @param s 字符串指针（i8*）
 * @return 新字符串指针（i8*），需要调用者释放内存
 * 
 * 编译器签名：i8* runtime_string_trim(i8* s)
 * 使用位置：stdlib/string/string.eo - string.trim()
 * 
 * 注意：返回的字符串是新分配的内存，需要由调用者或GC管理
 */
char* runtime_string_trim(const char* s);

/**
 * @brief 将字符串转换为大写
 * @param s 字符串指针（i8*）
 * @return 新字符串指针（i8*），需要调用者释放内存
 * 
 * 编译器签名：i8* runtime_string_to_upper(i8* s)
 * 使用位置：stdlib/string/string.eo - string.to_upper()
 * 
 * 注意：返回的字符串是新分配的内存，需要由调用者或GC管理
 */
char* runtime_string_to_upper(const char* s);

/**
 * @brief 将字符串转换为小写
 * @param s 字符串指针（i8*）
 * @return 新字符串指针（i8*），需要调用者释放内存
 * 
 * 编译器签名：i8* runtime_string_to_lower(i8* s)
 * 使用位置：stdlib/string/string.eo - string.to_lower()
 * 
 * 注意：返回的字符串是新分配的内存，需要由调用者或GC管理
 */
char* runtime_string_to_lower(const char* s);

/**
 * @brief 字符串到字节数组转换结果结构体
 */
typedef struct {
    uint8_t* bytes;  // 字节数组指针
    int32_t len;     // 字节数组长度
} StringToBytesResult;

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
StringToBytesResult* runtime_string_to_bytes(const char* s);

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
char* runtime_bytes_to_string(const uint8_t* bytes, int32_t len);

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
char* runtime_base64_encode_string(const char* s);

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
char* runtime_f64_to_string(double f);

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
double runtime_string_to_f64(const char* s);

/**
 * @brief 字符串类型（string_t）
 * 用于Echo语言的string类型
 */
#include "../shared/types/common_types.h"

/**
 * @brief 字符串分割结构体
 * 用于返回字符串分割结果
 */
typedef struct {
    char** strings;    // 字符串指针数组
    int32_t count;     // 字符串数量
} StringSplitResult;

/**
 * @brief 分割字符串
 * @param s 字符串指针（i8*）
 * @param delimiter 分隔符字符串指针（i8*）
 * @return 字符串分割结果结构体指针（需要调用者释放内存）
 * 
 * 编译器签名：i8* runtime_string_split(i8* s, i8* delimiter)
 * 使用位置：stdlib/string/string.eo - string.split()
 * 
 * 注意：
 * - 返回的结构体包含字符串指针数组和数量
 * - 所有分配的内存（结构体、字符串数组、每个字符串）都需要由调用者或GC管理
 * - 如果失败返回 NULL
 */
StringSplitResult* runtime_string_split(const char* s, const char* delimiter);

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
string_t runtime_char_ptr_to_string(char* ptr);

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
 * - 创建切片结构（在 LLVM IR 中，[]string 是 *i8）
 * - 返回切片指针
 * 
 * 注意：
 * - 如果 ptrs 为 NULL 或 count < 0，返回 NULL
 * - 返回的切片指针指向原始数据，不复制（零成本）
 * - 调用者需要确保 ptrs 指向的数据在切片使用期间有效
 */
void* runtime_char_ptr_array_to_string_slice(char** ptrs, int32_t count);

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
 */
void* runtime_realloc_slice(void* ptr, int32_t old_cap, int32_t new_cap, int32_t elem_size);

/**
 * @brief 分配新切片内存
 * @param cap 初始容量（i32）
 * @param elem_size 元素大小（字节）（i32）
 * @return 新分配的切片数据指针（i8*），失败返回 NULL
 * 
 * 编译器签名：i8* runtime_alloc_slice(i32 cap, i32 elem_size)
 * 使用位置：stdlib/collections/vec.eo - 初始化
 */
void* runtime_alloc_slice(int32_t cap, int32_t elem_size);

/**
 * @brief 释放切片内存
 * @param ptr 切片数据指针（i8*）
 * 
 * 编译器签名：void runtime_free_slice(i8* ptr)
 * 使用位置：stdlib/collections/vec.eo - 清理
 */
void runtime_free_slice(void* ptr);

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
void* runtime_slice_from_ptr_len(void* ptr, int32_t len);

/**
 * @brief 获取切片长度
 * @param slice_ptr 切片数据指针（i8*）
 * @return 切片长度（i32）
 * 
 * 编译器签名：i32 runtime_slice_len(i8* slice_ptr)
 */
int32_t runtime_slice_len(void* slice_ptr);

// ============================================================================
// P1 优先级运行时函数（标准库重要依赖）
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
void* runtime_open_file(const char* path);

/**
 * @brief 创建文件
 * @param path 文件路径（i8*）
 * @return Result[FileHandle, string] -> i8*
 * 
 * 编译器签名：i8* runtime_create_file(i8* path)
 * 使用位置：stdlib/io/file.eo - File.create()
 */
void* runtime_create_file(const char* path);

/**
 * @brief 关闭文件
 * @param handle 文件句柄（i32）
 * @return Result[void, string] -> i8*
 * 
 * 编译器签名：i8* runtime_close_file(i32 handle)
 * 使用位置：stdlib/io/file.eo - File.close()
 */
void* runtime_close_file(int32_t handle);

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
void* runtime_read_file(int32_t handle, void* buf_ptr, int32_t buf_len);

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
void* runtime_write_file(int32_t handle, const void* buf_ptr, int32_t buf_len);

/**
 * @brief 刷新文件缓冲区
 * @param handle 文件句柄（i32）
 * @return Result[void, string] -> i8*
 * 
 * 编译器签名：i8* runtime_flush_file(i32 handle)
 * 使用位置：stdlib/io/file.eo - File.flush()
 */
void* runtime_flush_file(int32_t handle);

// ==================== 同步原语 ====================

/**
 * @brief 创建互斥锁
 * @return MutexHandle -> i32
 * 
 * 编译器签名：i32 runtime_create_mutex()
 * 使用位置：stdlib/sync/mutex.eo - Mutex.new()
 */
int32_t runtime_create_mutex(void);

/**
 * @brief 锁定互斥锁（阻塞）
 * @param handle 互斥锁句柄（i32）
 * 
 * 编译器签名：void runtime_mutex_lock(i32 handle)
 * 使用位置：stdlib/sync/mutex.eo - Mutex.lock()
 */
void runtime_mutex_lock(int32_t handle);

/**
 * @brief 尝试锁定互斥锁（非阻塞）
 * @param handle 互斥锁句柄（i32）
 * @return bool -> i1（true 表示成功，false 表示失败）
 * 
 * 编译器签名：i1 runtime_mutex_try_lock(i32 handle)
 * 使用位置：stdlib/sync/mutex.eo - Mutex.try_lock()
 */
bool runtime_mutex_try_lock(int32_t handle);

/**
 * @brief 解锁互斥锁
 * @param handle 互斥锁句柄（i32）
 * 
 * 编译器签名：void runtime_mutex_unlock(i32 handle)
 * 使用位置：stdlib/sync/mutex.eo - Mutex.unlock()
 */
void runtime_mutex_unlock(int32_t handle);

/**
 * @brief 创建读写锁
 * @return RwLockHandle -> i32
 * 
 * 编译器签名：i32 runtime_create_rwlock()
 * 使用位置：stdlib/sync/rwlock.eo - RwLock.new()
 */
int32_t runtime_create_rwlock(void);

/**
 * @brief 获取读锁（阻塞）
 * @param handle 读写锁句柄（i32）
 * 
 * 编译器签名：void runtime_rwlock_read_lock(i32 handle)
 * 使用位置：stdlib/sync/rwlock.eo - RwLock.read_lock()
 */
void runtime_rwlock_read_lock(int32_t handle);

/**
 * @brief 尝试获取读锁（非阻塞）
 * @param handle 读写锁句柄（i32）
 * @return bool -> i1（true 表示成功，false 表示失败）
 * 
 * 编译器签名：i1 runtime_rwlock_try_read_lock(i32 handle)
 * 使用位置：stdlib/sync/rwlock.eo - RwLock.try_read_lock()
 */
bool runtime_rwlock_try_read_lock(int32_t handle);

/**
 * @brief 获取写锁（阻塞）
 * @param handle 读写锁句柄（i32）
 * 
 * 编译器签名：void runtime_rwlock_write_lock(i32 handle)
 * 使用位置：stdlib/sync/rwlock.eo - RwLock.write_lock()
 */
void runtime_rwlock_write_lock(int32_t handle);

/**
 * @brief 尝试获取写锁（非阻塞）
 * @param handle 读写锁句柄（i32）
 * @return bool -> i1（true 表示成功，false 表示失败）
 * 
 * 编译器签名：i1 runtime_rwlock_try_write_lock(i32 handle)
 * 使用位置：stdlib/sync/rwlock.eo - RwLock.try_write_lock()
 */
bool runtime_rwlock_try_write_lock(int32_t handle);

/**
 * @brief 释放读锁
 * @param handle 读写锁句柄（i32）
 * 
 * 编译器签名：void runtime_rwlock_read_unlock(i32 handle)
 * 使用位置：stdlib/sync/rwlock.eo - RwLock.read_unlock()
 */
void runtime_rwlock_read_unlock(int32_t handle);

/**
 * @brief 释放写锁
 * @param handle 读写锁句柄（i32）
 * 
 * 编译器签名：void runtime_rwlock_write_unlock(i32 handle)
 * 使用位置：stdlib/sync/rwlock.eo - RwLock.write_unlock()
 */
void runtime_rwlock_write_unlock(int32_t handle);

// ==================== 原子操作 ====================

/**
 * @brief 原子加载 i32
 * @param ptr 指针（*i32）
 * @return i32
 * 
 * 编译器签名：i32 runtime_atomic_load_i32(*i32 ptr)
 * 使用位置：stdlib/sync/atomic.eo - Atomic.load_i32()
 */
int32_t runtime_atomic_load_i32(int32_t* ptr);

/**
 * @brief 原子存储 i32
 * @param ptr 指针（*i32）
 * @param value 值（i32）
 * 
 * 编译器签名：void runtime_atomic_store_i32(*i32 ptr, i32 value)
 * 使用位置：stdlib/sync/atomic.eo - Atomic.store_i32()
 */
void runtime_atomic_store_i32(int32_t* ptr, int32_t value);

/**
 * @brief 原子交换 i32
 * @param ptr 指针（*i32）
 * @param value 新值（i32）
 * @return i32（旧值）
 * 
 * 编译器签名：i32 runtime_atomic_swap_i32(*i32 ptr, i32 value)
 * 使用位置：stdlib/sync/atomic.eo - Atomic.swap_i32()
 */
int32_t runtime_atomic_swap_i32(int32_t* ptr, int32_t value);

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
bool runtime_atomic_cas_i32(int32_t* ptr, int32_t expected, int32_t desired);

/**
 * @brief 原子加法 i32
 * @param ptr 指针（*i32）
 * @param delta 增量（i32）
 * @return i32（旧值）
 * 
 * 编译器签名：i32 runtime_atomic_add_i32(*i32 ptr, i32 delta)
 * 使用位置：stdlib/sync/atomic.eo - Atomic.add_i32()
 */
int32_t runtime_atomic_add_i32(int32_t* ptr, int32_t delta);

/**
 * @brief 原子加载 i64
 * @param ptr 指针（*i64）
 * @return i64
 * 
 * 编译器签名：i64 runtime_atomic_load_i64(*i64 ptr)
 * 使用位置：stdlib/sync/atomic.eo - Atomic.load_i64()
 */
int64_t runtime_atomic_load_i64(int64_t* ptr);

/**
 * @brief 原子存储 i64
 * @param ptr 指针（*i64）
 * @param value 值（i64）
 * 
 * 编译器签名：void runtime_atomic_store_i64(*i64 ptr, i64 value)
 * 使用位置：stdlib/sync/atomic.eo - Atomic.store_i64()
 */
void runtime_atomic_store_i64(int64_t* ptr, int64_t value);

/**
 * @brief 原子交换 i64
 * @param ptr 指针（*i64）
 * @param value 新值（i64）
 * @return i64（旧值）
 * 
 * 编译器签名：i64 runtime_atomic_swap_i64(*i64 ptr, i64 value)
 * 使用位置：stdlib/sync/atomic.eo - Atomic.swap_i64()
 */
int64_t runtime_atomic_swap_i64(int64_t* ptr, int64_t value);

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
bool runtime_atomic_cas_i64(int64_t* ptr, int64_t expected, int64_t desired);

/**
 * @brief 原子加法 i64
 * @param ptr 指针（*i64）
 * @param delta 增量（i64）
 * @return i64（旧值）
 * 
 * 编译器签名：i64 runtime_atomic_add_i64(*i64 ptr, i64 delta)
 * 使用位置：stdlib/sync/atomic.eo - Atomic.add_i64()
 */
int64_t runtime_atomic_add_i64(int64_t* ptr, int64_t delta);

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
bool runtime_atomic_load_bool(uint8_t* ptr);

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
void runtime_atomic_store_bool(uint8_t* ptr, uint8_t value);

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
bool runtime_atomic_swap_bool(uint8_t* ptr, uint8_t value);

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
bool runtime_atomic_cas_bool(uint8_t* ptr, uint8_t expected, uint8_t desired);

// ==================== 异步运行时 ====================

/**
 * @brief 创建任务
 * @param func_ptr 函数指针（i8*，泛型处理）
 * @param arg_ptr 参数指针（i8*）
 * @return TaskId -> i32
 * 
 * 编译器签名：i32 runtime_spawn_task(i8* func_ptr, i8* arg_ptr)
 * 使用位置：stdlib/async/task.eo - Task.spawn()
 */
int32_t runtime_spawn_task(void* func_ptr, void* arg_ptr);

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
char* runtime_task_status(int32_t task_id);

/**
 * @brief 取消任务
 * @param task_id 任务 ID（i32）
 * @return Result[void, string] -> i8*
 * 
 * 编译器签名：i8* runtime_cancel_task(i32 task_id)
 * 使用位置：stdlib/async/task.eo - Task.cancel()
 */
void* runtime_cancel_task(int32_t task_id);

/**
 * @brief 创建无缓冲通道
 * @return ChannelHandle -> i32
 * 
 * 编译器签名：i32 runtime_create_channel_unbuffered()
 * 使用位置：stdlib/async/channel.eo - Channel.unbuffered()
 */
int32_t runtime_create_channel_unbuffered(void);

/**
 * @brief 创建有缓冲通道
 * @param cap 容量（i32）
 * @return ChannelHandle -> i32
 * 
 * 编译器签名：i32 runtime_create_channel_buffered(i32 cap)
 * 使用位置：stdlib/async/channel.eo - Channel.buffered()
 */
int32_t runtime_create_channel_buffered(int32_t cap);

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
void* runtime_channel_send(int32_t handle, void* item_ptr, int32_t item_size);

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
void* runtime_channel_recv(int32_t handle, void* item_ptr, int32_t item_size);

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
void* runtime_channel_try_send(int32_t handle, void* item_ptr, int32_t item_size);

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
void* runtime_channel_try_recv(int32_t handle, void* item_ptr, int32_t item_size);

/**
 * @brief 关闭通道
 * @param handle 通道句柄（i32）
 * 
 * 编译器签名：void runtime_channel_close(i32 handle)
 * 使用位置：stdlib/async/channel.eo - Channel.close()
 */
void runtime_channel_close(int32_t handle);

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
char* runtime_channel_status(int32_t handle);

// ============================================================================
// P2 优先级运行时函数（增强功能）
// ============================================================================
// 这些函数是标准库的增强功能，包括网络 I/O 和数学函数

// ==================== 网络 I/O ====================

/**
 * @brief 建立 TCP 连接
 * @param addr 地址字符串（格式：host:port）
 * @return Result[SocketHandle, string] -> i8*
 * 
 * 编译器签名：i8* runtime_tcp_connect(i8* addr)
 * 使用位置：stdlib/net/tcp.eo - TcpStream.connect()
 */
void* runtime_tcp_connect(const char* addr);

/**
 * @brief 绑定地址并监听
 * @param addr 地址字符串（格式：host:port）
 * @return Result[SocketHandle, string] -> i8*
 * 
 * 编译器签名：i8* runtime_tcp_bind(i8* addr)
 * 使用位置：stdlib/net/tcp.eo - TcpListener.bind()
 */
void* runtime_tcp_bind(const char* addr);

/**
 * @brief 接受连接
 * @param listener 监听套接字句柄（i32）
 * @return Result[SocketHandle, string] -> i8*
 * 
 * 编译器签名：i8* runtime_tcp_accept(i32 listener)
 * 使用位置：stdlib/net/tcp.eo - TcpListener.accept()
 */
void* runtime_tcp_accept(int32_t listener);

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
void* runtime_tcp_read(int32_t socket, void* buf_ptr, int32_t buf_len);

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
void* runtime_tcp_write(int32_t socket, const void* buf_ptr, int32_t buf_len);

/**
 * @brief 关闭 TCP 连接
 * @param socket 套接字句柄（i32）
 * @return Result[void, string] -> i8*
 * 
 * 编译器签名：i8* runtime_tcp_close(i32 socket)
 * 使用位置：stdlib/net/tcp.eo - TcpStream.close(), TcpListener.close()
 */
void* runtime_tcp_close(int32_t socket);

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
void* runtime_tcp_flush(int32_t socket);

/**
 * @brief 获取 TCP 连接的本地地址
 * @param socket 套接字句柄（i32）
 * @return Result[string, string] -> i8*
 * 
 * 编译器签名：i8* runtime_tcp_local_addr(i32 socket)
 * 使用位置：stdlib/net/tcp.eo - TcpStream.local_addr()
 */
void* runtime_tcp_local_addr(int32_t socket);

/**
 * @brief 获取 TCP 连接的远程地址（对端地址）
 * @param socket 套接字句柄（i32）
 * @return Result[string, string] -> i8*
 * 
 * 编译器签名：i8* runtime_tcp_peer_addr(i32 socket)
 * 使用位置：stdlib/net/tcp.eo - TcpStream.peer_addr()
 */
void* runtime_tcp_peer_addr(int32_t socket);

// ==================== 数学函数 ====================

/**
 * @brief 平方根
 * @param x 输入值（f64）
 * @return f64
 * 
 * 编译器签名：f64 runtime_math_sqrt(f64 x)
 * 使用位置：stdlib/math/math.eo - Math.sqrt()
 */
double runtime_math_sqrt(double x);

/**
 * @brief 立方根
 * @param x 输入值（f64）
 * @return f64
 * 
 * 编译器签名：f64 runtime_math_cbrt(f64 x)
 * 使用位置：stdlib/math/math.eo - Math.cbrt()
 */
double runtime_math_cbrt(double x);

/**
 * @brief 幂运算
 * @param base 底数（f64）
 * @param exp 指数（f64）
 * @return f64
 * 
 * 编译器签名：f64 runtime_math_pow(f64 base, f64 exp)
 * 使用位置：stdlib/math/math.eo - Math.pow()
 */
double runtime_math_pow(double base, double exp);

/**
 * @brief 自然指数（e^x）
 * @param x 输入值（f64）
 * @return f64
 * 
 * 编译器签名：f64 runtime_math_exp(f64 x)
 * 使用位置：stdlib/math/math.eo - Math.exp()
 */
double runtime_math_exp(double x);

/**
 * @brief 自然对数（ln(x)）
 * @param x 输入值（f64）
 * @return f64
 * 
 * 编译器签名：f64 runtime_math_ln(f64 x)
 * 使用位置：stdlib/math/math.eo - Math.ln()
 */
double runtime_math_ln(double x);

/**
 * @brief 常用对数（log10(x)）
 * @param x 输入值（f64）
 * @return f64
 * 
 * 编译器签名：f64 runtime_math_log10(f64 x)
 * 使用位置：stdlib/math/math.eo - Math.log10()
 */
double runtime_math_log10(double x);

/**
 * @brief 二进制对数（log2(x)）
 * @param x 输入值（f64）
 * @return f64
 * 
 * 编译器签名：f64 runtime_math_log2(f64 x)
 * 使用位置：stdlib/math/math.eo - Math.log2()
 */
double runtime_math_log2(double x);

/**
 * @brief 正弦函数
 * @param x 输入值（f64，弧度）
 * @return f64
 * 
 * 编译器签名：f64 runtime_math_sin(f64 x)
 * 使用位置：stdlib/math/math.eo - Math.sin()
 */
double runtime_math_sin(double x);

/**
 * @brief 余弦函数
 * @param x 输入值（f64，弧度）
 * @return f64
 * 
 * 编译器签名：f64 runtime_math_cos(f64 x)
 * 使用位置：stdlib/math/math.eo - Math.cos()
 */
double runtime_math_cos(double x);

/**
 * @brief 正切函数
 * @param x 输入值（f64，弧度）
 * @return f64
 * 
 * 编译器签名：f64 runtime_math_tan(f64 x)
 * 使用位置：stdlib/math/math.eo - Math.tan()
 */
double runtime_math_tan(double x);

/**
 * @brief 反正弦函数
 * @param x 输入值（f64，范围：-1 到 1）
 * @return f64（弧度）
 * 
 * 编译器签名：f64 runtime_math_asin(f64 x)
 * 使用位置：stdlib/math/math.eo - Math.asin()
 */
double runtime_math_asin(double x);

/**
 * @brief 反余弦函数
 * @param x 输入值（f64，范围：-1 到 1）
 * @return f64（弧度）
 * 
 * 编译器签名：f64 runtime_math_acos(f64 x)
 * 使用位置：stdlib/math/math.eo - Math.acos()
 */
double runtime_math_acos(double x);

/**
 * @brief 反正切函数
 * @param x 输入值（f64）
 * @return f64（弧度）
 * 
 * 编译器签名：f64 runtime_math_atan(f64 x)
 * 使用位置：stdlib/math/math.eo - Math.atan()
 */
double runtime_math_atan(double x);

/**
 * @brief 向上取整
 * @param x 输入值（f64）
 * @return f64（向上取整后的值）
 * 
 * 编译器签名：f64 runtime_math_ceil(f64 x)
 * 使用位置：stdlib/math/math.eo - Math.ceil()
 */
double runtime_math_ceil(double x);

/**
 * @brief 向下取整
 * @param x 输入值（f64）
 * @return f64（向下取整后的值）
 * 
 * 编译器签名：f64 runtime_math_floor(f64 x)
 * 使用位置：stdlib/math/math.eo - Math.floor()
 */
double runtime_math_floor(double x);

/**
 * @brief 四舍五入
 * @param x 输入值（f64）
 * @return f64（四舍五入后的值）
 * 
 * 编译器签名：f64 runtime_math_round(f64 x)
 * 使用位置：stdlib/math/math.eo - Math.round()
 */
double runtime_math_round(double x);

/**
 * @brief 截断（向零取整）
 * @param x 输入值（f64）
 * @return f64（截断后的值）
 * 
 * 编译器签名：f64 runtime_math_trunc(f64 x)
 * 使用位置：stdlib/math/math.eo - Math.trunc()
 */
double runtime_math_trunc(double x);

/**
 * @brief 反正切函数（两个参数版本）
 * @param y y坐标（f64）
 * @param x x坐标（f64）
 * @return f64（弧度，范围 [-π, π]）
 * 
 * 编译器签名：f64 runtime_math_atan2(f64 y, f64 x)
 * 使用位置：stdlib/math/math.eo - Math.atan2()
 */
double runtime_math_atan2(double y, double x);

/**
 * @brief 双曲正弦函数
 * @param x 输入值（f64）
 * @return f64（双曲正弦值）
 * 
 * 编译器签名：f64 runtime_math_sinh(f64 x)
 * 使用位置：stdlib/math/math.eo - Math.sinh()
 */
double runtime_math_sinh(double x);

/**
 * @brief 双曲余弦函数
 * @param x 输入值（f64）
 * @return f64（双曲余弦值）
 * 
 * 编译器签名：f64 runtime_math_cosh(f64 x)
 * 使用位置：stdlib/math/math.eo - Math.cosh()
 */
double runtime_math_cosh(double x);

/**
 * @brief 双曲正切函数
 * @param x 输入值（f64）
 * @return f64（双曲正切值）
 * 
 * 编译器签名：f64 runtime_math_tanh(f64 x)
 * 使用位置：stdlib/math/math.eo - Math.tanh()
 */
double runtime_math_tanh(double x);

// ==================== HTTP 和 URL 处理 ====================

/**
 * @brief URL解析结果结构体
 * 用于返回URL解析结果
 */
typedef struct {
    char* scheme;   // 协议（如 "http", "https"）
    char* host;     // 主机名
    int32_t port;   // 端口号（-1表示使用默认端口）
    char* path;     // 路径
    char* query;    // 查询字符串（可选，可能为NULL）
    char* fragment; // 片段（可选，可能为NULL）
    bool success;  // 解析是否成功
} UrlParseResult;

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
 * 
 * 支持的URL格式：
 * - http://example.com:8080/path?query=value#fragment
 * - https://example.com/path
 * - http://example.com
 */
UrlParseResult* runtime_url_parse(const char* url);

/**
 * @brief HTTP请求解析结果结构体
 * 用于返回HTTP请求解析结果
 */
typedef struct {
    char* method;        // HTTP方法（如 "GET", "POST"）
    char* path;          // 请求路径
    char* version;       // HTTP版本（如 "HTTP/1.1"）
    char** header_keys;  // 头部键数组（字符串指针数组）
    char** header_values; // 头部值数组（字符串指针数组）
    int32_t header_count; // 头部数量
    uint8_t* body;      // 请求正文（字节数组）
    int32_t body_len;   // 正文长度
    bool success;        // 解析是否成功
} HttpParseRequestResult;

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
 * 注意：返回的结构体包含解析结果，需要由调用者或GC管理
 * 
 * 支持的HTTP请求格式：
 * - GET /path HTTP/1.1\r\n
 * - Host: example.com\r\n
 * - \r\n
 * - [body]
 */
HttpParseRequestResult* runtime_http_parse_request(const uint8_t* data, int32_t data_len);

/**
 * @brief HTTP响应构建结果结构体
 * 用于返回HTTP响应构建结果
 */
typedef struct {
    uint8_t* data;   // 响应字节数组
    int32_t data_len; // 响应长度
    bool success;     // 构建是否成功
} HttpBuildResponseResult;

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
 * 注意：返回的结构体包含构建结果，需要由调用者或GC管理
 */
HttpBuildResponseResult* runtime_http_build_response(
    int32_t status_code,
    const char* status_text,
    const char** header_keys,
    const char** header_values,
    int32_t header_count,
    const uint8_t* body,
    int32_t body_len
);

/**
 * @brief 返回 NULL 指针
 * @return NULL 指针（void*）
 * 
 * 编译器签名：i8* runtime_null_ptr()
 * 使用位置：stdlib/net/http/server.eo - build_response()
 * 
 * 实现：返回 NULL 指针，用于表示空指针
 * 注意：这是一个辅助函数，用于在Echo语言中表示空指针（因为语言暂不支持 nil 字面量）
 */
void* runtime_null_ptr(void);

/**
 * @brief Map迭代结果结构体
 * 用于返回map的键值对数组
 */
typedef struct {
    char** keys;    // 键数组指针（char**）
    char** values;  // 值数组指针（char**）
    int32_t count; // 键值对数量
} MapIterResult;

/**
 * @brief 获取map的键值对数组
 * @param map_ptr map指针（i8*，不透明指针）
 * @return MapIterResult* 结构体指针，包含键数组、值数组和数量，需要调用者释放内存
 * 
 * 编译器签名：i8* runtime_map_get_keys(i8* map_ptr)
 * 使用位置：编译器生成的map迭代循环代码（for (key, value) in obj）
 * 
 * 实现：从map中提取所有键值对，返回键数组和值数组
 * 注意：
 * - 返回的结构体包含键数组和值数组的指针，需要由调用者或GC管理
 * - 键数组和值数组是新分配的内存，需要释放
 * - 如果map为空，返回count=0的结果
 * - 如果map_ptr为NULL，返回NULL
 * 
 * 内部表示：
 * - map在Echo语言中可能使用hash_table_t或类似结构
 * - 需要遍历所有bucket和entry，收集键值对
 */
MapIterResult* runtime_map_get_keys(void* map_ptr);

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
void runtime_map_set(void* map_ptr, void* key, void* value);

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
void runtime_map_delete(void* map_ptr, void* key);

/**
 * @brief 获取 map[string]string 中的值
 * @param map_ptr map指针（i8*，不透明指针）
 * @param key 键（i8*，字符串）
 * @return 值（i8*，字符串指针），不存在返回 NULL
 *
 * 编译器签名：i8* runtime_map_get_string_string(i8* map_ptr, i8* key)
 * 使用位置：map[string]string 索引访问（map[key]）
 */
void* runtime_map_get_string_string(void* map_ptr, void* key);

/**
 * @brief 设置 map[string]string 中的键值对（与 runtime_map_set 同义，供编译器按类型名选择）
 * 编译器签名：void runtime_map_set_string_string(i8* map_ptr, i8* key, i8* value)
 */
void runtime_map_set_string_string(void* map_ptr, void* key, void* value);

// ============================================================================
// Map多类型支持：map[int]string
// ============================================================================

/**
 * @brief 计算32位整数哈希值
 * @param key 整数值（i32）
 * @return 哈希值（i64）
 * 
 * 编译器签名：i64 runtime_int32_hash(i32 key)
 * 使用位置：map[int]string 等整数键Map的哈希计算
 */
int64_t runtime_int32_hash(int32_t key);

/**
 * @brief 计算64位整数哈希值
 * @param key 整数值（i64）
 * @return 哈希值（i64）
 * 
 * 编译器签名：i64 runtime_int64_hash(i64 key)
 * 使用位置：map[i64]string 等64位整数键Map的哈希计算
 */
int64_t runtime_int64_hash(int64_t key);

/**
 * @brief 设置 map[int]string 中的键值对
 * @param map_ptr map指针（i8*，不透明指针）
 * @param key 键（i32，整数）
 * @param value 值（i8*，字符串）
 * @return void
 * 
 * 编译器签名：void runtime_map_set_int_string(i8* map_ptr, i32 key, i8* value)
 * 使用位置：编译器生成的map索引赋值代码（map[key] = value，其中map是map[int]string类型）
 */
void runtime_map_set_int_string(void* map_ptr, int32_t key, void* value);

/**
 * @brief 获取 map[int]string 中的值
 * @param map_ptr map指针（i8*，不透明指针）
 * @param key 键（i32，整数）
 * @return 值（i8*，字符串指针），如果不存在返回NULL
 * 
 * 编译器签名：i8* runtime_map_get_int_string(i8* map_ptr, i32 key)
 * 使用位置：编译器生成的map索引访问代码（map[key]，其中map是map[int]string类型）
 */
void* runtime_map_get_int_string(void* map_ptr, int32_t key);

/**
 * @brief 删除 map[int]string 中的键值对
 * @param map_ptr map指针（i8*，不透明指针）
 * @param key 键（i32，整数）
 * @return void
 * 
 * 编译器签名：void runtime_map_delete_int_string(i8* map_ptr, i32 key)
 * 使用位置：编译器生成的delete语句代码（delete(map, key)，其中map是map[int]string类型）
 */
void runtime_map_delete_int_string(void* map_ptr, int32_t key);

// ============================================================================
// Map多类型支持：map[string]int
// ============================================================================

/**
 * @brief 设置 map[string]int 中的键值对
 * @param map_ptr map指针（i8*，不透明指针）
 * @param key 键（i8*，字符串）
 * @param value 值（i32，整数）
 * @return void
 * 
 * 编译器签名：void runtime_map_set_string_int(i8* map_ptr, i8* key, i32 value)
 * 使用位置：编译器生成的map索引赋值代码（map[key] = value，其中map是map[string]int类型）
 */
void runtime_map_set_string_int(void* map_ptr, void* key, int32_t value);

/**
 * @brief 获取 map[string]int 中的值
 * @param map_ptr map指针（i8*，不透明指针）
 * @param key 键（i8*，字符串）
 * @return 值（i32*，整数指针），如果不存在返回NULL
 * 
 * 编译器签名：i32* runtime_map_get_string_int(i8* map_ptr, i8* key)
 * 使用位置：编译器生成的map索引访问代码（map[key]，其中map是map[string]int类型）
 */
int32_t* runtime_map_get_string_int(void* map_ptr, void* key);

/**
 * @brief 删除 map[string]int 中的键值对
 * @param map_ptr map指针（i8*，不透明指针）
 * @param key 键（i8*，字符串）
 * @return void
 * 
 * 编译器签名：void runtime_map_delete_string_int(i8* map_ptr, i8* key)
 * 使用位置：编译器生成的delete语句代码（delete(map, key)，其中map是map[string]int类型）
 */
void runtime_map_delete_string_int(void* map_ptr, void* key);

// ============================================================================
// Map多类型支持：map[int]int
// ============================================================================

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
void runtime_map_set_int_int(void* map_ptr, int32_t key, int32_t value);

/**
 * @brief 获取 map[int]int 中的值
 * @param map_ptr map指针（i8*，不透明指针）
 * @param key 键（i32，整数）
 * @return 值（i32*，整数指针），如果不存在返回NULL
 * 
 * 编译器签名：i32* runtime_map_get_int_int(i8* map_ptr, i32 key)
 * 使用位置：编译器生成的map索引访问代码（map[key]，其中map是map[int]int类型）
 */
int32_t* runtime_map_get_int_int(void* map_ptr, int32_t key);

/**
 * @brief 删除 map[int]int 中的键值对
 * @param map_ptr map指针（i8*，不透明指针）
 * @param key 键（i32，整数）
 * @return void
 * 
 * 编译器签名：void runtime_map_delete_int_int(i8* map_ptr, i32 key)
 * 使用位置：编译器生成的delete语句代码（delete(map, key)，其中map是map[int]int类型）
 */
void runtime_map_delete_int_int(void* map_ptr, int32_t key);

// ============================================================================
// Map多类型支持：map[float]string
// ============================================================================

/**
 * @brief 设置 map[float]string 中的键值对
 * @param map_ptr map指针（i8*，不透明指针）
 * @param key 键（f64，浮点数）
 * @param value 值（i8*，字符串）
 * @return void
 * 
 * 编译器签名：void runtime_map_set_float_string(i8* map_ptr, f64 key, i8* value)
 */
void runtime_map_set_float_string(void* map_ptr, double key, void* value);

/**
 * @brief 获取 map[float]string 中的值
 * @param map_ptr map指针（i8*，不透明指针）
 * @param key 键（f64，浮点数）
 * @return 值（i8*，字符串指针），如果不存在返回NULL
 * 
 * 编译器签名：i8* runtime_map_get_float_string(i8* map_ptr, f64 key)
 */
void* runtime_map_get_float_string(void* map_ptr, double key);

/**
 * @brief 删除 map[float]string 中的键值对
 * @param map_ptr map指针（i8*，不透明指针）
 * @param key 键（f64，浮点数）
 * @return void
 * 
 * 编译器签名：void runtime_map_delete_float_string(i8* map_ptr, f64 key)
 */
void runtime_map_delete_float_string(void* map_ptr, double key);

// ============================================================================
// Map多类型支持：map[string]float
// ============================================================================

/**
 * @brief 设置 map[string]float 中的键值对
 * @param map_ptr map指针（i8*，不透明指针）
 * @param key 键（i8*，字符串）
 * @param value 值（f64，浮点数）
 * @return void
 * 
 * 编译器签名：void runtime_map_set_string_float(i8* map_ptr, i8* key, f64 value)
 */
void runtime_map_set_string_float(void* map_ptr, void* key, double value);

/**
 * @brief 获取 map[string]float 中的值
 * @param map_ptr map指针（i8*，不透明指针）
 * @param key 键（i8*，字符串）
 * @return 值（f64*，浮点数指针），如果不存在返回NULL
 * 
 * 编译器签名：f64* runtime_map_get_string_float(i8* map_ptr, i8* key)
 */
double* runtime_map_get_string_float(void* map_ptr, void* key);

/**
 * @brief 删除 map[string]float 中的键值对
 * @param map_ptr map指针（i8*，不透明指针）
 * @param key 键（i8*，字符串）
 * @return void
 * 
 * 编译器签名：void runtime_map_delete_string_float(i8* map_ptr, i8* key)
 */
void runtime_map_delete_string_float(void* map_ptr, void* key);

// ============================================================================
// Map多类型支持：map[bool]string
// ============================================================================

/**
 * @brief 设置 map[bool]string 中的键值对
 * @param map_ptr map指针（i8*，不透明指针）
 * @param key 键（i1，布尔值）
 * @param value 值（i8*，字符串）
 * @return void
 * 
 * 编译器签名：void runtime_map_set_bool_string(i8* map_ptr, i1 key, i8* value)
 */
void runtime_map_set_bool_string(void* map_ptr, bool key, void* value);

/**
 * @brief 获取 map[bool]string 中的值
 * @param map_ptr map指针（i8*，不透明指针）
 * @param key 键（i1，布尔值）
 * @return 值（i8*，字符串指针），如果不存在返回NULL
 * 
 * 编译器签名：i8* runtime_map_get_bool_string(i8* map_ptr, i1 key)
 */
void* runtime_map_get_bool_string(void* map_ptr, bool key);

/**
 * @brief 删除 map[bool]string 中的键值对
 * @param map_ptr map指针（i8*，不透明指针）
 * @param key 键（i1，布尔值）
 * @return void
 * 
 * 编译器签名：void runtime_map_delete_bool_string(i8* map_ptr, i1 key)
 */
void runtime_map_delete_bool_string(void* map_ptr, bool key);

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
int32_t runtime_map_len(void* map_ptr);

/**
 * @brief 清空Map（删除所有条目）
 * @param map_ptr map指针（i8*，不透明指针）
 * @return void
 * 
 * 编译器签名：void runtime_map_clear(i8* map_ptr)
 * 
 * 注意：这是类型无关的函数，适用于所有Map类型
 */
void runtime_map_clear(void* map_ptr);

/**
 * @brief 检查 map[string]string 中是否包含指定的键
 * @param map_ptr map指针（i8*，不透明指针）
 * @param key 键（i8*，字符串）
 * @return 1 如果键存在，0 如果不存在
 * 
 * 编译器签名：i32 runtime_map_contains_string_string(i8* map_ptr, i8* key)
 */
int32_t runtime_map_contains_string_string(void* map_ptr, void* key);

/**
 * @brief 检查 map[int]string 中是否包含指定的键
 * @param map_ptr map指针（i8*，不透明指针）
 * @param key 键（i32，整数）
 * @return 1 如果键存在，0 如果不存在
 * 
 * 编译器签名：i32 runtime_map_contains_int_string(i8* map_ptr, i32 key)
 */
int32_t runtime_map_contains_int_string(void* map_ptr, int32_t key);

/**
 * @brief 检查 map[string]int 中是否包含指定的键
 * @param map_ptr map指针（i8*，不透明指针）
 * @param key 键（i8*，字符串）
 * @return 1 如果键存在，0 如果不存在
 * 
 * 编译器签名：i32 runtime_map_contains_string_int(i8* map_ptr, i8* key)
 */
int32_t runtime_map_contains_string_int(void* map_ptr, void* key);

/**
 * @brief 检查 map[int]int 中是否包含指定的键
 * @param map_ptr map指针（i8*，不透明指针）
 * @param key 键（i32，整数）
 * @return 1 如果键存在，0 如果不存在
 * 
 * 编译器签名：i32 runtime_map_contains_int_int(i8* map_ptr, i32 key)
 */
int32_t runtime_map_contains_int_int(void* map_ptr, int32_t key);

/**
 * @brief 检查 map[float]string 中是否包含指定的键
 * @param map_ptr map指针（i8*，不透明指针）
 * @param key 键（f64，浮点数）
 * @return 1 如果键存在，0 如果不存在
 * 
 * 编译器签名：i32 runtime_map_contains_float_string(i8* map_ptr, f64 key)
 */
int32_t runtime_map_contains_float_string(void* map_ptr, double key);

/**
 * @brief 检查 map[string]float 中是否包含指定的键
 * @param map_ptr map指针（i8*，不透明指针）
 * @param key 键（i8*，字符串）
 * @return 1 如果键存在，0 如果不存在
 * 
 * 编译器签名：i32 runtime_map_contains_string_float(i8* map_ptr, i8* key)
 */
int32_t runtime_map_contains_string_float(void* map_ptr, void* key);

/**
 * @brief 检查 map[bool]string 中是否包含指定的键
 * @param map_ptr map指针（i8*，不透明指针）
 * @param key 键（i1，布尔值）
 * @return 1 如果键存在，0 如果不存在
 * 
 * 编译器签名：i32 runtime_map_contains_bool_string(i8* map_ptr, i1 key)
 */
int32_t runtime_map_contains_bool_string(void* map_ptr, bool key);

// ============================================================================
// Map操作功能：keys() 函数（类型特定）
// ============================================================================

/**
 * @brief 获取 map[string]string 的键数组
 * @param map_ptr map指针（i8*，不透明指针）
 * @return 键数组指针（i8**，字符串数组）
 * 
 * 编译器签名：i8** runtime_map_keys_string_string(i8* map_ptr)
 */
char** runtime_map_keys_string_string(void* map_ptr);

/**
 * @brief 获取 map[int]string 的键数组
 * @param map_ptr map指针（i8*，不透明指针）
 * @return 键数组指针（i32*，整数数组）
 * 
 * 编译器签名：i32* runtime_map_keys_int_string(i8* map_ptr)
 */
int32_t* runtime_map_keys_int_string(void* map_ptr);

/**
 * @brief 获取 map[string]int 的键数组
 * @param map_ptr map指针（i8*，不透明指针）
 * @return 键数组指针（i8**，字符串数组）
 * 
 * 编译器签名：i8** runtime_map_keys_string_int(i8* map_ptr)
 */
char** runtime_map_keys_string_int(void* map_ptr);

/**
 * @brief 获取 map[int]int 的键数组
 * @param map_ptr map指针（i8*，不透明指针）
 * @return 键数组指针（i32*，整数数组）
 * 
 * 编译器签名：i32* runtime_map_keys_int_int(i8* map_ptr)
 */
int32_t* runtime_map_keys_int_int(void* map_ptr);

/**
 * @brief 获取 map[float]string 的键数组
 * @param map_ptr map指针（i8*，不透明指针）
 * @return 键数组指针（f64*，浮点数数组）
 * 
 * 编译器签名：f64* runtime_map_keys_float_string(i8* map_ptr)
 */
double* runtime_map_keys_float_string(void* map_ptr);

/**
 * @brief 获取 map[string]float 的键数组
 * @param map_ptr map指针（i8*，不透明指针）
 * @return 键数组指针（i8**，字符串数组）
 * 
 * 编译器签名：i8** runtime_map_keys_string_float(i8* map_ptr)
 */
char** runtime_map_keys_string_float(void* map_ptr);

/**
 * @brief 获取 map[bool]string 的键数组
 * @param map_ptr map指针（i8*，不透明指针）
 * @return 键数组指针（i1*，布尔数组）
 * 
 * 编译器签名：i1* runtime_map_keys_bool_string(i8* map_ptr)
 */
bool* runtime_map_keys_bool_string(void* map_ptr);

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
char** runtime_map_values_string_string(void* map_ptr);

/**
 * @brief 获取 map[int]string 的值数组
 * @param map_ptr map指针（i8*，不透明指针）
 * @return 值数组指针（i8**，字符串数组）
 * 
 * 编译器签名：i8** runtime_map_values_int_string(i8* map_ptr)
 */
char** runtime_map_values_int_string(void* map_ptr);

/**
 * @brief 获取 map[string]int 的值数组
 * @param map_ptr map指针（i8*，不透明指针）
 * @return 值数组指针（i32*，整数数组）
 * 
 * 编译器签名：i32* runtime_map_values_string_int(i8* map_ptr)
 */
int32_t* runtime_map_values_string_int(void* map_ptr);

/**
 * @brief 获取 map[int]int 的值数组
 * @param map_ptr map指针（i8*，不透明指针）
 * @return 值数组指针（i32*，整数数组）
 * 
 * 编译器签名：i32* runtime_map_values_int_int(i8* map_ptr)
 */
int32_t* runtime_map_values_int_int(void* map_ptr);

/**
 * @brief 获取 map[float]string 的值数组
 * @param map_ptr map指针（i8*，不透明指针）
 * @return 值数组指针（i8**，字符串数组）
 * 
 * 编译器签名：i8** runtime_map_values_float_string(i8* map_ptr)
 */
char** runtime_map_values_float_string(void* map_ptr);

/**
 * @brief 获取 map[string]float 的值数组
 * @param map_ptr map指针（i8*，不透明指针）
 * @return 值数组指针（f64*，浮点数数组）
 * 
 * 编译器签名：f64* runtime_map_values_string_float(i8* map_ptr)
 */
double* runtime_map_values_string_float(void* map_ptr);

/**
 * @brief 获取 map[bool]string 的值数组
 * @param map_ptr map指针（i8*，不透明指针）
 * @return 值数组指针（i8**，字符串数组）
 * 
 * 编译器签名：i8** runtime_map_values_bool_string(i8* map_ptr)
 */
char** runtime_map_values_bool_string(void* map_ptr);

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
 */
void runtime_map_set_string_slice_int(void* map_ptr, void* key, void* slice_ptr);

/**
 * @brief 获取 map[string][]int 中的值
 * @param map_ptr map指针（i8*，不透明指针）
 * @param key 键（i8*，字符串）
 * @return 值（i8*，切片指针），如果不存在返回NULL
 * 
 * 编译器签名：i8* runtime_map_get_string_slice_int(i8* map_ptr, i8* key)
 */
void* runtime_map_get_string_slice_int(void* map_ptr, void* key);

/**
 * @brief 设置 map[int][]string 中的键值对
 * @param map_ptr map指针（i8*，不透明指针）
 * @param key 键（i32，整数）
 * @param value 值（i8*，切片指针）
 * @return void
 * 
 * 编译器签名：void runtime_map_set_int_slice_string(i8* map_ptr, i32 key, i8* slice_ptr)
 */
void runtime_map_set_int_slice_string(void* map_ptr, int32_t key, void* slice_ptr);

/**
 * @brief 获取 map[int][]string 中的值
 * @param map_ptr map指针（i8*，不透明指针）
 * @param key 键（i32，整数）
 * @return 值（i8*，切片指针），如果不存在返回NULL
 * 
 * 编译器签名：i8* runtime_map_get_int_slice_string(i8* map_ptr, i32 key)
 */
void* runtime_map_get_int_slice_string(void* map_ptr, int32_t key);

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
 */
void runtime_map_set_string_map_int_string(void* map_ptr, void* key, void* inner_map_ptr);

/**
 * @brief 获取 map[string]map[int]string 中的值
 * @param map_ptr map指针（i8*，不透明指针）
 * @param key 键（i8*，字符串）
 * @return 值（i8*，内层Map指针），如果不存在返回NULL
 * 
 * 编译器签名：i8* runtime_map_get_string_map_int_string(i8* map_ptr, i8* key)
 */
void* runtime_map_get_string_map_int_string(void* map_ptr, void* key);

/**
 * @brief 设置 map[int]map[string]int 中的键值对
 * @param map_ptr map指针（i8*，不透明指针）
 * @param key 键（i32，整数）
 * @param value 值（i8*，内层Map指针）
 * @return void
 * 
 * 编译器签名：void runtime_map_set_int_map_string_int(i8* map_ptr, i32 key, i8* inner_map_ptr)
 */
void runtime_map_set_int_map_string_int(void* map_ptr, int32_t key, void* inner_map_ptr);

/**
 * @brief 获取 map[int]map[string]int 中的值
 * @param map_ptr map指针（i8*，不透明指针）
 * @param key 键（i32，整数）
 * @return 值（i8*，内层Map指针），如果不存在返回NULL
 * 
 * 编译器签名：i8* runtime_map_get_int_map_string_int(i8* map_ptr, i32 key)
 */
void* runtime_map_get_int_map_string_int(void* map_ptr, int32_t key);

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
 * 注意：支持所有结构体类型（User, Person等），通过指针传递
 */
void runtime_map_set_string_struct(void* map_ptr, void* key, void* struct_ptr);

/**
 * @brief 获取 map[string]struct 中的值（通用结构体值）
 * @param map_ptr map指针（i8*，不透明指针）
 * @param key 键（i8*，字符串）
 * @return 值（i8*，结构体指针），如果不存在返回NULL
 * 
 * 编译器签名：i8* runtime_map_get_string_struct(i8* map_ptr, i8* key)
 */
void* runtime_map_get_string_struct(void* map_ptr, void* key);

/**
 * @brief 设置 map[int]struct 中的键值对（通用结构体值）
 * @param map_ptr map指针（i8*，不透明指针）
 * @param key 键（i32，整数）
 * @param struct_ptr 值（i8*，结构体指针）
 * @return void
 * 
 * 编译器签名：void runtime_map_set_int_struct(i8* map_ptr, i32 key, i8* struct_ptr)
 */
void runtime_map_set_int_struct(void* map_ptr, int32_t key, void* struct_ptr);

/**
 * @brief 获取 map[int]struct 中的值（通用结构体值）
 * @param map_ptr map指针（i8*，不透明指针）
 * @param key 键（i32，整数）
 * @return 值（i8*，结构体指针），如果不存在返回NULL
 * 
 * 编译器签名：i8* runtime_map_get_int_struct(i8* map_ptr, i32 key)
 */
void* runtime_map_get_int_struct(void* map_ptr, int32_t key);

// ============================================================================
// Map操作功能：结构体作为键支持（map[User]string等）
// ============================================================================

// 哈希函数指针类型（用于结构体键）
typedef int64_t (*map_key_hash_func_t)(const void* key);

// 比较函数指针类型（用于结构体键）
typedef int (*map_key_equals_func_t)(const void* key1, const void* key2);

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
 */
void runtime_map_set_struct_string(void* map_ptr, void* key_ptr, void* value, map_key_hash_func_t hash_func, map_key_equals_func_t equals_func);

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
void* runtime_map_get_struct_string(void* map_ptr, void* key_ptr, map_key_hash_func_t hash_func, map_key_equals_func_t equals_func);

#endif // RUNTIME_API_H
