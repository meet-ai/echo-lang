/**
 * @file metric_collector.h
 * @brief 指标收集器接口定义
 *
 * 定义指标收集器的标准接口，负责收集和导出监控指标
 */

#ifndef METRIC_COLLECTOR_H
#define METRIC_COLLECTOR_H

#include "metric.h"
#include <stdbool.h>

// 收集器类型
typedef enum {
    COLLECTOR_TYPE_PUSH,     // 推送模式
    COLLECTOR_TYPE_PULL,     // 拉取模式
    COLLECTOR_TYPE_STREAM    // 流式模式
} collector_type_t;

// 导出格式
typedef enum {
    EXPORT_FORMAT_PROMETHEUS,  // Prometheus格式
    EXPORT_FORMAT_JSON,        // JSON格式
    EXPORT_FORMAT_GRAPHITE,    // Graphite格式
    EXPORT_FORMAT_INFLUXDB     // InfluxDB格式
} export_format_t;

// 收集器配置
typedef struct {
    collector_type_t type;           // 收集器类型
    export_format_t format;          // 导出格式
    const char* endpoint;            // 导出端点
    uint32_t interval_seconds;       // 收集间隔（秒）
    uint32_t timeout_seconds;        // 超时时间（秒）
    size_t max_batch_size;           // 最大批处理大小
    bool enable_compression;         // 是否启用压缩
} metric_collector_config_t;

// 指标收集器接口
typedef struct metric_collector_t {
    // 配置
    metric_collector_config_t config;

    // 方法
    bool (*init)(struct metric_collector_t* self, const metric_collector_config_t* config);
    bool (*collect)(struct metric_collector_t* self, metric_t* metric);
    bool (*flush)(struct metric_collector_t* self);
    bool (*export)(struct metric_collector_t* self, const char** output, size_t* output_size);
    void (*destroy)(struct metric_collector_t* self);

    // 私有数据
    void* private_data;
} metric_collector_t;

// 收集器工厂函数
typedef metric_collector_t* (*metric_collector_factory_t)(void);

// 注册和获取收集器
bool register_metric_collector(const char* name, metric_collector_factory_t factory);
metric_collector_t* get_metric_collector(const char* name);

// 便利函数
metric_collector_t* create_prometheus_collector(void);
metric_collector_t* create_json_collector(void);
metric_collector_t* create_graphite_collector(void);

// 默认收集器
extern metric_collector_t* default_metric_collector;

#endif // METRIC_COLLECTOR_H
