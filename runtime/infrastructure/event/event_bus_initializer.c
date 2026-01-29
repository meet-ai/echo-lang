/**
 * @file event_bus_initializer.c
 * @brief EventBus初始化器实现（基础设施层）
 *
 * 负责创建全局EventBus实例并注入到各个适配层。
 */

#include "event_bus_initializer.h"
#include "bus_impl.h"  // 包含event_bus_create
#include "../../domain/task_scheduling/adapter/scheduler_adapter.h"  // scheduler_adapter_set_event_bus
#include "../../domain/channel_communication/adapter/channel_adapter.h"  // channel_adapter_set_event_bus
#include <stdio.h>
#include <pthread.h>

// ==================== 全局EventBus管理 ====================

static EventBus* global_event_bus = NULL;
static pthread_mutex_t global_event_bus_lock = PTHREAD_MUTEX_INITIALIZER;

/**
 * @brief 初始化全局EventBus并注入到适配层
 */
EventBus* event_bus_initializer_setup(uint32_t buffer_size) {
    pthread_mutex_lock(&global_event_bus_lock);
    
    // 如果已经初始化，直接返回
    if (global_event_bus != NULL) {
        pthread_mutex_unlock(&global_event_bus_lock);
        printf("DEBUG: EventBus already initialized, returning existing instance\n");
        return global_event_bus;
    }
    
    // 创建EventBus实例
    if (buffer_size == 0) {
        buffer_size = 100;  // 默认缓冲大小
    }
    
    EventBus* event_bus = event_bus_create(buffer_size);
    if (!event_bus) {
        fprintf(stderr, "ERROR: Failed to create EventBus\n");
        pthread_mutex_unlock(&global_event_bus_lock);
        return NULL;
    }
    
    global_event_bus = event_bus;
    
    // 注入到适配层
    scheduler_adapter_set_event_bus(event_bus);
    channel_adapter_set_event_bus(event_bus);
    
    printf("DEBUG: EventBus initialized and injected into adapters (buffer_size=%u)\n", buffer_size);
    
    pthread_mutex_unlock(&global_event_bus_lock);
    return event_bus;
}

/**
 * @brief 清理全局EventBus
 */
void event_bus_initializer_cleanup(EventBus* event_bus) {
    if (!event_bus) return;
    
    pthread_mutex_lock(&global_event_bus_lock);
    
    // 清理适配层中的引用
    scheduler_adapter_set_event_bus(NULL);
    channel_adapter_set_event_bus(NULL);
    
    // 销毁EventBus实例
    if (event_bus == global_event_bus) {
        event_bus_destroy(event_bus);
        global_event_bus = NULL;
        printf("DEBUG: EventBus destroyed and cleaned up\n");
    }
    
    pthread_mutex_unlock(&global_event_bus_lock);
}

/**
 * @brief 获取全局EventBus实例
 */
EventBus* event_bus_initializer_get(void) {
    pthread_mutex_lock(&global_event_bus_lock);
    EventBus* bus = global_event_bus;
    pthread_mutex_unlock(&global_event_bus_lock);
    return bus;
}
