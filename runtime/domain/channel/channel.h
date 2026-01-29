#ifndef CHANNEL_H
#define CHANNEL_H

#include <stdint.h>
#include <pthread.h>

// 通道状态（旧版本，用于向后兼容）
// 注意：新的Channel聚合根使用CHANNEL_STATE_OPEN和CHANNEL_STATE_CLOSED
typedef enum {
    CHANNEL_OPEN,
    CHANNEL_CLOSED
} LegacyChannelState;

// 通道结构体
typedef struct Channel {
    uint64_t id;
    void** buffer;             // 环形缓冲区数组（用于有缓冲通道）
    void* temp_value;          // 临时值存储（用于无缓冲通道的直接传递）
    uint32_t capacity;         // 容量（0表示无缓冲）
    uint32_t size;             // 当前大小
    uint32_t read_pos;         // 读指针（环形缓冲区）
    uint32_t write_pos;        // 写指针（环形缓冲区）
    LegacyChannelState state;        // 通道状态（旧版本，用于向后兼容）
    pthread_mutex_t lock;
    pthread_cond_t send_cond;  // 发送条件变量
    pthread_cond_t recv_cond;  // 接收条件变量

    // 等待发送和接收的协程队列
    struct Coroutine* sender_queue;
    struct Coroutine* receiver_queue;
} Channel;

// 函数声明
void* channel_create_impl();  // 创建无缓冲通道
void* channel_create_buffered_impl(uint32_t capacity);  // 创建有缓冲通道
void channel_send_impl(Channel* channel, void* value);
void* channel_receive_impl(Channel* channel);
int32_t channel_select_impl(int32_t num_cases, Channel** channels, int32_t* operations, int32_t* result_ptr);
void channel_close_impl(Channel* channel);
void channel_destroy_impl(Channel* channel);

// 非阻塞操作函数
uint32_t channel_get_message_count(Channel* channel);
int channel_try_send(Channel* channel, void* value);
void* channel_try_receive(Channel* channel);

#endif // CHANNEL_H
