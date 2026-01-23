#ifndef RUNTIME_TYPES_H
#define RUNTIME_TYPES_H

#include <stdint.h>
#include <stdbool.h>
#include <stddef.h>

// ============================================================================
// 基本类型定义
// ============================================================================

// 运行时句柄（前向声明已在runtime_api.h中）
typedef struct RuntimeHandle RuntimeHandle;

// 任务相关类型
typedef struct TaskHandle TaskHandle;
typedef struct TaskResult TaskResult;

// 协程相关类型
typedef struct CoroutineHandle CoroutineHandle;
typedef struct CoroutineResult CoroutineResult;

// 异步编程类型
typedef struct FutureHandle FutureHandle;
typedef struct PromiseHandle PromiseHandle;

// 并发原语类型
typedef struct MutexHandle MutexHandle;
typedef struct SemaphoreHandle SemaphoreHandle;
typedef struct ConditionHandle ConditionHandle;

// 通道类型
typedef struct ChannelHandle ChannelHandle;

// ============================================================================
// 枚举类型定义
// ============================================================================

// 任务状态枚举
typedef enum {
    TASK_STATUS_NEW,        // 新建
    TASK_STATUS_READY,      // 就绪
    TASK_STATUS_RUNNING,    // 运行中
    TASK_STATUS_SUSPENDED,  // 挂起
    TASK_STATUS_COMPLETED,  // 已完成
    TASK_STATUS_FAILED,     // 失败
    TASK_STATUS_CANCELLED   // 已取消
} TaskStatus;

// 协程状态枚举
typedef enum {
    COROUTINE_STATUS_NEW,       // 新建
    COROUTINE_STATUS_RUNNING,   // 运行中
    COROUTINE_STATUS_SUSPENDED, // 挂起
    COROUTINE_STATUS_COMPLETED, // 已完成
    COROUTINE_STATUS_FAILED     // 失败
} CoroutineStatus;

// Future状态枚举
typedef enum {
    FUTURE_STATUS_PENDING,  // 等待中
    FUTURE_STATUS_RESOLVED, // 已解决
    FUTURE_STATUS_REJECTED  // 已拒绝
} FutureStatus;

// 通道模式枚举
typedef enum {
    CHANNEL_MODE_UNBUFFERED, // 无缓冲
    CHANNEL_MODE_BUFFERED    // 有缓冲
} ChannelMode;

// ============================================================================
// 结构体类型定义
// ============================================================================

// 任务结果结构体
struct TaskResult {
    TaskStatus status;          // 任务状态
    void* result_data;          // 结果数据
    size_t result_size;         // 结果大小
    char* error_message;        // 错误信息
    uint64_t execution_time_ns; // 执行时间（纳秒）
    uint64_t start_time;        // 开始时间
    uint64_t end_time;          // 结束时间
};

// 协程结果结构体
struct CoroutineResult {
    CoroutineStatus status;     // 协程状态
    void* result_data;          // 结果数据
    size_t result_size;         // 结果大小
    char* error_message;        // 错误信息
    uint32_t yield_count;       // yield次数
    size_t stack_peak_usage;    // 栈峰值使用量
};

// ============================================================================
// 常量定义
// ============================================================================

// 默认值常量
#define RUNTIME_DEFAULT_HEAP_SIZE           (64 * 1024 * 1024)    // 64MB
#define RUNTIME_DEFAULT_STACK_SIZE          (1024 * 1024)         // 1MB
#define RUNTIME_DEFAULT_MAX_TASKS           10000
#define RUNTIME_DEFAULT_MAX_COROUTINES      100000
#define RUNTIME_DEFAULT_CHANNEL_BUFFER_SIZE 1024

// 时间常量
#define RUNTIME_DEFAULT_TIMEOUT_MS          5000    // 5秒
#define RUNTIME_DEFAULT_CONNECT_TIMEOUT_MS  10000   // 10秒
#define RUNTIME_DEFAULT_READ_TIMEOUT_MS     30000   // 30秒

// 内存常量
#define RUNTIME_DEFAULT_MEMORY_ALIGNMENT    16      // 16字节对齐
#define RUNTIME_DEFAULT_MEMORY_BLOCK_SIZE   (4 * 1024) // 4KB

// ============================================================================
// 宏定义
// ============================================================================

// 版本宏
#define RUNTIME_VERSION_MAJOR 1
#define RUNTIME_VERSION_MINOR 0
#define RUNTIME_VERSION_PATCH 0
#define RUNTIME_VERSION_STRING "1.0.0"

// 编译时断言宏
#define RUNTIME_STATIC_ASSERT(cond, msg) typedef char static_assertion_##msg[(cond)?1:-1]

// 便捷宏
#define RUNTIME_ARRAY_SIZE(arr) (sizeof(arr) / sizeof((arr)[0]))

// ============================================================================
// 类型别名
// ============================================================================

// 基本类型别名
typedef uint8_t  u8;
typedef uint16_t u16;
typedef uint32_t u32;
typedef uint64_t u64;
typedef int8_t   i8;
typedef int16_t  i16;
typedef int32_t  i32;
typedef int64_t  i64;
typedef float    f32;
typedef double   f64;

