#include "system_calls.h"
#include "../../shared/types/common_types.h"
#include <stdlib.h>
#include <stdio.h>
#include <string.h>
#include <unistd.h>
#include <sys/types.h>
#include <sys/wait.h>
#include <sys/stat.h>
#include <sys/time.h>
#include <sys/resource.h>
#include <signal.h>
#include <errno.h>
#include <dirent.h>
#include <fcntl.h>
#include <pthread.h>
#include <time.h>

// 内部实现结构体
typedef struct SystemCallsImpl {
    SystemCalls base;

    // 配置
    SystemCallConfig config;

    // 统计信息
    SystemCallsStats stats;

    // 同步原语
    pthread_mutex_t mutex;
    bool initialized;
} SystemCallsImpl;

// 创建系统调用接口
SystemCalls* system_calls_create(const SystemCallConfig* config) {
    SystemCallsImpl* impl = (SystemCallsImpl*)malloc(sizeof(SystemCallsImpl));
    if (!impl) {
        fprintf(stderr, "ERROR: Failed to allocate memory for system calls\n");
        return NULL;
    }

    memset(impl, 0, sizeof(SystemCallsImpl));

    // 设置默认配置
    if (config) {
        memcpy(&impl->config, config, sizeof(SystemCallConfig));
    } else {
        impl->config.enable_process_monitoring = true;
        impl->config.enable_signal_handling = true;
        impl->config.enable_resource_monitoring = true;
        impl->config.max_process_info_cache = 1000;
        impl->config.process_info_cache_ttl_seconds = 300;
        impl->config.enable_system_info_caching = true;
        impl->config.system_info_cache_ttl_seconds = 60;
    }

    // 初始化统计信息
    memset(&impl->stats, 0, sizeof(SystemCallsStats));
    impl->stats.created_at = time(NULL);

    // 初始化同步原语
    pthread_mutex_init(&impl->mutex, NULL);

    impl->initialized = true;

    return (SystemCalls*)impl;
}

// 销毁系统调用接口
void system_calls_destroy(SystemCalls* syscalls) {
    if (!syscalls) return;

    SystemCallsImpl* impl = (SystemCallsImpl*)syscalls;

    if (impl->initialized) {
        pthread_mutex_destroy(&impl->mutex);
    }

    free(impl);
}

// 获取进程信息
bool system_calls_get_process_info(SystemCalls* syscalls, pid_t pid, ProcessInfo* info) {
    if (!syscalls || !info) return false;

    SystemCallsImpl* impl = (SystemCallsImpl*)syscalls;
    if (!impl->initialized) return false;

    memset(info, 0, sizeof(ProcessInfo));

    // 读取/proc/[pid]/stat
    char stat_path[256];
    snprintf(stat_path, sizeof(stat_path), "/proc/%d/stat", pid);

    FILE* stat_file = fopen(stat_path, "r");
    if (!stat_file) return false;

    // 解析进程状态信息
    char comm[256];
    char state;
    pid_t ppid;
    unsigned long utime, stime;
    long priority, nice;

    if (fscanf(stat_file, "%d %s %c %d %*d %*d %*d %*d %*u %*u %*u %*u %*u %lu %lu %*d %*d %ld %ld",
               &info->pid, comm, &state, &ppid, &utime, &stime, &priority, &nice) == 8) {
        info->ppid = ppid;
        info->user_time = utime;
        info->system_time = stime;
        info->priority = priority;
        info->nice_value = nice;
        info->state[0] = state;
        info->state[1] = '\0';
    }
    fclose(stat_file);

    // 读取/proc/[pid]/status获取更多信息
    char status_path[256];
    snprintf(status_path, sizeof(status_path), "/proc/%d/status", pid);

    FILE* status_file = fopen(status_path, "r");
    if (status_file) {
        char line[256];
        while (fgets(line, sizeof(line), status_file)) {
            if (strncmp(line, "Uid:", 4) == 0) {
                sscanf(line, "Uid:\t%d", &info->uid);
            } else if (strncmp(line, "Gid:", 4) == 0) {
                sscanf(line, "Gid:\t%d", &info->gid);
            } else if (strncmp(line, "Threads:", 8) == 0) {
                sscanf(line, "Threads:\t%d", &info->thread_count);
            }
        }
        fclose(status_file);
    }

    // 读取命令行
    char cmdline_path[256];
    snprintf(cmdline_path, sizeof(cmdline_path), "/proc/%d/cmdline", pid);

    FILE* cmdline_file = fopen(cmdline_path, "r");
    if (cmdline_file) {
        size_t len = fread(info->cmdline, 1, sizeof(info->cmdline) - 1, cmdline_file);
        if (len > 0) {
            // 将null字节替换为空格
            for (size_t i = 0; i < len - 1; i++) {
                if (info->cmdline[i] == '\0') {
                    info->cmdline[i] = ' ';
                }
            }
            info->cmdline[len] = '\0';
        }
        fclose(cmdline_file);
    }

    // 读取可执行文件路径
    char exe_path[256];
    snprintf(exe_path, sizeof(exe_path), "/proc/%d/exe", pid);
    ssize_t exe_len = readlink(exe_path, info->executable, sizeof(info->executable) - 1);
    if (exe_len > 0) {
        info->executable[exe_len] = '\0';
    }

    // 读取当前工作目录
    char cwd_path[256];
    snprintf(cwd_path, sizeof(cwd_path), "/proc/%d/cwd", pid);
    ssize_t cwd_len = readlink(cwd_path, info->cwd, sizeof(info->cwd) - 1);
    if (cwd_len > 0) {
        info->cwd[cwd_len] = '\0';
    }

    // 获取启动时间
    struct stat st;
    if (stat(stat_path, &st) == 0) {
        info->start_time = st.st_ctime;
    }

    // 更新统计信息
    pthread_mutex_lock(&impl->mutex);
    impl->stats.total_process_info_queries++;
    pthread_mutex_unlock(&impl->mutex);

    return true;
}

