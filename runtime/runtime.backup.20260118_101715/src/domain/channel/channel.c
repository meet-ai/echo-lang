/**
 * @file channel.c
 * @brief Channel 聚合根实现
 *
 * 实现协程间的类型安全通信，支持阻塞和非阻塞操作。
 */

#include "channel.h"
#include <stdlib.h>
#include <stdio.h>
#include <string.h>
#include <pthread.h>

// 全局通道ID计数器
static uint64_t next_channel_id = 1;

// 默认最大等待者数量
#define DEFAULT_MAX_WAITERS 16

// ============================================================================
// Channel 聚合根实现
// ============================================================================

/**
 * @brief 创建通道
 */
channel_t* channel_create(size_t capacity) {
    channel_t* channel = (channel_t*)malloc(sizeof(channel_t));
    if (!channel) {
        fprintf(stderr, "Failed to allocate channel\n");
        return NULL;
    }

    memset(channel, 0, sizeof(channel_t));

    channel->id = next_channel_id++;
    channel->capacity = capacity;
    channel->type = (capacity == 0) ? CHANNEL_TYPE_UNBUFFERED : CHANNEL_TYPE_BUFFERED;
    channel->size = 0;
    channel->buffer_head = 0;
    channel->buffer_tail = 0;
    channel->is_closed = false;
    channel->pending_message = NULL;
    channel->has_pending_message = false;
    channel->max_waiters = DEFAULT_MAX_WAITERS;
    channel->messages_sent = 0;
    channel->messages_received = 0;

    // 分配缓冲区
    if (capacity > 0) {
        channel->buffer = (void**)malloc(sizeof(void*) * capacity);
        if (!channel->buffer) {
            fprintf(stderr, "Failed to allocate channel buffer\n");
            free(channel);
            return NULL;
        }
    }

    // 分配等待队列
    channel->send_waiters = (struct Task**)malloc(sizeof(struct Task*) * DEFAULT_MAX_WAITERS);
    channel->recv_waiters = (struct Task**)malloc(sizeof(struct Task*) * DEFAULT_MAX_WAITERS);
    if (!channel->send_waiters || !channel->recv_waiters) {
        fprintf(stderr, "Failed to allocate waiter queues\n");
        free(channel->buffer);
        free(channel->send_waiters);
        free(channel->recv_waiters);
        free(channel);
        return NULL;
    }

    memset(channel->send_waiters, 0, sizeof(struct Task*) * DEFAULT_MAX_WAITERS);
    memset(channel->recv_waiters, 0, sizeof(struct Task*) * DEFAULT_MAX_WAITERS);

    // 初始化同步机制
    if (pthread_mutex_init(&channel->mutex, NULL) != 0) {
        fprintf(stderr, "Failed to initialize channel mutex\n");
        goto cleanup;
    }

    if (pthread_cond_init(&channel->send_cond, NULL) != 0) {
        fprintf(stderr, "Failed to initialize send condition\n");
        pthread_mutex_destroy(&channel->mutex);
        goto cleanup;
    }

    if (pthread_cond_init(&channel->recv_cond, NULL) != 0) {
        fprintf(stderr, "Failed to initialize receive condition\n");
        pthread_cond_destroy(&channel->send_cond);
        pthread_mutex_destroy(&channel->mutex);
        goto cleanup;
    }

    printf("DEBUG: Created channel %llu with capacity %zu\n", channel->id, capacity);
    return channel;

cleanup:
    free(channel->buffer);
    free(channel->send_waiters);
    free(channel->recv_waiters);
    free(channel);
    return NULL;
}

/**
 * @brief 销毁通道
 */
void channel_destroy(channel_t* channel) {
    if (!channel) return;

    // 关闭通道以唤醒所有等待者
    channel_close(channel);

    // 清理同步机制
    pthread_cond_destroy(&channel->recv_cond);
    pthread_cond_destroy(&channel->send_cond);
    pthread_mutex_destroy(&channel->mutex);

    // 清理缓冲区
    if (channel->buffer) {
        free(channel->buffer);
    }

    // 清理pending_message（注意：消息内容由发送者负责清理）
    channel->pending_message = NULL;
    channel->has_pending_message = false;

    // 清理等待队列
    free(channel->send_waiters);
    free(channel->recv_waiters);

    printf("DEBUG: Destroyed channel %llu\n", channel->id);
    free(channel);
}

