#include "channel.h"
#include "../events/channel_events.h"  // 领域事件定义
#include "../../shared/events/bus.h"  // 事件总线接口
#include "../../task_execution/repository/task_repository.h"  // 用于通过TaskID查找Task
#include "../../scheduler/scheduler.h"  // 用于current_task
#include <stdio.h>
#include <stdlib.h>
#include <string.h>
#include <assert.h>

// ==================== 内部结构定义 ====================
// ChannelBuffer：内部实体（值对象），用于有缓冲通道
typedef struct ChannelBuffer {
    void** data;              // 缓冲区数据数组
    uint32_t capacity;        // 容量
    uint32_t size;           // 当前大小
    uint32_t read_pos;       // 读指针（环形缓冲区）
    uint32_t write_pos;      // 写指针（环形缓冲区）
} ChannelBuffer;

// Channel聚合根内部结构
struct Channel {
    // 聚合根标识
    uint64_t id;
    
    // 值对象
    ChannelState state;       // 通道状态
    
    // 内部实体
    ChannelBuffer* buffer;    // 缓冲区（有缓冲通道使用）
    void* temp_value;         // 临时值存储（无缓冲通道的直接传递）
    
    // TaskID等待者队列（改为TaskID列表）
    TaskIDNode* sender_queue;    // 等待发送的TaskID列表
    TaskIDNode* receiver_queue;  // 等待接收的TaskID列表
    
    // 同步机制
    pthread_mutex_t lock;
    pthread_cond_t send_cond;  // 发送条件变量
    pthread_cond_t recv_cond;  // 接收条件变量
    
    // 通道属性
    uint32_t capacity;        // 容量（0表示无缓冲）
    uint32_t size;           // 当前大小
    
    // 领域事件列表（待发布）
    void** domain_events;
    size_t domain_events_count;
    size_t domain_events_capacity;
    
    // 事件总线（可选，用于发布领域事件）
    struct EventBus* event_bus;  // 事件总线实例
};

// ==================== 辅助函数 ====================
// 创建TaskIDNode
static TaskIDNode* task_id_node_create(TaskID task_id) {
    TaskIDNode* node = (TaskIDNode*)malloc(sizeof(TaskIDNode));
    if (!node) {
        return NULL;
    }
    node->task_id = task_id;
    node->next = NULL;
    return node;
}

// 释放TaskIDNode列表
static void task_id_node_list_free(TaskIDNode* head) {
    while (head) {
        TaskIDNode* next = head->next;
        free(head);
        head = next;
    }
}

// 添加TaskID到队列
static void task_id_queue_add(TaskIDNode** queue, TaskID task_id) {
    TaskIDNode* node = task_id_node_create(task_id);
    if (!node) {
        return;
    }
    
    if (*queue == NULL) {
        *queue = node;
    } else {
        TaskIDNode* current = *queue;
        while (current->next != NULL) {
            current = current->next;
        }
        current->next = node;
    }
}

// 从队列移除并返回第一个TaskID
static TaskID task_id_queue_remove(TaskIDNode** queue) {
    if (*queue == NULL) {
        return 0;  // 无效的TaskID
    }
    
    TaskIDNode* first = *queue;
    TaskID task_id = first->task_id;
    *queue = first->next;
    free(first);
    
    return task_id;
}

// 创建ChannelBuffer
static ChannelBuffer* channel_buffer_create(uint32_t capacity) {
    if (capacity == 0) {
        return NULL;
    }
    
    ChannelBuffer* buffer = (ChannelBuffer*)malloc(sizeof(ChannelBuffer));
    if (!buffer) {
        return NULL;
    }
    
    buffer->data = (void**)malloc(sizeof(void*) * capacity);
    if (!buffer->data) {
        free(buffer);
        return NULL;
    }
    
    buffer->capacity = capacity;
    buffer->size = 0;
    buffer->read_pos = 0;
    buffer->write_pos = 0;
    
    // 初始化缓冲区为NULL
    for (uint32_t i = 0; i < capacity; i++) {
        buffer->data[i] = NULL;
    }
    
    return buffer;
}

// 销毁ChannelBuffer
static void channel_buffer_destroy(ChannelBuffer* buffer) {
    if (buffer) {
        if (buffer->data) {
            free(buffer->data);
        }
        free(buffer);
    }
}

