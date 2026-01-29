#include "channel_adapter.h"
// 事件总线接口
#include "../../shared/events/bus.h"
#include <stdio.h>
#include <stdlib.h>
#include <string.h>
#include <pthread.h>

// ==================== 全局TaskRepository（暂时方案）====================
// TODO: 阶段4后续重构：应该通过依赖注入传递TaskRepository，而不是使用全局变量
static TaskRepository* g_task_repository = NULL;

// ==================== 全局EventBus（暂时方案）====================
// TODO: 阶段7后续重构：应该通过依赖注入传递EventBus，而不是使用全局变量
static EventBus* g_event_bus = NULL;
static pthread_mutex_t g_event_bus_lock = PTHREAD_MUTEX_INITIALIZER;

// ==================== Channel映射表（暂时方案）====================
// 将旧的Channel*映射到新的Channel聚合根
// 使用简单的数组实现，后续可以改为哈希表
#define MAX_CHANNEL_MAPPINGS 1024
typedef struct {
    void* legacy_channel;  // 旧的Channel*指针
    Channel* new_channel;   // 新的Channel聚合根
} ChannelMapping;

static ChannelMapping g_channel_mappings[MAX_CHANNEL_MAPPINGS];
static size_t g_channel_mappings_count = 0;

/**
 * @brief 注册Channel映射（将旧的Channel*映射到新的Channel聚合根）
 * @param legacy_channel 旧的Channel*指针
 * @param new_channel 新的Channel聚合根
 * @return 成功返回0，失败返回-1
 */
int channel_adapter_register_mapping(void* legacy_channel, Channel* new_channel) {
    if (!legacy_channel || !new_channel) {
        return -1;
    }
    
    if (g_channel_mappings_count >= MAX_CHANNEL_MAPPINGS) {
        printf("DEBUG: channel_adapter_register_mapping: Mapping table full\n");
        return -1;
    }
    
    // 检查是否已存在映射
    for (size_t i = 0; i < g_channel_mappings_count; i++) {
        if (g_channel_mappings[i].legacy_channel == legacy_channel) {
            // 更新现有映射
            g_channel_mappings[i].new_channel = new_channel;
            return 0;
        }
    }
    
    // 添加新映射
    g_channel_mappings[g_channel_mappings_count].legacy_channel = legacy_channel;
    g_channel_mappings[g_channel_mappings_count].new_channel = new_channel;
    g_channel_mappings_count++;
    
    return 0;
}

/**
 * @brief 查找Channel映射（根据旧的Channel*查找新的Channel聚合根）
 * @param legacy_channel 旧的Channel*指针
 * @return 新的Channel聚合根，如果未找到返回NULL
 */
static Channel* channel_adapter_find_mapping(void* legacy_channel) {
    if (!legacy_channel) {
        return NULL;
    }
    
    for (size_t i = 0; i < g_channel_mappings_count; i++) {
        if (g_channel_mappings[i].legacy_channel == legacy_channel) {
            return g_channel_mappings[i].new_channel;
        }
    }
    
    return NULL;
}

/**
 * @brief 设置全局TaskRepository（用于适配层）
 * @param task_repo TaskRepository实例
 * 
 * 注意：这是暂时方案，后续应该通过依赖注入传递TaskRepository
 */
void channel_adapter_set_task_repository(TaskRepository* task_repo) {
    g_task_repository = task_repo;
}

/**
 * @brief 获取全局TaskRepository
 * @return TaskRepository实例，如果未设置返回NULL
 */
TaskRepository* channel_adapter_get_task_repository(void) {
    return g_task_repository;
}

/**
 * @brief 设置全局EventBus（用于适配层）
 * @param event_bus EventBus实例
 * 
 * 注意：这是暂时方案，后续应该通过依赖注入传递EventBus
 */
void channel_adapter_set_event_bus(EventBus* event_bus) {
    pthread_mutex_lock(&g_event_bus_lock);
    g_event_bus = event_bus;
    pthread_mutex_unlock(&g_event_bus_lock);
}

/**
 * @brief 获取全局EventBus
 * @return EventBus实例，如果未设置返回NULL
 */
