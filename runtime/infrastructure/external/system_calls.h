#ifndef SYSTEM_CALLS_H
#define SYSTEM_CALLS_H

#include <stdint.h>
#include <stdbool.h>
#include <time.h>
#include <signal.h>
#include <sys/types.h>
#include <sys/wait.h>

// 前向声明
struct SystemCalls;
struct ProcessInfo;
struct SystemResourceUsage;
struct SignalInfo;
struct SystemCallConfig;

// 进程信息
typedef struct ProcessInfo {
    pid_t pid;                   // 进程ID
    pid_t ppid;                  // 父进程ID
    uid_t uid;                   // 用户ID
    gid_t gid;                   // 组ID
    char executable[1024];       // 可执行文件路径
    char cmdline[4096];          // 命令行
    char cwd[1024];              // 当前工作目录
    time_t start_time;           // 启动时间
    uint64_t user_time;          // 用户CPU时间
    uint64_t system_time;        // 系统CPU时间
    uint64_t virtual_memory;     // 虚拟内存使用量
    uint64_t resident_memory;    // 常驻内存使用量
    double cpu_usage;            // CPU使用率
    int priority;                // 进程优先级
    int nice_value;              // nice值
    char state[16];              // 进程状态
    uint32_t thread_count;       // 线程数量
    uint64_t open_files;         // 打开文件数
} ProcessInfo;

// 系统资源使用情况
typedef struct SystemResourceUsage {
    // CPU信息
    uint32_t cpu_count;          // CPU核心数
    double cpu_usage_total;      // CPU总使用率
    double* cpu_usage_per_core;  // 每核心CPU使用率
    uint32_t cpu_count_cores;    // CPU核心数量

    // 内存信息
    uint64_t total_memory;       // 总内存
    uint64_t available_memory;   // 可用内存
    uint64_t used_memory;        // 已用内存
    uint64_t free_memory;        // 空闲内存
    uint64_t buffered_memory;    // 缓冲内存
    uint64_t cached_memory;      // 缓存内存
    double memory_usage_percent; // 内存使用率

    // 磁盘信息
    uint64_t total_disk_space;   // 总磁盘空间
    uint64_t available_disk_space; // 可用磁盘空间
    uint64_t used_disk_space;    // 已用磁盘空间
    double disk_usage_percent;   // 磁盘使用率

    // 网络信息
    uint64_t bytes_sent;         // 发送字节数
    uint64_t bytes_received;     // 接收字节数
    uint64_t packets_sent;       // 发送包数
    uint64_t packets_received;   // 接收包数
    uint32_t active_connections; // 活跃连接数

    // 负载信息
    double load_average_1m;      // 1分钟负载平均值
    double load_average_5m;      // 5分钟负载平均值
    double load_average_15m;     // 15分钟负载平均值

    time_t collected_at;         // 收集时间
} SystemResourceUsage;

// 信号信息
typedef struct SignalInfo {
    int signal_number;           // 信号编号
    char signal_name[32];        // 信号名称
    char description[256];       // 信号描述
    bool can_be_caught;          // 是否可以捕获
    bool can_be_ignored;         // 是否可以忽略
    bool generates_core_dump;    // 是否产生核心转储
    bool terminates_process;     // 是否终止进程
} SignalInfo;

// 系统调用配置
typedef struct SystemCallConfig {
    uint32_t max_processes;      // 最大进程数限制
    uint32_t max_threads;        // 最大线程数限制
    uint64_t max_memory_usage;   // 最大内存使用量
    uint32_t max_open_files;     // 最大打开文件数
    bool enable_process_monitoring; // 是否启用进程监控
    bool enable_resource_monitoring; // 是否启用资源监控
    uint32_t monitoring_interval_ms; // 监控间隔
    bool enable_signal_handling; // 是否启用信号处理
    char log_file[512];          // 日志文件路径
} SystemCallConfig;

