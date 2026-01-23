/**
 * @file metric.h
 * @brief 监控指标接口定义 - 共享内核
 *
 * 定义统一的监控指标接口，所有上下文都可以使用标准化的监控机制
 */

#ifndef METRIC_H
#define METRIC_H

#include <stdint.h>
#include <stdbool.h>

// 指标类型
typedef enum {
    METRIC_TYPE_COUNTER,    // 计数器（单调递增）
    METRIC_TYPE_GAUGE,      // 仪表（可增可减）
    METRIC_TYPE_HISTOGRAM,  // 直方图（分布统计）
    METRIC_TYPE_SUMMARY     // 摘要（百分位数）
} metric_type_t;

// 指标值
typedef union {
    int64_t counter;        // 计数器值
    double gauge;           // 仪表值
    struct {
        uint64_t count;     // 样本数量
        double sum;         // 样本总和
        double min;         // 最小值
        double max;         // 最大值
        double mean;        // 平均值
        double p50;         // 中位数
        double p95;         // 95百分位数
        double p99;         // 99百分位数
    } histogram;
} metric_value_t;

// 指标标签
typedef struct {
    const char* key;
    const char* value;
} metric_tag_t;

// 指标定义
typedef struct metric_t {
    const char* name;           // 指标名称
    const char* description;    // 指标描述
    metric_type_t type;         // 指标类型
    metric_tag_t* tags;         // 标签数组
    size_t tag_count;           // 标签数量
    metric_value_t value;       // 指标值
    uint64_t timestamp;         // 时间戳
} metric_t;

// 指标接口
typedef struct {
    // 创建指标
    metric_t* (*create)(const char* name, const char* description, metric_type_t type);

    // 设置标签
    bool (*set_tag)(metric_t* metric, const char* key, const char* value);

    // 更新值
    bool (*update_counter)(metric_t* metric, int64_t value);
    bool (*update_gauge)(metric_t* metric, double value);
    bool (*record_histogram)(metric_t* metric, double value);

    // 获取值
    bool (*get_value)(metric_t* metric, metric_value_t* value);

    // 销毁指标
    void (*destroy)(metric_t* metric);
} metric_interface_t;

// 全局指标接口实例
extern metric_interface_t* global_metric_interface;

// 便利函数
metric_t* metric_create_counter(const char* name, const char* description);
metric_t* metric_create_gauge(const char* name, const char* description);
metric_t* metric_create_histogram(const char* name, const char* description);

bool metric_set_tag(metric_t* metric, const char* key, const char* value);
bool metric_increment_counter(metric_t* metric);
bool metric_set_gauge(metric_t* metric, double value);
bool metric_record_value(metric_t* metric, double value);

#endif // METRIC_H

