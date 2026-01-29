/**
 * @file test_event_bus_integration.c
 * @brief EventBus与聚合根集成测试
 * 
 * 验证EventBus在实际场景中的使用：
 * 1. Task创建时发布TaskCreated事件
 * 2. EventBus正确分发事件
 * 3. 事件处理器被正确调用
 */

#include <stdio.h>
#include <stdlib.h>
#include <string.h>
#include <stdbool.h>
#include <pthread.h>
#include <unistd.h>
#include <stdint.h>
#include <time.h>

// 包含EventBus接口和实现
#include "../../domain/shared/events/bus.h"
#include "../../infrastructure/event/bus_impl.h"

// 包含Task聚合根和事件
#include "../../domain/task_execution/aggregate/task.h"
#include "../../domain/task_execution/events/task_events.h"

// ==================== 事件处理器 ====================

/**
 * @brief 事件处理器数据
 */
typedef struct TaskEventHandlerData {
    int task_created_count;
    int task_started_count;
    int task_completed_count;
    uint64_t last_task_id;
    pthread_mutex_t mutex;
} TaskEventHandlerData;

/**
 * @brief 事件处理函数
 */
static int task_event_handler_handle(EventHandler* handler, const DomainEvent* event) {
    if (!handler || !event) return -1;
    
    TaskEventHandlerData* data = (TaskEventHandlerData*)handler->data;
    if (!data) return -1;
    
    const char* event_type = event->event_type(event);
    if (!event_type) return -1;
    
    pthread_mutex_lock(&data->mutex);
    
    if (strcmp(event_type, "TaskCreated") == 0) {
        data->task_created_count++;
        // 从事件数据中获取TaskID
        if (event->data) {
            TaskCreatedEvent* te = (TaskCreatedEvent*)event->data;
            data->last_task_id = te->task_id;
        }
        printf("  [处理器] 收到TaskCreated事件: task_id=%llu\n", data->last_task_id);
    } else if (strcmp(event_type, "TaskStarted") == 0) {
        data->task_started_count++;
        if (event->data) {
            TaskStartedEvent* te = (TaskStartedEvent*)event->data;
            data->last_task_id = te->task_id;
        }
        printf("  [处理器] 收到TaskStarted事件: task_id=%llu\n", data->last_task_id);
    } else if (strcmp(event_type, "TaskCompleted") == 0) {
        data->task_completed_count++;
        if (event->data) {
            TaskCompletedEvent* te = (TaskCompletedEvent*)event->data;
            data->last_task_id = te->task_id;
        }
        printf("  [处理器] 收到TaskCompleted事件: task_id=%llu\n", data->last_task_id);
    }
    
    pthread_mutex_unlock(&data->mutex);
    return 0;
}

/**
 * @brief 创建Task事件处理器
 */
static EventHandler* task_event_handler_create(void) {
    EventHandler* handler = (EventHandler*)calloc(1, sizeof(EventHandler));
    if (!handler) return NULL;
    
    TaskEventHandlerData* data = (TaskEventHandlerData*)calloc(1, sizeof(TaskEventHandlerData));
    if (!data) {
        free(handler);
        return NULL;
    }
    
    data->task_created_count = 0;
    data->task_started_count = 0;
    data->task_completed_count = 0;
    data->last_task_id = 0;
    pthread_mutex_init(&data->mutex, NULL);
    
    handler->handle = task_event_handler_handle;
    handler->data = data;
    
    return handler;
}

/**
 * @brief 销毁Task事件处理器
 */
static void task_event_handler_destroy(EventHandler* handler) {
    if (!handler) return;
    
    TaskEventHandlerData* data = (TaskEventHandlerData*)handler->data;
    if (data) {
        pthread_mutex_destroy(&data->mutex);
        free(data);
    }
    free(handler);
}

/**
 * @brief 获取处理器统计
 */
static void task_event_handler_get_stats(EventHandler* handler, int* created, int* started, int* completed) {
    if (!handler || !handler->data) {
        if (created) *created = 0;
        if (started) *started = 0;
        if (completed) *completed = 0;
        return;
    }
    
    TaskEventHandlerData* data = (TaskEventHandlerData*)handler->data;
    pthread_mutex_lock(&data->mutex);
    if (created) *created = data->task_created_count;
    if (started) *started = data->task_started_count;
    if (completed) *completed = data->task_completed_count;
    pthread_mutex_unlock(&data->mutex);
}

// ==================== 测试辅助函数 ====================

/**
 * @brief 简单的任务入口函数（用于测试）
 */
static void test_task_entry_point(void* arg) {
    // 简单的任务函数
    volatile int sum = 0;
    for (int i = 0; i < 100; i++) {
        sum += i;
    }
}

// ==================== 测试函数 ====================

/**
 * @brief 测试1: Task创建事件发布
 */
