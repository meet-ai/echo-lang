/**
 * @file test_event_bus.c
 * @brief EventBus机制验证程序
 * 
 * 验证EventBus的基本功能：
 * 1. EventBus的创建和销毁
 * 2. 事件的发布和订阅
 * 3. 事件处理器的注册和调用
 * 4. 多个事件类型的处理
 */

#include <stdio.h>
#include <stdlib.h>
#include <string.h>
#include <stdbool.h>
#include <stddef.h>  // for offsetof
#include <pthread.h>
#include <unistd.h>

// 包含EventBus接口和实现
#include "../../domain/shared/events/bus.h"
#include "../../infrastructure/event/bus_impl.h"

// ==================== 测试事件定义 ====================

/**
 * @brief 测试事件结构
 */
typedef struct TestEvent {
    DomainEvent base;  // 实现DomainEvent接口
    uint64_t event_id;
    const char* event_type_str;
    time_t occurred_at;
    int test_data;  // 测试数据
} TestEvent;

// 实现DomainEvent接口
// 注意：TestEvent的base字段在结构体开头，可以直接转换
static const char* test_event_get_type(const DomainEvent* event) {
    if (!event || !event->data) return NULL;
    TestEvent* te = (TestEvent*)event->data;
    return te->event_type_str;
}

static time_t test_event_get_occurred_at(const DomainEvent* event) {
    if (!event || !event->data) return 0;
    TestEvent* te = (TestEvent*)event->data;
    return te->occurred_at;
}

static uint64_t test_event_get_id(const DomainEvent* event) {
    if (!event || !event->data) return 0;
    TestEvent* te = (TestEvent*)event->data;
    return te->event_id;
}

static void test_event_destroy(DomainEvent* event) {
    if (!event || !event->data) return;
    TestEvent* te = (TestEvent*)event->data;
    free(te);
}

/**
 * @brief 创建测试事件
 */
static TestEvent* test_event_create(const char* event_type, int test_data) {
    TestEvent* event = (TestEvent*)calloc(1, sizeof(TestEvent));
    if (!event) return NULL;
    
    // 初始化DomainEvent接口
    // 注意：base字段在结构体开头，可以直接转换
    event->base.event_type = test_event_get_type;
    event->base.occurred_at = test_event_get_occurred_at;
    event->base.event_id = test_event_get_id;
    event->base.destroy = test_event_destroy;
    event->base.data = event;  // 存储TestEvent指针，便于访问
    
    // 设置事件数据
    event->event_id = (uint64_t)time(NULL) * 1000000 + (uint64_t)rand();
    event->event_type_str = event_type;
    event->occurred_at = time(NULL);
    event->test_data = test_data;
    
    return event;
}

// ==================== 测试事件处理器 ====================

/**
 * @brief 测试事件处理器数据
 */
typedef struct TestEventHandlerData {
    int received_count;  // 接收到的事件数量
    int last_test_data;  // 最后一个事件的test_data
    char last_event_type[64];  // 最后一个事件类型
    pthread_mutex_t mutex;  // 保护数据
} TestEventHandlerData;

/**
 * @brief 事件处理函数
 */
static int test_event_handler_handle(EventHandler* handler, const DomainEvent* event) {
    if (!handler || !event) return -1;
    
    TestEventHandlerData* data = (TestEventHandlerData*)handler->data;
    if (!data) return -1;
    
    pthread_mutex_lock(&data->mutex);
    
    data->received_count++;
    const char* event_type = event->event_type(event);
    if (event_type) {
        strncpy(data->last_event_type, event_type, sizeof(data->last_event_type) - 1);
        data->last_event_type[sizeof(data->last_event_type) - 1] = '\0';
    }
    
    // 获取test_data（如果事件是TestEvent）
    if (event->data) {
        TestEvent* te = (TestEvent*)event->data;
        data->last_test_data = te->test_data;
    }
    
    pthread_mutex_unlock(&data->mutex);
    
    printf("  [处理器] 收到事件: type=%s, test_data=%d\n", 
           event_type ? event_type : "unknown", 
           data->last_test_data);
    
    return 0;
}

/**
 * @brief 创建测试事件处理器
 */
static EventHandler* test_event_handler_create(void) {
    EventHandler* handler = (EventHandler*)calloc(1, sizeof(EventHandler));
    if (!handler) return NULL;
    
    TestEventHandlerData* data = (TestEventHandlerData*)calloc(1, sizeof(TestEventHandlerData));
    if (!data) {
        free(handler);
        return NULL;
    }
    
    data->received_count = 0;
    data->last_test_data = -1;
    data->last_event_type[0] = '\0';
    pthread_mutex_init(&data->mutex, NULL);
    
    handler->handle = test_event_handler_handle;
    handler->data = data;
    
    return handler;
}

