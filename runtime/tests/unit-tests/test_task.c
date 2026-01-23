#include "../unit-tests/test_framework.h"
#include "../../domain/task/entity/task.h"
#include "../../domain/task/service/task_scheduler.h"
#include <string.h>

// 测试套件
TestSuite* task_test_suite(void);

// 测试用例
void test_task_creation(void);
void test_task_state_transition(void);
void test_task_scheduler_basic(void);
void test_task_scheduler_priority(void);
void test_task_event_publishing(void);

// 任务创建测试
void test_task_creation(void) {
    // 创建任务
    Task* task = task_create("test_task", task_function_example, NULL, 1024);

    TEST_ASSERT_NOT_NULL(task);
    TEST_ASSERT_STR_EQUAL("test_task", task->name);
    TEST_ASSERT_EQUAL(TASK_STATE_NEW, task->state);
    TEST_ASSERT(task->stack_size >= 1024);

    // 验证任务ID唯一性
    Task* task2 = task_create("test_task_2", task_function_example, NULL, 1024);
    TEST_ASSERT(task->id != task2->id);

    // 清理
    task_destroy(task);
    task_destroy(task2);
}

// 任务状态转换测试
void test_task_state_transition(void) {
    Task* task = task_create("state_test", task_function_example, NULL, 1024);

    // NEW -> READY
    TEST_ASSERT(task_set_state(task, TASK_STATE_READY));
    TEST_ASSERT_EQUAL(TASK_STATE_READY, task->state);

    // READY -> RUNNING
    TEST_ASSERT(task_set_state(task, TASK_STATE_RUNNING));
    TEST_ASSERT_EQUAL(TASK_STATE_RUNNING, task->state);

    // RUNNING -> COMPLETED
    TEST_ASSERT(task_set_state(task, TASK_STATE_COMPLETED));
    TEST_ASSERT_EQUAL(TASK_STATE_COMPLETED, task->state);

    // 验证非法状态转换
    Task* task2 = task_create("invalid_test", task_function_example, NULL, 1024);
    task_set_state(task2, TASK_STATE_COMPLETED);
    TEST_ASSERT(!task_set_state(task2, TASK_STATE_RUNNING)); // COMPLETED不能回到RUNNING

    task_destroy(task);
    task_destroy(task2);
}

// 任务调度器基本测试
void test_task_scheduler_basic(void) {
    TaskScheduler* scheduler = task_scheduler_create();

    // 创建任务
    Task* task1 = task_create("task1", task_function_example, NULL, 1024);
    Task* task2 = task_create("task2", task_function_example, NULL, 1024);

    // 提交任务
    TEST_ASSERT(task_scheduler_submit(scheduler, task1));
    TEST_ASSERT(task_scheduler_submit(scheduler, task2));

    // 验证队列大小
    TEST_ASSERT_EQUAL(2, task_scheduler_get_queue_size(scheduler));

    // 调度任务
    Task* scheduled = task_scheduler_schedule_next(scheduler);
    TEST_ASSERT_NOT_NULL(scheduled);
    TEST_ASSERT_EQUAL(1, task_scheduler_get_queue_size(scheduler));

    // 清理
    task_destroy(task1);
    task_destroy(task2);
    task_scheduler_destroy(scheduler);
}

// 任务调度器优先级测试
void test_task_scheduler_priority(void) {
    TaskScheduler* scheduler = task_scheduler_create();

    // 创建不同优先级的任务
    Task* low_priority = task_create("low", task_function_example, NULL, 1024);
    Task* high_priority = task_create("high", task_function_example, NULL, 1024);

    task_set_priority(low_priority, 1);
    task_set_priority(high_priority, 10);

    // 提交任务（低优先级先提交）
    TEST_ASSERT(task_scheduler_submit(scheduler, low_priority));
    TEST_ASSERT(task_scheduler_submit(scheduler, high_priority));

    // 高优先级任务应该先被调度
    Task* scheduled = task_scheduler_schedule_next(scheduler);
    TEST_ASSERT_NOT_NULL(scheduled);
    TEST_ASSERT_STR_EQUAL("high", scheduled->name);

    // 清理
    task_destroy(low_priority);
    task_destroy(high_priority);
    task_scheduler_destroy(scheduler);
}

// 任务事件发布测试
void test_task_event_publishing(void) {
    // 创建事件收集器
    EventCollector* collector = event_collector_create();

    // 订阅任务事件
    event_bus_subscribe(event_bus_get_global(), TASK_EVENT_ALL, collector);

    // 创建并启动任务
    Task* task = task_create("event_test", task_function_example, NULL, 1024);
    TEST_ASSERT(task_start(task));

    // 等待任务完成
    task_wait(task, 5000);

    // 验证事件发布
    TEST_ASSERT(event_collector_has_event(collector, TASK_EVENT_STARTED));
    TEST_ASSERT(event_collector_has_event(collector, TASK_EVENT_COMPLETED));

    // 清理
    task_destroy(task);
    event_collector_destroy(collector);
}

// 示例任务函数
void task_function_example(void* arg) {
    // 简单的任务执行
    volatile int sum = 0;
    for (int i = 0; i < 1000; i++) {
        sum += i;
    }
    // 模拟一些工作
    for (volatile int i = 0; i < 10000; i++) {
        // busy wait
    }
}

// 任务测试套件
TestSuite* task_test_suite(void) {
    TestSuite* suite = TEST_SUITE("Task Management", "任务管理相关测试");

    TEST_ADD(suite, "task_creation", "测试任务创建功能", test_task_creation);
    TEST_ADD(suite, "task_state_transition", "测试任务状态转换", test_task_state_transition);
    TEST_ADD(suite, "task_scheduler_basic", "测试任务调度器基本功能", test_task_scheduler_basic);
    TEST_ADD(suite, "task_scheduler_priority", "测试任务调度器优先级调度", test_task_scheduler_priority);
    TEST_ADD(suite, "task_event_publishing", "测试任务事件发布", test_task_event_publishing);

    return suite;
}

// 事件收集器（用于测试事件发布）
typedef struct EventCollector {
    TaskEvent* events;
    size_t count;
    size_t capacity;
} EventCollector;

EventCollector* event_collector_create(void) {
    EventCollector* collector = TEST_MALLOC(sizeof(EventCollector));
    collector->capacity = 16;
    collector->events = TEST_MALLOC(sizeof(TaskEvent) * collector->capacity);
    collector->count = 0;
    return collector;
}

void event_collector_destroy(EventCollector* collector) {
    TEST_FREE(collector->events);
    TEST_FREE(collector);
}

void event_collector_on_event(void* user_data, TaskEvent* event) {
    EventCollector* collector = (EventCollector*)user_data;

    if (collector->count >= collector->capacity) {
        collector->capacity *= 2;
        collector->events = TEST_REALLOC(collector->events,
                                       sizeof(TaskEvent) * collector->capacity);
    }

    collector->events[collector->count++] = *event;
}

bool event_collector_has_event(EventCollector* collector, TaskEventType type) {
    for (size_t i = 0; i < collector->count; i++) {
        if (collector->events[i].type == type) {
            return true;
        }
    }
    return false;
}
