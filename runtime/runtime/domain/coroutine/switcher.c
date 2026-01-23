/**
 * @file switcher.c
 * @brief ContextSwitcher 领域服务实现
 *
 * ContextSwitcher 负责协程间的上下文切换逻辑。
 */

#include "context.h"
#include <stdio.h>

// ============================================================================
// ContextSwitcher 领域服务
// ============================================================================

/**
 * @brief 切换到目标上下文
 */
void context_switcher_switch_to(context_t* from, context_t* to) {
    if (!from || !to) {
        fprintf(stderr, "ERROR: Invalid contexts for switching\n");
        return;
    }

    printf("DEBUG: ContextSwitcher switching from %llu to %llu\n",
           context_get_id(from), context_get_id(to));

    context_switch(from, to);
}

/**
 * @brief 保存当前上下文
 */
void context_switcher_save(context_t* context) {
    if (!context) return;

    printf("DEBUG: ContextSwitcher saving context %llu\n", context_get_id(context));
    context_save(context);
}

/**
 * @brief 恢复指定上下文
 */
void context_switcher_restore(context_t* context) {
    if (!context) return;

    printf("DEBUG: ContextSwitcher restoring context %llu\n", context_get_id(context));
    context_restore(context);
}

/**
 * @brief 初始化协程上下文
 */
void context_switcher_init_coroutine(context_t* context, void (*entry_point)(void*), void* arg) {
    if (!context) return;

    printf("DEBUG: ContextSwitcher initializing coroutine context %llu\n", context_get_id(context));
    context_init(context, entry_point, arg);
}

/**
 * @brief 验证上下文切换能力
 */
bool context_switcher_validate(void) {
    // 创建两个测试上下文
    context_t* ctx1 = context_create(4096);
    context_t* ctx2 = context_create(4096);

    if (!ctx1 || !ctx2) {
        fprintf(stderr, "Failed to create test contexts\n");
        return false;
    }

    bool success = context_is_valid(ctx1) && context_is_valid(ctx2);

    context_destroy(ctx1);
    context_destroy(ctx2);

    return success;
}

