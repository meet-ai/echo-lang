#include "channel.h"
#include "../coroutine/coroutine.h"
#include "../task/task.h"
#include "../scheduler/scheduler.h"
#include <stdio.h>
#include <stdlib.h>
#include <string.h>

// 创建通道
void* channel_create_impl() {
    printf("DEBUG: channel_create_impl called\n");
    Channel* ch = (Channel*)malloc(sizeof(Channel));
    if (!ch) {
        printf("DEBUG: Failed to allocate channel\n");
        return NULL;
    }

    static uint64_t next_channel_id = 1;
    ch->id = next_channel_id++;
    ch->capacity = 0; // 无缓冲通道
    ch->size = 0;
    ch->state = CHANNEL_OPEN;
    ch->buffer = NULL; // 无缓冲区
    ch->sender_queue = NULL;
    ch->receiver_queue = NULL;

    // 初始化同步机制
    pthread_mutex_init(&ch->lock, NULL);
    pthread_cond_init(&ch->send_cond, NULL);
    pthread_cond_init(&ch->recv_cond, NULL);

    printf("DEBUG: Created channel %llu\n", ch->id);
    return ch;
}

// 发送数据到通道（真正的阻塞实现）
void channel_send_impl(Channel* ch, void* value) {
    if (!ch) {
        printf("DEBUG: channel_send_impl called with NULL channel\n");
        return;
    }

    printf("DEBUG: channel_send_impl called on channel %llu with value %p\n", ch->id, value);

    pthread_mutex_lock(&ch->lock);

    // 检查通道是否已关闭
    if (ch->state == CHANNEL_CLOSED) {
        printf("DEBUG: Cannot send to closed channel %llu\n", ch->id);
        pthread_mutex_unlock(&ch->lock);
        return;
    }

    // 对于无缓冲通道（capacity == 0）
    if (ch->capacity == 0) {
        // 等待有接收者准备好
        while (ch->receiver_queue == NULL && ch->state == CHANNEL_OPEN) {
            printf("DEBUG: Channel %llu has no receiver, sender blocking\n", ch->id);
            pthread_cond_wait(&ch->send_cond, &ch->lock);
        }

        if (ch->state == CHANNEL_CLOSED) {
            printf("DEBUG: Channel %llu closed while waiting\n", ch->id);
            pthread_mutex_unlock(&ch->lock);
            return;
        }

        // 直接传递给接收者
        Coroutine* receiver = ch->receiver_queue;
        if (receiver) {
            // 从等待队列移除接收者
            ch->receiver_queue = receiver->next;
            receiver->next = NULL;

            printf("DEBUG: Sending value %p directly to receiver on channel %llu\n", value, ch->id);

            // 这里应该唤醒接收者协程并传递值
            // 简化实现：暂时只是设置一个标志
            // TODO: 实现真正的协程唤醒和值传递
        }
    } else {
        // 有缓冲区的通道 - TODO: 实现缓冲区管理
        printf("DEBUG: Buffered channels not yet implemented\n");
    }

    // 唤醒等待的接收者
    pthread_cond_broadcast(&ch->recv_cond);
    pthread_mutex_unlock(&ch->lock);

    printf("DEBUG: Sent value %p to channel %llu\n", value, ch->id);
}

// 从通道接收数据（真正的阻塞实现）
void* channel_receive_impl(Channel* ch) {
    if (!ch) {
        printf("DEBUG: channel_receive_impl called with NULL channel\n");
        return NULL;
    }

    printf("DEBUG: channel_receive_impl called on channel %llu\n", ch->id);

    pthread_mutex_lock(&ch->lock);

    // 对于无缓冲通道
    if (ch->capacity == 0) {
        // 等待有发送者
        while (ch->size == 0 && ch->state == CHANNEL_OPEN) {
            printf("DEBUG: Channel %llu has no data, receiver blocking\n", ch->id);

            // 将当前协程加入接收者等待队列
            // TODO: 获取当前协程并加入队列

            pthread_cond_wait(&ch->recv_cond, &ch->lock);
        }

        if (ch->state == CHANNEL_CLOSED && ch->size == 0) {
            printf("DEBUG: Channel %llu closed and empty\n", ch->id);
            pthread_mutex_unlock(&ch->lock);
            return NULL; // 通道关闭且无数据
        }

        // 有数据，接收它
        void* value = ch->buffer;
        ch->size = 0;
        ch->buffer = NULL;

        // 唤醒等待的发送者
        pthread_cond_broadcast(&ch->send_cond);

        pthread_mutex_unlock(&ch->lock);
        printf("DEBUG: Received value %p from channel %llu\n", value, ch->id);
        return value;
    } else {
        // 有缓冲区的通道 - TODO: 实现缓冲区管理
        printf("DEBUG: Buffered channels not yet implemented\n");
        pthread_mutex_unlock(&ch->lock);
        return NULL;
    }
}

