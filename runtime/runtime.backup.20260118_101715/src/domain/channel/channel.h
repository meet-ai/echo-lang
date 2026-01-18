/**
 * @file channel.h
 * @brief Channel 聚合根定义
 *
 * Channel 是协程间的通信原语，支持类型安全的消息传递。
 */

#ifndef CHANNEL_H
#define CHANNEL_H

#include "../../../include/echo/runtime.h"
#include <pthread.h>

// 前向声明
struct Task;

// ============================================================================
// Channel 聚合根定义
// ============================================================================

/**
 * @brief Channel 聚合根结构体
 * 实现协程间的类型安全通信
 */
typedef struct Channel {
    uint64_t id;                    // 通道唯一标识
    channel_type_t type;           // 通道类型（缓冲/无缓冲）
    size_t capacity;               // 缓冲区容量（0表示无缓冲）
    size_t size;                   // 当前缓冲区大小

    // 缓冲区（环形队列）
    void** buffer;                 // 消息缓冲区
    size_t buffer_head;            // 缓冲区头
    size_t buffer_tail;            // 缓冲区尾

    // 等待队列
    struct Task** send_waiters;    // 发送等待任务队列
    struct Task** recv_waiters;    // 接收等待任务队列
    size_t send_waiter_count;      // 发送等待者数量
    size_t recv_waiter_count;      // 接收等待者数量
    size_t max_waiters;            // 最大等待者数量

    bool is_closed;                // 通道是否已关闭

    // 无缓冲通道同步传递
    void* pending_message;         // 待传递的消息
    bool has_pending_message;      // 是否有待传递的消息

    // 同步机制
    pthread_mutex_t mutex;         // 通道锁
    pthread_cond_t send_cond;      // 发送条件变量
    pthread_cond_t recv_cond;      // 接收条件变量

    // 统计信息
    uint64_t messages_sent;        // 已发送消息总数
    uint64_t messages_received;    // 已接收消息总数
} channel_t;

// ============================================================================
// Channel 聚合根行为接口
// ============================================================================

/**
 * @brief 创建通道
 * @param capacity 缓冲区容量（0表示无缓冲）
 * @return 新创建的通道
 */
channel_t* channel_create(size_t capacity);

/**
 * @brief 销毁通道
 * @param channel 要销毁的通道
 */
void channel_destroy(channel_t* channel);

/**
 * @brief 发送消息到通道（阻塞）
 * @param channel 目标通道
 * @param message 要发送的消息
 * @return 成功返回0，失败返回-1
 */
int channel_send(channel_t* channel, void* message);

/**
 * @brief 从通道接收消息（阻塞）
 * @param channel 源通道
 * @return 接收到的消息，NULL表示通道已关闭
 */
void* channel_receive(channel_t* channel);

/**
 * @brief 发送消息到通道（非阻塞）
 * @param channel 目标通道
 * @param message 要发送的消息
 * @return 成功返回0，失败返回-1
 */
int channel_try_send(channel_t* channel, void* message);

/**
 * @brief 从通道接收消息（非阻塞）
 * @param channel 源通道
 * @return 接收到的消息，NULL表示无可用消息或通道已关闭
 */
void* channel_try_receive(channel_t* channel);

/**
 * @brief 关闭通道
 * @param channel 要关闭的通道
 */
void channel_close(channel_t* channel);

/**
 * @brief 检查通道是否已关闭
 * @param channel 通道
 * @return true如果已关闭
 */
bool channel_is_closed(const channel_t* channel);

/**
 * @brief 获取通道缓冲区大小
 * @param channel 通道
 * @return 缓冲区大小
 */
size_t channel_get_buffer_size(const channel_t* channel);

/**
 * @brief 获取通道当前消息数量
 * @param channel 通道
 * @return 当前消息数量
 */
size_t channel_get_message_count(const channel_t* channel);

/**
 * @brief 获取通道统计信息
 * @param channel 通道
 * @param sent 已发送消息数
 * @param received 已接收消息数
 */
void channel_get_stats(const channel_t* channel,
                      uint64_t* sent,
                      uint64_t* received);

#endif // CHANNEL_H
