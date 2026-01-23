#ifndef ECHO_GC_PORTS_H
#define ECHO_GC_PORTS_H

#include <stdint.h>
#include <stdbool.h>

// 前向声明
struct echo_gc_heap;
struct echo_gc_collector;

// 事件发布端口
typedef void (*echo_gc_event_publisher_t)(const char* event_type, const void* event_data);

// 自适应GC提供者端口（防腐层）
typedef struct echo_adaptive_gc_provider {
    echo_gc_error_t (*analyze_workload)(
        void* provider,
        const echo_gc_metrics_t* metrics,
        echo_gc_workload_type_t* workload_type,
        double* confidence
    );

    echo_gc_error_t (*make_decisions)(
        void* provider,
        const echo_gc_metrics_t* metrics,
        echo_gc_decision_t** decisions,
        uint32_t* decision_count
    );

    echo_gc_error_t (*apply_decisions)(
        void* provider,
        const echo_gc_decision_t* decisions,
        uint32_t decision_count
    );
} echo_adaptive_gc_provider_t;

// 监控服务端口
typedef struct echo_monitoring_service {
    echo_gc_error_t (*collect_metrics)(
        void* service,
        echo_gc_metrics_t* metrics
    );

    echo_gc_error_t (*detect_anomalies)(
        void* service,
        const echo_gc_metrics_t* metrics,
        echo_gc_anomaly_t** anomalies,
        uint32_t* anomaly_count
    );

    echo_gc_error_t (*generate_report)(
        void* service,
        const echo_gc_metrics_t* metrics,
        echo_gc_report_t* report
    );
} echo_monitoring_service_t;

// 增强GC模块端口
typedef struct echo_enhanced_gc_module {
    const char* (*get_name)(void* module);
    const char* (*get_version)(void* module);

    echo_gc_error_t (*init)(void* module, const echo_gc_config_t* config);
    echo_gc_error_t (*start)(void* module);
    echo_gc_error_t (*stop)(void* module);
    echo_gc_error_t (*cleanup)(void* module);

    echo_gc_error_t (*before_gc)(void* module, struct echo_gc_heap* heap);
    echo_gc_error_t (*after_gc)(void* module, struct echo_gc_heap* heap, const echo_gc_metrics_t* metrics);

    echo_gc_error_t (*get_stats)(void* module, void* stats_buffer, size_t buffer_size);
} echo_enhanced_gc_module_t;

// 工作负载类型枚举
typedef enum {
    ECHO_GC_WORKLOAD_TRANSACTIONAL = 0,
    ECHO_GC_WORKLOAD_BATCH_PROCESSING = 1,
    ECHO_GC_WORKLOAD_CACHE_SERVER = 2,
    ECHO_GC_WORKLOAD_MIXED = 3
} echo_gc_workload_type_t;

// GC决策结构
typedef struct echo_gc_decision {
    char parameter_name[64];
    union {
        double double_value;
        uint64_t uint_value;
        int64_t int_value;
        bool bool_value;
    } value;
    char reason[256];
} echo_gc_decision_t;

// 异常结构
typedef struct echo_gc_anomaly {
    char type[64];
    char description[256];
    double severity;  // 0.0 - 1.0
    struct timespec detected_at;
} echo_gc_anomaly_t;

// 报告结构
typedef struct echo_gc_report {
    char title[128];
    char summary[1024];
    char recommendations[2048];
    struct timespec generated_at;
    uint32_t anomaly_count;
    echo_gc_anomaly_t anomalies[16];  // 最多16个异常
} echo_gc_report_t;

// 端口注册函数
echo_gc_error_t echo_gc_register_adaptive_provider(echo_adaptive_gc_provider_t* provider, void* provider_instance);
echo_gc_error_t echo_gc_register_monitoring_service(echo_monitoring_service_t* service, void* service_instance);
echo_gc_error_t echo_gc_register_enhanced_module(echo_enhanced_gc_module_t* module, void* module_instance);

// 端口查询函数
echo_gc_error_t echo_gc_get_adaptive_provider(echo_adaptive_gc_provider_t** provider, void** instance);
echo_gc_error_t echo_gc_get_monitoring_service(echo_monitoring_service_t** service, void** instance);
echo_gc_error_t echo_gc_get_enhanced_modules(echo_enhanced_gc_module_t*** modules, void*** instances, uint32_t* count);

#endif // ECHO_GC_PORTS_H
