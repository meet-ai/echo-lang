#ifndef CHANNEL_EVENTS_H
#define CHANNEL_EVENTS_H

#include "../../shared/events/bus.h"  // 使用新的DomainEvent接口
#include <stdint.h>
#include <time.h>

// 注意：旧的DomainEvent定义已移除，现在使用shared/events/bus.h中的DomainEvent接口

// ==================== 领域事件定义 ====================

/**
 * @brief ChannelDataSent 事件
 * 
 * 触发时机：数据成功发送到通道
 * 携带数据：通道ID、发送的任务ID、数据指针
 */
typedef struct ChannelDataSentEvent {
    uint64_t event_id;
    uint64_t channel_id;
    uint64_t sender_task_id;
    void* data;
    time_t occurred_at;
} ChannelDataSentEvent;

/**
 * @brief ChannelDataReceived 事件
 * 
 * 触发时机：数据成功从通道接收
 * 携带数据：通道ID、接收的任务ID、数据指针
 */
typedef struct ChannelDataReceivedEvent {
    uint64_t event_id;
    uint64_t channel_id;
    uint64_t receiver_task_id;
    void* data;
    time_t occurred_at;
} ChannelDataReceivedEvent;

/**
 * @brief ChannelClosed 事件
 * 
 * 触发时机：通道被关闭
 * 携带数据：通道ID、关闭时等待的发送者数量、接收者数量
 */
typedef struct ChannelClosedEvent {
    uint64_t event_id;
    uint64_t channel_id;
    size_t waiting_senders_count;
    size_t waiting_receivers_count;
    time_t occurred_at;
} ChannelClosedEvent;

// ==================== 事件创建函数 ====================
ChannelDataSentEvent* channel_data_sent_event_create(
    uint64_t channel_id,
    uint64_t sender_task_id,
    void* data
);

ChannelDataReceivedEvent* channel_data_received_event_create(
    uint64_t channel_id,
    uint64_t receiver_task_id,
    void* data
);

ChannelClosedEvent* channel_closed_event_create(
    uint64_t channel_id,
    size_t waiting_senders_count,
    size_t waiting_receivers_count
);

// ==================== 事件销毁函数 ====================
void channel_data_sent_event_destroy(ChannelDataSentEvent* event);
void channel_data_received_event_destroy(ChannelDataReceivedEvent* event);
void channel_closed_event_destroy(ChannelClosedEvent* event);

// ==================== DomainEvent接口实现 ====================

/**
 * @brief 将ChannelDataSentEvent转换为DomainEvent接口
 * 
 * 用于通过EventBus发布事件
 */
DomainEvent* channel_data_sent_event_to_domain_event(ChannelDataSentEvent* event);

/**
 * @brief 将ChannelDataReceivedEvent转换为DomainEvent接口
 * 
 * 用于通过EventBus发布事件
 */
DomainEvent* channel_data_received_event_to_domain_event(ChannelDataReceivedEvent* event);

/**
 * @brief 将ChannelClosedEvent转换为DomainEvent接口
 * 
 * 用于通过EventBus发布事件
 */
DomainEvent* channel_closed_event_to_domain_event(ChannelClosedEvent* event);

#endif // CHANNEL_EVENTS_H
