#ifndef CHANNEL_ADAPTER_H
#define CHANNEL_ADAPTER_H

#include "../aggregate/channel.h"
#include "../../task_execution/repository/task_repository.h"
#include "../../task_execution/aggregate/task.h"
#include "../../scheduler/scheduler.h"  // 用于current_task
// 事件总线接口
#include "../../shared/events/bus.h"

/**
 * @brief Channel适配层：用于向后兼容旧的Channel接口
 * 
 * 这个适配层将旧的Channel*类型转换为新的Channel聚合根，
 * 并提供包装函数，从current_task获取TaskID，然后调用新的聚合根方法。
 * 
 * 使用场景：
 * - 在迁移期间，旧的代码可以继续使用旧的接口
 * - 新的代码应该直接使用新的聚合根方法
 * 
 * TODO: 阶段4后续重构：完全迁移到新的聚合根方法后，可以移除这个适配层
 */

// ==================== 全局TaskRepository管理（暂时方案）====================
// TODO: 阶段4后续重构：应该通过依赖注入传递TaskRepository，而不是使用全局变量

/**
 * @brief 设置全局TaskRepository（用于适配层）
 * @param task_repo TaskRepository实例
 */
void channel_adapter_set_task_repository(TaskRepository* task_repo);

/**
 * @brief 获取全局TaskRepository
 * @return TaskRepository实例，如果未设置返回NULL
 */
TaskRepository* channel_adapter_get_task_repository(void);

// ==================== 全局EventBus管理（暂时方案）====================
// TODO: 阶段7后续重构：应该通过依赖注入传递EventBus，而不是使用全局变量

/**
 * @brief 设置全局EventBus（用于适配层）
 * @param event_bus EventBus实例
 */
void channel_adapter_set_event_bus(EventBus* event_bus);

/**
 * @brief 获取全局EventBus
 * @return EventBus实例，如果未设置返回NULL
 */
EventBus* channel_adapter_get_event_bus(void);

// ==================== Channel映射管理 ====================

/**
 * @brief 注册Channel映射（将旧的Channel*映射到新的Channel聚合根）
 * @param legacy_channel 旧的Channel*指针
 * @param new_channel 新的Channel聚合根
 * @return 成功返回0，失败返回-1
 */
int channel_adapter_register_mapping(void* legacy_channel, Channel* new_channel);

// ==================== 类型转换 ====================
// 将旧的Channel*转换为新的Channel聚合根
// 注意：这需要两个Channel结构兼容，如果不兼容，需要创建新的Channel聚合根
Channel* channel_adapter_from_legacy(void* legacy_channel);

// ==================== 包装函数（向后兼容）====================
// 这些函数从current_task获取TaskID，然后调用新的聚合根方法

// channel_adapter_send 发送数据到通道（适配层）
// @param legacy_channel 旧的Channel*指针
// @param task_repo TaskRepository实例
// @param value 要发送的数据
// @return 成功返回0，失败返回-1
// 注意：这个函数从current_task获取TaskID，然后调用channel_send
int channel_adapter_send(void* legacy_channel, TaskRepository* task_repo, void* value);

// channel_adapter_receive 从通道接收数据（适配层）
// @param legacy_channel 旧的Channel*指针
// @param task_repo TaskRepository实例
// @return 接收到的数据，如果通道已关闭且无数据返回NULL
// 注意：这个函数从current_task获取TaskID，然后调用channel_receive
void* channel_adapter_receive(void* legacy_channel, TaskRepository* task_repo);

// channel_adapter_close 关闭通道（适配层）
// @param legacy_channel 旧的Channel*指针
// @param task_repo TaskRepository实例
// 注意：这个函数调用channel_close
void channel_adapter_close(void* legacy_channel, TaskRepository* task_repo);

#endif // CHANNEL_ADAPTER_H
