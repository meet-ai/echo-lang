/**
 * @file context.h
 * @brief Context 实体定义
 *
 * Context 管理协程的执行上下文，包括栈指针、指令指针等。
 */

#ifndef CONTEXT_H
#define CONTEXT_H

#include "../../shared/types/common_types.h"

// ============================================================================
// Context 实体定义
// ============================================================================

/**
 * @brief Context 实体结构体
 * 实体模式：具有唯一标识，封装协程上下文状态
 */
typedef struct Context {
    uint64_t id;                 // 上下文ID（用于调试）
    void* stack_pointer;         // 栈指针
    void* instruction_pointer;   // 指令指针
    void* base_pointer;          // 基址指针
    size_t stack_size;           // 栈大小
    char* stack;                 // 栈内存
    bool is_main_context;        // 是否为主上下文
} context_t;

// ============================================================================
// Context 实体行为接口
// ============================================================================

/**
 * @brief 创建上下文
 * @param stack_size 栈大小
 * @return 新创建的上下文
 */
context_t* context_create(size_t stack_size);

/**
 * @brief 销毁上下文
 * @param context 要销毁的上下文
 */
void context_destroy(context_t* context);

/**
 * @brief 初始化上下文
 * @param context 上下文
 * @param entry_point 入口函数
 * @param arg 函数参数
 */
void context_init(context_t* context, void (*entry_point)(void*), void* arg);

/**
 * @brief 保存当前上下文
 * @param context 要保存到的上下文
 */
void context_save(context_t* context);

/**
 * @brief 恢复上下文
 * @param context 要恢复的上下文
 */
void context_restore(const context_t* context);

/**
 * @brief 切换上下文
 * @param from 当前上下文
 * @param to 目标上下文
 */
void context_switch(context_t* from, context_t* to);

/**
 * @brief 获取上下文ID
 * @param context 上下文
 * @return 上下文ID
 */
uint64_t context_get_id(const context_t* context);

/**
 * @brief 检查上下文是否有效
 * @param context 上下文
 * @return true如果有效
 */
bool context_is_valid(const context_t* context);

#endif // CONTEXT_H