// ==================== 工厂方法 ====================
Channel* channel_factory_create_unbuffered(struct EventBus* event_bus) {
    Channel* ch = (Channel*)calloc(1, sizeof(Channel));
    if (!ch) {
        fprintf(stderr, "ERROR: Failed to allocate memory for channel\n");
        return NULL;
    }
    
    static uint64_t next_channel_id = 1;
    ch->id = next_channel_id++;
    ch->capacity = 0;  // 无缓冲通道
    ch->size = 0;
    ch->state = CHANNEL_STATE_OPEN;
    ch->buffer = NULL;  // 无缓冲通道不需要缓冲区
    ch->temp_value = NULL;
    ch->sender_queue = NULL;
    ch->receiver_queue = NULL;
    ch->domain_events = NULL;
    ch->domain_events_count = 0;
    ch->domain_events_capacity = 0;
    ch->event_bus = event_bus;  // 初始化事件总线
    
    // 初始化同步机制
    if (pthread_mutex_init(&ch->lock, NULL) != 0) {
        fprintf(stderr, "ERROR: Failed to initialize channel mutex\n");
        free(ch);
        return NULL;
    }
    if (pthread_cond_init(&ch->send_cond, NULL) != 0) {
        fprintf(stderr, "ERROR: Failed to initialize send condition variable\n");
        pthread_mutex_destroy(&ch->lock);
        free(ch);
        return NULL;
    }
    if (pthread_cond_init(&ch->recv_cond, NULL) != 0) {
        fprintf(stderr, "ERROR: Failed to initialize receive condition variable\n");
        pthread_cond_destroy(&ch->send_cond);
        pthread_mutex_destroy(&ch->lock);
        free(ch);
        return NULL;
    }
    
    printf("DEBUG: Created unbuffered channel %llu\n", ch->id);
    return ch;
}

Channel* channel_factory_create_buffered(uint32_t capacity, struct EventBus* event_bus) {
    if (capacity == 0) {
        fprintf(stderr, "ERROR: Cannot create buffered channel with capacity 0\n");
        return NULL;
    }
    
    Channel* ch = (Channel*)calloc(1, sizeof(Channel));
    if (!ch) {
        fprintf(stderr, "ERROR: Failed to allocate memory for channel\n");
        return NULL;
    }
    
    static uint64_t next_channel_id = 1;
    ch->id = next_channel_id++;
    ch->capacity = capacity;
    ch->size = 0;
    ch->state = CHANNEL_STATE_OPEN;
    ch->temp_value = NULL;
    ch->sender_queue = NULL;
    ch->receiver_queue = NULL;
    ch->domain_events = NULL;
    ch->domain_events_count = 0;
    ch->domain_events_capacity = 0;
    ch->event_bus = event_bus;  // 初始化事件总线
    
    // 创建缓冲区
    ch->buffer = channel_buffer_create(capacity);
    if (!ch->buffer) {
        fprintf(stderr, "ERROR: Failed to allocate buffer for channel %llu\n", ch->id);
        free(ch);
        return NULL;
    }
    
    // 初始化同步机制
    if (pthread_mutex_init(&ch->lock, NULL) != 0) {
        fprintf(stderr, "ERROR: Failed to initialize channel mutex\n");
        channel_buffer_destroy(ch->buffer);
        free(ch);
        return NULL;
    }
    if (pthread_cond_init(&ch->send_cond, NULL) != 0) {
        fprintf(stderr, "ERROR: Failed to initialize send condition variable\n");
        pthread_mutex_destroy(&ch->lock);
        channel_buffer_destroy(ch->buffer);
        free(ch);
        return NULL;
    }
    if (pthread_cond_init(&ch->recv_cond, NULL) != 0) {
        fprintf(stderr, "ERROR: Failed to initialize receive condition variable\n");
        pthread_cond_destroy(&ch->send_cond);
        pthread_mutex_destroy(&ch->lock);
        channel_buffer_destroy(ch->buffer);
        free(ch);
        return NULL;
    }
    
    printf("DEBUG: Created buffered channel %llu with capacity %u\n", ch->id, capacity);
    return ch;
}

// ==================== 查询方法 ====================
uint64_t channel_get_id(const Channel* channel) {
    if (!channel) {
        return 0;
    }
    return channel->id;
}

ChannelState channel_get_state(const Channel* channel) {
    if (!channel) {
        return CHANNEL_STATE_CLOSED;
    }
    return channel->state;
}

