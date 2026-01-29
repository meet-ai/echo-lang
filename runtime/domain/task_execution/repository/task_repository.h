/**
 * @file task_repository.h
 * @brief TaskExecution 聚合仓储接口
 *
 * 仓储接口定义在领域层，遵循依赖倒置原则。
 * 具体实现在基础设施层（如内存实现、数据库实现等）。
 *
 * 使用场景：
 * - 应用服务通过仓储接口保存和查询Task聚合
 * - 支持多种持久化实现（内存、数据库、文件等）
 *
 * 示例：
 * ```c
 * // 在应用服务中使用仓储接口
 * TaskRepository* repo = task_repository_create();
 * Task* task = task_factory_create(entry_point, arg, stack_size);
 * task_repository_save(repo, task);
 *
 * // 查询任务
 * Task* found = task_repository_find_by_id(repo, task_id);
 * ```
 */

#ifndef TASK_EXECUTION_REPOSITORY_TASK_REPOSITORY_H
#define TASK_EXECUTION_REPOSITORY_TASK_REPOSITORY_H

#include "../aggregate/task.h"
#include <stdbool.h>

// ============================================================================
// 仓储接口定义
// ============================================================================

/**
 * @brief Task仓储接口
 *
 * 提供Task聚合的持久化操作，遵循集合语义。
 * 实现应该在基础设施层。
 */
typedef struct TaskRepository TaskRepository;

/**
 * @brief 根据ID查找Task聚合
 *
 * @param repo 仓储实例
 * @param task_id 任务ID
 * @return Task* 找到的Task聚合，如果不存在返回NULL
 */
Task* task_repository_find_by_id(TaskRepository* repo, TaskID task_id);

/**
 * @brief 保存Task聚合
 *
 * @param repo 仓储实例
 * @param task 要保存的Task聚合
 * @return int 成功返回0，失败返回-1
 */
int task_repository_save(TaskRepository* repo, Task* task);

/**
 * @brief 删除Task聚合
 *
 * @param repo 仓储实例
 * @param task_id 要删除的任务ID
 * @return int 成功返回0，失败返回-1
 */
int task_repository_delete(TaskRepository* repo, TaskID task_id);

/**
 * @brief 检查Task是否存在
 *
 * @param repo 仓储实例
 * @param task_id 任务ID
 * @return bool 存在返回true，不存在返回false
 */
bool task_repository_exists(TaskRepository* repo, TaskID task_id);

/**
 * @brief 获取所有Task聚合（用于调试和监控）
 *
 * @param repo 仓储实例
 * @param tasks 输出参数，Task指针数组
 * @param count 输出参数，Task数量
 * @return int 成功返回0，失败返回-1
 *
 * 注意：调用者负责释放返回的Task数组（但不释放Task对象本身）
 */
int task_repository_find_all(TaskRepository* repo, Task*** tasks, size_t* count);

/**
 * @brief 根据状态查找Task聚合列表
 *
 * @param repo 仓储实例
 * @param status 任务状态
 * @param tasks 输出参数，Task指针数组
 * @param count 输出参数，Task数量
 * @return int 成功返回0，失败返回-1
 */
int task_repository_find_by_status(
    TaskRepository* repo,
    TaskStatus status,
    Task*** tasks,
    size_t* count
);

/**
 * @brief 销毁仓储实例
 *
 * @param repo 仓储实例
 */
void task_repository_destroy(TaskRepository* repo);

#endif // TASK_EXECUTION_REPOSITORY_TASK_REPOSITORY_H
