/**
 * @file test_channel_adapter_integration.c
 * @brief Channel适配层集成测试
 * 
 * 验证Channel适配层的向后兼容性：
 * 1. Channel创建和映射注册
 * 2. channel_adapter_send功能
 * 3. channel_adapter_receive功能
 * 4. channel_adapter_close功能
 * 
 * 测试场景：
 * - 创建Channel聚合根并注册映射
 * - 创建Task用于测试
 * - 通过适配层发送/接收数据
 * - 验证适配层正确调用聚合根方法
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

// 包含Task聚合根和仓储
#include "../../domain/task_execution/aggregate/task.h"
#include "../../domain/task_execution/repository/task_repository.h"
#include "../../domain/task_execution/repository/task_repository_memory.h"

// 包含Channel聚合根和适配层
#include "../../domain/channel_communication/aggregate/channel.h"
#include "../../domain/channel_communication/adapter/channel_adapter.h"

// 包含scheduler.h以访问current_task声明
#include "../../domain/scheduler/scheduler.h"

// ==================== 测试辅助函数 ====================

/**
 * @brief 测试任务入口点（空任务）
 */
static void test_task_entry_point(void* arg) {
    // 空任务，仅用于测试
    (void)arg;
}

/**
 * @brief 设置测试用的current_task
 * 
 * 注意：current_task在scheduler.h中声明为extern
 * 在测试中，我们直接设置这个全局变量
 */
static void set_test_current_task(Task* task) {
    current_task = task;  // current_task在scheduler.h中声明为extern
}

// ==================== 测试用例 ====================

/**
 * @brief 测试1: Channel创建和映射注册
 */
static bool test_channel_creation_and_mapping(void) {
    printf("\n=== 测试1: Channel创建和映射注册 ===\n");

    // 创建EventBus
    EventBus* bus = event_bus_create(100);
    if (!bus) {
        printf("  ❌ 失败: 无法创建EventBus\n");
        return false;
    }
    printf("  ✅ 成功: EventBus创建成功\n");

    // 创建Channel聚合根（无缓冲）
    Channel* channel = channel_factory_create_unbuffered(bus);
    if (!channel) {
        printf("  ❌ 失败: 无法创建Channel聚合根\n");
        event_bus_destroy(bus);
        return false;
    }
    printf("  ✅ 成功: Channel聚合根创建成功 (id=%llu)\n", channel_get_id(channel));

    // 注册映射（模拟旧的Channel*指针）
    void* legacy_channel = (void*)0x12345678;  // 模拟旧的Channel*指针
    if (channel_adapter_register_mapping(legacy_channel, channel) != 0) {
        printf("  ❌ 失败: 无法注册Channel映射\n");
        channel_destroy(channel);
        event_bus_destroy(bus);
        return false;
    }
    printf("  ✅ 成功: Channel映射注册成功\n");

    // 验证映射查找
    Channel* found = channel_adapter_from_legacy(legacy_channel);
    if (!found || found != channel) {
        printf("  ❌ 失败: 映射查找失败\n");
        channel_destroy(channel);
        event_bus_destroy(bus);
        return false;
    }
    printf("  ✅ 成功: 映射查找成功\n");

    // 清理
    channel_destroy(channel);
    event_bus_destroy(bus);

    return true;
}

/**
 * @brief 测试2: channel_adapter_send功能验证
 */
