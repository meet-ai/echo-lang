#ifndef CHANNEL_H
#define CHANNEL_H

#include <stdint.h>
#include <stdbool.h>
#include <pthread.h>

// 前向声明
struct Channel;
struct ChannelBuffer;

// 通道类型枚举
typedef enum {
    CHANNEL_TYPE_UNBUFFERED,    // 无缓冲通道
    CHANNEL_TYPE_BUFFERED       // 有缓冲通道
} ChannelType;

// 通道状态枚举
typedef enum {
    CHANNEL_STATUS_OPEN,        // 通道开放
    CHANNEL_STATUS_CLOSED       // 通道关闭
} ChannelStatus;

// 通道实体 - 实现CSP并发模型
typedef struct Channel {
    uint64_t id;                    // 通道唯一标识
    ChannelType type;               // 通道类型
    ChannelStatus status;           // 通道状态
    size_t buffer_size;             // 缓冲区大小（无缓冲为0）
    size_t element_size;            // 元素大小

    // 内部实现
    struct ChannelBuffer* buffer;   // 缓冲区
    pthread_mutex_t mutex;          // 同步锁
    pthread_cond_t not_empty;       // 非空条件变量
    pthread_cond_t not_full;        // 非满条件变量
    size_t send_wait_count;         // 发送等待者数量
    size_t recv_wait_count;         // 接收等待者数量

    // 统计信息
    uint64_t send_count;            // 发送操作次数
    uint64_t recv_count;            // 接收操作次数
    time_t created_at;              // 创建时间
} Channel;

// 通道缓冲区（内部使用）
typedef struct ChannelBuffer {
    void* data;                     // 缓冲区数据
    size_t capacity;                // 容量
    size_t size;                    // 当前大小
    size_t front;                   // 队头索引
    size_t rear;                    // 队尾索引
} ChannelBuffer;

// 通道操作结果
typedef enum {
    CHANNEL_OP_SUCCESS,             // 操作成功
    CHANNEL_OP_CLOSED,              // 通道已关闭
    CHANNEL_OP_TIMEOUT,             // 操作超时
    CHANNEL_OP_WOULD_BLOCK,         // 操作会阻塞（非阻塞模式）
    CHANNEL_OP_INVALID_ARG          // 无效参数
} ChannelOpResult;

// 通道选项
typedef struct {
    size_t buffer_size;             // 缓冲区大小
    bool blocking;                  // 是否阻塞模式
    uint32_t timeout_ms;            // 超时时间（毫秒）
} ChannelOptions;

// 默认通道选项
extern const ChannelOptions CHANNEL_DEFAULT_OPTIONS;

// 通道方法
Channel* channel_create(size_t element_size, const ChannelOptions* options);
void channel_destroy(Channel* channel);

// 基本操作
ChannelOpResult channel_send(Channel* channel, const void* data, uint32_t timeout_ms);
ChannelOpResult channel_receive(Channel* channel, void* data, uint32_t timeout_ms);

// 非阻塞操作
ChannelOpResult channel_try_send(Channel* channel, const void* data);
ChannelOpResult channel_try_receive(Channel* channel, void* data);

// 高级操作
bool channel_close(Channel* channel);
bool channel_is_closed(const Channel* channel);
size_t channel_len(const Channel* channel);
size_t channel_cap(const Channel* channel);

// 状态查询
ChannelStatus channel_get_status(const Channel* channel);
uint64_t channel_get_send_count(const Channel* channel);
uint64_t channel_get_recv_count(const Channel* channel);

// 工具函数
ChannelOpResult channel_select_send(Channel** channels, size_t count, const void* data, uint32_t timeout_ms, size_t* selected_index);
ChannelOpResult channel_select_receive(Channel** channels, size_t count, void* data, uint32_t timeout_ms, size_t* selected_index);

#endif // CHANNEL_H