// 函数指针类型别名
typedef void (*TaskFunction)(void*);                    // 任务函数类型
typedef void (*CoroutineFunction)(void*);              // 协程函数类型
typedef void (*CallbackFunction)(void*, void*);        // 回调函数类型
typedef bool (*PredicateFunction)(void*);              // 谓词函数类型
typedef void (*CleanupFunction)(void*);                // 清理函数类型

// ============================================================================
// 错误码定义
// ============================================================================

#define RUNTIME_ERROR_SUCCESS              0
#define RUNTIME_ERROR_INVALID_ARGUMENT     -1
#define RUNTIME_ERROR_OUT_OF_MEMORY        -2
#define RUNTIME_ERROR_NOT_INITIALIZED      -3
#define RUNTIME_ERROR_ALREADY_EXISTS       -4
#define RUNTIME_ERROR_NOT_FOUND            -5
#define RUNTIME_ERROR_TIMEOUT              -6
#define RUNTIME_ERROR_PERMISSION_DENIED    -7
#define RUNTIME_ERROR_SYSTEM_ERROR         -8
#define RUNTIME_ERROR_NOT_SUPPORTED        -9
#define RUNTIME_ERROR_BUSY                 -10
#define RUNTIME_ERROR_INTERRUPTED          -11

// ============================================================================
// 平台相关类型定义
// ============================================================================

#if defined(_WIN32) || defined(_WIN64)
    // Windows平台
    typedef void* HANDLE;
    typedef unsigned long DWORD;
    typedef long LONG;
    #define RUNTIME_PLATFORM_WINDOWS 1
    #define RUNTIME_PATH_SEPARATOR '\\'
    #define RUNTIME_PATH_SEPARATOR_STR "\\"

#elif defined(__linux__) || defined(__unix__) || defined(__APPLE__)
    // Unix-like平台
    typedef int HANDLE;
    typedef unsigned int DWORD;
    typedef long LONG;
    #if defined(__linux__)
        #define RUNTIME_PLATFORM_LINUX 1
    #elif defined(__APPLE__)
        #define RUNTIME_PLATFORM_MACOS 1
    #else
        #define RUNTIME_PLATFORM_UNIX 1
    #endif
    #define RUNTIME_PATH_SEPARATOR '/'
    #define RUNTIME_PATH_SEPARATOR_STR "/"

#else
    // 未知平台
    typedef void* HANDLE;
    typedef unsigned int DWORD;
    typedef long LONG;
    #define RUNTIME_PLATFORM_UNKNOWN 1
    #define RUNTIME_PATH_SEPARATOR '/'
    #define RUNTIME_PATH_SEPARATOR_STR "/"

#endif

// ============================================================================
// 原子操作类型（如果编译器支持）
// ============================================================================

#if defined(__STDC_VERSION__) && __STDC_VERSION__ >= 201112L && !defined(__STDC_NO_ATOMICS__)
    // C11原子操作支持
    #include <stdatomic.h>
    #define RUNTIME_HAS_ATOMICS 1

    typedef atomic_int atomic_int32_t;
    typedef atomic_long atomic_int64_t;
    typedef atomic_intptr_t atomic_intptr_t;

#else
    // 回退到volatile
    #define RUNTIME_HAS_ATOMICS 0

    typedef volatile int atomic_int32_t;
    typedef volatile long atomic_int64_t;
    typedef volatile intptr_t atomic_intptr_t;

#endif

// ============================================================================
// 内联函数定义
// ============================================================================

// 便捷的错误检查宏
#define RUNTIME_CHECK_HANDLE(handle) \
    if (!(handle)) { \
        return false; \
    }

// 便捷的内存分配宏
#define RUNTIME_ALLOC(type) (type*)malloc(sizeof(type))
#define RUNTIME_ALLOC_ARRAY(type, count) (type*)malloc(sizeof(type) * (count))
#define RUNTIME_FREE(ptr) do { if (ptr) { free(ptr); ptr = NULL; } } while(0)

// ============================================================================
// 调试和诊断宏
// ============================================================================

#ifdef RUNTIME_DEBUG
    #define RUNTIME_DEBUG_LOG(format, ...) printf("[DEBUG] " format "\n", ##__VA_ARGS__)
    #define RUNTIME_TRACE_CALL() printf("[TRACE] %s:%d %s()\n", __FILE__, __LINE__, __func__)
#else
    #define RUNTIME_DEBUG_LOG(format, ...)
    #define RUNTIME_TRACE_CALL()
#endif

// ============================================================================
// 编译时配置检查
// ============================================================================

// 检查指针大小
RUNTIME_STATIC_ASSERT(sizeof(void*) == 8 || sizeof(void*) == 4, pointer_size_must_be_32_or_64_bit);

// 检查基本类型大小
RUNTIME_STATIC_ASSERT(sizeof(u8) == 1, u8_must_be_1_byte);
RUNTIME_STATIC_ASSERT(sizeof(u16) == 2, u16_must_be_2_bytes);
RUNTIME_STATIC_ASSERT(sizeof(u32) == 4, u32_must_be_4_bytes);
RUNTIME_STATIC_ASSERT(sizeof(u64) == 8, u64_must_be_8_bytes);

#endif // RUNTIME_TYPES_H
