#ifndef ECHO_RUNTIME_H
#define ECHO_RUNTIME_H

#include <stdint.h>
#include <stdlib.h>

// 内存管理
void* echo_alloc(size_t size);
void echo_free(void* ptr);

// 基础I/O
void echo_print_int(int32_t value);
void echo_print_string(const char* str);

// 智能体系统
typedef int agent_id_t;

// 智能体创建和销毁
agent_id_t echo_agent_create(void);
void echo_agent_destroy(agent_id_t agent_id);

// 消息传递
void echo_agent_send(agent_id_t from_agent, agent_id_t to_agent, const char* message);
char* echo_agent_receive(agent_id_t agent_id);

// 智能体生命周期
void echo_agent_start(agent_id_t agent_id);
void echo_agent_stop(agent_id_t agent_id);
int echo_agent_is_alive(agent_id_t agent_id);

#endif
