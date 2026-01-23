#ifndef FUTURE_COMBINATOR_H
#define FUTURE_COMBINATOR_H

#include "../entity/future.h"
#include <stdint.h>
#include <stdbool.h>

// 前向声明
struct FutureCombinator;

// Future组合器领域服务 - 提供Future的高级组合操作
typedef struct FutureCombinator {
    uint32_t max_concurrent_operations;    // 最大并发操作数
    uint32_t active_operations;            // 当前活动操作数
} FutureCombinator;

// Future组合器方法
FutureCombinator* future_combinator_create(uint32_t max_concurrent);
void future_combinator_destroy(FutureCombinator* combinator);

// Future组合操作

// all: 等待所有Future完成
Future* future_combinator_all(FutureCombinator* combinator, Future** futures, size_t count);

// race: 返回第一个完成的Future的结果
Future* future_combinator_race(FutureCombinator* combinator, Future** futures, size_t count);

// any: 返回第一个成功的Future的结果，忽略失败的
Future* future_combinator_any(FutureCombinator* combinator, Future** futures, size_t count);

// allSettled: 等待所有Future完成，无论成功还是失败
Future* future_combinator_all_settled(FutureCombinator* combinator, Future** futures, size_t count);

// 超时操作
Future* future_combinator_timeout(FutureCombinator* combinator, Future* future, uint32_t timeout_ms);

// 重试操作
Future* future_combinator_retry(FutureCombinator* combinator,
                                AsyncTaskFunc task_func,
                                void* arg,
                                uint32_t max_retries,
                                uint32_t delay_ms);

// 延迟操作
Future* future_combinator_delay(FutureCombinator* combinator, void* value, uint32_t delay_ms);

// 条件等待
Future* future_combinator_when(FutureCombinator* combinator,
                               bool (*condition)(void* context),
                               void* context,
                               uint32_t check_interval_ms);

// Future变换操作

// map: 对Future的结果应用函数
Future* future_combinator_map(FutureCombinator* combinator,
                              Future* future,
                              void* (*mapper)(void* input));

// flatMap: 对Future的结果应用返回Future的函数
Future* future_combinator_flat_map(FutureCombinator* combinator,
                                   Future* future,
                                   Future* (*flat_mapper)(void* input));

// filter: 过滤Future的结果
Future* future_combinator_filter(FutureCombinator* combinator,
                                 Future* future,
                                 bool (*predicate)(void* input));

// recover: 处理Future的错误
Future* future_combinator_recover(FutureCombinator* combinator,
                                  Future* future,
                                  void* (*recovery_func)(const char* error));

// 批处理操作

// batch: 将多个值批处理成单个Future
Future* future_combinator_batch(FutureCombinator* combinator,
                                void** values,
                                size_t count,
                                size_t batch_size);

// window: 在时间窗口内收集结果
Future* future_combinator_window(FutureCombinator* combinator,
                                 Future** futures,
                                 size_t count,
                                 uint32_t window_ms);

#endif // FUTURE_COMBINATOR_H
