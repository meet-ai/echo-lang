/**
 * @file coroutine.h
 * @brief Coroutine 聚合根定义
 *
 * Coroutine 是用户空间轻量级线程，负责上下文切换和栈管理。
 */

#ifndef COROUTINE_H
#define COROUTINE_H

#include "../../../include/echo/runtime.h"

// 前向声明
struct Context;
typedef struct Context context_t;

// 注意：set_current_coroutine函数在yield.c中定义

// ============================================================================
// Coroutine 聚合根定义
// ============================================================================

/**
 * @brief Coroutine 聚合根结构体
 */
typedef struct Coroutine {
    coroutine_id_t id;           // 协程唯一标识
    coroutine_state_t state;     // 协程状态
    struct Context* context;     // 协程上下文
    struct Context* scheduler_context;  // 调度器上下文（用于协程完成时切换回）
    void* stack;                 // 栈内存
    size_t stack_size;           // 栈大小
    void (*entry_point)(void*);  // 入口函数
    void* arg;                   // 函数参数
    void* result;                // 执行结果
} coroutine_t;

// ============================================================================
// Coroutine 聚合根行为接口
// ============================================================================

/**
 * @brief 创建协程
 * @param entry_point 入口函数
 * @param arg 函数参数
 * @param stack_size 栈大小
 * @return 新创建的协程
 */
coroutine_t* coroutine_create(void (*entry_point)(void*), void* arg, size_t stack_size, context_t* scheduler_context);

/**
 * @brief 销毁协程
 * @param coroutine 要销毁的协程
 */
void coroutine_destroy(coroutine_t* coroutine);

/**
 * @brief 恢复协程执行
 * @param coroutine 要恢复的协程
 */
void coroutine_resume(coroutine_t* coroutine);

/**
 * @brief 挂起协程
 * @param coroutine 要挂起的协程
 */
void coroutine_suspend(coroutine_t* coroutine);

/**
 * @brief 让出协程执行权
 * @param coroutine 当前协程
 */
void coroutine_yield(coroutine_t* coroutine);

/**
 * @brief 检查协程是否完成
 * @param coroutine 协程
 * @return true如果已完成
 */
bool coroutine_is_completed(const coroutine_t* coroutine);

/**
 * @brief 获取协程ID
 * @param coroutine 协程
 * @return 协程ID
 */
coroutine_id_t coroutine_get_id(const coroutine_t* coroutine);

/**
 * @brief 获取协程状态
 * @param coroutine 协程
 * @return 协程状态
 */
coroutine_state_t coroutine_get_state(const coroutine_t* coroutine);

/**
 * @brief 获取协程上下文
 * @param coroutine 协程
 * @return 协程上下文
 */
struct Context* coroutine_get_context(const coroutine_t* coroutine);

#endif // COROUTINE_H