// 获取当前进程信息
bool system_calls_get_current_process_info(SystemCalls* syscalls, ProcessInfo* info) {
    return system_calls_get_process_info(syscalls, getpid(), info);
}

// 获取系统资源使用情况
bool system_calls_get_resource_usage(SystemCalls* syscalls, SystemResourceUsage* usage) {
    if (!syscalls || !usage) return false;

    SystemCallsImpl* impl = (SystemCallsImpl*)syscalls;
    if (!impl->initialized) return false;

    memset(usage, 0, sizeof(SystemResourceUsage));

    // 获取CPU信息
    usage->cpu_count = sysconf(_SC_NPROCESSORS_ONLN);

    // 读取/proc/stat获取CPU使用率
    FILE* stat_file = fopen("/proc/stat", "r");
    if (stat_file) {
        char line[256];
        while (fgets(line, sizeof(line), stat_file)) {
            if (strncmp(line, "cpu ", 4) == 0) {
                unsigned long user, nice, system, idle, iowait, irq, softirq;
                if (sscanf(line, "cpu %lu %lu %lu %lu %lu %lu %lu",
                          &user, &nice, &system, &idle, &iowait, &irq, &softirq) == 7) {
                    unsigned long total = user + nice + system + idle + iowait + irq + softirq;
                    unsigned long idle_total = idle + iowait;
                    if (total > 0) {
                        usage->cpu_usage_total = 100.0 * (1.0 - (double)idle_total / (double)total);
                    }
                }
            }
        }
        fclose(stat_file);
    }

    // 获取内存信息
    FILE* meminfo_file = fopen("/proc/meminfo", "r");
    if (meminfo_file) {
        char line[256];
        while (fgets(line, sizeof(line), meminfo_file)) {
            if (strncmp(line, "MemTotal:", 9) == 0) {
                sscanf(line, "MemTotal: %lu kB", &usage->total_memory);
                usage->total_memory *= 1024; // 转换为字节
            } else if (strncmp(line, "MemAvailable:", 13) == 0) {
                sscanf(line, "MemAvailable: %lu kB", &usage->available_memory);
                usage->available_memory *= 1024;
            }
        }
        fclose(meminfo_file);
    }

    // 获取磁盘信息
    FILE* disk_file = popen("df / | tail -1", "r");
    if (disk_file) {
        char line[256];
        if (fgets(line, sizeof(line), disk_file)) {
            sscanf(line, "%*s %lu %lu %lu", &usage->total_disk_space,
                   &usage->used_disk_space, &usage->available_disk_space);
            usage->total_disk_space *= 1024; // 转换为字节
            usage->used_disk_space *= 1024;
            usage->available_disk_space *= 1024;
        }
        pclose(disk_file);
    }

    // 获取网络信息（简化实现）
    FILE* net_file = popen("cat /proc/net/dev | grep -E 'eth|wlan' | head -1", "r");
    if (net_file) {
        char line[256];
        if (fgets(line, sizeof(line), net_file)) {
            // 解析网络统计信息
            char iface[16];
            unsigned long rx_bytes, tx_bytes;
            if (sscanf(line, "%s %lu %*u %*u %*u %*u %*u %*u %*u %lu",
                      iface, &rx_bytes, &tx_bytes) == 3) {
                usage->network_rx_bytes = rx_bytes;
                usage->network_tx_bytes = tx_bytes;
            }
        }
        pclose(net_file);
    }

    // 更新统计信息
    pthread_mutex_lock(&impl->mutex);
    impl->stats.total_resource_queries++;
    pthread_mutex_unlock(&impl->mutex);

    return true;
}