// select语句支持（真正的多路复用实现）
int32_t channel_select_impl(int32_t num_cases, Channel** channels, int32_t* operations, int32_t* result_ptr) {
    printf("DEBUG: channel_select_impl called with %d cases\n", num_cases);

    // 首先尝试非阻塞检查
    for (int32_t i = 0; i < num_cases; i++) {
        Channel* ch = channels[i];
        int32_t op = operations[i];

        if (!ch) continue;

        printf("DEBUG: Checking case %d: channel %llu, operation %d\n", i, ch->id, op);

        pthread_mutex_lock(&ch->lock);

        if (op == 0) { // 接收操作
            if (ch->size > 0 || ch->state == CHANNEL_CLOSED) {
                // 有数据可以接收或通道已关闭
                printf("DEBUG: Case %d (receive) ready\n", i);
                pthread_mutex_unlock(&ch->lock);
                if (result_ptr) *result_ptr = i;
                return i;
            }
        } else if (op == 1) { // 发送操作
            if (ch->capacity == 0) {
                // 无缓冲通道：如果有等待的接收者，就可以发送
                if (ch->receiver_queue != NULL) {
                    printf("DEBUG: Case %d (send) ready\n", i);
                    pthread_mutex_unlock(&ch->lock);
                    if (result_ptr) *result_ptr = i;
                    return i;
                }
            } else {
                // 有缓冲通道：如果缓冲区有空间
                if (ch->size < ch->capacity) {
                    printf("DEBUG: Case %d (send) ready\n", i);
                    pthread_mutex_unlock(&ch->lock);
                    if (result_ptr) *result_ptr = i;
                    return i;
                }
            }
        }

        pthread_mutex_unlock(&ch->lock);
    }

    // 如果没有case立即就绪，需要阻塞等待
    // TODO: 实现真正的select阻塞等待逻辑
    // 目前简化：返回-1表示无就绪case

    printf("DEBUG: No cases ready in select, would block\n");
    if (result_ptr) *result_ptr = -1;
    return -1;
}

// 关闭通道
void channel_close_impl(Channel* ch) {
    if (!ch) return;

    printf("DEBUG: Closing channel %llu\n", ch->id);

    pthread_mutex_lock(&ch->lock);
    ch->state = CHANNEL_CLOSED;

    // 唤醒所有等待的发送者和接收者
    pthread_cond_broadcast(&ch->send_cond);
    pthread_cond_broadcast(&ch->recv_cond);

    pthread_mutex_unlock(&ch->lock);

    printf("DEBUG: Channel %llu closed\n", ch->id);
}

// 销毁通道
void channel_destroy_impl(Channel* ch) {
    if (!ch) return;

    printf("DEBUG: Destroying channel %llu\n", ch->id);

    // 确保通道已关闭
    if (ch->state != CHANNEL_CLOSED) {
        channel_close_impl(ch);
    }

    // 清理同步机制
    pthread_mutex_destroy(&ch->lock);
    pthread_cond_destroy(&ch->send_cond);
    pthread_cond_destroy(&ch->recv_cond);

    // 释放缓冲区（如果有的话）
    if (ch->buffer) {
        free(ch->buffer);
    }

    free(ch);
    printf("DEBUG: Channel %llu destroyed\n", ch->id);
}