/**
 * @brief 发送消息到通道（阻塞）
 */
int channel_send(channel_t* channel, void* message) {
    if (!channel) return -1;

    pthread_mutex_lock(&channel->mutex);

    // 检查通道是否已关闭
    if (channel->is_closed) {
        pthread_mutex_unlock(&channel->mutex);
        return -1;
    }

    // 对于无缓冲通道，实现真正的同步传递
    if (channel->capacity == 0) {
        // 无缓冲通道：发送者和接收者必须同时准备好
        // 1. 检查是否有等待的接收者
        if (channel->recv_waiter_count == 0) {
            // 没有接收者，发送者等待
            channel->send_waiter_count++;
            while (channel->recv_waiter_count == 0 && !channel->is_closed) {
                pthread_cond_wait(&channel->send_cond, &channel->mutex);
            }
            channel->send_waiter_count--;
        }

        if (channel->is_closed) {
            pthread_mutex_unlock(&channel->mutex);
            return -1;
        }

        // 2. 直接传递消息给等待的接收者
        // 由于是无缓冲，我们假设消息已经被接收者获取
        // 这里只是记录发送成功
        channel->pending_message = message;
        channel->has_pending_message = true;

        // 3. 唤醒一个等待的接收者
        if (channel->recv_waiter_count > 0) {
            pthread_cond_signal(&channel->recv_cond);
            // 注意：这里我们不直接唤醒协程，而是通过条件变量
            // GMP调度器应该监听这些条件变量的变化
        }

        printf("DEBUG: Channel %llu synchronous send completed\n", channel->id);
    } else {
        // 对于有缓冲通道，等待缓冲区有空间
        while (channel->size >= channel->capacity && !channel->is_closed) {
            pthread_cond_wait(&channel->send_cond, &channel->mutex);
        }

        if (channel->is_closed) {
            pthread_mutex_unlock(&channel->mutex);
            return -1;
        }

        // 放入缓冲区
        channel->buffer[channel->buffer_tail] = message;
        channel->buffer_tail = (channel->buffer_tail + 1) % channel->capacity;
        channel->size++;

        printf("DEBUG: Channel %llu buffered message, size: %zu/%zu\n",
               channel->id, channel->size, channel->capacity);

        // 唤醒接收等待者
        if (channel->recv_waiter_count > 0) {
            pthread_cond_signal(&channel->recv_cond);
        }
    }

    channel->messages_sent++;
    pthread_mutex_unlock(&channel->mutex);
    return 0;
}

/**
 * @brief 从通道接收消息（阻塞）
 */
void* channel_receive(channel_t* channel) {
    if (!channel) return NULL;

    pthread_mutex_lock(&channel->mutex);

    void* message = NULL;

    if (channel->capacity == 0) {
        // 无缓冲通道：接收者和发送者必须同时准备好
        // 1. 检查是否有待处理的消息
        if (!channel->has_pending_message) {
            // 没有消息，接收者等待
            channel->recv_waiter_count++;
            while (!channel->has_pending_message && !channel->is_closed) {
                pthread_cond_wait(&channel->recv_cond, &channel->mutex);
            }
            channel->recv_waiter_count--;
        }

        if (channel->is_closed && !channel->has_pending_message) {
            pthread_mutex_unlock(&channel->mutex);
            return NULL; // 通道已关闭且无消息
        }

        // 2. 获取待处理的消息
        message = channel->pending_message;
        channel->pending_message = NULL;
        channel->has_pending_message = false;

        // 3. 唤醒等待的发送者（如果有的话）
        if (channel->send_waiter_count > 0) {
            pthread_cond_signal(&channel->send_cond);
        }
    } else {
        // 有缓冲通道：等待消息或通道关闭
        while (channel->size == 0 && !channel->is_closed) {
            pthread_cond_wait(&channel->recv_cond, &channel->mutex);
        }

        if (channel->is_closed && channel->size == 0) {
            pthread_mutex_unlock(&channel->mutex);
            return NULL; // 通道已关闭且无消息
        }

        // 从缓冲区取出消息
        message = channel->buffer[channel->buffer_head];
        channel->buffer_head = (channel->buffer_head + 1) % channel->capacity;
        channel->size--;
    }

    printf("DEBUG: Channel %llu received message, size: %zu/%zu\n",
           channel->id, channel->size, channel->capacity);

    channel->messages_received++;

    // 唤醒发送等待者
    if (channel->send_waiter_count > 0) {
        pthread_cond_signal(&channel->send_cond);
    }

    pthread_mutex_unlock(&channel->mutex);
    return message;
}