bool channel_is_closed(const Channel* channel) {
    if (!channel) {
        return true;
    }
    return channel->state == CHANNEL_STATE_CLOSED;
}

uint32_t channel_get_capacity(const Channel* channel) {
    if (!channel) {
        return 0;
    }
    return channel->capacity;
}

uint32_t channel_get_size(const Channel* channel) {
    if (!channel) {
        return 0;
    }
    return channel->size;
}

// ==================== 聚合根方法（业务操作）====================
// TODO: 实现channel_send, channel_receive, channel_close
// 这些方法需要：
// 1. 使用TaskID列表而不是Coroutine*队列
// 2. 通过TaskRepository查找Task
// 3. 通过Task聚合根获取Coroutine
// 4. 发布领域事件

// ==================== 不变条件验证 ====================
int channel_validate_invariants(const Channel* channel) {
    if (!channel) {
        return -1;  // 无效通道
    }
    
    // 不变条件1：容量必须 >= 当前大小
    if (channel->size > channel->capacity) {
        return 1;  // 违反不变条件
    }
    
    // 不变条件2：如果通道已关闭，不能有新的发送/接收操作
    // （这个在方法中检查，这里只检查基本结构）
    
    // 不变条件3：缓冲区大小必须 <= 容量
    if (channel->buffer && channel->buffer->size > channel->buffer->capacity) {
        return 2;  // 违反不变条件
    }
    
    return 0;  // 所有不变条件满足
}

// ==================== 领域事件管理 ====================

/**
 * @brief 发布领域事件（通过事件总线或存储到列表）
 *
 * 如果event_bus存在，通过EventBus发布事件。
 * 同时存储到domain_events列表（用于get_domain_events方法）。
 */
static int channel_add_domain_event(Channel* channel, DomainEvent* domain_event) {
    if (!channel || !domain_event) {
        return -1;
    }

    // 如果事件总线存在，通过EventBus发布
    if (channel->event_bus && channel->event_bus->publish) {
        int result = channel->event_bus->publish(channel->event_bus, domain_event);
        if (result != 0) {
            fprintf(stderr, "WARNING: Failed to publish event through EventBus for channel %llu\n", channel->id);
            // 继续执行，至少存储到列表
        }
    }

    // 同时存储到domain_events列表（用于get_domain_events方法）
    // 扩展事件数组容量
    if (channel->domain_events_count >= channel->domain_events_capacity) {
        size_t new_capacity = channel->domain_events_capacity == 0 ? 4 : channel->domain_events_capacity * 2;
        void** new_events = realloc(
            channel->domain_events,
            new_capacity * sizeof(void*)
        );
        if (!new_events) {
            // 如果存储失败，至少事件已经通过EventBus发布了
            return -1;
        }
        channel->domain_events = new_events;
        channel->domain_events_capacity = new_capacity;
    }

    // 添加事件到列表
    channel->domain_events[channel->domain_events_count++] = domain_event;
    return 0;
}

void** channel_get_domain_events(Channel* channel, size_t* count) {
    if (!channel || !count) {
        return NULL;
    }
    
    *count = channel->domain_events_count;
    void** events = channel->domain_events;
    
    // 清空事件列表
    channel->domain_events = NULL;
    channel->domain_events_count = 0;
    channel->domain_events_capacity = 0;
    
    return events;
}

// ==================== 销毁方法 ====================
void channel_aggregate_destroy(Channel* channel) {
    if (!channel) {
        return;
    }
    
    // 释放TaskID队列
    task_id_node_list_free(channel->sender_queue);
    task_id_node_list_free(channel->receiver_queue);
    
    // 释放缓冲区
    channel_buffer_destroy(channel->buffer);
    
    // 释放领域事件
    if (channel->domain_events) {
        free(channel->domain_events);
    }
    
    // 销毁同步机制
    pthread_cond_destroy(&channel->recv_cond);
    pthread_cond_destroy(&channel->send_cond);
    pthread_mutex_destroy(&channel->lock);
    
    // 释放通道
    free(channel);
}