// 系统调用接口
typedef struct {
    // 进程管理
    bool (*get_current_pid)(struct SystemCalls* sys, pid_t* pid);
    bool (*get_process_info)(struct SystemCalls* sys, pid_t pid, struct ProcessInfo* info);
    bool (*get_all_processes)(struct SystemCalls* sys, struct ProcessInfo** processes, size_t* count);
    bool (*kill_process)(struct SystemCalls* sys, pid_t pid, int signal);
    bool (*wait_for_process)(struct SystemCalls* sys, pid_t pid, int* status, int options);

    // 进程创建
    bool (*fork_process)(struct SystemCalls* sys, pid_t* child_pid);
    bool (*exec_process)(struct SystemCalls* sys, const char* path, char* const argv[], char* const envp[]);
    bool (*spawn_process)(struct SystemCalls* sys, const char* path, char* const argv[], char* const envp[],
                         pid_t* child_pid, int* stdin_fd, int* stdout_fd, int* stderr_fd);

    // 线程管理
    bool (*get_current_tid)(struct SystemCalls* sys, pid_t* tid);
    bool (*create_thread)(struct SystemCalls* sys, void* (*start_routine)(void*), void* arg, void** thread_handle);
    bool (*join_thread)(struct SystemCalls* sys, void* thread_handle, void** retval);
    bool (*detach_thread)(struct SystemCalls* sys, void* thread_handle);
    bool (*cancel_thread)(struct SystemCalls* sys, void* thread_handle);

    // 信号处理
    bool (*send_signal)(struct SystemCalls* sys, pid_t pid, int signal);
    bool (*set_signal_handler)(struct SystemCalls* sys, int signal,
                              void (*handler)(int, siginfo_t*, void*), void* user_data);
    bool (*ignore_signal)(struct SystemCalls* sys, int signal);
    bool (*reset_signal_handler)(struct SystemCalls* sys, int signal);
    bool (*get_signal_info)(struct SystemCalls* sys, int signal, struct SignalInfo* info);

    // 系统资源监控
    bool (*get_system_resource_usage)(struct SystemCalls* sys, struct SystemResourceUsage* usage);
    bool (*get_process_resource_usage)(struct SystemCalls* sys, pid_t pid, struct SystemResourceUsage* usage);
    bool (*set_resource_limits)(struct SystemCalls* sys, pid_t pid, const char* resource, uint64_t limit);
    bool (*get_resource_limits)(struct SystemCalls* sys, pid_t pid, const char* resource, uint64_t* limit);

    // 文件和I/O
    bool (*open_file)(struct SystemCalls* sys, const char* path, int flags, mode_t mode, int* fd);
    bool (*close_file)(struct SystemCalls* sys, int fd);
    bool (*read_file)(struct SystemCalls* sys, int fd, void* buffer, size_t count, ssize_t* bytes_read);
    bool (*write_file)(struct SystemCalls* sys, int fd, const void* buffer, size_t count, ssize_t* bytes_written);
    bool (*seek_file)(struct SystemCalls* sys, int fd, off_t offset, int whence, off_t* new_offset);
    bool (*stat_file)(struct SystemCalls* sys, const char* path, struct stat* statbuf);
    bool (*unlink_file)(struct SystemCalls* sys, const char* path);
    bool (*rename_file)(struct SystemCalls* sys, const char* old_path, const char* new_path);

    // 网络操作
    bool (*socket_create)(struct SystemCalls* sys, int domain, int type, int protocol, int* socket_fd);
    bool (*socket_bind)(struct SystemCalls* sys, int socket_fd, const struct sockaddr* addr, socklen_t addrlen);
    bool (*socket_listen)(struct SystemCalls* sys, int socket_fd, int backlog);
    bool (*socket_accept)(struct SystemCalls* sys, int socket_fd, struct sockaddr* addr, socklen_t* addrlen, int* client_fd);
    bool (*socket_connect)(struct SystemCalls* sys, int socket_fd, const struct sockaddr* addr, socklen_t addrlen);
    bool (*socket_send)(struct SystemCalls* sys, int socket_fd, const void* buffer, size_t len, int flags, ssize_t* bytes_sent);
    bool (*socket_recv)(struct SystemCalls* sys, int socket_fd, void* buffer, size_t len, int flags, ssize_t* bytes_received);
    bool (*socket_close)(struct SystemCalls* sys, int socket_fd);

    // 时间和定时器
    bool (*get_current_time)(struct SystemCalls* sys, time_t* seconds, long* nanoseconds);
    bool (*get_monotonic_time)(struct SystemCalls* sys, uint64_t* nanoseconds);
    bool (*sleep_seconds)(struct SystemCalls* sys, unsigned int seconds);
    bool (*sleep_nanoseconds)(struct SystemCalls* sys, uint64_t nanoseconds);
    bool (*create_timer)(struct SystemCalls* sys, clockid_t clock_id, int flags,
                        const struct sigevent* sevp, timer_t* timer_id);
    bool (*set_timer)(struct SystemCalls* sys, timer_t timer_id, int flags,
                     const struct itimerspec* new_value, struct itimerspec* old_value);
    bool (*delete_timer)(struct SystemCalls* sys, timer_t timer_id);

    // 内存管理
    bool (*get_memory_info)(struct SystemCalls* sys, uint64_t* total, uint64_t* available, uint64_t* used);
    bool (*allocate_memory)(struct SystemCalls* sys, size_t size, void** ptr);
    bool (*deallocate_memory)(struct SystemCalls* sys, void* ptr);
    bool (*protect_memory)(struct SystemCalls* sys, void* addr, size_t len, int prot);
    bool (*lock_memory)(struct SystemCalls* sys, void* addr, size_t len);
    bool (*unlock_memory)(struct SystemCalls* sys, void* addr, size_t len);

    // 系统信息
    bool (*get_hostname)(struct SystemCalls* sys, char* hostname, size_t size);
    bool (*get_os_info)(struct SystemCalls* sys, char* os_name, size_t name_size, char* os_version, size_t version_size);
    bool (*get_cpu_info)(struct SystemCalls* sys, uint32_t* cpu_count, char* cpu_model, size_t model_size);
    bool (*get_load_average)(struct SystemCalls* sys, double* load1, double* load5, double* load15);

    // 权限和安全
    bool (*get_current_uid)(struct SystemCalls* sys, uid_t* uid);
    bool (*get_current_gid)(struct SystemCalls* sys, gid_t* gid);
    bool (*set_uid)(struct SystemCalls* sys, uid_t uid);
    bool (*set_gid)(struct SystemCalls* sys, gid_t gid);
    bool (*drop_privileges)(struct SystemCalls* sys);
    bool (*check_capability)(struct SystemCalls* sys, const char* capability);
} SystemCallsInterface;