/**
 * @brief 发送消息到通道（非阻塞）
 */
int channel_try_send(channel_t* channel, void* message) {
    if (!channel) return -1;

    pthread_mutex_lock(&channel->mutex);

    if (channel->is_closed) {
        pthread_mutex_unlock(&channel->mutex);
        return -1;
    }

    // 检查是否有空间
    if (channel->capacity > 0) {
        if (channel->size >= channel->capacity) {
            pthread_mutex_unlock(&channel->mutex);
            return -1; // 缓冲区满
        }
    } else {
        // 无缓冲通道：只有在有接收者等待时才能发送
        if (channel->recv_waiter_count == 0) {
            pthread_mutex_unlock(&channel->mutex);
            return -1; // 无接收者等待
        }

        // 设置待传递消息并唤醒接收者
        channel->pending_message = message;
        channel->has_pending_message = true;
        channel->messages_sent++;

        // 唤醒一个接收者
        pthread_cond_signal(&channel->recv_cond);

        pthread_mutex_unlock(&channel->mutex);
        return 0;
    }

    // 发送消息
    if (channel->capacity > 0) {
        // 有缓冲通道
        channel->buffer[channel->buffer_tail] = message;
        channel->buffer_tail = (channel->buffer_tail + 1) % channel->capacity;
        channel->size++;
        printf("DEBUG: Channel %llu buffered message, size: %zu/%zu\n",
               channel->id, channel->size, channel->capacity);

        // 唤醒接收等待者
        if (channel->recv_waiter_count > 0) {
            pthread_cond_signal(&channel->recv_cond);
        }
    }

    channel->messages_sent++;
    pthread_mutex_unlock(&channel->mutex);
    return 0;
}

/**
 * @brief 从通道接收消息（非阻塞）
 */
void* channel_try_receive(channel_t* channel) {
    if (!channel) return NULL;

    pthread_mutex_lock(&channel->mutex);

    if (channel->size == 0) {
        pthread_mutex_unlock(&channel->mutex);
        return NULL; // 无消息
    }

    // 直接接收消息（已经有锁了）
    void* message = channel->buffer[channel->buffer_head];
    channel->buffer_head = (channel->buffer_head + 1) % channel->capacity;
    channel->size--;

    printf("DEBUG: Channel %llu received message, size: %zu/%zu\n",
           channel->id, channel->size, channel->capacity);

    channel->messages_received++;

    // 唤醒发送等待者
    if (channel->send_waiter_count > 0) {
        pthread_cond_signal(&channel->send_cond);
    }

    pthread_mutex_unlock(&channel->mutex);
    return message;
}

/**
 * @brief 关闭通道
 */
void channel_close(channel_t* channel) {
    if (!channel) return;

    pthread_mutex_lock(&channel->mutex);

    if (!channel->is_closed) {
        channel->is_closed = true;

        // 唤醒所有等待者
        pthread_cond_broadcast(&channel->send_cond);
        pthread_cond_broadcast(&channel->recv_cond);

        printf("DEBUG: Channel %llu closed\n", channel->id);
    }

    pthread_mutex_unlock(&channel->mutex);
}

/**
 * @brief 检查通道是否已关闭
 */
bool channel_is_closed(const channel_t* channel) {
    return channel && channel->is_closed;
}

/**
 * @brief 获取通道缓冲区大小
 */
size_t channel_get_buffer_size(const channel_t* channel) {
    return channel ? channel->capacity : 0;
}

/**
 * @brief 获取通道当前消息数量
 */
size_t channel_get_message_count(const channel_t* channel) {
    return channel ? channel->size : 0;
}

/**
 * @brief 获取通道统计信息
 */
void channel_get_stats(const channel_t* channel,
                      uint64_t* sent,
                      uint64_t* received) {
    if (!channel) return;

    if (sent) *sent = channel->messages_sent;
    if (received) *received = channel->messages_received;
}