// ==================== 聚合根方法（业务操作）====================
// channel_send 发送数据到通道
// 业务规则：
// 1. 如果通道已关闭，返回错误
// 2. 如果无缓冲通道且有接收者，直接传递数据并唤醒接收者
// 3. 如果无缓冲通道且无接收者，阻塞当前任务（添加到sender_queue）
// 4. 如果有缓冲通道且缓冲区未满，直接写入缓冲区
// 5. 如果有缓冲通道且缓冲区已满，阻塞当前任务（添加到sender_queue）
int channel_aggregate_send(Channel* channel, TaskRepository* task_repo, TaskID sender_task_id, void* value) {
    if (!channel || !value || !task_repo || sender_task_id == 0) {
        return -1;  // 无效参数
    }
    
    pthread_mutex_lock(&channel->lock);
    
    // 业务规则1：检查通道是否已关闭
    if (channel->state == CHANNEL_STATE_CLOSED) {
        printf("DEBUG: Cannot send to closed channel %llu\n", channel->id);
        pthread_mutex_unlock(&channel->lock);
        return -1;
    }
    
    // 通过TaskRepository查找Task
    Task* sender_task = task_repository_find_by_id(task_repo, sender_task_id);
    if (!sender_task) {
        printf("DEBUG: Task %llu not found for channel send\n", sender_task_id);
        pthread_mutex_unlock(&channel->lock);
        return -1;
    }
    
    // 对于无缓冲通道（capacity == 0）
    if (channel->capacity == 0) {
        // 业务规则2：检查是否有接收者在等待
        if (channel->receiver_queue != NULL) {
            // 有接收者，直接传递数据
            TaskID receiver_task_id = task_id_queue_remove(&channel->receiver_queue);
            
            // 通过TaskRepository查找接收者Task
            Task* receiver_task = task_repository_find_by_id(task_repo, receiver_task_id);
            if (receiver_task) {
                // 通过Task聚合根获取Coroutine
                const struct Coroutine* receiver_coroutine = task_get_coroutine(receiver_task);
                if (receiver_coroutine) {
                    // TODO: 阶段4后续重构：应该通过Task聚合根方法操作Coroutine状态
                    // 当前暂时直接操作Coroutine结构（需要进一步封装）
                    struct Coroutine* coroutine = (struct Coroutine*)receiver_coroutine;
                    coroutine->state = COROUTINE_READY;  // 设置协程状态为就绪
                    
                    printf("DEBUG: Sending value %p directly to receiver task %llu (coroutine %llu) on channel %llu\n", 
                           value, receiver_task_id, coroutine->id, channel->id);
                }
            }
            
            channel->temp_value = value;  // 临时存储值
            channel->size = 1;  // 标记有数据
            
            // 发布领域事件：数据已发送
            ChannelDataSentEvent* event = channel_data_sent_event_create(
                channel->id,
                sender_task_id,
                value
            );
            if (event) {
                DomainEvent* domain_event = channel_data_sent_event_to_domain_event(event);
                if (domain_event) {
                    channel_add_domain_event(channel, domain_event);
                } else {
                    // 转换失败，销毁原始事件
                    channel_data_sent_event_destroy(event);
                }
            }
            
            // 唤醒接收者
            // TODO: 阶段4后续重构：应该通过Task聚合根方法唤醒任务
            // 当前暂时使用条件变量
            pthread_cond_signal(&channel->recv_cond);
            pthread_mutex_unlock(&channel->lock);
            return 0;  // 成功
        } else {
            // 业务规则3：无接收者，阻塞当前任务
            task_id_queue_add(&channel->sender_queue, sender_task_id);
            channel->temp_value = value;  // 临时存储值
            
            printf("DEBUG: No receiver, blocking sender task %llu on channel %llu\n", 
                   sender_task_id, channel->id);
            
            // TODO: 阶段4后续重构：应该通过Task聚合根方法设置任务状态为等待
            // 当前暂时使用条件变量阻塞
            // 阻塞等待接收者
            while (channel->receiver_queue == NULL && channel->state == CHANNEL_STATE_OPEN) {
                pthread_cond_wait(&channel->send_cond, &channel->lock);
            }
            
            // 检查通道是否在等待期间被关闭
            if (channel->state == CHANNEL_STATE_CLOSED) {
                // 从等待队列移除
                // TODO: 从sender_queue中移除sender_task_id（需要遍历队列）
                pthread_mutex_unlock(&channel->lock);
                return -1;  // 通道已关闭
            }
            
            // 有接收者了，传递数据
            TaskID receiver_task_id = task_id_queue_remove(&channel->receiver_queue);
            
            // 通过TaskRepository查找接收者Task并唤醒
            Task* receiver_task = task_repository_find_by_id(task_repo, receiver_task_id);
            if (receiver_task) {
                const struct Coroutine* receiver_coroutine = task_get_coroutine(receiver_task);
                if (receiver_coroutine) {
                    struct Coroutine* coroutine = (struct Coroutine*)receiver_coroutine;
                    coroutine->state = COROUTINE_READY;
                }
            }
            
            printf("DEBUG: Waking sender task %llu, receiver task %llu ready on channel %llu\n",
                   sender_task_id, receiver_task_id, channel->id);
            
            // 发布领域事件：数据已发送
            ChannelDataSentEvent* event = channel_data_sent_event_create(
                channel->id,
                sender_task_id,
                value
            );
            if (event) {
                DomainEvent* domain_event = channel_data_sent_event_to_domain_event(event);
                if (domain_event) {
                    channel_add_domain_event(channel, domain_event);
                } else {
                    // 转换失败，销毁原始事件
                    channel_data_sent_event_destroy(event);
                }
            }
            
            // 唤醒接收者
            pthread_cond_signal(&channel->recv_cond);
            pthread_mutex_unlock(&channel->lock);
            return 0;  // 成功
        }
    } else {
        // 有缓冲通道
        ChannelBuffer* buffer = channel->buffer;
        
        // 业务规则4：如果缓冲区未满，直接写入
        if (buffer->size < buffer->capacity) {
            buffer->data[buffer->write_pos] = value;
            buffer->write_pos = (buffer->write_pos + 1) % buffer->capacity;
            buffer->size++;
            channel->size = buffer->size;
            
            printf("DEBUG: Sent value %p to buffered channel %llu, size: %u/%u\n",
                   value, channel->id, buffer->size, buffer->capacity);
            
            // 发布领域事件：数据已发送
            ChannelDataSentEvent* event = channel_data_sent_event_create(
                channel->id,
                sender_task_id,
                value
            );
            if (event) {
                DomainEvent* domain_event = channel_data_sent_event_to_domain_event(event);
                if (domain_event) {
                    channel_add_domain_event(channel, domain_event);
                } else {
                    // 转换失败，销毁原始事件
                    channel_data_sent_event_destroy(event);
                }
            }
            
            // 唤醒等待的接收者
            if (channel->receiver_queue != NULL) {
                pthread_cond_signal(&channel->recv_cond);
            }
            
            pthread_mutex_unlock(&channel->lock);
            return 0;  // 成功
        } else {
            // 业务规则5：缓冲区已满，阻塞当前任务
            task_id_queue_add(&channel->sender_queue, sender_task_id);
            
            printf("DEBUG: Buffer full, blocking sender task %llu on channel %llu\n",
                   sender_task_id, channel->id);
            
            // TODO: 阶段4后续重构：应该通过Task聚合根方法设置任务状态为等待
            // 阻塞等待缓冲区有空间
            while (buffer->size >= buffer->capacity && channel->state == CHANNEL_STATE_OPEN) {
                pthread_cond_wait(&channel->send_cond, &channel->lock);
            }
            
            // 检查通道是否在等待期间被关闭
            if (channel->state == CHANNEL_STATE_CLOSED) {
                // TODO: 从sender_queue中移除sender_task_id（需要遍历队列）
                pthread_mutex_unlock(&channel->lock);
                return -1;  // 通道已关闭
            }
            
            // 缓冲区有空间了，写入数据
            buffer->data[buffer->write_pos] = value;
            buffer->write_pos = (buffer->write_pos + 1) % buffer->capacity;
            buffer->size++;
            channel->size = buffer->size;
            
            printf("DEBUG: Waking sender task %llu, sent value %p to buffered channel %llu\n",
                   sender_task_id, value, channel->id);
            
            // 发布领域事件：数据已发送
            ChannelDataSentEvent* event = channel_data_sent_event_create(
                channel->id,
                sender_task_id,
                value
            );
            if (event) {
                DomainEvent* domain_event = channel_data_sent_event_to_domain_event(event);
                if (domain_event) {
                    channel_add_domain_event(channel, domain_event);
                } else {
                    // 转换失败，销毁原始事件
                    channel_data_sent_event_destroy(event);
                }
            }
            
            // 唤醒等待的接收者
            if (channel->receiver_queue != NULL) {
                pthread_cond_signal(&channel->recv_cond);
            }
            
            pthread_mutex_unlock(&channel->lock);
            return 0;  // 成功
        }
    }
}