// 发送信号到进程
bool system_calls_send_signal(SystemCalls* syscalls, pid_t pid, int signal) {
    if (!syscalls) return false;

    SystemCallsImpl* impl = (SystemCallsImpl*)syscalls;
    if (!impl->initialized) return false;

    int result = kill(pid, signal);

    // 更新统计信息
    pthread_mutex_lock(&impl->mutex);
    impl->stats.total_signals_sent++;
    if (result == 0) {
        impl->stats.successful_signals++;
    }
    pthread_mutex_unlock(&impl->mutex);

    return result == 0;
}

// 等待进程结束
pid_t system_calls_wait_process(SystemCalls* syscalls, pid_t pid, int* status, int options) {
    if (!syscalls) return -1;

    SystemCallsImpl* impl = (SystemCallsImpl*)syscalls;
    if (!impl->initialized) return -1;

    pid_t result = waitpid(pid, status, options);

    // 更新统计信息
    pthread_mutex_lock(&impl->mutex);
    impl->stats.total_process_waits++;
    pthread_mutex_unlock(&impl->mutex);

    return result;
}

// 创建子进程
pid_t system_calls_fork_process(SystemCalls* syscalls) {
    if (!syscalls) return -1;

    SystemCallsImpl* impl = (SystemCallsImpl*)syscalls;
    if (!impl->initialized) return -1;

    pid_t pid = fork();

    // 更新统计信息
    if (pid == 0) {
        // 子进程
        pthread_mutex_lock(&impl->mutex);
        impl->stats.total_processes_created++;
        pthread_mutex_unlock(&impl->mutex);
    }

    return pid;
}

// 执行外部命令
int system_calls_execute_command(SystemCalls* syscalls, const char* command,
                                char* output, size_t output_size,
                                char* error_output, size_t error_size) {
    if (!syscalls || !command) return -1;

    SystemCallsImpl* impl = (SystemCallsImpl*)syscalls;
    if (!impl->initialized) return -1;

    FILE* pipe = popen(command, "r");
    if (!pipe) return -1;

    // 读取输出
    size_t output_read = 0;
    if (output && output_size > 0) {
        output_read = fread(output, 1, output_size - 1, pipe);
        if (output_read > 0) {
            output[output_read] = '\0';
        }
    }

    int status = pclose(pipe);

    // 更新统计信息
    pthread_mutex_lock(&impl->mutex);
    impl->stats.total_commands_executed++;
    if (WIFEXITED(status) && WEXITSTATUS(status) == 0) {
        impl->stats.successful_commands++;
    }
    pthread_mutex_unlock(&impl->mutex);

    return WIFEXITED(status) ? WEXITSTATUS(status) : -1;
}

// 获取环境变量
const char* system_calls_get_environment_variable(SystemCalls* syscalls, const char* name) {
    if (!syscalls || !name) return NULL;

    SystemCallsImpl* impl = (SystemCallsImpl*)syscalls;
    if (!impl->initialized) return NULL;

    return getenv(name);
}

// 设置环境变量
bool system_calls_set_environment_variable(SystemCalls* syscalls, const char* name, const char* value) {
    if (!syscalls || !name) return false;

    SystemCallsImpl* impl = (SystemCallsImpl*)syscalls;
    if (!impl->initialized) return false;

    int result = setenv(name, value ? value : "", 1);

    // 更新统计信息
    pthread_mutex_lock(&impl->mutex);
    impl->stats.total_env_vars_set++;
    pthread_mutex_unlock(&impl->mutex);

    return result == 0;
}

// 获取当前工作目录
const char* system_calls_get_current_working_directory(SystemCalls* syscalls, char* buffer, size_t size) {
    if (!syscalls || !buffer || size == 0) return NULL;

    SystemCallsImpl* impl = (SystemCallsImpl*)syscalls;
    if (!impl->initialized) return NULL;

    if (getcwd(buffer, size)) {
        return buffer;
    }

    return NULL;
}

// 改变当前工作目录
bool system_calls_change_working_directory(SystemCalls* syscalls, const char* path) {
    if (!syscalls || !path) return false;

    SystemCallsImpl* impl = (SystemCallsImpl*)syscalls;
    if (!impl->initialized) return false;

    int result = chdir(path);

    // 更新统计信息
    pthread_mutex_lock(&impl->mutex);
    impl->stats.total_directory_changes++;
    pthread_mutex_unlock(&impl->mutex);

    return result == 0;
}

