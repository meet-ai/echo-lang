#ifndef CHANNEL_EVENTS_H
#define CHANNEL_EVENTS_H

#include <stdint.h>
#include <time.h>

// 通道创建事件
typedef struct {
    uint64_t event_id;
    uint64_t channel_id;
    size_t element_size;
    size_t buffer_size;
    time_t created_at;
} ChannelCreatedEvent;

// 通道关闭事件
typedef struct {
    uint64_t event_id;
    uint64_t channel_id;
    uint64_t messages_sent;
    uint64_t messages_received;
    time_t closed_at;
} ChannelClosedEvent;

// 消息发送事件
typedef struct {
    uint64_t event_id;
    uint64_t channel_id;
    time_t sent_at;
} MessageSentEvent;

// 消息接收事件
typedef struct {
    uint64_t event_id;
    uint64_t channel_id;
    time_t received_at;
} MessageReceivedEvent;

// 通道阻塞事件
typedef struct {
    uint64_t event_id;
    uint64_t channel_id;
    bool is_send_blocked;    // true=发送阻塞，false=接收阻塞
    time_t blocked_at;
} ChannelBlockedEvent;

// 通道解除阻塞事件
typedef struct {
    uint64_t event_id;
    uint64_t channel_id;
    time_t unblocked_at;
} ChannelUnblockedEvent;

// 事件类型枚举
typedef enum {
    EVENT_TYPE_CHANNEL_CREATED,
    EVENT_TYPE_CHANNEL_CLOSED,
    EVENT_TYPE_MESSAGE_SENT,
    EVENT_TYPE_MESSAGE_RECEIVED,
    EVENT_TYPE_CHANNEL_BLOCKED,
    EVENT_TYPE_CHANNEL_UNBLOCKED
} ChannelEventType;

// 通用事件联合体
typedef union {
    ChannelCreatedEvent created;
    ChannelClosedEvent closed;
    MessageSentEvent sent;
    MessageReceivedEvent received;
    ChannelBlockedEvent blocked;
    ChannelUnblockedEvent unblocked;
} ChannelEventData;

// 通用通道事件
typedef struct {
    ChannelEventType type;
    ChannelEventData data;
} ChannelEvent;

// 事件处理函数类型
typedef void (*ChannelEventHandler)(const ChannelEvent* event, void* context);

// 事件发布器接口
typedef struct ChannelEventPublisher {
    void (*publish)(struct ChannelEventPublisher* publisher, const ChannelEvent* event);
} ChannelEventPublisher;

// 创建事件函数
ChannelEvent* channel_event_create_created(uint64_t channel_id, size_t element_size, size_t buffer_size);
ChannelEvent* channel_event_create_closed(uint64_t channel_id, uint64_t sent_count, uint64_t recv_count);
ChannelEvent* channel_event_create_message_sent(uint64_t channel_id);
ChannelEvent* channel_event_create_message_received(uint64_t channel_id);
ChannelEvent* channel_event_create_blocked(uint64_t channel_id, bool is_send);
ChannelEvent* channel_event_create_unblocked(uint64_t channel_id);

// 销毁事件
void channel_event_destroy(ChannelEvent* event);

// 获取事件名称
const char* channel_event_type_name(ChannelEventType type);

#endif // CHANNEL_EVENTS_H
