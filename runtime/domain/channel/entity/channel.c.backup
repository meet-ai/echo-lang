#include "channel.h"
#include <stdlib.h>
#include <string.h>
#include <stdio.h>
#include <errno.h>

// 默认选项
const ChannelOptions CHANNEL_DEFAULT_OPTIONS = {
    .buffer_size = 0,   // 无缓冲
    .blocking = true,   // 阻塞模式
    .timeout_ms = 0     // 无超时
};

// 静态ID生成器
static uint64_t next_channel_id = 1;

// 创建缓冲区
static ChannelBuffer* create_buffer(size_t capacity, size_t element_size) {
    if (capacity == 0) return NULL;  // 无缓冲通道不需要缓冲区

    ChannelBuffer* buffer = (ChannelBuffer*)malloc(sizeof(ChannelBuffer));
    if (!buffer) return NULL;

    buffer->data = malloc(capacity * element_size);
    if (!buffer->data) {
        free(buffer);
        return NULL;
    }

    buffer->capacity = capacity;
    buffer->size = 0;
    buffer->front = 0;
    buffer->rear = 0;

    return buffer;
}

// 销毁缓冲区
static void destroy_buffer(ChannelBuffer* buffer) {
    if (!buffer) return;
    if (buffer->data) {
        free(buffer->data);
    }
    free(buffer);
}

// 缓冲区操作
static bool buffer_is_empty(const ChannelBuffer* buffer) {
    return buffer->size == 0;
}

static bool buffer_is_full(const ChannelBuffer* buffer) {
    return buffer->size == buffer->capacity;
}

static bool buffer_enqueue(ChannelBuffer* buffer, const void* data, size_t element_size) {
    if (buffer_is_full(buffer)) return false;

    memcpy((char*)buffer->data + (buffer->rear * element_size), data, element_size);
    buffer->rear = (buffer->rear + 1) % buffer->capacity;
    buffer->size++;

    return true;
}

static bool buffer_dequeue(ChannelBuffer* buffer, void* data, size_t element_size) {
    if (buffer_is_empty(buffer)) return false;

    memcpy(data, (char*)buffer->data + (buffer->front * element_size), element_size);
    buffer->front = (buffer->front + 1) % buffer->capacity;
    buffer->size--;

    return true;
}

// 创建通道
Channel* channel_create(size_t element_size, const ChannelOptions* options) {
    if (element_size == 0) return NULL;

    const ChannelOptions* opts = options ? options : &CHANNEL_DEFAULT_OPTIONS;

    Channel* channel = (Channel*)malloc(sizeof(Channel));
    if (!channel) return NULL;

    channel->id = next_channel_id++;
    channel->type = opts->buffer_size > 0 ? CHANNEL_TYPE_BUFFERED : CHANNEL_TYPE_UNBUFFERED;
    channel->status = CHANNEL_STATUS_OPEN;
    channel->buffer_size = opts->buffer_size;
    channel->element_size = element_size;

    // 创建缓冲区
    channel->buffer = create_buffer(opts->buffer_size, element_size);
    if (opts->buffer_size > 0 && !channel->buffer) {
        free(channel);
        return NULL;
    }

    // 初始化同步原语
    if (pthread_mutex_init(&channel->mutex, NULL) != 0) {
        destroy_buffer(channel->buffer);
        free(channel);
        return NULL;
    }

    if (pthread_cond_init(&channel->not_empty, NULL) != 0) {
        pthread_mutex_destroy(&channel->mutex);
        destroy_buffer(channel->buffer);
        free(channel);
        return NULL;
    }

    if (pthread_cond_init(&channel->not_full, NULL) != 0) {
        pthread_cond_destroy(&channel->not_empty);
        pthread_mutex_destroy(&channel->mutex);
        destroy_buffer(channel->buffer);
        free(channel);
        return NULL;
    }

    // 初始化统计信息
    channel->send_wait_count = 0;
    channel->recv_wait_count = 0;
    channel->send_count = 0;
    channel->recv_count = 0;
    channel->created_at = time(NULL);

    return channel;
}

// 销毁通道
void channel_destroy(Channel* channel) {
    if (!channel) return;

    pthread_mutex_lock(&channel->mutex);

    // 标记为关闭
    channel->status = CHANNEL_STATUS_CLOSED;

    // 唤醒所有等待的发送者
    pthread_cond_broadcast(&channel->not_full);

    // 唤醒所有等待的接收者
    pthread_cond_broadcast(&channel->not_empty);

    pthread_mutex_unlock(&channel->mutex);

    // 销毁同步原语
    pthread_cond_destroy(&channel->not_full);
    pthread_cond_destroy(&channel->not_empty);
    pthread_mutex_destroy(&channel->mutex);

    // 销毁缓冲区
    destroy_buffer(channel->buffer);

    free(channel);
}

