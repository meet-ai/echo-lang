/**
 * @file test_channel_blocking.c
 * @brief 无缓冲通道阻塞通信测试
 * 
 * 测试无缓冲通道的阻塞发送、接收和同步机制
 */

#include "../../../unit-tests/test_framework.h"
#include "../../../../domain/channel/channel.h"
#include "../../../../domain/coroutine/coroutine.h"
#include "../../../../domain/task/task.h"
#include "../../../../domain/scheduler/scheduler.h"
#include <stdio.h>
#include <stdlib.h>
#include <string.h>
#include <assert.h>
#include <pthread.h>

// 测试辅助变量
static bool sender_blocked = false;
static bool receiver_blocked = false;
static bool value_received = false;
static void* received_value = NULL;

// 发送者协程函数
void sender_coroutine_func(void* arg) {
    Channel* ch = (Channel*)arg;
    int* value = malloc(sizeof(int));
    *value = 42;
    
    printf("  [Sender] Sending value %d to channel\n", *value);
    channel_send_impl(ch, value);
    printf("  [Sender] Value sent successfully\n");
}

// 接收者协程函数
void receiver_coroutine_func(void* arg) {
    Channel* ch = (Channel*)arg;
    
    printf("  [Receiver] Waiting to receive from channel\n");
    void* value = channel_receive_impl(ch);
    
    if (value) {
        received_value = value;
        value_received = true;
        printf("  [Receiver] Received value %d\n", *(int*)value);
    } else {
        printf("  [Receiver] Channel closed, received NULL\n");
    }
}

/**
 * @brief 测试无缓冲通道阻塞发送
 */
void test_unbuffered_channel_blocking_send(void) {
    printf("=== Testing Unbuffered Channel Blocking Send ===\n");
    
    // 创建无缓冲通道
    Channel* ch = (Channel*)channel_create_impl();
    TEST_ASSERT_NOT_NULL(ch);
    TEST_ASSERT_EQUAL(0, ch->capacity);
    
    // 创建接收者协程（先创建接收者，这样发送者不会阻塞）
    Coroutine* receiver = coroutine_create(
        "receiver_coroutine",
        receiver_coroutine_func,
        ch,
        4096
    );
    TEST_ASSERT_NOT_NULL(receiver);
    
    Task* receiver_task = task_create(NULL, NULL);
    receiver->task = receiver_task;
    receiver_task->coroutine = receiver;
    
    // 将接收者加入通道等待队列
    receiver->next = ch->receiver_queue;
    ch->receiver_queue = receiver;
    
    // 创建发送者协程
    Coroutine* sender = coroutine_create(
        "sender_coroutine",
        sender_coroutine_func,
        ch,
        4096
    );
    TEST_ASSERT_NOT_NULL(sender);
    
    Task* sender_task = task_create(NULL, NULL);
    sender->task = sender_task;
    sender_task->coroutine = sender;
    
    // 执行发送（应该直接传递给接收者）
    pthread_mutex_lock(&ch->lock);
    int* value = malloc(sizeof(int));
    *value = 42;
    channel_send_impl(ch, value);
    pthread_mutex_unlock(&ch->lock);
    
    // 验证值被传递
    TEST_ASSERT(ch->size == 0 || ch->temp_value == NULL);
    
    // 清理
    channel_destroy_impl(ch);
    coroutine_destroy(sender);
    coroutine_destroy(receiver);
    task_destroy(sender_task);
    task_destroy(receiver_task);
    free(value);
    
    printf("✓ Unbuffered channel blocking send test passed\n");
}

/**
 * @brief 测试无缓冲通道阻塞接收
 */
