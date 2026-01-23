/**
 * @file select.h
 * @brief Select 多路复用接口
 *
 * Select 允许同时监听多个通道的操作，是并发编程的核心原语。
 */

#ifndef SELECT_H
#define SELECT_H

#include "channel.h"
#include <stdbool.h>

// ============================================================================
// Select 操作定义
// ============================================================================

/**
 * @brief Select case 类型
 */
typedef enum select_case_type {
    SELECT_CASE_SEND,    // 发送操作
    SELECT_CASE_RECV     // 接收操作
} select_case_type_t;

/**
 * @brief Select case 定义
 */
typedef struct select_case {
    Channel* channel;             // 操作的通道
    select_case_type_t type;      // 操作类型
    void* value;                  // 发送的值（对发送操作）或接收的值（对接收操作）
    bool ready;                   // 该case是否准备好
} select_case_t;

/**
 * @brief Select 操作结果
 */
typedef struct select_result {
    int selected_index;           // 选中的case索引，-1表示无选择
    void* received_value;         // 接收到的值（对接收操作）
    bool has_timeout;             // 是否因超时而返回
} select_result_t;

// ============================================================================
// Select 接口
// ============================================================================

/**
 * @brief 执行select操作（阻塞）
 *
 * 同时监听多个通道的操作，选择第一个准备好的执行。
 *
 * @param cases select case数组
 * @param num_cases case数量
 * @return select结果
 */
select_result_t select_execute(select_case_t* cases, int num_cases);

/**
 * @brief 执行select操作（带超时）
 *
 * @param cases select case数组
 * @param num_cases case数量
 * @param timeout_ms 超时时间（毫秒），0表示不阻塞，-1表示无限等待
 * @return select结果
 */
select_result_t select_execute_timeout(select_case_t* cases, int num_cases, int timeout_ms);

/**
 * @brief 检查select cases是否有准备好的
 *
 * @param cases select case数组
 * @param num_cases case数量
 * @return true如果至少有一个case准备好
 */
bool select_has_ready(select_case_t* cases, int num_cases);

/**
 * @brief 查找第一个准备好的case
 *
 * @param cases select case数组
 * @param num_cases case数量
 * @return 准备好的case索引，-1表示无
 */
int select_find_ready(select_case_t* cases, int num_cases);

#endif // SELECT_H

