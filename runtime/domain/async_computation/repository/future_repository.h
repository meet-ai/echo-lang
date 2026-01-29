/**
 * @file future_repository.h
 * @brief FutureRepository 仓储接口定义
 *
 * FutureRepository 负责 Future 聚合的持久化和查询。
 * 接口定义在领域层，实现在基础设施层。
 */

#ifndef ASYNC_COMPUTATION_REPOSITORY_FUTURE_REPOSITORY_H
#define ASYNC_COMPUTATION_REPOSITORY_FUTURE_REPOSITORY_H

#include "../aggregate/future.h"
#include <stdbool.h>

// ============================================================================
// FutureRepository 仓储接口
// ============================================================================

/**
 * @brief FutureRepository 仓储接口（不透明类型）
 * 
 * 职责：
 * - 保存和查询 Future 聚合根
 * - 提供类似集合的接口
 * - 隐藏数据访问细节
 * 
 * 实际结构在实现文件中定义（如future_repository_memory.c）
 */
typedef struct FutureRepository FutureRepository;

/**
 * @brief 根据ID查找Future聚合
 *
 * @param repo 仓储实例
 * @param id Future ID
 * @return Future* 找到的Future，如果不存在返回NULL
 */
Future* future_repository_find_by_id(FutureRepository* repo, FutureID id);

/**
 * @brief 保存Future聚合
 *
 * @param repo 仓储实例
 * @param future 要保存的Future聚合
 * @return bool 成功返回true，失败返回false
 */
bool future_repository_save(FutureRepository* repo, Future* future);

/**
 * @brief 删除Future聚合
 *
 * @param repo 仓储实例
 * @param id 要删除的Future ID
 * @return bool 成功返回true，失败返回false
 */
bool future_repository_delete(FutureRepository* repo, FutureID id);

/**
 * @brief 获取所有Future聚合（用于调试和监控）
 *
 * @param repo 仓储实例
 * @param futures 输出参数，Future指针数组
 * @param count 输出参数，Future数量
 * @return bool 成功返回true，失败返回false
 *
 * 注意：调用者负责释放返回的Future数组（但不释放Future对象本身）
 */
bool future_repository_find_all(FutureRepository* repo, Future*** futures, size_t* count);

/**
 * @brief 根据状态查找Future聚合列表
 *
 * @param repo 仓储实例
 * @param state Future状态
 * @param futures 输出参数，Future指针数组
 * @param count 输出参数，Future数量
 * @return bool 成功返回true，失败返回false
 */
bool future_repository_find_by_state(
    FutureRepository* repo,
    FutureState state,
    Future*** futures,
    size_t* count
);

/**
 * @brief 销毁仓储实例
 *
 * @param repo 仓储实例
 */
void future_repository_destroy(FutureRepository* repo);

#endif // ASYNC_COMPUTATION_REPOSITORY_FUTURE_REPOSITORY_H