static bool test_channel_adapter_send(void) {
    printf("\n=== 测试2: channel_adapter_send功能验证 ===\n");

    // 创建EventBus
    EventBus* bus = event_bus_create(100);
    if (!bus) {
        printf("  ❌ 失败: 无法创建EventBus\n");
        return false;
    }

    // 创建TaskRepository
    TaskRepository* repo = task_repository_create();
    if (!repo) {
        printf("  ❌ 失败: 无法创建TaskRepository\n");
        event_bus_destroy(bus);
        return false;
    }
    printf("  ✅ 成功: TaskRepository创建成功\n");

    // 设置全局TaskRepository
    channel_adapter_set_task_repository(repo);
    channel_adapter_set_event_bus(bus);

    // 创建Task用于测试
    Task* sender_task = task_factory_create(test_task_entry_point, NULL, 4096, bus);
    if (!sender_task) {
        printf("  ❌ 失败: 无法创建Task\n");
        task_repository_destroy(repo);
        event_bus_destroy(bus);
        return false;
    }
    printf("  ✅ 成功: Task创建成功 (id=%llu)\n", task_get_id(sender_task));

    // 保存Task到Repository
    if (task_repository_save(repo, sender_task) != 0) {
        printf("  ❌ 失败: 无法保存Task到Repository\n");
        task_destroy(sender_task);
        task_repository_destroy(repo);
        event_bus_destroy(bus);
        return false;
    }

    // 设置测试用的current_task
    set_test_current_task(sender_task);

    // 创建Channel聚合根（无缓冲）
    Channel* channel = channel_factory_create_unbuffered(bus);
    if (!channel) {
        printf("  ❌ 失败: 无法创建Channel聚合根\n");
        set_test_current_task(NULL);
        task_destroy(sender_task);
        task_repository_destroy(repo);
        event_bus_destroy(bus);
        return false;
    }
    printf("  ✅ 成功: Channel聚合根创建成功 (id=%llu)\n", channel_get_id(channel));

    // 注册映射
    void* legacy_channel = (void*)0x12345678;
    if (channel_adapter_register_mapping(legacy_channel, channel) != 0) {
        printf("  ❌ 失败: 无法注册Channel映射\n");
        channel_destroy(channel);
        set_test_current_task(NULL);
        task_destroy(sender_task);
        task_repository_destroy(repo);
        event_bus_destroy(bus);
        return false;
    }

    // 测试发送数据（注意：无缓冲通道需要接收者，这里只测试基本功能）
    // 由于无缓冲通道在没有接收者时会阻塞，我们创建一个接收者Task
    Task* receiver_task = task_factory_create(test_task_entry_point, NULL, 4096, bus);
    if (!receiver_task) {
        printf("  ❌ 失败: 无法创建接收者Task\n");
        channel_destroy(channel);
        set_test_current_task(NULL);
        task_destroy(sender_task);
        task_repository_destroy(repo);
        event_bus_destroy(bus);
        return false;
    }
    task_repository_save(repo, receiver_task);

    // 先让接收者等待（通过直接调用聚合根方法）
    // 注意：这里我们直接调用聚合根方法，因为适配层需要current_task
    TaskID receiver_id = task_get_id(receiver_task);
    void* test_value = (void*)"test_data";
    
    // 由于无缓冲通道需要接收者先等待，我们直接测试适配层的基本功能
    // 这里我们测试有缓冲通道，更容易测试
    channel_destroy(channel);
    
    // 创建有缓冲通道
    Channel* buffered_channel = channel_factory_create_buffered(10, bus);
    if (!buffered_channel) {
        printf("  ❌ 失败: 无法创建有缓冲Channel\n");
        set_test_current_task(NULL);
        task_destroy(sender_task);
        task_destroy(receiver_task);
        task_repository_destroy(repo);
        event_bus_destroy(bus);
        return false;
    }
    printf("  ✅ 成功: 有缓冲Channel创建成功 (id=%llu)\n", channel_get_id(buffered_channel));

    // 注册新映射
    void* legacy_buffered = (void*)0x87654321;
    channel_adapter_register_mapping(legacy_buffered, buffered_channel);

    // 测试发送（有缓冲通道可以直接发送）
    int result = channel_adapter_send(legacy_buffered, repo, test_value);
    if (result != 0) {
        printf("  ❌ 失败: channel_adapter_send返回错误 (result=%d)\n", result);
        channel_destroy(buffered_channel);
        set_test_current_task(NULL);
        task_destroy(sender_task);
        task_destroy(receiver_task);
        task_repository_destroy(repo);
        event_bus_destroy(bus);
        return false;
    }
    printf("  ✅ 成功: channel_adapter_send成功\n");

    // 清理
    channel_destroy(buffered_channel);
    set_test_current_task(NULL);
    task_destroy(sender_task);
    task_destroy(receiver_task);
    task_repository_destroy(repo);
    event_bus_destroy(bus);

    return true;
}

/**
 * @brief 测试3: channel_adapter_receive功能验证
 */
