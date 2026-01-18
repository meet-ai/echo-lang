/**
 * @file event_types.h
 * @brief 事件类型定义 - 共享内核
 *
 * 定义系统中所有标准事件类型的常量和枚举
 */

#ifndef EVENT_TYPES_H
#define EVENT_TYPES_H

// 事件类型前缀
#define EVENT_PREFIX_COROUTINE   "coroutine"
#define EVENT_PREFIX_SCHEDULER   "scheduler"
#define EVENT_PREFIX_GC          "gc"
#define EVENT_PREFIX_NETWORK     "network"
#define EVENT_PREFIX_FILESYSTEM  "filesystem"
#define EVENT_PREFIX_TIMER       "timer"
#define EVENT_PREFIX_EXCEPTION   "exception"
#define EVENT_PREFIX_DEBUG       "debug"

// 协程相关事件类型
#define EVENT_TYPE_COROUTINE_CREATED     EVENT_PREFIX_COROUTINE ".created"
#define EVENT_TYPE_COROUTINE_STARTED     EVENT_PREFIX_COROUTINE ".started"
#define EVENT_TYPE_COROUTINE_SUSPENDED   EVENT_PREFIX_COROUTINE ".suspended"
#define EVENT_TYPE_COROUTINE_RESUMED     EVENT_PREFIX_COROUTINE ".resumed"
#define EVENT_TYPE_COROUTINE_COMPLETED   EVENT_PREFIX_COROUTINE ".completed"
#define EVENT_TYPE_COROUTINE_FAILED      EVENT_PREFIX_COROUTINE ".failed"

// 调度器相关事件类型
#define EVENT_TYPE_TASK_SCHEDULED        EVENT_PREFIX_SCHEDULER ".task.scheduled"
#define EVENT_TYPE_TASK_STARTED          EVENT_PREFIX_SCHEDULER ".task.started"
#define EVENT_TYPE_TASK_COMPLETED        EVENT_PREFIX_SCHEDULER ".task.completed"
#define EVENT_TYPE_TASK_FAILED           EVENT_PREFIX_SCHEDULER ".task.failed"
#define EVENT_TYPE_WORK_STOLEN           EVENT_PREFIX_SCHEDULER ".work.stolen"
#define EVENT_TYPE_PROCESSOR_OVERLOADED  EVENT_PREFIX_SCHEDULER ".processor.overloaded"

// GC相关事件类型
#define EVENT_TYPE_GC_STARTED            EVENT_PREFIX_GC ".started"
#define EVENT_TYPE_GC_COMPLETED          EVENT_PREFIX_GC ".completed"
#define EVENT_TYPE_GC_MARK_PHASE_START   EVENT_PREFIX_GC ".mark.started"
#define EVENT_TYPE_GC_MARK_PHASE_END     EVENT_PREFIX_GC ".mark.completed"
#define EVENT_TYPE_GC_SWEEP_PHASE_START  EVENT_PREFIX_GC ".sweep.started"
#define EVENT_TYPE_GC_SWEEP_PHASE_END    EVENT_PREFIX_GC ".sweep.completed"
#define EVENT_TYPE_GC_STATS_UPDATED      EVENT_PREFIX_GC ".stats.updated"

// 网络相关事件类型
#define EVENT_TYPE_CONNECTION_ESTABLISHED EVENT_PREFIX_NETWORK ".connection.established"
#define EVENT_TYPE_CONNECTION_CLOSED      EVENT_PREFIX_NETWORK ".connection.closed"
#define EVENT_TYPE_DATA_RECEIVED          EVENT_PREFIX_NETWORK ".data.received"
#define EVENT_TYPE_DATA_SENT              EVENT_PREFIX_NETWORK ".data.sent"
#define EVENT_TYPE_NETWORK_ERROR          EVENT_PREFIX_NETWORK ".error"

// 文件系统相关事件类型
#define EVENT_TYPE_FILE_OPENED            EVENT_PREFIX_FILESYSTEM ".file.opened"
#define EVENT_TYPE_FILE_CLOSED            EVENT_PREFIX_FILESYSTEM ".file.closed"
#define EVENT_TYPE_FILE_READ              EVENT_PREFIX_FILESYSTEM ".file.read"
#define EVENT_TYPE_FILE_WRITTEN           EVENT_PREFIX_FILESYSTEM ".file.written"
#define EVENT_TYPE_FILE_ERROR             EVENT_PREFIX_FILESYSTEM ".file.error"

// 定时器相关事件类型
#define EVENT_TYPE_TIMER_SET              EVENT_PREFIX_TIMER ".set"
#define EVENT_TYPE_TIMER_TRIGGERED        EVENT_PREFIX_TIMER ".triggered"
#define EVENT_TYPE_TIMER_CANCELLED        EVENT_PREFIX_TIMER ".cancelled"

// 异常相关事件类型
#define EVENT_TYPE_EXCEPTION_THROWN       EVENT_PREFIX_EXCEPTION ".thrown"
#define EVENT_TYPE_EXCEPTION_CAUGHT       EVENT_PREFIX_EXCEPTION ".caught"
#define EVENT_TYPE_EXCEPTION_UNCAUGHT     EVENT_PREFIX_EXCEPTION ".uncaught"

// 调试监控相关事件类型
#define EVENT_TYPE_DEBUG_BREAKPOINT       EVENT_PREFIX_DEBUG ".breakpoint"
#define EVENT_TYPE_DEBUG_STEP             EVENT_PREFIX_DEBUG ".step"
#define EVENT_TYPE_DEBUG_VARIABLE         EVENT_PREFIX_DEBUG ".variable"
#define EVENT_TYPE_METRIC_UPDATED         EVENT_PREFIX_DEBUG ".metric.updated"

// 事件类型验证函数
bool is_valid_event_type(const char* event_type);
const char* get_event_category(const char* event_type);
bool is_coroutine_event(const char* event_type);
bool is_scheduler_event(const char* event_type);
bool is_gc_event(const char* event_type);
bool is_network_event(const char* event_type);
bool is_filesystem_event(const char* event_type);

#endif // EVENT_TYPES_H
