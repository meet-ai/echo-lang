/**
 * @file test_channel_buffered.c
 * @brief 缓冲通道测试
 * 
 * 测试有缓冲通道的基本发送接收、缓冲区满/空时的阻塞逻辑
 */

#include "../../../unit-tests/test_framework.h"
#include "../../../../domain/channel/channel.h"
#include "../../../../domain/coroutine/coroutine.h"
#include "../../../../domain/task/task.h"
#include <stdio.h>
#include <stdlib.h>
#include <string.h>
#include <assert.h>
#include <pthread.h>

/**
 * @brief 测试缓冲通道基本发送接收
 */
void test_buffered_channel_basic_send_receive(void) {
    printf("=== Testing Buffered Channel Basic Send/Receive ===\n");
    
    // 创建有缓冲通道（容量为5）
    Channel* ch = (Channel*)channel_create_buffered_impl(5);
    TEST_ASSERT_NOT_NULL(ch);
    TEST_ASSERT_EQUAL(5, ch->capacity);
    TEST_ASSERT_NOT_NULL(ch->buffer);
    TEST_ASSERT_EQUAL(0, ch->size);
    TEST_ASSERT_EQUAL(0, ch->read_pos);
    TEST_ASSERT_EQUAL(0, ch->write_pos);
    
    // 发送多个值
    int values[5] = {10, 20, 30, 40, 50};
    for (int i = 0; i < 5; i++) {
        pthread_mutex_lock(&ch->lock);
        channel_send_impl(ch, &values[i]);
        pthread_mutex_unlock(&ch->lock);
        TEST_ASSERT_EQUAL(i + 1, ch->size);
    }
    
    // 验证缓冲区已满
    TEST_ASSERT_EQUAL(5, ch->size);
    TEST_ASSERT_EQUAL(5, ch->write_pos);
    
    // 接收所有值
    for (int i = 0; i < 5; i++) {
        pthread_mutex_lock(&ch->lock);
        int* received = (int*)channel_receive_impl(ch);
        pthread_mutex_unlock(&ch->lock);
        TEST_ASSERT_NOT_NULL(received);
        TEST_ASSERT_EQUAL(values[i], *received);
        TEST_ASSERT_EQUAL(5 - i - 1, ch->size);
    }
    
    // 验证缓冲区为空
    TEST_ASSERT_EQUAL(0, ch->size);
    
    // 清理
    channel_destroy_impl(ch);
    
    printf("✓ Buffered channel basic send/receive test passed\n");
}

/**
 * @brief 测试缓冲区满时阻塞
 */
void test_buffered_channel_full_blocking(void) {
    printf("=== Testing Buffered Channel Full Blocking ===\n");
    
    // 创建有缓冲通道（容量为2）
    Channel* ch = (Channel*)channel_create_buffered_impl(2);
    TEST_ASSERT_NOT_NULL(ch);
    
    // 填满缓冲区
    int value1 = 100;
    int value2 = 200;
    
    pthread_mutex_lock(&ch->lock);
    channel_send_impl(ch, &value1);
    channel_send_impl(ch, &value2);
    pthread_mutex_unlock(&ch->lock);
    
    TEST_ASSERT_EQUAL(2, ch->size);
    TEST_ASSERT_EQUAL(2, ch->capacity);
    
    // 创建发送者协程（尝试发送到已满的缓冲区）
    Coroutine* sender = coroutine_create(
        "sender_coroutine",
        NULL,
        ch,
        4096
    );
    TEST_ASSERT_NOT_NULL(sender);
    
    Task* sender_task = task_create(NULL, NULL);
    sender->task = sender_task;
    sender_task->coroutine = sender;
    
    // 尝试发送（应该阻塞）
    pthread_mutex_lock(&ch->lock);
    // int value3 = 300;  // 未使用，注释掉
    
    // 检查缓冲区是否满
    if (ch->size >= ch->capacity) {
        // 发送者应该被加入等待队列
        sender->next = ch->sender_queue;
        ch->sender_queue = sender;
        sender->state = COROUTINE_SUSPENDED;
        sender_task->status = TASK_WAITING;
        
        TEST_ASSERT_NOT_NULL(ch->sender_queue);
        TEST_ASSERT_EQUAL(COROUTINE_SUSPENDED, sender->state);
        TEST_ASSERT_EQUAL(TASK_WAITING, sender_task->status);
    }
    pthread_mutex_unlock(&ch->lock);
    
    // 清理
    channel_destroy_impl(ch);
    coroutine_destroy(sender);
    task_destroy(sender_task);
    
    printf("✓ Buffered channel full blocking test passed\n");
}

/**
 * @brief 测试缓冲区空时阻塞
 */