// 获取系统负载
bool system_calls_get_system_load(SystemCalls* syscalls, double* load1, double* load5, double* load15) {
    if (!syscalls) return false;

    SystemCallsImpl* impl = (SystemCallsImpl*)syscalls;
    if (!impl->initialized) return false;

    double loads[3];
    if (getloadavg(loads, 3) == 3) {
        if (load1) *load1 = loads[0];
        if (load5) *load5 = loads[1];
        if (load15) *load15 = loads[2];

        // 更新统计信息
        pthread_mutex_lock(&impl->mutex);
        impl->stats.total_load_queries++;
        pthread_mutex_unlock(&impl->mutex);

        return true;
    }

    return false;
}

// 获取进程列表
ProcessInfo* system_calls_get_process_list(SystemCalls* syscalls, size_t* count) {
    if (!syscalls || !count) return NULL;

    SystemCallsImpl* impl = (SystemCallsImpl*)syscalls;
    if (!impl->initialized) return NULL;

    // 读取/proc目录获取所有进程ID
    DIR* proc_dir = opendir("/proc");
    if (!proc_dir) return NULL;

    // 估算进程数量
    size_t capacity = 1000;
    ProcessInfo* processes = (ProcessInfo*)malloc(sizeof(ProcessInfo) * capacity);
    if (!processes) {
        closedir(proc_dir);
        return NULL;
    }

    size_t process_count = 0;
    struct dirent* entry;

    while ((entry = readdir(proc_dir)) != NULL && process_count < capacity) {
        // 检查是否为进程目录（纯数字）
        char* endptr;
        pid_t pid = (pid_t)strtol(entry->d_name, &endptr, 10);
        if (*endptr == '\0' && pid > 0) {
            if (system_calls_get_process_info((SystemCalls*)impl, pid, &processes[process_count])) {
                process_count++;
            }
        }
    }

    closedir(proc_dir);

    *count = process_count;
    return processes;
}

// 释放进程列表
void system_calls_free_process_list(ProcessInfo* processes) {
    free(processes);
}

// 获取系统调用统计信息
bool system_calls_get_stats(const SystemCalls* syscalls, SystemCallsStats* stats) {
    if (!syscalls || !stats) return false;

    const SystemCallsImpl* impl = (SystemCallsImpl*)syscalls;

    pthread_mutex_lock(&impl->mutex);
    memcpy(stats, &impl->stats, sizeof(SystemCallsStats));
    pthread_mutex_unlock(&impl->mutex);

    return true;
}

// 设置信号处理器
bool system_calls_set_signal_handler(SystemCalls* syscalls, int signal, void (*handler)(int)) {
    if (!syscalls) return false;

    SystemCallsImpl* impl = (SystemCallsImpl*)syscalls;
    if (!impl->initialized) return false;

    struct sigaction sa;
    memset(&sa, 0, sizeof(sa));
    sa.sa_handler = handler;

    int result = sigaction(signal, &sa, NULL);

    // 更新统计信息
    pthread_mutex_lock(&impl->mutex);
    impl->stats.total_signal_handlers_set++;
    pthread_mutex_unlock(&impl->mutex);

    return result == 0;
}

// 发送信号到进程组
bool system_calls_send_signal_to_group(SystemCalls* syscalls, pid_t pgid, int signal) {
    if (!syscalls) return false;

    SystemCallsImpl* impl = (SystemCallsImpl*)syscalls;
    if (!impl->initialized) return false;

    int result = killpg(pgid, signal);

    // 更新统计信息
    pthread_mutex_lock(&impl->mutex);
    impl->stats.total_group_signals_sent++;
    if (result == 0) {
        impl->stats.successful_group_signals++;
    }
    pthread_mutex_unlock(&impl->mutex);

    return result == 0;
}

// 获取系统正常运行时间
bool system_calls_get_uptime(SystemCalls* syscalls, time_t* uptime_seconds) {
    if (!syscalls || !uptime_seconds) return false;

    SystemCallsImpl* impl = (SystemCallsImpl*)syscalls;
    if (!impl->initialized) return false;

    FILE* uptime_file = fopen("/proc/uptime", "r");
    if (!uptime_file) return false;

    double uptime;
    if (fscanf(uptime_file, "%lf", &uptime) == 1) {
        *uptime_seconds = (time_t)uptime;
        fclose(uptime_file);

        // 更新统计信息
        pthread_mutex_lock(&impl->mutex);
        impl->stats.total_uptime_queries++;
        pthread_mutex_unlock(&impl->mutex);

        return true;
    }

    fclose(uptime_file);
    return false;
}
