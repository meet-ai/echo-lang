#ifndef CHANNEL_MANAGER_H
#define CHANNEL_MANAGER_H

#include "../entity/channel.h"
#include <stdint.h>
#include <stdbool.h>

// 前向声明
struct ChannelManager;
struct ChannelRegistry;

// 通道管理器领域服务 - 管理所有通道的生命周期
typedef struct ChannelManager {
    struct ChannelRegistry* registry;    // 通道注册表
    uint32_t max_channels;               // 最大通道数
    uint32_t active_channels;            // 活动通道数
} ChannelManager;

// 通道注册表（内部使用）
typedef struct ChannelRegistry {
    Channel** channels;                  // 通道数组
    size_t capacity;                     // 容量
    size_t size;                         // 当前大小
    pthread_mutex_t mutex;               // 同步锁
} ChannelRegistry;

// 通道管理器方法
ChannelManager* channel_manager_create(uint32_t max_channels);
void channel_manager_destroy(ChannelManager* manager);

// 通道生命周期管理
Channel* channel_manager_create_channel(ChannelManager* manager, size_t element_size, const ChannelOptions* options);
bool channel_manager_destroy_channel(ChannelManager* manager, uint64_t channel_id);

// 通道查询
Channel* channel_manager_get_channel(const ChannelManager* manager, uint64_t channel_id);
Channel** channel_manager_get_all_channels(const ChannelManager* manager, size_t* count);

// 状态监控
uint32_t channel_manager_get_active_count(const ChannelManager* manager);
uint32_t channel_manager_get_total_created(const ChannelManager* manager);

// 批量操作
bool channel_manager_close_all_channels(ChannelManager* manager);
bool channel_manager_broadcast_message(ChannelManager* manager, const void* message, size_t message_size);

#endif // CHANNEL_MANAGER_H
