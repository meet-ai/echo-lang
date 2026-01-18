#ifndef CHANNEL_H
#define CHANNEL_H

#include <stdint.h>
#include <pthread.h>

// 通道状态
typedef enum {
    CHANNEL_OPEN,
    CHANNEL_CLOSED
} ChannelState;

// 通道结构体
typedef struct Channel {
    uint64_t id;
    void* buffer;              // 缓冲区（暂时未实现）
    uint32_t capacity;         // 容量（0表示无缓冲）
    uint32_t size;             // 当前大小
    ChannelState state;        // 通道状态
    pthread_mutex_t lock;
    pthread_cond_t send_cond;  // 发送条件变量
    pthread_cond_t recv_cond;  // 接收条件变量

    // 等待发送和接收的协程队列
    struct Coroutine* sender_queue;
    struct Coroutine* receiver_queue;
} Channel;

// 函数声明
void* channel_create_impl();
void channel_send_impl(Channel* channel, void* value);
void* channel_receive_impl(Channel* channel);
int32_t channel_select_impl(int32_t num_cases, Channel** channels, int32_t* operations, int32_t* result_ptr);
void channel_close_impl(Channel* channel);
void channel_destroy_impl(Channel* channel);

#endif // CHANNEL_H
