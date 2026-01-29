#include "channel_events.h"
#include "../../shared/events/bus.h"  // 事件总线接口
#include <stdio.h>
#include <stdlib.h>
#include <time.h>
#include <string.h>
#include <stdint.h>

// 生成事件ID
static uint64_t generate_event_id(void) {
    static uint64_t next_event_id = 1;
    return next_event_id++;
}

// 事件类型常量
#define EVENT_TYPE_CHANNEL_DATA_SENT "channel.data.sent"
#define EVENT_TYPE_CHANNEL_DATA_RECEIVED "channel.data.received"
#define EVENT_TYPE_CHANNEL_CLOSED "channel.closed"

// ==================== 事件创建函数 ====================
ChannelDataSentEvent* channel_data_sent_event_create(
    uint64_t channel_id,
    uint64_t sender_task_id,
    void* data
) {
    ChannelDataSentEvent* event = (ChannelDataSentEvent*)malloc(sizeof(ChannelDataSentEvent));
    if (!event) {
        return NULL;
    }
    
    event->event_id = generate_event_id();
    event->channel_id = channel_id;
    event->sender_task_id = sender_task_id;
    event->data = data;
    event->occurred_at = time(NULL);
    
    return event;
}

ChannelDataReceivedEvent* channel_data_received_event_create(
    uint64_t channel_id,
    uint64_t receiver_task_id,
    void* data
) {
    ChannelDataReceivedEvent* event = (ChannelDataReceivedEvent*)malloc(sizeof(ChannelDataReceivedEvent));
    if (!event) {
        return NULL;
    }
    
    event->event_id = generate_event_id();
    event->channel_id = channel_id;
    event->receiver_task_id = receiver_task_id;
    event->data = data;
    event->occurred_at = time(NULL);
    
    return event;
}

ChannelClosedEvent* channel_closed_event_create(
    uint64_t channel_id,
    size_t waiting_senders_count,
    size_t waiting_receivers_count
) {
    ChannelClosedEvent* event = (ChannelClosedEvent*)malloc(sizeof(ChannelClosedEvent));
    if (!event) {
        return NULL;
    }
    
    event->event_id = generate_event_id();
    event->channel_id = channel_id;
    event->waiting_senders_count = waiting_senders_count;
    event->waiting_receivers_count = waiting_receivers_count;
    event->occurred_at = time(NULL);
    
    return event;
}

// ==================== 事件销毁函数 ====================
void channel_data_sent_event_destroy(ChannelDataSentEvent* event) {
    if (event) {
        free(event);
    }
}

void channel_data_received_event_destroy(ChannelDataReceivedEvent* event) {
    if (event) {
        free(event);
    }
}

void channel_closed_event_destroy(ChannelClosedEvent* event) {
    if (event) {
        free(event);
    }
}

// ============================================================================
// DomainEvent接口实现
// ============================================================================

// ChannelDataSentEvent的DomainEvent接口实现
static const char* channel_data_sent_event_get_type(const DomainEvent* event) {
    (void)event;  // 未使用
    return EVENT_TYPE_CHANNEL_DATA_SENT;
}

static time_t channel_data_sent_event_get_occurred_at(const DomainEvent* event) {
    if (!event || !event->data) return 0;
    ChannelDataSentEvent* sent_event = (ChannelDataSentEvent*)event->data;
    return sent_event->occurred_at;
}

static uint64_t channel_data_sent_event_get_id(const DomainEvent* event) {
    if (!event || !event->data) return 0;
    ChannelDataSentEvent* sent_event = (ChannelDataSentEvent*)event->data;
    return sent_event->event_id;
}

static void channel_data_sent_event_destroy_wrapper(DomainEvent* event) {
    if (!event || !event->data) return;
    ChannelDataSentEvent* sent_event = (ChannelDataSentEvent*)event->data;
    channel_data_sent_event_destroy(sent_event);
    free(event);
}

// ChannelDataReceivedEvent的DomainEvent接口实现
static const char* channel_data_received_event_get_type(const DomainEvent* event) {
    (void)event;  // 未使用
    return EVENT_TYPE_CHANNEL_DATA_RECEIVED;
}

