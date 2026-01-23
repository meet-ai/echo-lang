#ifndef CHANNEL_DTOS_H
#define CHANNEL_DTOS_H

#include <stdint.h>
#include <stdbool.h>
#include <time.h>

// 前向声明
struct Channel;

// 通道DTO - 数据传输对象
typedef struct {
    uint64_t id;                    // 通道ID
    char name[256];                 // 通道名称
    char type[32];                  // 通道类型 (e.g., "buffered", "unbuffered")
    uint32_t capacity;              // 通道容量
    uint32_t size;                  // 当前大小
    bool is_closed;                 // 是否已关闭
    time_t created_at;              // 创建时间
} ChannelDTO;

// 通道创建结果DTO
typedef struct {
    bool success;
    uint64_t channel_id;
    char message[512];
    int error_code;
} ChannelCreationResultDTO;

// 发送结果DTO
typedef struct {
    bool success;
    char message[512];
    int error_code;
    uint32_t bytes_sent;
} SendResultDTO;

// 接收结果DTO
typedef struct {
    bool success;
    void* data;
    size_t data_size;
    char message[512];
    int error_code;
    uint32_t bytes_received;
} ReceiveResultDTO;

// 通道列表DTO
typedef struct {
    ChannelDTO* channels;           // 通道数组
    size_t count;                   // 通道数量
    size_t total_count;             // 总通道数（用于分页）
    uint32_t page;                  // 当前页码
    uint32_t page_size;             // 每页大小
} ChannelListDTO;

// 通道统计DTO
typedef struct {
    uint32_t total_channels;        // 总通道数
    uint32_t active_channels;       // 活跃通道数
    uint32_t closed_channels;       // 已关闭通道数
    uint64_t total_messages_sent;   // 总发送消息数
    uint64_t total_messages_received; // 总接收消息数
    uint64_t total_bytes_sent;      // 总发送字节数
    uint64_t total_bytes_received;  // 总接收字节数
    time_t last_updated;            // 最后更新时间
} ChannelStatisticsDTO;

// DTO创建和销毁函数
ChannelDTO* channel_dto_create(uint64_t id, const char* name, const char* type, uint32_t capacity, uint32_t size, bool is_closed, time_t created_at);
void channel_dto_destroy(ChannelDTO* dto);

ChannelCreationResultDTO* channel_creation_result_dto_create(bool success, uint64_t channel_id, const char* message, int error_code);
void channel_creation_result_dto_destroy(ChannelCreationResultDTO* dto);

SendResultDTO* send_result_dto_create(bool success, const char* message, int error_code, uint32_t bytes_sent);
void send_result_dto_destroy(SendResultDTO* dto);

ReceiveResultDTO* receive_result_dto_create(bool success, void* data, size_t data_size, const char* message, int error_code, uint32_t bytes_received);
void receive_result_dto_destroy(ReceiveResultDTO* dto);

ChannelListDTO* channel_list_dto_create(ChannelDTO* channels, size_t count, size_t total_count, uint32_t page, uint32_t page_size);
void channel_list_dto_destroy(ChannelListDTO* dto);

ChannelStatisticsDTO* channel_statistics_dto_create(uint32_t total, uint32_t active, uint32_t closed, uint64_t sent_msgs, uint64_t recv_msgs, uint64_t sent_bytes, uint64_t recv_bytes, time_t last_updated);
void channel_statistics_dto_destroy(ChannelStatisticsDTO* dto);

// 转换函数
ChannelDTO* channel_to_dto(const struct Channel* channel);
ChannelListDTO* channels_to_list_dto(struct Channel** channels, size_t count, size_t total, uint32_t page, uint32_t page_size);
ChannelStatisticsDTO* calculate_channel_statistics(struct Channel** channels, size_t count);

#endif // CHANNEL_DTOS_H
