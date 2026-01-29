/**
 * @file event_bus_initializer.h
 * @brief EventBus初始化器（基础设施层）
 *
 * 负责创建全局EventBus实例并注入到各个适配层。
 * 这是阶段7的临时方案，后续应该通过依赖注入传递EventBus。
 */

#ifndef INFRASTRUCTURE_EVENT_EVENT_BUS_INITIALIZER_H
#define INFRASTRUCTURE_EVENT_EVENT_BUS_INITIALIZER_H

#include "../../domain/shared/events/bus.h"

/**
 * @brief 初始化全局EventBus并注入到适配层
 * 
 * 创建EventBus实例，并设置到：
 * - scheduler_adapter
 * - channel_adapter
 * - async_application_service（通过initialize函数）
 * 
 * @param buffer_size EventBus缓冲大小（默认100）
 * @return 成功返回EventBus实例，失败返回NULL
 */
EventBus* event_bus_initializer_setup(uint32_t buffer_size);

/**
 * @brief 清理全局EventBus
 * 
 * 销毁EventBus实例并清理适配层中的引用
 * 
 * @param event_bus EventBus实例
 */
void event_bus_initializer_cleanup(EventBus* event_bus);

/**
 * @brief 获取全局EventBus实例
 * 
 * @return EventBus实例，如果未初始化返回NULL
 */
EventBus* event_bus_initializer_get(void);

#endif // INFRASTRUCTURE_EVENT_EVENT_BUS_INITIALIZER_H
