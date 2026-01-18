/**
 * @file runtime.h
 * @brief Runtime 组装服务 - 异步运行时入口
 *
 * Runtime 提供统一的异步运行时入口，组装Executor和Reactor。
 */

#ifndef RUNTIME_H
#define RUNTIME_H

#include "../../include/echo/task.h"

// 前向声明
struct Executor;
struct Reactor;

// ============================================================================
// Runtime 组装服务定义
// ============================================================================

/**
 * @brief Runtime 结构体
 */
typedef struct Runtime {
    struct Executor* executor;  // 执行器
    struct Reactor* reactor;    // 反应器
    bool initialized;           // 是否已初始化
} runtime_t;

// ============================================================================
// Runtime 组装服务接口
// ============================================================================

/**
 * @brief 创建Runtime实例
 * 工厂方法模式：组装完整的异步运行时
 * @return 新创建的Runtime实例
 */
runtime_t* runtime_create(void);

/**
 * @brief 销毁Runtime实例
 * @param runtime 要销毁的Runtime
 */
void runtime_destroy(runtime_t* runtime);

/**
 * @brief 运行主任务
 * 统一入口：启动异步运行时并执行主任务
 * @param runtime Runtime实例
 * @param main_task 主任务
 * @return 成功返回0，失败返回-1
 */
int runtime_run(runtime_t* runtime, task_t* main_task);

/**
 * @brief 停止Runtime
 * @param runtime Runtime实例
 */
void runtime_shutdown(runtime_t* runtime);

/**
 * @brief 检查Runtime是否正在运行
 * @param runtime Runtime实例
 * @return true如果正在运行
 */
bool runtime_is_running(const runtime_t* runtime);

#endif // RUNTIME_H

