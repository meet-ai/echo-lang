/**
 * @file switcher.h
 * @brief ContextSwitcher 领域服务接口
 *
 * ContextSwitcher 协调多个协程间的上下文切换。
 */

#ifndef SWITCHER_H
#define SWITCHER_H

#include "context.h"

// ============================================================================
// ContextSwitcher 领域服务接口
// ============================================================================

/**
 * @brief 切换到目标上下文
 * @param from 当前上下文
 * @param to 目标上下文
 */
void context_switcher_switch_to(context_t* from, context_t* to);

/**
 * @brief 保存当前上下文
 * @param context 要保存的上下文
 */
void context_switcher_save(context_t* context);

/**
 * @brief 恢复指定上下文
 * @param context 要恢复的上下文
 */
void context_switcher_restore(context_t* context);

/**
 * @brief 初始化协程上下文
 * @param context 要初始化的上下文
 * @param entry_point 入口函数
 * @param arg 函数参数
 */
void context_switcher_init_coroutine(context_t* context, void (*entry_point)(void*), void* arg);

/**
 * @brief 验证上下文切换能力
 * @return true如果验证通过
 */
bool context_switcher_validate(void);

#endif // SWITCHER_H

