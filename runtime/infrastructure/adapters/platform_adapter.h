#ifndef PLATFORM_ADAPTER_H
#define PLATFORM_ADAPTER_H

#include <stdint.h>
#include <stdbool.h>

// 前向声明
struct PlatformAdapter;

// 平台类型枚举
typedef enum {
    PLATFORM_LINUX,
    PLATFORM_MACOS,
    PLATFORM_WINDOWS,
    PLATFORM_FREEBSD,
    PLATFORM_UNKNOWN
} PlatformType;

// 架构类型枚举
typedef enum {
    ARCH_X86_64,
    ARCH_ARM64,
    ARCH_X86,
    ARCH_ARM32,
    ARCH_UNKNOWN
} ArchitectureType;

// 平台适配器接口 - 统一不同平台的系统调用和特性
typedef struct {
    // 平台信息查询
    PlatformType (*get_platform_type)(const struct PlatformAdapter* adapter);
    ArchitectureType (*get_architecture_type)(const struct PlatformAdapter* adapter);
    const char* (*get_platform_name)(const struct PlatformAdapter* adapter);
    const char* (*get_architecture_name)(const struct PlatformAdapter* adapter);

    // 内存管理适配
    void* (*aligned_alloc)(struct PlatformAdapter* adapter, size_t alignment, size_t size);
    void (*aligned_free)(struct PlatformAdapter* adapter, void* ptr);
    size_t (*get_page_size)(const struct PlatformAdapter* adapter);
    bool (*protect_memory)(struct PlatformAdapter* adapter, void* addr, size_t size, int protection);

    // 线程和并发适配
    bool (*create_thread)(struct PlatformAdapter* adapter,
                         void* (*start_routine)(void*), void* arg, void** thread_handle);
    bool (*join_thread)(struct PlatformAdapter* adapter, void* thread_handle);
    bool (*detach_thread)(struct PlatformAdapter* adapter, void* thread_handle);
    void (*yield_thread)(struct PlatformAdapter* adapter);
    uint64_t (*get_current_thread_id)(const struct PlatformAdapter* adapter);

    // 同步原语适配
    bool (*create_mutex)(struct PlatformAdapter* adapter, void** mutex_handle);
    bool (*destroy_mutex)(struct PlatformAdapter* adapter, void* mutex_handle);
    bool (*lock_mutex)(struct PlatformAdapter* adapter, void* mutex_handle);
    bool (*try_lock_mutex)(struct PlatformAdapter* adapter, void* mutex_handle);
    bool (*unlock_mutex)(struct PlatformAdapter* adapter, void* mutex_handle);

    bool (*create_cond)(struct PlatformAdapter* adapter, void** cond_handle);
    bool (*destroy_cond)(struct PlatformAdapter* adapter, void* cond_handle);
    bool (*wait_cond)(struct PlatformAdapter* adapter, void* cond_handle, void* mutex_handle);
    bool (*timed_wait_cond)(struct PlatformAdapter* adapter, void* cond_handle,
                           void* mutex_handle, uint64_t timeout_ns);
    bool (*signal_cond)(struct PlatformAdapter* adapter, void* cond_handle);
    bool (*broadcast_cond)(struct PlatformAdapter* adapter, void* cond_handle);

    // 原子操作适配
    bool (*atomic_compare_exchange)(struct PlatformAdapter* adapter,
                                   volatile uint64_t* ptr, uint64_t expected, uint64_t desired);
    uint64_t (*atomic_fetch_add)(struct PlatformAdapter* adapter, volatile uint64_t* ptr, uint64_t value);
    uint64_t (*atomic_load)(struct PlatformAdapter* adapter, volatile const uint64_t* ptr);
    void (*atomic_store)(struct PlatformAdapter* adapter, volatile uint64_t* ptr, uint64_t value);

    // 网络适配
    bool (*create_socket)(struct PlatformAdapter* adapter, int domain, int type, int protocol, int* socket_fd);
    bool (*close_socket)(struct PlatformAdapter* adapter, int socket_fd);
    bool (*set_socket_nonblocking)(struct PlatformAdapter* adapter, int socket_fd);
    bool (*bind_socket)(struct PlatformAdapter* adapter, int socket_fd,
                       const struct sockaddr* addr, socklen_t addrlen);
    bool (*listen_socket)(struct PlatformAdapter* adapter, int socket_fd, int backlog);
    bool (*accept_socket)(struct PlatformAdapter* adapter, int socket_fd,
                         struct sockaddr* addr, socklen_t* addrlen, int* client_fd);

    // 事件循环适配
    bool (*create_event_loop)(struct PlatformAdapter* adapter, void** loop_handle);
    bool (*destroy_event_loop)(struct PlatformAdapter* adapter, void* loop_handle);
    bool (*add_event)(struct PlatformAdapter* adapter, void* loop_handle, int fd,
                     int events, void* callback, void* user_data);
    bool (*remove_event)(struct PlatformAdapter* adapter, void* loop_handle, int fd);
    bool (*run_event_loop)(struct PlatformAdapter* adapter, void* loop_handle, int timeout_ms);

    // 文件系统适配
    bool (*create_file)(struct PlatformAdapter* adapter, const char* path, int flags, int mode, int* fd);
    bool (*open_file)(struct PlatformAdapter* adapter, const char* path, int flags, int* fd);
    bool (*close_file)(struct PlatformAdapter* adapter, int fd);
    ssize_t (*read_file)(struct PlatformAdapter* adapter, int fd, void* buffer, size_t count);
    ssize_t (*write_file)(struct PlatformAdapter* adapter, int fd, const void* buffer, size_t count);
    bool (*seek_file)(struct PlatformAdapter* adapter, int fd, off_t offset, int whence);
    bool (*stat_file)(struct PlatformAdapter* adapter, const char* path, struct stat* statbuf);

    // 时间和定时器适配
    uint64_t (*get_monotonic_time_ns)(const struct PlatformAdapter* adapter);
    uint64_t (*get_wall_time_ns)(const struct PlatformAdapter* adapter);
    bool (*create_timer)(struct PlatformAdapter* adapter,
                        void (*callback)(void*), void* user_data,
                        uint64_t initial_delay_ns, uint64_t interval_ns, void** timer_handle);
    bool (*destroy_timer)(struct PlatformAdapter* adapter, void* timer_handle);

    // 系统信息适配
    uint32_t (*get_cpu_count)(const struct PlatformAdapter* adapter);
    uint64_t (*get_total_memory)(const struct PlatformAdapter* adapter);
    uint64_t (*get_available_memory)(const struct PlatformAdapter* adapter);
    const char* (*get_hostname)(const struct PlatformAdapter* adapter);
    const char* (*get_os_version)(const struct PlatformAdapter* adapter);
} PlatformAdapterInterface;

// 平台适配器基类
typedef struct PlatformAdapter {
    PlatformAdapterInterface* vtable;  // 虚函数表
    PlatformType platform_type;        // 平台类型
    ArchitectureType arch_type;        // 架构类型
    char platform_name[64];            // 平台名称
    char arch_name[32];                // 架构名称
} PlatformAdapter;

// 平台适配器工厂函数
PlatformAdapter* create_platform_adapter(void);
void destroy_platform_adapter(PlatformAdapter* adapter);

// 便捷函数（直接调用虚函数表）
static inline PlatformType platform_adapter_get_platform_type(const PlatformAdapter* adapter) {
    return adapter->vtable->get_platform_type(adapter);
}

static inline void* platform_adapter_aligned_alloc(PlatformAdapter* adapter, size_t alignment, size_t size) {
    return adapter->vtable->aligned_alloc(adapter, alignment, size);
}

static inline bool platform_adapter_create_thread(PlatformAdapter* adapter,
                                                 void* (*start_routine)(void*), void* arg, void** thread_handle) {
    return adapter->vtable->create_thread(adapter, start_routine, arg, thread_handle);
}

#endif // PLATFORM_ADAPTER_H
