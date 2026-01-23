#ifndef ENVIRONMENT_CONFIG_H
#define ENVIRONMENT_CONFIG_H

#include <stdint.h>
#include <stdbool.h>

// 前向声明
struct EnvironmentConfig;
struct SystemEnvironment;
struct RuntimeEnvironment;
struct ExternalEnvironment;

// 系统环境信息
typedef struct SystemEnvironment {
    char hostname[256];          // 主机名
    char os_name[128];           // 操作系统名称
    char os_version[128];        // 操作系统版本
    char architecture[64];       // 系统架构
    uint32_t cpu_count;          // CPU核心数
    uint64_t total_memory;       // 总内存（字节）
    uint64_t available_memory;   // 可用内存（字节）
    uint64_t total_disk_space;   // 总磁盘空间
    uint64_t available_disk_space; // 可用磁盘空间
    char timezone[64];           // 时区
    char locale[64];             // 区域设置
    bool is_virtual_machine;     // 是否为虚拟机
    bool has_hypervisor;         // 是否有虚拟化层
} SystemEnvironment;

// 运行时环境信息
typedef struct RuntimeEnvironment {
    char runtime_version[32];    // 运行时版本
    char compiler_version[64];   // 编译器版本
    char build_date[32];         // 构建日期
    char build_time[32];         // 构建时间
    char git_commit[64];         // Git提交哈希
    char optimization_level[16]; // 优化级别
    bool debug_build;            // 是否为调试构建
    bool profiling_enabled;      // 是否启用性能分析
    char target_platform[64];    // 目标平台
    char target_architecture[32]; // 目标架构
} RuntimeEnvironment;

// 外部环境信息
typedef struct ExternalEnvironment {
    // 网络环境
    bool internet_connected;     // 是否连接互联网
    char public_ip[64];          // 公网IP
    char network_interfaces[1024]; // 网络接口信息（JSON）

    // 云环境
    bool is_cloud_environment;   // 是否为云环境
    char cloud_provider[64];     // 云提供商
    char instance_type[128];     // 实例类型
    char region[64];             // 区域
    char availability_zone[64];  // 可用区
    char instance_id[128];       // 实例ID

    // 容器环境
    bool is_containerized;       // 是否容器化
    char container_runtime[64];  // 容器运行时
    char container_image[256];   // 容器镜像
    char container_id[128];      // 容器ID

    // 外部服务
    char external_services[2048]; // 外部服务连接信息（JSON）
} ExternalEnvironment;

// 环境配置
typedef struct EnvironmentConfig {
    char config_id[128];         // 配置ID
    time_t collected_at;         // 收集时间
    char environment_type[32];   // 环境类型 ("development", "staging", "production")

    // 子环境信息
    struct SystemEnvironment system;
    struct RuntimeEnvironment runtime;
    struct ExternalEnvironment external;

    // 环境变量
    char** environment_variables; // 环境变量数组
    size_t env_var_count;         // 环境变量数量

    // 资源限制
    uint64_t max_memory_usage;    // 最大内存使用量
    uint64_t max_cpu_usage;       // 最大CPU使用率
    uint64_t max_disk_usage;      // 最大磁盘使用量
    uint32_t max_file_descriptors; // 最大文件描述符数
    uint32_t max_threads;         // 最大线程数

    // 安全设置
    bool sandbox_enabled;         // 是否启用沙箱
    char security_policy[1024];   // 安全策略（JSON）
    bool privilege_dropping;      // 是否降权运行
    char user_id[64];             // 运行用户ID
    char group_id[64];            // 运行组ID

    // 监控和日志
    bool monitoring_enabled;      // 是否启用监控
    bool structured_logging;      // 是否启用结构化日志
    char log_level[16];           // 日志级别
    char metrics_endpoint[256];   // 指标端点
    char traces_endpoint[256];    // 追踪端点
} EnvironmentConfig;

// 环境配置管理函数
EnvironmentConfig* environment_config_create(void);
void environment_config_destroy(EnvironmentConfig* config);

// 环境信息收集
bool environment_config_collect_system_info(EnvironmentConfig* config);
bool environment_config_collect_runtime_info(EnvironmentConfig* config);
bool environment_config_collect_external_info(EnvironmentConfig* config);
bool environment_config_collect_all(EnvironmentConfig* config);

// 环境检测
bool environment_config_detect_cloud_provider(const EnvironmentConfig* config,
                                             char* provider, size_t provider_size);
bool environment_config_is_development(const EnvironmentConfig* config);
bool environment_config_is_production(const EnvironmentConfig* config);
bool environment_config_is_containerized(const EnvironmentConfig* config);

// 环境变量管理
bool environment_config_get_env_var(const EnvironmentConfig* config,
                                   const char* key, char* value, size_t value_size);
bool environment_config_set_env_var(EnvironmentConfig* config, const char* key, const char* value);
bool environment_config_unset_env_var(EnvironmentConfig* config, const char* key);

// 资源限制检查
bool environment_config_check_memory_limit(const EnvironmentConfig* config, uint64_t requested_memory);
bool environment_config_check_cpu_limit(const EnvironmentConfig* config, double requested_cpu);
bool environment_config_check_disk_limit(const EnvironmentConfig* config, uint64_t requested_disk);

// 安全检查
bool environment_config_validate_security(const EnvironmentConfig* config,
                                         char* issues, size_t issues_size);

// 配置序列化
char* environment_config_to_json(const EnvironmentConfig* config);
EnvironmentConfig* environment_config_from_json(const char* json_config);

// 配置比较（用于环境变更检测）
typedef struct EnvironmentDiff {
    char** changed_fields;       // 变更字段列表
    size_t changed_count;        // 变更数量
    char* diff_details;          // 详细差异（JSON）
} EnvironmentDiff;

EnvironmentDiff* environment_config_diff(const EnvironmentConfig* old_config,
                                        const EnvironmentConfig* new_config);
void environment_diff_destroy(EnvironmentDiff* diff);

// 环境健康检查
typedef enum {
    HEALTH_OK,                  // 健康
    HEALTH_WARNING,             // 警告
    HEALTH_CRITICAL,            // 严重
    HEALTH_UNKNOWN              // 未知
} EnvironmentHealthStatus;

typedef struct EnvironmentHealthCheck {
    EnvironmentHealthStatus overall_status; // 整体状态
    char** warnings;             // 警告列表
    size_t warning_count;        // 警告数量
    char** critical_issues;      // 严重问题列表
    size_t critical_count;       // 严重问题数量
    time_t checked_at;           // 检查时间
} EnvironmentHealthCheck;

EnvironmentHealthCheck* environment_config_health_check(const EnvironmentConfig* config);
void environment_health_check_destroy(EnvironmentHealthCheck* check);

// 便捷访问函数
static inline bool environment_config_is_cloud(const EnvironmentConfig* config) {
    return config->external.is_cloud_environment;
}

static inline bool environment_config_is_container(const EnvironmentConfig* config) {
    return config->external.is_containerized;
}

static inline uint32_t environment_config_get_cpu_count(const EnvironmentConfig* config) {
    return config->system.cpu_count;
}

static inline uint64_t environment_config_get_total_memory(const EnvironmentConfig* config) {
    return config->system.total_memory;
}

#endif // ENVIRONMENT_CONFIG_H