// 系统调用实现
typedef struct SystemCalls {
    SystemCallsInterface* vtable;  // 虚函数表

    // 配置
    struct SystemCallConfig config;

    // 状态
    bool initialized;
    time_t started_at;
    char sys_name[128];

    // 内部数据结构
    void* process_table;          // 进程表
    void* thread_table;           // 线程表
    void* signal_handlers;        // 信号处理器表
    void* resource_limits;        // 资源限制表

    // 同步原语
    void* mutex;

    // 统计
    uint64_t total_syscalls;
    uint64_t failed_syscalls;
    time_t last_monitoring_at;
} SystemCalls;

// 系统调用工厂函数
SystemCalls* system_calls_create(const struct SystemCallConfig* config);
void system_calls_destroy(SystemCalls* sys);

// 便捷构造函数
struct SystemCallConfig* system_call_config_create(void);
void system_call_config_destroy(struct SystemCallConfig* config);

// 预定义配置
struct SystemCallConfig* system_call_config_create_default(void);
struct SystemCallConfig* system_call_config_create_secure(void);
struct SystemCallConfig* system_call_config_create_high_performance(void);

// 便捷函数（直接调用虚函数表）
static inline bool system_calls_get_current_pid(SystemCalls* sys, pid_t* pid) {
    return sys->vtable->get_current_pid(sys, pid);
}

static inline bool system_calls_get_process_info(SystemCalls* sys, pid_t pid, struct ProcessInfo* info) {
    return sys->vtable->get_process_info(sys, pid, info);
}

static inline bool system_calls_kill_process(SystemCalls* sys, pid_t pid, int signal) {
    return sys->vtable->kill_process(sys, pid, signal);
}

static inline bool system_calls_get_system_resource_usage(SystemCalls* sys, struct SystemResourceUsage* usage) {
    return sys->vtable->get_system_resource_usage(sys, usage);
}

// 全局系统调用实例（可选）
extern SystemCalls* g_global_system_calls;

// 全局便捷函数
static inline bool system_calls_get_pid_global(pid_t* pid) {
    if (g_global_system_calls) {
        return system_calls_get_current_pid(g_global_system_calls, pid);
    }
    return false;
}

#endif // SYSTEM_CALLS_H