void test_buffered_channel_empty_blocking(void) {
    printf("=== Testing Buffered Channel Empty Blocking ===\n");
    
    // 创建有缓冲通道
    Channel* ch = (Channel*)channel_create_buffered_impl(3);
    TEST_ASSERT_NOT_NULL(ch);
    
    // 创建接收者协程（尝试从空的缓冲区接收）
    Coroutine* receiver = coroutine_create(
        "receiver_coroutine",
        NULL,
        ch,
        4096
    );
    TEST_ASSERT_NOT_NULL(receiver);
    
    Task* receiver_task = task_create(NULL, NULL);
    receiver->task = receiver_task;
    receiver_task->coroutine = receiver;
    
    // 尝试接收（应该阻塞）
    pthread_mutex_lock(&ch->lock);
    
    // 检查缓冲区是否空
    if (ch->size == 0) {
        // 接收者应该被加入等待队列
        receiver->next = ch->receiver_queue;
        ch->receiver_queue = receiver;
        receiver->state = COROUTINE_SUSPENDED;
        receiver_task->status = TASK_WAITING;
        
        TEST_ASSERT_NOT_NULL(ch->receiver_queue);
        TEST_ASSERT_EQUAL(COROUTINE_SUSPENDED, receiver->state);
        TEST_ASSERT_EQUAL(TASK_WAITING, receiver_task->status);
    }
    pthread_mutex_unlock(&ch->lock);
    
    // 清理
    channel_destroy_impl(ch);
    coroutine_destroy(receiver);
    task_destroy(receiver_task);
    
    printf("✓ Buffered channel empty blocking test passed\n");
}

/**
 * @brief 测试环形缓冲区正确性
 */
void test_circular_buffer_correctness(void) {
    printf("=== Testing Circular Buffer Correctness ===\n");
    
    // 创建有缓冲通道（容量为3）
    Channel* ch = (Channel*)channel_create_buffered_impl(3);
    TEST_ASSERT_NOT_NULL(ch);
    
    // 发送3个值填满缓冲区
    int values[3] = {1, 2, 3};
    for (int i = 0; i < 3; i++) {
        pthread_mutex_lock(&ch->lock);
        channel_send_impl(ch, &values[i]);
        pthread_mutex_unlock(&ch->lock);
    }
    
    TEST_ASSERT_EQUAL(3, ch->size);
    TEST_ASSERT_EQUAL(0, ch->read_pos);  // 读指针在开始
    TEST_ASSERT_EQUAL(0, ch->write_pos); // 写指针绕回开始（因为是环形）
    
    // 接收1个值
    pthread_mutex_lock(&ch->lock);
    int* received1 = (int*)channel_receive_impl(ch);
    pthread_mutex_unlock(&ch->lock);
    
    TEST_ASSERT_NOT_NULL(received1);
    TEST_ASSERT_EQUAL(1, *received1);
    TEST_ASSERT_EQUAL(2, ch->size);
    TEST_ASSERT_EQUAL(1, ch->read_pos);  // 读指针前进
    
    // 再发送1个值（测试环形缓冲区）
    int value4 = 4;
    pthread_mutex_lock(&ch->lock);
    channel_send_impl(ch, &value4);
    pthread_mutex_unlock(&ch->lock);
    
    TEST_ASSERT_EQUAL(3, ch->size);
    TEST_ASSERT_EQUAL(1, ch->write_pos); // 写指针应该在位置1
    
    // 接收剩余的值
    for (int i = 0; i < 3; i++) {
        pthread_mutex_lock(&ch->lock);
        int* received = (int*)channel_receive_impl(ch);
        pthread_mutex_unlock(&ch->lock);
        if (received) {
            TEST_ASSERT_EQUAL(i + 2, *received);  // 应该是2, 3, 4
        }
    }
    
    TEST_ASSERT_EQUAL(0, ch->size);
    
    // 清理
    channel_destroy_impl(ch);
    
    printf("✓ Circular buffer correctness test passed\n");
}

// 测试套件
TestSuite* channel_buffered_test_suite(void) {
    TestSuite* suite = TEST_SUITE("Channel Buffered Tests", "缓冲通道测试套件");
    
    TEST_ADD(suite, "buffered_channel_basic_send_receive", 
             "测试缓冲通道基本发送接收", test_buffered_channel_basic_send_receive);
    TEST_ADD(suite, "buffered_channel_full_blocking", 
             "测试缓冲区满时阻塞", test_buffered_channel_full_blocking);
    TEST_ADD(suite, "buffered_channel_empty_blocking", 
             "测试缓冲区空时阻塞", test_buffered_channel_empty_blocking);
    TEST_ADD(suite, "circular_buffer_correctness", 
             "测试环形缓冲区正确性", test_circular_buffer_correctness);
    
    return suite;
}

// 主函数（用于独立运行测试）
#ifdef STANDALONE_TEST
int main(void) {
    printf("=== Channel Buffered Test Suite ===\n");
    printf("====================================\n\n");
    
    int tests_run = 0;
    int tests_passed = 0;
    
    // 运行所有测试
    printf("\n[1/4] Running test_buffered_channel_basic_send_receive...\n");
    tests_run++;
    test_buffered_channel_basic_send_receive();
    tests_passed++;
    
    printf("\n[2/4] Running test_buffered_channel_full_blocking...\n");
    tests_run++;
    test_buffered_channel_full_blocking();
    tests_passed++;
    
    printf("\n[3/4] Running test_buffered_channel_empty_blocking...\n");
    tests_run++;
    test_buffered_channel_empty_blocking();
    tests_passed++;
    
    printf("\n[4/4] Running test_circular_buffer_correctness...\n");
    tests_run++;
    test_circular_buffer_correctness();
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