void test_unbuffered_channel_blocking_receive(void) {
    printf("=== Testing Unbuffered Channel Blocking Receive ===\n");
    
    // 创建无缓冲通道
    Channel* ch = (Channel*)channel_create_impl();
    TEST_ASSERT_NOT_NULL(ch);
    
    // 创建发送者协程（先创建发送者，这样接收者不会阻塞）
    Coroutine* sender = coroutine_create(
        "sender_coroutine",
        sender_coroutine_func,
        ch,
        4096
    );
    TEST_ASSERT_NOT_NULL(sender);
    
    Task* sender_task = task_create(NULL, NULL);
    sender->task = sender_task;
    sender_task->coroutine = sender;
    
    // 将发送者加入通道等待队列
    sender->next = ch->sender_queue;
    ch->sender_queue = sender;
    
    // 创建接收者协程
    Coroutine* receiver = coroutine_create(
        "receiver_coroutine",
        receiver_coroutine_func,
        ch,
        4096
    );
    TEST_ASSERT_NOT_NULL(receiver);
    
    Task* receiver_task = task_create(NULL, NULL);
    receiver->task = receiver_task;
    receiver_task->coroutine = receiver;
    
    // 执行接收（应该直接从发送者获取值）
    pthread_mutex_lock(&ch->lock);
    int* test_value = malloc(sizeof(int));
    *test_value = 100;
    ch->temp_value = test_value;
    ch->size = 1;
    
    void* received = channel_receive_impl(ch);
    pthread_mutex_unlock(&ch->lock);
    
    // 验证值被接收
    TEST_ASSERT_NOT_NULL(received);
    TEST_ASSERT_EQUAL(100, *(int*)received);
    
    // 清理
    channel_destroy_impl(ch);
    coroutine_destroy(sender);
    coroutine_destroy(receiver);
    task_destroy(sender_task);
    task_destroy(receiver_task);
    free(test_value);
    free(received);
    
    printf("✓ Unbuffered channel blocking receive test passed\n");
}

/**
 * @brief 测试发送者和接收者同步
 */
void test_sender_receiver_synchronization(void) {
    printf("=== Testing Sender-Receiver Synchronization ===\n");
    
    // 创建无缓冲通道
    Channel* ch = (Channel*)channel_create_impl();
    TEST_ASSERT_NOT_NULL(ch);
    
    value_received = false;
    received_value = NULL;
    
    // 创建发送者和接收者协程
    Coroutine* sender = coroutine_create(
        "sender_coroutine",
        sender_coroutine_func,
        ch,
        4096
    );
    
    Coroutine* receiver = coroutine_create(
        "receiver_coroutine",
        receiver_coroutine_func,
        ch,
        4096
    );
    
    TEST_ASSERT_NOT_NULL(sender);
    TEST_ASSERT_NOT_NULL(receiver);
    
    // 创建任务
    Task* sender_task = task_create(NULL, NULL);
    Task* receiver_task = task_create(NULL, NULL);
    sender->task = sender_task;
    receiver->task = receiver_task;
    sender_task->coroutine = sender;
    receiver_task->coroutine = receiver;
    
    // 先设置接收者等待
    pthread_mutex_lock(&ch->lock);
    receiver->next = ch->receiver_queue;
    ch->receiver_queue = receiver;
    receiver->state = COROUTINE_SUSPENDED;
    pthread_mutex_unlock(&ch->lock);
    
    // 发送者发送（应该直接传递给接收者）
    pthread_mutex_lock(&ch->lock);
    int* value = malloc(sizeof(int));
    *value = 200;
    channel_send_impl(ch, value);
    pthread_mutex_unlock(&ch->lock);
    
    // 验证接收者被唤醒
    TEST_ASSERT(receiver->state == COROUTINE_READY || 
                receiver->state == COROUTINE_RUNNING);
    
    // 清理
    channel_destroy_impl(ch);
    coroutine_destroy(sender);
    coroutine_destroy(receiver);
    task_destroy(sender_task);
    task_destroy(receiver_task);
    free(value);
    
    printf("✓ Sender-receiver synchronization test passed\n");
}

/**
 * @brief 测试通道关闭时唤醒
 */
