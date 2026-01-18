/**
 * @file machine.h
 * @brief Machine (M) 实体定义
 *
 * Machine 是GMP模型中的M，代表操作系统线程。
 */

#ifndef MACHINE_H
#define MACHINE_H

#include "../../../include/echo/runtime.h"
#include <pthread.h>

// 前向声明
struct Processor;

// ============================================================================
// Machine 实体定义
// ============================================================================

/**
 * @brief Machine 实体结构体
 * GMP模型中的M：操作系统线程，执行处理器调度
 */
typedef struct Machine {
    uint32_t id;                    // 机器ID
    pthread_t thread;               // POSIX线程
    struct Processor* bound_processor; // 绑定的处理器

    // 状态管理
    bool is_running;                // 是否正在运行
    bool should_stop;               // 是否应该停止

    // 统计信息
    uint64_t tasks_processed;       // 处理的任务总数

    // 同步机制
    pthread_mutex_t state_mutex;    // 状态保护锁
    pthread_cond_t state_cond;      // 状态条件变量
} machine_t;

// ============================================================================
// Machine 实体行为接口
// ============================================================================

/**
 * @brief 创建机器
 * @param id 机器ID
 * @return 新创建的机器
 */
machine_t* machine_create(uint32_t id);

/**
 * @brief 销毁机器
 * @param machine 要销毁的机器
 */
void machine_destroy(machine_t* machine);

/**
 * @brief 绑定处理器
 * @param machine 机器
 * @param processor 要绑定的处理器
 */
void machine_bind_processor(machine_t* machine, struct Processor* processor);

/**
 * @brief 启动机器线程
 * @param machine 机器
 * @return 成功返回0，失败返回-1
 */
int machine_start(machine_t* machine);

/**
 * @brief 停止机器线程
 * @param machine 机器
 */
void machine_stop(machine_t* machine);

/**
 * @brief 等待机器线程结束
 * @param machine 机器
 * @return 成功返回0，失败返回-1
 */
int machine_join(machine_t* machine);

/**
 * @brief 检查机器是否正在运行
 * @param machine 机器
 * @return true如果正在运行
 */
bool machine_is_running(const machine_t* machine);

/**
 * @brief 获取机器统计信息
 * @param machine 机器
 * @param processed 已处理任务数
 */
void machine_get_stats(const machine_t* machine, uint64_t* processed);

#endif // MACHINE_H