// channel_receive 从通道接收数据
// 业务规则：
// 1. 如果通道已关闭且无数据，返回NULL
// 2. 如果无缓冲通道且有发送者，直接接收数据并唤醒发送者
// 3. 如果无缓冲通道且无发送者，阻塞当前任务（添加到receiver_queue）
// 4. 如果有缓冲通道且缓冲区有数据，直接读取数据
// 5. 如果有缓冲通道且缓冲区为空，阻塞当前任务（添加到receiver_queue）
void* channel_aggregate_receive(Channel* channel, TaskRepository* task_repo, TaskID receiver_task_id) {
    if (!channel || !task_repo || receiver_task_id == 0) {
        return NULL;
    }
    
    pthread_mutex_lock(&channel->lock);
    
    // 通过TaskRepository查找Task
    Task* receiver_task = task_repository_find_by_id(task_repo, receiver_task_id);
    if (!receiver_task) {
        printf("DEBUG: Task %llu not found for channel receive\n", receiver_task_id);
        pthread_mutex_unlock(&channel->lock);
        return NULL;
    }
    
    // 对于无缓冲通道（capacity == 0）
    if (channel->capacity == 0) {
        // 业务规则1：检查通道是否已关闭且无数据
        if (channel->state == CHANNEL_STATE_CLOSED && channel->size == 0) {
            printf("DEBUG: Channel %llu is closed and empty\n", channel->id);
            pthread_mutex_unlock(&channel->lock);
            return NULL;
        }
        
        // 业务规则2：检查是否有发送者在等待
        if (channel->sender_queue != NULL) {
            // 有发送者，直接接收数据
            TaskID sender_task_id = task_id_queue_remove(&channel->sender_queue);
            void* value = channel->temp_value;
            channel->temp_value = NULL;
            channel->size = 0;
            
            // 通过TaskRepository查找发送者Task并唤醒
            Task* sender_task = task_repository_find_by_id(task_repo, sender_task_id);
            if (sender_task) {
                const struct Coroutine* sender_coroutine = task_get_coroutine(sender_task);
                if (sender_coroutine) {
                    // TODO: 阶段4后续重构：应该通过Task聚合根方法操作Coroutine状态
                    struct Coroutine* coroutine = (struct Coroutine*)sender_coroutine;
                    coroutine->state = COROUTINE_READY;
                }
            }
            
            printf("DEBUG: Receiving value %p directly from sender task %llu on channel %llu\n",
                   value, sender_task_id, channel->id);
            
            // 发布领域事件：数据已接收
            ChannelDataReceivedEvent* event = channel_data_received_event_create(
                channel->id,
                receiver_task_id,
                value
            );
            if (event) {
                DomainEvent* domain_event = channel_data_received_event_to_domain_event(event);
                if (domain_event) {
                    channel_add_domain_event(channel, domain_event);
                } else {
                    // 转换失败，销毁原始事件
                    channel_data_received_event_destroy(event);
                }
            }
            
            // 唤醒发送者
            pthread_cond_signal(&channel->send_cond);
            pthread_mutex_unlock(&channel->lock);
            return value;
        } else {
            // 业务规则3：无发送者，阻塞当前任务
            task_id_queue_add(&channel->receiver_queue, receiver_task_id);
            
            printf("DEBUG: No sender, blocking receiver task %llu on channel %llu\n",
                   receiver_task_id, channel->id);
            
            // TODO: 阶段4后续重构：应该通过Task聚合根方法设置任务状态为等待
            // 阻塞等待发送者
            while (channel->sender_queue == NULL && channel->state == CHANNEL_STATE_OPEN) {
                pthread_cond_wait(&channel->recv_cond, &channel->lock);
            }
            
            // 检查通道是否在等待期间被关闭
            if (channel->state == CHANNEL_STATE_CLOSED && channel->size == 0) {
                // TODO: 从receiver_queue中移除receiver_task_id（需要遍历队列）
                pthread_mutex_unlock(&channel->lock);
                return NULL;  // 通道已关闭且无数据
            }
            
            // 有发送者了，接收数据
            TaskID sender_task_id = task_id_queue_remove(&channel->sender_queue);
            void* value = channel->temp_value;
            channel->temp_value = NULL;
            channel->size = 0;
            
            // 通过TaskRepository查找发送者Task并唤醒
            Task* sender_task = task_repository_find_by_id(task_repo, sender_task_id);
            if (sender_task) {
                const struct Coroutine* sender_coroutine = task_get_coroutine(sender_task);
                if (sender_coroutine) {
                    struct Coroutine* coroutine = (struct Coroutine*)sender_coroutine;
                    coroutine->state = COROUTINE_READY;
                }
            }
            
            printf("DEBUG: Waking receiver task %llu, sender task %llu ready on channel %llu\n",
                   receiver_task_id, sender_task_id, channel->id);
            
            // 发布领域事件：数据已接收
            ChannelDataReceivedEvent* event = channel_data_received_event_create(
                channel->id,
                receiver_task_id,
                value
            );
            if (event) {
                DomainEvent* domain_event = channel_data_received_event_to_domain_event(event);
                if (domain_event) {
                    channel_add_domain_event(channel, domain_event);
                } else {
                    // 转换失败，销毁原始事件
                    channel_data_received_event_destroy(event);
                }
            }
            
            // 唤醒发送者
            pthread_cond_signal(&channel->send_cond);
            pthread_mutex_unlock(&channel->lock);
            return value;
        }
    } else {
        // 有缓冲通道
        ChannelBuffer* buffer = channel->buffer;
        
        // 业务规则1：检查通道是否已关闭且无数据
        if (channel->state == CHANNEL_STATE_CLOSED && buffer->size == 0) {
            printf("DEBUG: Buffered channel %llu is closed and empty\n", channel->id);
            pthread_mutex_unlock(&channel->lock);
            return NULL;
        }
        
        // 业务规则4：如果缓冲区有数据，直接读取
        if (buffer->size > 0) {
            void* value = buffer->data[buffer->read_pos];
            buffer->data[buffer->read_pos] = NULL;  // 清除引用
            buffer->read_pos = (buffer->read_pos + 1) % buffer->capacity;
            buffer->size--;
            channel->size = buffer->size;
            
            printf("DEBUG: Received value %p from buffered channel %llu, size: %u/%u\n",
                   value, channel->id, buffer->size, buffer->capacity);
            
            // 发布领域事件：数据已接收
            ChannelDataReceivedEvent* event = channel_data_received_event_create(
                channel->id,
                receiver_task_id,
                value
            );
            if (event) {
                DomainEvent* domain_event = channel_data_received_event_to_domain_event(event);
                if (domain_event) {
                    channel_add_domain_event(channel, domain_event);
                } else {
                    // 转换失败，销毁原始事件
                    channel_data_received_event_destroy(event);
                }
            }
            
            // 唤醒等待的发送者
            if (channel->sender_queue != NULL) {
                pthread_cond_signal(&channel->send_cond);
            }
            
            pthread_mutex_unlock(&channel->lock);
            return value;
        } else {
            // 业务规则5：缓冲区为空，阻塞当前任务
            task_id_queue_add(&channel->receiver_queue, receiver_task_id);
            
            printf("DEBUG: Buffer empty, blocking receiver task %llu on channel %llu\n",
                   receiver_task_id, channel->id);
            
            // TODO: 阶段4后续重构：应该通过Task聚合根方法设置任务状态为等待
            // 阻塞等待缓冲区有数据
            while (buffer->size == 0 && channel->state == CHANNEL_STATE_OPEN) {
                pthread_cond_wait(&channel->recv_cond, &channel->lock);
            }
            
            // 检查通道是否在等待期间被关闭
            if (channel->state == CHANNEL_STATE_CLOSED && buffer->size == 0) {
                // TODO: 从receiver_queue中移除receiver_task_id（需要遍历队列）
                pthread_mutex_unlock(&channel->lock);
                return NULL;  // 通道已关闭且无数据
            }
            
            // 缓冲区有数据了，读取数据
            void* value = buffer->data[buffer->read_pos];
            buffer->data[buffer->read_pos] = NULL;  // 清除引用
            buffer->read_pos = (buffer->read_pos + 1) % buffer->capacity;
            buffer->size--;
            channel->size = buffer->size;
            
            printf("DEBUG: Waking receiver task %llu, received value %p from buffered channel %llu\n",
                   receiver_task_id, value, channel->id);
            
            // 发布领域事件：数据已接收
            ChannelDataReceivedEvent* event = channel_data_received_event_create(
                channel->id,
                receiver_task_id,
                value
            );
            if (event) {
                DomainEvent* domain_event = channel_data_received_event_to_domain_event(event);
                if (domain_event) {
                    channel_add_domain_event(channel, domain_event);
                } else {
                    // 转换失败，销毁原始事件
                    channel_data_received_event_destroy(event);
                }
            }
            
            // 唤醒等待的发送者
            if (channel->sender_queue != NULL) {
                pthread_cond_signal(&channel->send_cond);
            }
            
            pthread_mutex_unlock(&channel->lock);
            return value;
        }
    }
}