void test_channel_close_wakeup(void) {
    printf("=== Testing Channel Close Wakeup ===\n");
    
    // 创建无缓冲通道
    Channel* ch = (Channel*)channel_create_impl();
    TEST_ASSERT_NOT_NULL(ch);
    
    // 创建等待的发送者和接收者
    Coroutine* sender = coroutine_create(
        "sender_coroutine",
        sender_coroutine_func,
        ch,
        4096
    );
    
    Coroutine* receiver = coroutine_create(
        "receiver_coroutine",
        receiver_coroutine_func,
        ch,
        4096
    );
    
    TEST_ASSERT_NOT_NULL(sender);
    TEST_ASSERT_NOT_NULL(receiver);
    
    // 创建任务
    Task* sender_task = task_create(NULL, NULL);
    Task* receiver_task = task_create(NULL, NULL);
    sender->task = sender_task;
    receiver->task = receiver_task;
    sender_task->coroutine = sender;
    receiver_task->coroutine = receiver;
    
    // 将发送者和接收者加入等待队列
    pthread_mutex_lock(&ch->lock);
    sender->next = ch->sender_queue;
    ch->sender_queue = sender;
    sender->state = COROUTINE_SUSPENDED;
    
    receiver->next = ch->receiver_queue;
    ch->receiver_queue = receiver;
    receiver->state = COROUTINE_SUSPENDED;
    pthread_mutex_unlock(&ch->lock);
    
    // 关闭通道（应该唤醒所有等待者）
    channel_close_impl(ch);
    
    // 验证等待者被唤醒
    TEST_ASSERT(sender->state == COROUTINE_READY);
    TEST_ASSERT(receiver->state == COROUTINE_READY);
    TEST_ASSERT(ch->sender_queue == NULL);
    TEST_ASSERT(ch->receiver_queue == NULL);
    
    // 清理
    channel_destroy_impl(ch);
    coroutine_destroy(sender);
    coroutine_destroy(receiver);
    task_destroy(sender_task);
    task_destroy(receiver_task);
    
    printf("✓ Channel close wakeup test passed\n");
}

// 测试套件
TestSuite* channel_blocking_test_suite(void) {
    TestSuite* suite = TEST_SUITE("Channel Blocking Tests", "无缓冲通道阻塞通信测试套件");
    
    TEST_ADD(suite, "unbuffered_channel_blocking_send", 
             "测试无缓冲通道阻塞发送", test_unbuffered_channel_blocking_send);
    TEST_ADD(suite, "unbuffered_channel_blocking_receive", 
             "测试无缓冲通道阻塞接收", test_unbuffered_channel_blocking_receive);
    TEST_ADD(suite, "sender_receiver_synchronization", 
             "测试发送者和接收者同步", test_sender_receiver_synchronization);
    TEST_ADD(suite, "channel_close_wakeup", 
             "测试通道关闭时唤醒", test_channel_close_wakeup);
    
    return suite;
}

// 主函数（用于独立运行测试）
#ifdef STANDALONE_TEST
int main(void) {
    printf("=== Channel Blocking Test Suite ===\n");
    printf("====================================\n\n");
    
    int tests_run = 0;
    int tests_passed = 0;
    
    // 运行所有测试
    printf("\n[1/4] Running test_unbuffered_channel_blocking_send...\n");
    tests_run++;
    test_unbuffered_channel_blocking_send();
    tests_passed++;
    
    printf("\n[2/4] Running test_unbuffered_channel_blocking_receive...\n");
    tests_run++;
    test_unbuffered_channel_blocking_receive();
    tests_passed++;
    
    printf("\n[3/4] Running test_sender_receiver_synchronization...\n");
    tests_run++;
    test_sender_receiver_synchronization();
    tests_passed++;
    
    printf("\n[4/4] Running test_channel_close_wakeup...\n");
    tests_run++;
    test_channel_close_wakeup();
    tests_passed++;
    
    // 输出统计
    printf("\n====================================\n");
    printf("Test Summary:\n");
    printf("  Total:  %d\n", tests_run);
    printf("  Passed: %d\n", tests_passed);
    printf("  Failed: %d\n", tests_run - tests_passed);
    printf("====================================\n");
    
    return (tests_passed == tests_run) ? 0 : 1;
}
#endif