/**
 * @brief 销毁测试事件处理器
 */
static void test_event_handler_destroy(EventHandler* handler) {
    if (!handler) return;
    
    TestEventHandlerData* data = (TestEventHandlerData*)handler->data;
    if (data) {
        pthread_mutex_destroy(&data->mutex);
        free(data);
    }
    free(handler);
}

/**
 * @brief 获取处理器接收的事件数量
 */
static int test_event_handler_get_count(EventHandler* handler) {
    if (!handler || !handler->data) return -1;
    
    TestEventHandlerData* data = (TestEventHandlerData*)handler->data;
    pthread_mutex_lock(&data->mutex);
    int count = data->received_count;
    pthread_mutex_unlock(&data->mutex);
    return count;
}

// ==================== 测试函数 ====================

/**
 * @brief 测试1: EventBus创建和销毁
 */
static bool test_event_bus_create_destroy(void) {
    printf("\n=== 测试1: EventBus创建和销毁 ===\n");
    
    // 创建EventBus
    EventBus* bus = event_bus_create(100);
    if (!bus) {
        printf("  ❌ 失败: 无法创建EventBus\n");
        return false;
    }
    printf("  ✅ 成功: EventBus创建成功\n");
    
    // 销毁EventBus
    event_bus_destroy(bus);
    printf("  ✅ 成功: EventBus销毁成功\n");
    
    return true;
}

/**
 * @brief 测试2: 事件发布和订阅
 */
static bool test_event_publish_subscribe(void) {
    printf("\n=== 测试2: 事件发布和订阅 ===\n");
    
    // 创建EventBus
    EventBus* bus = event_bus_create(100);
    if (!bus) {
        printf("  ❌ 失败: 无法创建EventBus\n");
        return false;
    }
    
    // 创建事件处理器
    EventHandler* handler = test_event_handler_create();
    if (!handler) {
        printf("  ❌ 失败: 无法创建事件处理器\n");
        event_bus_destroy(bus);
        return false;
    }
    
    // 订阅事件
    const char* event_type = "TestEvent";
    if (bus->subscribe(bus, event_type, handler) != 0) {
        printf("  ❌ 失败: 无法订阅事件\n");
        test_event_handler_destroy(handler);
        event_bus_destroy(bus);
        return false;
    }
    printf("  ✅ 成功: 事件订阅成功 (type=%s)\n", event_type);
    
    // 创建并发布事件
    TestEvent* event = test_event_create(event_type, 42);
    if (!event) {
        printf("  ❌ 失败: 无法创建测试事件\n");
        test_event_handler_destroy(handler);
        event_bus_destroy(bus);
        return false;
    }
    
    printf("  [发布] 发布事件: type=%s, test_data=%d\n", event_type, event->test_data);
    if (bus->publish(bus, (DomainEvent*)&event->base) != 0) {
        printf("  ❌ 失败: 无法发布事件\n");
        test_event_destroy((DomainEvent*)&event->base);
        test_event_handler_destroy(handler);
        event_bus_destroy(bus);
        return false;
    }
    
    // 等待事件处理（异步处理需要一点时间）
    usleep(100000);  // 100ms
    
    // 验证处理器被调用
    int count = test_event_handler_get_count(handler);
    if (count != 1) {
        printf("  ❌ 失败: 处理器应该被调用1次，实际调用%d次\n", count);
        test_event_handler_destroy(handler);
        event_bus_destroy(bus);
        return false;
    }
    printf("  ✅ 成功: 处理器被正确调用 (count=%d)\n", count);
    
    // 清理
    test_event_handler_destroy(handler);
    event_bus_destroy(bus);
    
    return true;
}

/**
 * @brief 测试3: 多个事件类型
 */