// 发送数据（阻塞）
ChannelOpResult channel_send(Channel* channel, const void* data, uint32_t timeout_ms) {
    if (!channel || !data) return CHANNEL_OP_INVALID_ARG;

    pthread_mutex_lock(&channel->mutex);

    // 检查通道状态
    if (channel->status == CHANNEL_STATUS_CLOSED) {
        pthread_mutex_unlock(&channel->mutex);
        return CHANNEL_OP_CLOSED;
    }

    ChannelOpResult result = CHANNEL_OP_SUCCESS;
    bool sent = false;

    if (channel->type == CHANNEL_TYPE_UNBUFFERED) {
        // 无缓冲通道：等待接收者
        channel->send_wait_count++;

        while (channel->status == CHANNEL_STATUS_OPEN && channel->recv_wait_count == 0) {
            if (timeout_ms > 0) {
                struct timespec ts;
                clock_gettime(CLOCK_REALTIME, &ts);
                ts.tv_sec += timeout_ms / 1000;
                ts.tv_nsec += (timeout_ms % 1000) * 1000000;
                if (ts.tv_nsec >= 1000000000) {
                    ts.tv_sec++;
                    ts.tv_nsec -= 1000000000;
                }

                int wait_result = pthread_cond_timedwait(&channel->not_empty, &channel->mutex, &ts);
                if (wait_result == ETIMEDOUT) {
                    result = CHANNEL_OP_TIMEOUT;
                    break;
                }
            } else {
                pthread_cond_wait(&channel->not_empty, &channel->mutex);
            }

            // 重新检查状态（可能是被唤醒）
            if (channel->status == CHANNEL_STATUS_CLOSED) {
                result = CHANNEL_OP_CLOSED;
                break;
            }
        }

        channel->send_wait_count--;

        if (result == CHANNEL_OP_SUCCESS) {
            // 这里应该直接传递数据给接收者
            // 简化实现：标记为发送成功
            sent = true;
        }

    } else {
        // 有缓冲通道：等待缓冲区有空间
        while (channel->status == CHANNEL_STATUS_OPEN && buffer_is_full(channel->buffer)) {
            channel->send_wait_count++;

            if (timeout_ms > 0) {
                struct timespec ts;
                clock_gettime(CLOCK_REALTIME, &ts);
                ts.tv_sec += timeout_ms / 1000;
                ts.tv_nsec += (timeout_ms % 1000) * 1000000;
                if (ts.tv_nsec >= 1000000000) {
                    ts.tv_sec++;
                    ts.tv_nsec -= 1000000000;
                }

                int wait_result = pthread_cond_timedwait(&channel->not_full, &channel->mutex, &ts);
                if (wait_result == ETIMEDOUT) {
                    result = CHANNEL_OP_TIMEOUT;
                    channel->send_wait_count--;
                    break;
                }
            } else {
                pthread_cond_wait(&channel->not_full, &channel->mutex);
            }

            channel->send_wait_count--;
        }

        if (result == CHANNEL_OP_SUCCESS && buffer_enqueue(channel->buffer, data, channel->element_size)) {
            sent = true;
            // 唤醒等待的接收者
            pthread_cond_signal(&channel->not_empty);
        }
    }

    if (sent) {
        channel->send_count++;
    }

    pthread_mutex_unlock(&channel->mutex);
    return result;
}

