/**
 * @file select.c
 * @brief Select 多路复用实现
 *
 * 实现select多路复用的核心逻辑，支持同时监听多个通道操作。
 */

#include "select.h"
#include <stdlib.h>
#include <stdio.h>
#include <string.h>
#include <pthread.h>
#include <sys/time.h>
#include <errno.h>
#include <unistd.h>

// ============================================================================
// 内部辅助函数
// ============================================================================

/**
 * @brief 检查单个case是否准备好（不消耗消息）
 */
static bool is_case_ready(select_case_t* case_ptr) {
    if (!case_ptr || !case_ptr->channel) return false;

    switch (case_ptr->type) {
        case SELECT_CASE_SEND:
            // 简化实现：假设发送总是准备好的
            return true;

        case SELECT_CASE_RECV: {
            // 接收操作：检查通道是否有消息（不实际接收）
            return channel_get_message_count(case_ptr->channel) > 0;
        }

        default:
            return false;
    }
}

/**
 * @brief 执行单个case操作
 */
static bool execute_case(select_case_t* case_ptr) {
    if (!case_ptr || !case_ptr->channel) return false;

    switch (case_ptr->type) {
        case SELECT_CASE_SEND:
            // 发送操作
            return channel_try_send(case_ptr->channel, case_ptr->value) == 0;

        case SELECT_CASE_RECV: {
            // 接收操作
            void* received = channel_try_receive(case_ptr->channel);
            if (received) {
                // 将接收到的值存储在case中供调用者使用
                case_ptr->value = received;
                return true;
            }
            return false;
        }

        default:
            return false;
    }
}

/**
 * @brief 创建条件变量数组用于等待
 */
static pthread_cond_t* create_wait_conds(int num_cases) {
    pthread_cond_t* conds = (pthread_cond_t*)malloc(sizeof(pthread_cond_t) * num_cases);
    if (!conds) return NULL;

    for (int i = 0; i < num_cases; i++) {
        if (pthread_cond_init(&conds[i], NULL) != 0) {
            // 清理已初始化的条件变量
            for (int j = 0; j < i; j++) {
                pthread_cond_destroy(&conds[j]);
            }
            free(conds);
            return NULL;
        }
    }

    return conds;
}

/**
 * @brief 销毁条件变量数组
 */
static void destroy_wait_conds(pthread_cond_t* conds, int num_cases) {
    if (!conds) return;

    for (int i = 0; i < num_cases; i++) {
        pthread_cond_destroy(&conds[i]);
    }
    free(conds);
}

/**
 * @brief 获取当前时间加上超时
 */
static struct timespec get_timeout_time(int timeout_ms) {
    struct timespec timeout;
    clock_gettime(CLOCK_REALTIME, &timeout);

    timeout.tv_sec += timeout_ms / 1000;
    timeout.tv_nsec += (timeout_ms % 1000) * 1000000;

    if (timeout.tv_nsec >= 1000000000) {
        timeout.tv_sec += 1;
        timeout.tv_nsec -= 1000000000;
    }

    return timeout;
}

// ============================================================================
// Select 接口实现
// ============================================================================

/**
 * @brief 执行select操作（阻塞）
 */
select_result_t select_execute(select_case_t* cases, int num_cases) {
    return select_execute_timeout(cases, num_cases, -1); // 无限等待
}

/**
 * @brief 执行select操作（带超时）
 */
select_result_t select_execute_timeout(select_case_t* cases, int num_cases, int timeout_ms) {
    select_result_t result = { .selected_index = -1, .received_value = NULL, .has_timeout = false };

    if (!cases || num_cases <= 0) {
        return result;
    }

    // 第一遍：检查是否有立即准备好的case
    for (int i = 0; i < num_cases; i++) {
        if (is_case_ready(&cases[i])) {
            // 找到准备好的case，执行它
            if (execute_case(&cases[i])) {
                result.selected_index = i;
                if (cases[i].type == SELECT_CASE_RECV) {
                    result.received_value = cases[i].value;
                }
                printf("DEBUG: Select chose case %d immediately\n", i);
                return result;
            }
        }
    }

    // 第二遍：没有立即准备好的，等待
    // 创建等待条件变量
    pthread_cond_t* wait_conds = create_wait_conds(num_cases);
    if (!wait_conds) {
        fprintf(stderr, "Failed to create wait conditions for select\n");
        return result;
    }

    // 为每个通道注册等待条件（简化实现）
    // 实际实现应该修改Channel结构以支持select等待

    // 等待逻辑
    if (timeout_ms == 0) {
        // 非阻塞模式，直接返回无选择
        result.has_timeout = true;
    } else {
        // 阻塞等待
        struct timespec timeout_time;
        if (timeout_ms > 0) {
            timeout_time = get_timeout_time(timeout_ms);
        }

        // 简化实现：轮询等待
        // 实际应该使用条件变量等待
        const int max_attempts = (timeout_ms > 0) ? (timeout_ms / 10) : 1000;
        int attempts = 0;

        while (attempts < max_attempts) {
            for (int i = 0; i < num_cases; i++) {
                if (is_case_ready(&cases[i])) {
                    if (execute_case(&cases[i])) {
                        result.selected_index = i;
                        if (cases[i].type == SELECT_CASE_RECV) {
                            result.received_value = cases[i].value;
                        }
                        printf("DEBUG: Select chose case %d after waiting\n", i);
                        goto cleanup;
                    }
                }
            }

            // 小睡一会儿再检查
            usleep(10000); // 10ms
            attempts++;

            if (timeout_ms > 0 && attempts * 10 >= timeout_ms) {
                result.has_timeout = true;
                break;
            }
        }
    }

cleanup:
    destroy_wait_conds(wait_conds, num_cases);
    return result;
}

/**
 * @brief 检查select cases是否有准备好的
 */
bool select_has_ready(select_case_t* cases, int num_cases) {
    if (!cases || num_cases <= 0) return false;

    for (int i = 0; i < num_cases; i++) {
        if (is_case_ready(&cases[i])) {
            return true;
        }
    }

    return false;
}

/**
 * @brief 查找第一个准备好的case
 */
int select_find_ready(select_case_t* cases, int num_cases) {
    if (!cases || num_cases <= 0) return -1;

    for (int i = 0; i < num_cases; i++) {
        if (is_case_ready(&cases[i])) {
            return i;
        }
    }

    return -1;
}