// channel_close 关闭通道
// 业务规则：
// 1. 设置通道状态为CLOSED
// 2. 唤醒所有等待的发送者和接收者（通过TaskRepository查找Task，然后通过Task聚合根方法唤醒）
// 3. 发布ChannelClosed领域事件
void channel_aggregate_close(Channel* channel, TaskRepository* task_repo) {
    if (!channel || !task_repo) {
        return;
    }
    
    pthread_mutex_lock(&channel->lock);
    
    if (channel->state == CHANNEL_STATE_CLOSED) {
        // 已经关闭，直接返回
        pthread_mutex_unlock(&channel->lock);
        return;
    }
    
    // 业务规则1：设置通道状态为CLOSED
    channel->state = CHANNEL_STATE_CLOSED;
    
    // 统计等待的任务数量
    size_t waiting_senders_count = 0;
    size_t waiting_receivers_count = 0;
    TaskIDNode* node = channel->sender_queue;
    while (node) {
        waiting_senders_count++;
        node = node->next;
    }
    node = channel->receiver_queue;
    while (node) {
        waiting_receivers_count++;
        node = node->next;
    }
    
    printf("DEBUG: Closing channel %llu (waiting: %zu senders, %zu receivers)\n", 
           channel->id, waiting_senders_count, waiting_receivers_count);
    
    // 业务规则2：唤醒所有等待的发送者和接收者
    // 唤醒所有等待的发送者
    while (channel->sender_queue != NULL) {
        TaskID task_id = task_id_queue_remove(&channel->sender_queue);
        
        // 通过TaskRepository查找Task并唤醒
        Task* task = task_repository_find_by_id(task_repo, task_id);
        if (task) {
            const struct Coroutine* coroutine = task_get_coroutine(task);
            if (coroutine) {
                // TODO: 阶段4后续重构：应该通过Task聚合根方法操作Coroutine状态
                struct Coroutine* coro = (struct Coroutine*)coroutine;
                coro->state = COROUTINE_READY;
            }
            // TODO: 阶段4后续重构：应该通过Task聚合根方法唤醒任务
            // 当前暂时使用条件变量
        }
        
        printf("DEBUG: Waking sender task %llu due to channel %llu close\n", task_id, channel->id);
    }
    pthread_cond_broadcast(&channel->send_cond);
    
    // 唤醒所有等待的接收者
    while (channel->receiver_queue != NULL) {
        TaskID task_id = task_id_queue_remove(&channel->receiver_queue);
        
        // 通过TaskRepository查找Task并唤醒
        Task* task = task_repository_find_by_id(task_repo, task_id);
        if (task) {
            const struct Coroutine* coroutine = task_get_coroutine(task);
            if (coroutine) {
                // TODO: 阶段4后续重构：应该通过Task聚合根方法操作Coroutine状态
                struct Coroutine* coro = (struct Coroutine*)coroutine;
                coro->state = COROUTINE_READY;
            }
            // TODO: 阶段4后续重构：应该通过Task聚合根方法唤醒任务
            // 当前暂时使用条件变量
        }
        
        printf("DEBUG: Waking receiver task %llu due to channel %llu close\n", task_id, channel->id);
    }
    pthread_cond_broadcast(&channel->recv_cond);
    
    // 业务规则3：发布ChannelClosed领域事件
    // TODO: 实现领域事件发布机制
    // 发布领域事件：通道已关闭
    ChannelClosedEvent* event = channel_closed_event_create(
        channel->id,
        waiting_senders_count,
        waiting_receivers_count
    );
    if (event) {
        DomainEvent* domain_event = channel_closed_event_to_domain_event(event);
        if (domain_event) {
            channel_add_domain_event(channel, domain_event);
        } else {
            // 转换失败，销毁原始事件
            channel_closed_event_destroy(event);
        }
    }
    
    pthread_mutex_unlock(&channel->lock);
}