static bool test_task_created_event(void) {
    printf("\n=== 测试1: Task创建事件发布 ===\n");
    
    // 创建EventBus
    EventBus* bus = event_bus_create(100);
    if (!bus) {
        printf("  ❌ 失败: 无法创建EventBus\n");
        return false;
    }
    
    // 创建事件处理器并订阅
    EventHandler* handler = task_event_handler_create();
    if (!handler) {
        printf("  ❌ 失败: 无法创建事件处理器\n");
        event_bus_destroy(bus);
        return false;
    }
    
    bus->subscribe(bus, "TaskCreated", handler);
    printf("  ✅ 成功: 订阅TaskCreated事件\n");
    
    // 创建Task（传入EventBus）
    // 注意：task_factory_create的签名是 (entry_point, arg, stack_size, event_bus)
    // stack_size必须至少4096字节
    Task* task = task_factory_create(test_task_entry_point, NULL, 4096, bus);
    if (!task) {
        printf("  ❌ 失败: 无法创建Task\n");
        task_event_handler_destroy(handler);
        event_bus_destroy(bus);
        return false;
    }
    printf("  ✅ 成功: Task创建成功 (id=%llu)\n", task->id);
    
    // 获取并发布领域事件
    DomainEvent** events;
    size_t event_count;
    task_get_domain_events(task, &events, &event_count);
    
    if (event_count == 0) {
        printf("  ⚠️  警告: Task没有发布任何领域事件\n");
    } else {
        printf("  [发布] 发布 %zu 个领域事件\n", event_count);
        for (size_t i = 0; i < event_count; i++) {
            const char* event_type = events[i]->event_type(events[i]);
            printf("    - 事件 %zu: type=%s\n", i + 1, event_type ? event_type : "unknown");
            bus->publish(bus, events[i]);
        }
    }
    
    // 等待事件处理
    usleep(200000);  // 200ms
    
    // 验证处理器被调用
    int created, started, completed;
    task_event_handler_get_stats(handler, &created, &started, &completed);
    
    if (created < 1) {
        printf("  ❌ 失败: TaskCreated事件应该被处理，实际处理%d次\n", created);
        task_destroy(task);
        task_event_handler_destroy(handler);
        event_bus_destroy(bus);
        return false;
    }
    printf("  ✅ 成功: TaskCreated事件被正确处理 (count=%d)\n", created);
    
    // 清理
    task_destroy(task);
    task_event_handler_destroy(handler);
    event_bus_destroy(bus);
    
    return true;
}

/**
 * @brief 测试2: TaskCreated事件自动发布验证
 * 
 * 验证Task创建时，事件是否通过EventBus自动发布（无需手动调用publish）
 */
static bool test_task_created_auto_publish(void) {
    printf("\n=== 测试2: TaskCreated事件自动发布验证 ===\n");
    
    // 创建EventBus
    EventBus* bus = event_bus_create(100);
    if (!bus) {
        printf("  ❌ 失败: 无法创建EventBus\n");
        return false;
    }
    
    // 创建事件处理器并订阅
    EventHandler* handler = task_event_handler_create();
    if (!handler) {
        printf("  ❌ 失败: 无法创建事件处理器\n");
        event_bus_destroy(bus);
        return false;
    }
    
    bus->subscribe(bus, "TaskCreated", handler);
    printf("  ✅ 成功: 订阅TaskCreated事件\n");
    
    // 创建Task（传入EventBus）
    // Task创建时会自动通过EventBus发布TaskCreated事件（在task_add_domain_event中）
    Task* task = task_factory_create(test_task_entry_point, NULL, 4096, bus);
    if (!task) {
        printf("  ❌ 失败: 无法创建Task\n");
        task_event_handler_destroy(handler);
        event_bus_destroy(bus);
        return false;
    }
    printf("  ✅ 成功: Task创建成功 (id=%llu)\n", task->id);
    printf("  ℹ️  说明: Task创建时，事件应该通过EventBus自动发布\n");
    
    // 等待事件处理（EventBus是异步的）
    usleep(200000);  // 200ms
    
    // 验证处理器被调用（事件应该已经通过EventBus自动发布）
    int created, started, completed;
    task_event_handler_get_stats(handler, &created, &started, &completed);
    
    if (created < 1) {
        printf("  ❌ 失败: TaskCreated事件应该被自动发布和处理，实际处理%d次\n", created);
        task_destroy(task);
        task_event_handler_destroy(handler);
        event_bus_destroy(bus);
        return false;
    }
    printf("  ✅ 成功: TaskCreated事件自动发布并处理 (count=%d)\n", created);
    printf("  ℹ️  说明: Task创建时，事件通过EventBus自动发布，无需手动调用publish\n");
    
    // 清理
    task_destroy(task);
    task_event_handler_destroy(handler);
    event_bus_destroy(bus);
    
    return true;
}

// ==================== 主函数 ====================

int main(void) {
    printf("========================================\n");
    printf("EventBus与聚合根集成测试\n");
    printf("========================================\n");
    
    int passed = 0;
    int total = 2;
    
    // 运行所有测试
    if (test_task_created_event()) passed++;
    if (test_task_created_auto_publish()) passed++;
    
    // 输出结果
    printf("\n========================================\n");
    printf("测试结果: %d/%d 通过\n", passed, total);
    printf("========================================\n");
    
    if (passed == total) {
        printf("✅ 所有测试通过！EventBus与聚合根集成正常。\n");
        return 0;
    } else {
        printf("❌ 部分测试失败，请检查EventBus与聚合根的集成。\n");
        return 1;
    }
}