EventBus* channel_adapter_get_event_bus(void) {
    pthread_mutex_lock(&g_event_bus_lock);
    EventBus* bus = g_event_bus;
    pthread_mutex_unlock(&g_event_bus_lock);
    return bus;
}

// ==================== 包装函数（向后兼容）====================

/**
 * @brief 从current_task获取TaskID
 * @return TaskID，如果current_task为NULL返回0
 */
static TaskID get_current_task_id(void) {
    extern Task* current_task;  // 声明外部变量
    if (!current_task) {
        return 0;
    }
    return task_get_id(current_task);
}

/**
 * @brief channel_adapter_send 发送数据到通道（适配层）
 * 
 * 这个函数从current_task获取TaskID，然后调用新的聚合根方法channel_send。
 * 用于向后兼容旧的channel_send_impl接口。
 */
int channel_adapter_send(void* legacy_channel, TaskRepository* task_repo, void* value) {
    if (!legacy_channel || !value) {
        return -1;
    }
    
    // 如果没有传递task_repo，使用全局TaskRepository
    if (!task_repo) {
        task_repo = g_task_repository;
    }
    
    if (!task_repo) {
        printf("DEBUG: channel_adapter_send: TaskRepository not available\n");
        return -1;
    }
    
    // 从current_task获取TaskID
    TaskID sender_task_id = get_current_task_id();
    if (sender_task_id == 0) {
        printf("DEBUG: channel_adapter_send: No current task\n");
        return -1;
    }
    
    // 查找新的Channel聚合根
    Channel* channel = channel_adapter_find_mapping(legacy_channel);
    if (!channel) {
        printf("DEBUG: channel_adapter_send: Channel mapping not found for %p\n", legacy_channel);
        // TODO: 阶段4后续重构：如果找不到映射，可能需要创建新的Channel聚合根
        // 或者更新channel_create_impl，让它同时创建旧的Channel和新的Channel聚合根
        return -1;
    }
    
    // 调用新的聚合根方法
    return channel_aggregate_send(channel, task_repo, sender_task_id, value);
}

/**
 * @brief channel_adapter_receive 从通道接收数据（适配层）
 * 
 * 这个函数从current_task获取TaskID，然后调用新的聚合根方法channel_receive。
 * 用于向后兼容旧的channel_receive_impl接口。
 */
void* channel_adapter_receive(void* legacy_channel, TaskRepository* task_repo) {
    if (!legacy_channel) {
        return NULL;
    }
    
    // 如果没有传递task_repo，使用全局TaskRepository
    if (!task_repo) {
        task_repo = g_task_repository;
    }
    
    if (!task_repo) {
        printf("DEBUG: channel_adapter_receive: TaskRepository not available\n");
        return NULL;
    }
    
    // 从current_task获取TaskID
    TaskID receiver_task_id = get_current_task_id();
    if (receiver_task_id == 0) {
        printf("DEBUG: channel_adapter_receive: No current task\n");
        return NULL;
    }
    
    // 查找新的Channel聚合根
    Channel* channel = channel_adapter_find_mapping(legacy_channel);
    if (!channel) {
        printf("DEBUG: channel_adapter_receive: Channel mapping not found for %p\n", legacy_channel);
        return NULL;
    }
    
    // 调用新的聚合根方法
    return channel_aggregate_receive(channel, task_repo, receiver_task_id);
}

/**
 * @brief channel_adapter_close 关闭通道（适配层）
 * 
 * 这个函数调用新的聚合根方法channel_close。
 * 用于向后兼容旧的channel_close接口。
 */
void channel_adapter_close(void* legacy_channel, TaskRepository* task_repo) {
    if (!legacy_channel) {
        return;
    }
    
    // 如果没有传递task_repo，使用全局TaskRepository
    if (!task_repo) {
        task_repo = g_task_repository;
    }
    
    if (!task_repo) {
        printf("DEBUG: channel_adapter_close: TaskRepository not available\n");
        return;
    }
    
    // 查找新的Channel聚合根
    Channel* channel = channel_adapter_find_mapping(legacy_channel);
    if (!channel) {
        printf("DEBUG: channel_adapter_close: Channel mapping not found for %p\n", legacy_channel);
        return;
    }
    
    // 调用新的聚合根方法
    channel_aggregate_close(channel, task_repo);
}