static time_t channel_data_received_event_get_occurred_at(const DomainEvent* event) {
    if (!event || !event->data) return 0;
    ChannelDataReceivedEvent* received_event = (ChannelDataReceivedEvent*)event->data;
    return received_event->occurred_at;
}

static uint64_t channel_data_received_event_get_id(const DomainEvent* event) {
    if (!event || !event->data) return 0;
    ChannelDataReceivedEvent* received_event = (ChannelDataReceivedEvent*)event->data;
    return received_event->event_id;
}

static void channel_data_received_event_destroy_wrapper(DomainEvent* event) {
    if (!event || !event->data) return;
    ChannelDataReceivedEvent* received_event = (ChannelDataReceivedEvent*)event->data;
    channel_data_received_event_destroy(received_event);
    free(event);
}

// ChannelClosedEvent的DomainEvent接口实现
static const char* channel_closed_event_get_type(const DomainEvent* event) {
    (void)event;  // 未使用
    return EVENT_TYPE_CHANNEL_CLOSED;
}

static time_t channel_closed_event_get_occurred_at(const DomainEvent* event) {
    if (!event || !event->data) return 0;
    ChannelClosedEvent* closed_event = (ChannelClosedEvent*)event->data;
    return closed_event->occurred_at;
}

static uint64_t channel_closed_event_get_id(const DomainEvent* event) {
    if (!event || !event->data) return 0;
    ChannelClosedEvent* closed_event = (ChannelClosedEvent*)event->data;
    return closed_event->event_id;
}

static void channel_closed_event_destroy_wrapper(DomainEvent* event) {
    if (!event || !event->data) return;
    ChannelClosedEvent* closed_event = (ChannelClosedEvent*)event->data;
    channel_closed_event_destroy(closed_event);
    free(event);
}

// 转换函数：将具体事件转换为DomainEvent接口
DomainEvent* channel_data_sent_event_to_domain_event(ChannelDataSentEvent* event) {
    if (!event) return NULL;

    DomainEvent* domain_event = (DomainEvent*)calloc(1, sizeof(DomainEvent));
    if (!domain_event) {
        fprintf(stderr, "ERROR: Failed to allocate memory for DomainEvent wrapper\n");
        return NULL;
    }

    domain_event->event_type = channel_data_sent_event_get_type;
    domain_event->occurred_at = channel_data_sent_event_get_occurred_at;
    domain_event->event_id = channel_data_sent_event_get_id;
    domain_event->destroy = channel_data_sent_event_destroy_wrapper;
    domain_event->data = event;

    return domain_event;
}

DomainEvent* channel_data_received_event_to_domain_event(ChannelDataReceivedEvent* event) {
    if (!event) return NULL;

    DomainEvent* domain_event = (DomainEvent*)calloc(1, sizeof(DomainEvent));
    if (!domain_event) {
        fprintf(stderr, "ERROR: Failed to allocate memory for DomainEvent wrapper\n");
        return NULL;
    }

    domain_event->event_type = channel_data_received_event_get_type;
    domain_event->occurred_at = channel_data_received_event_get_occurred_at;
    domain_event->event_id = channel_data_received_event_get_id;
    domain_event->destroy = channel_data_received_event_destroy_wrapper;
    domain_event->data = event;

    return domain_event;
}

DomainEvent* channel_closed_event_to_domain_event(ChannelClosedEvent* event) {
    if (!event) return NULL;

    DomainEvent* domain_event = (DomainEvent*)calloc(1, sizeof(DomainEvent));
    if (!domain_event) {
        fprintf(stderr, "ERROR: Failed to allocate memory for DomainEvent wrapper\n");
        return NULL;
    }

    domain_event->event_type = channel_closed_event_get_type;
    domain_event->occurred_at = channel_closed_event_get_occurred_at;
    domain_event->event_id = channel_closed_event_get_id;
    domain_event->destroy = channel_closed_event_destroy_wrapper;
    domain_event->data = event;

    return domain_event;
}
