/**
 * @file yield.h
 * @brief Yield 操作接口
 */

#ifndef YIELD_H
#define YIELD_H

#include "coroutine.h"

// ============================================================================
// Yield 操作接口
// ============================================================================

/**
 * @brief 设置调度器上下文
 * @param context 调度器上下文
 */
void yield_set_scheduler_context(context_t* context);

/**
 * @brief 执行yield操作
 * @param coroutine 执行yield的协程
 */
void coroutine_yield_operation(coroutine_t* coroutine);

/**
 * @brief 从调度器恢复协程执行
 * @param coroutine 要恢复的协程
 */
void coroutine_resume_from_yield(coroutine_t* coroutine);

#endif // YIELD_H
