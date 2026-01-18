#include "echo_runtime.h"
#include <stdio.h>
#include <stdlib.h>
#include <string.h>

// 内存管理实现
void* echo_alloc(size_t size) {
    return malloc(size);
}

void echo_free(void* ptr) {
    free(ptr);
}

// 基础I/O实现
void echo_print_int(int32_t value) {
    printf("%d\n", value);
}

void echo_print_string(const char* str) {
    printf("%s", str);
}

// 智能体系统实现
#define MAX_AGENTS 100
#define MAX_MESSAGE_SIZE 1024

typedef struct {
    int is_alive;
    char inbox[MAX_MESSAGE_SIZE];
    int has_message;
} Agent;

static Agent agents[MAX_AGENTS];
static int next_agent_id = 1;

// 智能体创建和销毁
agent_id_t echo_agent_create(void) {
    for (int i = 0; i < MAX_AGENTS; i++) {
        if (!agents[i].is_alive) {
            agents[i].is_alive = 1;
            agents[i].has_message = 0;
            agents[i].inbox[0] = '\0';
            return next_agent_id++;
        }
    }
    return -1; // 创建失败
}

void echo_agent_destroy(agent_id_t agent_id) {
    int index = agent_id - 1;
    if (index >= 0 && index < MAX_AGENTS) {
        agents[index].is_alive = 0;
        agents[index].has_message = 0;
    }
}

// 消息传递
void echo_agent_send(agent_id_t from_agent, agent_id_t to_agent, const char* message) {
    int to_index = to_agent - 1;
    if (to_index >= 0 && to_index < MAX_AGENTS && agents[to_index].is_alive) {
        // 简单实现：覆盖之前的消息
        strncpy(agents[to_index].inbox, message, MAX_MESSAGE_SIZE - 1);
        agents[to_index].inbox[MAX_MESSAGE_SIZE - 1] = '\0';
        agents[to_index].has_message = 1;
    }
}

char* echo_agent_receive(agent_id_t agent_id) {
    int index = agent_id - 1;
    if (index >= 0 && index < MAX_AGENTS && agents[index].is_alive && agents[index].has_message) {
        agents[index].has_message = 0;
        return agents[index].inbox;
    }
    return NULL; // 没有消息
}

// 智能体生命周期
void echo_agent_start(agent_id_t agent_id) {
    // 简单实现：标记为活跃
    int index = agent_id - 1;
    if (index >= 0 && index < MAX_AGENTS) {
        agents[index].is_alive = 1;
    }
}

void echo_agent_stop(agent_id_t agent_id) {
    // 简单实现：标记为非活跃
    int index = agent_id - 1;
    if (index >= 0 && index < MAX_AGENTS) {
        agents[index].is_alive = 0;
    }
}

int echo_agent_is_alive(agent_id_t agent_id) {
    int index = agent_id - 1;
    if (index >= 0 && index < MAX_AGENTS) {
        return agents[index].is_alive;
    }
    return 0;
}
