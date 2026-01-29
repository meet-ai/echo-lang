#ifndef CHANNEL_AGGREGATE_H
#define CHANNEL_AGGREGATE_H

#include <stdint.h>
#include <stdbool.h>
#include <pthread.h>
#include "../../task_execution/aggregate/task.h"  // 使用TaskID
#include "../../task_execution/repository/task_repository.h"  // 使用TaskRepository
#include "../../task_execution/aggregate/coroutine.h"  // 使用Coroutine结构

// 前向声明
struct Channel;
struct ChannelBuffer;
struct EventBus;  // 事件总线

// 通道状态枚举（值对象）
typedef enum {
    CHANNEL_STATE_OPEN,
    CHANNEL_STATE_CLOSED
} ChannelState;

// TaskIDNode定义在task.h中（共享值对象）

// 通道聚合根（不透明指针）
typedef struct Channel Channel;

// ==================== 工厂方法 ====================
// 创建无缓冲通道
Channel* channel_factory_create_unbuffered(struct EventBus* event_bus);

// 创建有缓冲通道
Channel* channel_factory_create_buffered(uint32_t capacity, struct EventBus* event_bus);

// ==================== 聚合根方法（业务操作）====================
// channel_send 发送数据到通道
// @param channel 通道聚合根
// @param task_repo TaskRepository实例（用于通过TaskID查找Task）
// @param sender_task_id 发送任务的TaskID
// @param value 要发送的数据
// @return 成功返回0，失败返回-1
// 业务规则：
// 1. 如果通道已关闭，返回错误
// 2. 如果无接收者且无缓冲，阻塞当前任务（添加到sender_queue）
// 3. 如果有接收者，唤醒接收者并传递数据
// 4. 如果有缓冲通道且缓冲区未满，直接写入缓冲区
// 5. 如果有缓冲通道且缓冲区已满，阻塞当前任务（添加到sender_queue）
int channel_aggregate_send(Channel* channel, TaskRepository* task_repo, TaskID sender_task_id, void* value);

// channel_receive 从通道接收数据
// @param channel 通道聚合根
// @param task_repo TaskRepository实例（用于通过TaskID查找Task）
// @param receiver_task_id 接收任务的TaskID
// @return 接收到的数据，如果通道已关闭且无数据返回NULL
// 业务规则：
// 1. 如果通道已关闭且无数据，返回NULL
// 2. 如果无发送者且无缓冲，阻塞当前任务（添加到receiver_queue）
// 3. 如果有发送者，唤醒发送者并接收数据
// 4. 如果有缓冲通道且缓冲区有数据，直接读取数据
// 5. 如果有缓冲通道且缓冲区为空，阻塞当前任务（添加到receiver_queue）
void* channel_aggregate_receive(Channel* channel, TaskRepository* task_repo, TaskID receiver_task_id);

// channel_close 关闭通道
// @param channel 通道聚合根
// @param task_repo TaskRepository实例（用于通过TaskID查找Task并唤醒）
// 业务规则：
// 1. 设置通道状态为CLOSED
// 2. 唤醒所有等待的发送者和接收者（通过TaskRepository查找Task，然后通过Task聚合根方法唤醒）
// 3. 发布ChannelClosed领域事件
void channel_aggregate_close(Channel* channel, TaskRepository* task_repo);

// ==================== 查询方法 ====================
// channel_get_id 获取通道ID
uint64_t channel_get_id(const Channel* channel);

// channel_get_state 获取通道状态
ChannelState channel_get_state(const Channel* channel);

// channel_is_closed 检查通道是否已关闭
bool channel_is_closed(const Channel* channel);

// channel_get_capacity 获取通道容量
uint32_t channel_get_capacity(const Channel* channel);

// channel_get_size 获取当前数据数量
uint32_t channel_get_size(const Channel* channel);

// ==================== 领域事件管理 ====================
// channel_get_domain_events 获取并清空领域事件列表
// 返回事件数组，调用者负责释放内存
void** channel_get_domain_events(Channel* channel, size_t* count);

// ==================== 不变条件验证 ====================
// channel_validate_invariants 验证聚合不变条件
// 返回0表示有效，非0表示违反不变条件
int channel_validate_invariants(const Channel* channel);

// ==================== 销毁方法 ====================
// channel_destroy 销毁通道聚合根
void channel_aggregate_destroy(Channel* channel);

#endif // CHANNEL_AGGREGATE_H