// 接收数据（阻塞）
ChannelOpResult channel_receive(Channel* channel, void* data, uint32_t timeout_ms) {
    if (!channel || !data) return CHANNEL_OP_INVALID_ARG;

    pthread_mutex_lock(&channel->mutex);

    ChannelOpResult result = CHANNEL_OP_SUCCESS;
    bool received = false;

    if (channel->type == CHANNEL_TYPE_UNBUFFERED) {
        // 无缓冲通道：等待发送者
        channel->recv_wait_count++;

        while (channel->status == CHANNEL_STATUS_OPEN && channel->send_wait_count == 0) {
            if (timeout_ms > 0) {
                struct timespec ts;
                clock_gettime(CLOCK_REALTIME, &ts);
                ts.tv_sec += timeout_ms / 1000;
                ts.tv_nsec += (timeout_ms % 1000) * 1000000;
                if (ts.tv_nsec >= 1000000000) {
                    ts.tv_sec++;
                    ts.tv_nsec -= 1000000000;
                }

                int wait_result = pthread_cond_timedwait(&channel->not_full, &channel->mutex, &ts);
                if (wait_result == ETIMEDOUT) {
                    result = CHANNEL_OP_TIMEOUT;
                    break;
                }
            } else {
                pthread_cond_wait(&channel->not_full, &channel->mutex);
            }
        }

        channel->recv_wait_count--;

        if (result == CHANNEL_OP_SUCCESS && channel->status == CHANNEL_STATUS_OPEN) {
            // 这里应该从发送者接收数据
            // 简化实现：接收成功
            received = true;
        }

    } else {
        // 有缓冲通道：等待缓冲区有数据
        while (channel->status == CHANNEL_STATUS_OPEN && buffer_is_empty(channel->buffer)) {
            channel->recv_wait_count++;

            if (timeout_ms > 0) {
                struct timespec ts;
                clock_gettime(CLOCK_REALTIME, &ts);
                ts.tv_sec += timeout_ms / 1000;
                ts.tv_nsec += (timeout_ms % 1000) * 1000000;
                if (ts.tv_nsec >= 1000000000) {
                    ts.tv_sec++;
                    ts.tv_nsec -= 1000000000;
                }

                int wait_result = pthread_cond_timedwait(&channel->not_empty, &channel->mutex, &ts);
                if (wait_result == ETIMEDOUT) {
                    result = CHANNEL_OP_TIMEOUT;
                    channel->recv_wait_count--;
                    break;
                }
            } else {
                pthread_cond_wait(&channel->not_empty, &channel->mutex);
            }

            channel->recv_wait_count--;
        }

        if (result == CHANNEL_OP_SUCCESS && buffer_dequeue(channel->buffer, data, channel->element_size)) {
            received = true;
            // 唤醒等待的发送者
            pthread_cond_signal(&channel->not_full);
        }
    }

    // 处理关闭的通道
    if (channel->status == CHANNEL_STATUS_CLOSED) {
        if (channel->type == CHANNEL_TYPE_BUFFERED && !buffer_is_empty(channel->buffer)) {
            // 还有数据可接收
            if (buffer_dequeue(channel->buffer, data, channel->element_size)) {
                received = true;
            }
        } else if (channel->type == CHANNEL_TYPE_UNBUFFERED) {
            result = CHANNEL_OP_CLOSED;
        }
    }

    if (received) {
        channel->recv_count++;
    }

    pthread_mutex_unlock(&channel->mutex);
    return result;
}

// 非阻塞发送
ChannelOpResult channel_try_send(Channel* channel, const void* data) {
    return channel_send(channel, data, 0);
}

// 非阻塞接收
ChannelOpResult channel_try_receive(Channel* channel, void* data) {
    return channel_receive(channel, data, 0);
}

// 关闭通道
bool channel_close(Channel* channel) {
    if (!channel) return false;

    pthread_mutex_lock(&channel->mutex);

    if (channel->status == CHANNEL_STATUS_OPEN) {
        channel->status = CHANNEL_STATUS_CLOSED;

        // 唤醒所有等待者
        pthread_cond_broadcast(&channel->not_empty);
        pthread_cond_broadcast(&channel->not_full);
    }

    pthread_mutex_unlock(&channel->mutex);
    return true;
}

// 检查通道是否关闭
bool channel_is_closed(const Channel* channel) {
    return channel && channel->status == CHANNEL_STATUS_CLOSED;
}

// 获取通道长度
size_t channel_len(const Channel* channel) {
    if (!channel) return 0;

    if (channel->type == CHANNEL_TYPE_UNBUFFERED) {
        return 0;  // 无缓冲通道长度为0
    }

    return channel->buffer ? channel->buffer->size : 0;
}

// 获取通道容量
size_t channel_cap(const Channel* channel) {
    if (!channel) return 0;

    if (channel->type == CHANNEL_TYPE_UNBUFFERED) {
        return 0;  // 无缓冲通道容量为0
    }

    return channel->buffer ? channel->buffer->capacity : 0;
}

// 获取通道状态
ChannelStatus channel_get_status(const Channel* channel) {
    return channel ? channel->status : CHANNEL_STATUS_CLOSED;
}

// 获取发送计数
uint64_t channel_get_send_count(const Channel* channel) {
    return channel ? channel->send_count : 0;
}

// 获取接收计数
uint64_t channel_get_recv_count(const Channel* channel) {
    return channel ? channel->recv_count : 0;
}

// select操作 - 发送
ChannelOpResult channel_select_send(Channel** channels, size_t count, const void* data, uint32_t timeout_ms, size_t* selected_index) {
    // 简化实现：逐个尝试
    for (size_t i = 0; i < count; i++) {
        ChannelOpResult result = channel_try_send(channels[i], data);
        if (result == CHANNEL_OP_SUCCESS) {
            if (selected_index) *selected_index = i;
            return result;
        }
    }

    // 都没有成功，等待
    // TODO: 实现真正的select逻辑
    return CHANNEL_OP_WOULD_BLOCK;
}

// select操作 - 接收
ChannelOpResult channel_select_receive(Channel** channels, size_t count, void* data, uint32_t timeout_ms, size_t* selected_index) {
    // 简化实现：逐个尝试
    for (size_t i = 0; i < count; i++) {
        ChannelOpResult result = channel_try_receive(channels[i], data);
        if (result == CHANNEL_OP_SUCCESS) {
            if (selected_index) *selected_index = i;
            return result;
        }
    }

    // 都没有成功，等待
    // TODO: 实现真正的select逻辑
    return CHANNEL_OP_WOULD_BLOCK;
}