static bool test_channel_adapter_receive(void) {
    printf("\n=== 测试3: channel_adapter_receive功能验证 ===\n");

    // 创建EventBus和TaskRepository
    EventBus* bus = event_bus_create(100);
    TaskRepository* repo = task_repository_create();
    if (!bus || !repo) {
        printf("  ❌ 失败: 无法创建EventBus或TaskRepository\n");
        if (bus) event_bus_destroy(bus);
        if (repo) task_repository_destroy(repo);
        return false;
    }

    channel_adapter_set_task_repository(repo);
    channel_adapter_set_event_bus(bus);

    // 创建接收者Task
    Task* receiver_task = task_factory_create(test_task_entry_point, NULL, 4096, bus);
    if (!receiver_task) {
        printf("  ❌ 失败: 无法创建接收者Task\n");
        task_repository_destroy(repo);
        event_bus_destroy(bus);
        return false;
    }
    task_repository_save(repo, receiver_task);
    set_test_current_task(receiver_task);
    printf("  ✅ 成功: 接收者Task创建成功 (id=%llu)\n", task_get_id(receiver_task));

    // 创建有缓冲Channel并发送数据
    Channel* channel = channel_factory_create_buffered(10, bus);
    if (!channel) {
        printf("  ❌ 失败: 无法创建Channel\n");
        set_test_current_task(NULL);
        task_destroy(receiver_task);
        task_repository_destroy(repo);
        event_bus_destroy(bus);
        return false;
    }

    void* legacy_channel = (void*)0xABCDEF00;
    channel_adapter_register_mapping(legacy_channel, channel);

    // 先发送数据（使用另一个Task作为发送者）
    Task* sender_task = task_factory_create(test_task_entry_point, NULL, 4096, bus);
    task_repository_save(repo, sender_task);
    set_test_current_task(sender_task);

    void* test_value = (void*)"test_receive_data";
    if (channel_adapter_send(legacy_channel, repo, test_value) != 0) {
        printf("  ❌ 失败: 无法发送数据\n");
        channel_destroy(channel);
        set_test_current_task(NULL);
        task_destroy(sender_task);
        task_destroy(receiver_task);
        task_repository_destroy(repo);
        event_bus_destroy(bus);
        return false;
    }
    printf("  ✅ 成功: 数据发送成功\n");

    // 切换回接收者Task并接收数据
    set_test_current_task(receiver_task);
    void* received = channel_adapter_receive(legacy_channel, repo);
    if (received != test_value) {
        printf("  ❌ 失败: 接收到的数据不匹配 (expected=%p, got=%p)\n", test_value, received);
        channel_destroy(channel);
        set_test_current_task(NULL);
        task_destroy(sender_task);
        task_destroy(receiver_task);
        task_repository_destroy(repo);
        event_bus_destroy(bus);
        return false;
    }
    printf("  ✅ 成功: 数据接收成功 (value=%s)\n", (char*)received);

    // 清理
    channel_destroy(channel);
    set_test_current_task(NULL);
    task_destroy(sender_task);
    task_destroy(receiver_task);
    task_repository_destroy(repo);
    event_bus_destroy(bus);

    return true;
}

/**
 * @brief 测试4: channel_adapter_close功能验证
 */
static bool test_channel_adapter_close(void) {
    printf("\n=== 测试4: channel_adapter_close功能验证 ===\n");

    // 创建EventBus和TaskRepository
    EventBus* bus = event_bus_create(100);
    TaskRepository* repo = task_repository_create();
    if (!bus || !repo) {
        printf("  ❌ 失败: 无法创建EventBus或TaskRepository\n");
        if (bus) event_bus_destroy(bus);
        if (repo) task_repository_destroy(repo);
        return false;
    }

    channel_adapter_set_task_repository(repo);
    channel_adapter_set_event_bus(bus);

    // 创建Channel
    Channel* channel = channel_factory_create_buffered(10, bus);
    if (!channel) {
        printf("  ❌ 失败: 无法创建Channel\n");
        task_repository_destroy(repo);
        event_bus_destroy(bus);
        return false;
    }
    printf("  ✅ 成功: Channel创建成功 (id=%llu)\n", channel_get_id(channel));

    // 验证Channel状态为OPEN
    if (channel_is_closed(channel)) {
        printf("  ❌ 失败: Channel初始状态应该是OPEN\n");
        channel_destroy(channel);
        task_repository_destroy(repo);
        event_bus_destroy(bus);
        return false;
    }
    printf("  ✅ 成功: Channel初始状态为OPEN\n");

    // 注册映射
    void* legacy_channel = (void*)0xCLOSETEST;
    channel_adapter_register_mapping(legacy_channel, channel);

    // 通过适配层关闭Channel
    channel_adapter_close(legacy_channel, repo);

    // 验证Channel状态为CLOSED
    if (!channel_is_closed(channel)) {
        printf("  ❌ 失败: Channel关闭后状态应该是CLOSED\n");
        channel_destroy(channel);
        task_repository_destroy(repo);
        event_bus_destroy(bus);
        return false;
    }
    printf("  ✅ 成功: Channel关闭后状态为CLOSED\n");

    // 清理
    channel_destroy(channel);
    task_repository_destroy(repo);
    event_bus_destroy(bus);

    return true;
}

// ==================== 主函数 ====================

int main() {
    printf("========================================\n");
    printf("Channel适配层集成测试\n");
    printf("========================================\n");

    int passed = 0;
    int total = 4;

    // 运行所有测试
    if (test_channel_creation_and_mapping()) passed++;
    if (test_channel_adapter_send()) passed++;
    if (test_channel_adapter_receive()) passed++;
    if (test_channel_adapter_close()) passed++;

    printf("\n========================================\n");
    printf("测试结果: %d/%d 通过\n", passed, total);
    printf("========================================\n");

    if (passed == total) {
        printf("✅ 所有测试通过！Channel适配层功能正常。\n");
        return 0;
    } else {
        printf("❌ 部分测试失败，请检查适配层实现。\n");
        return 1;
    }
}