static bool test_multiple_event_types(void) {
    printf("\n=== 测试3: 多个事件类型 ===\n");
    
    // 创建EventBus
    EventBus* bus = event_bus_create(100);
    if (!bus) {
        printf("  ❌ 失败: 无法创建EventBus\n");
        return false;
    }
    
    // 创建两个处理器
    EventHandler* handler1 = test_event_handler_create();
    EventHandler* handler2 = test_event_handler_create();
    if (!handler1 || !handler2) {
        printf("  ❌ 失败: 无法创建事件处理器\n");
        if (handler1) test_event_handler_destroy(handler1);
        if (handler2) test_event_handler_destroy(handler2);
        event_bus_destroy(bus);
        return false;
    }
    
    // 订阅不同的事件类型
    const char* event_type1 = "EventType1";
    const char* event_type2 = "EventType2";
    
    bus->subscribe(bus, event_type1, handler1);
    bus->subscribe(bus, event_type2, handler2);
    printf("  ✅ 成功: 订阅两个事件类型 (type1=%s, type2=%s)\n", event_type1, event_type2);
    
    // 发布事件1
    TestEvent* event1 = test_event_create(event_type1, 100);
    printf("  [发布] 发布事件1: type=%s, test_data=%d\n", event_type1, event1->test_data);
    bus->publish(bus, (DomainEvent*)&event1->base);
    
    // 发布事件2
    TestEvent* event2 = test_event_create(event_type2, 200);
    printf("  [发布] 发布事件2: type=%s, test_data=%d\n", event_type2, event2->test_data);
    bus->publish(bus, (DomainEvent*)&event2->base);
    
    // 等待事件处理
    usleep(200000);  // 200ms
    
    // 验证处理器调用
    int count1 = test_event_handler_get_count(handler1);
    int count2 = test_event_handler_get_count(handler2);
    
    if (count1 != 1 || count2 != 1) {
        printf("  ❌ 失败: 处理器1调用%d次，处理器2调用%d次（应该都是1次）\n", count1, count2);
        test_event_handler_destroy(handler1);
        test_event_handler_destroy(handler2);
        event_bus_destroy(bus);
        return false;
    }
    printf("  ✅ 成功: 两个处理器都被正确调用 (handler1=%d, handler2=%d)\n", count1, count2);
    
    // 清理
    test_event_handler_destroy(handler1);
    test_event_handler_destroy(handler2);
    event_bus_destroy(bus);
    
    return true;
}

/**
 * @brief 测试4: 多个处理器订阅同一事件类型
 */
static bool test_multiple_handlers_same_type(void) {
    printf("\n=== 测试4: 多个处理器订阅同一事件类型 ===\n");
    
    // 创建EventBus
    EventBus* bus = event_bus_create(100);
    if (!bus) {
        printf("  ❌ 失败: 无法创建EventBus\n");
        return false;
    }
    
    // 创建两个处理器，订阅同一事件类型
    EventHandler* handler1 = test_event_handler_create();
    EventHandler* handler2 = test_event_handler_create();
    if (!handler1 || !handler2) {
        printf("  ❌ 失败: 无法创建事件处理器\n");
        if (handler1) test_event_handler_destroy(handler1);
        if (handler2) test_event_handler_destroy(handler2);
        event_bus_destroy(bus);
        return false;
    }
    
    const char* event_type = "SharedEventType";
    bus->subscribe(bus, event_type, handler1);
    bus->subscribe(bus, event_type, handler2);
    printf("  ✅ 成功: 两个处理器订阅同一事件类型 (type=%s)\n", event_type);
    
    // 发布事件
    TestEvent* event = test_event_create(event_type, 300);
    printf("  [发布] 发布事件: type=%s, test_data=%d\n", event_type, event->test_data);
    bus->publish(bus, (DomainEvent*)&event->base);
    
    // 等待事件处理
    usleep(200000);  // 200ms
    
    // 验证两个处理器都被调用
    int count1 = test_event_handler_get_count(handler1);
    int count2 = test_event_handler_get_count(handler2);
    
    if (count1 != 1 || count2 != 1) {
        printf("  ❌ 失败: 处理器1调用%d次，处理器2调用%d次（应该都是1次）\n", count1, count2);
        test_event_handler_destroy(handler1);
        test_event_handler_destroy(handler2);
        event_bus_destroy(bus);
        return false;
    }
    printf("  ✅ 成功: 两个处理器都被正确调用 (handler1=%d, handler2=%d)\n", count1, count2);
    
    // 清理
    test_event_handler_destroy(handler1);
    test_event_handler_destroy(handler2);
    event_bus_destroy(bus);
    
    return true;
}

// ==================== 主函数 ====================

int main(void) {
    printf("========================================\n");
    printf("EventBus机制验证程序\n");
    printf("========================================\n");
    
    int passed = 0;
    int total = 4;
    
    // 运行所有测试
    if (test_event_bus_create_destroy()) passed++;
    if (test_event_publish_subscribe()) passed++;
    if (test_multiple_event_types()) passed++;
    if (test_multiple_handlers_same_type()) passed++;
    
    // 输出结果
    printf("\n========================================\n");
    printf("测试结果: %d/%d 通过\n", passed, total);
    printf("========================================\n");
    
    if (passed == total) {
        printf("✅ 所有测试通过！EventBus机制工作正常。\n");
        return 0;
    } else {
        printf("❌ 部分测试失败，请检查EventBus实现。\n");
        return 1;
    }
}
