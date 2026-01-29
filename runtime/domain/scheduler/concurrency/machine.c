#include "machine.h"
#include "scheduler.h"
#include "processor.h"
#include "coroutine.h"
#include <stdio.h>
#include <stdlib.h>
#include <string.h>
#include <unistd.h>

// 线程入口函数
void* machine_thread_entry(void* arg) {
    Machine* machine = (Machine*)arg;
    if (!machine) {
        fprintf(stderr, "ERROR: machine_thread_entry called with NULL machine\n");
        return NULL;
    }

    printf("DEBUG: Machine %u thread started\n", machine->id);

    // 设置当前机器
    // TODO: 设置线程本地存储

    // 绑定到处理器
    if (!machine->processor) {
        fprintf(stderr, "ERROR: Machine %u has no bound processor\n", machine->id);
        return NULL;
    }

    // 运行处理器调度循环
    while (!machine->should_stop) {
        printf("DEBUG: Machine %u running processor schedule\n", machine->id);
        processor_schedule(machine->processor);

        // 如果处理器没有工作，短暂休眠避免忙等待
        usleep(1000); // 1ms
    }

    printf("DEBUG: Machine %u thread exiting\n", machine->id);
    return NULL;
}

// 创建Machine
Machine* machine_create(uint32_t id, struct Scheduler* scheduler) {
    Machine* machine = (Machine*)malloc(sizeof(Machine));
    if (!machine) {
        fprintf(stderr, "ERROR: Failed to allocate memory for Machine\n");
        return NULL;
    }

    memset(machine, 0, sizeof(Machine));
    machine->id = id;
    machine->scheduler = scheduler;
    machine->is_running = false;
    machine->should_stop = false;

    // 分配并初始化上下文（用于可能的上下文切换）
    // 注意：machine->context 是 context_t* 类型，需要先分配内存
    machine->context = (context_t*)malloc(sizeof(context_t));
    if (machine->context) {
        context_init(machine->context, NULL, NULL);
    }

    printf("DEBUG: Created Machine %u\n", id);
    return machine;
}

// 销毁Machine
void machine_destroy(Machine* machine) {
    if (!machine) return;

    printf("DEBUG: Destroying Machine %u\n", machine->id);

    // 释放上下文内存
    if (machine->context) {
        free(machine->context);
        machine->context = NULL;
    }

    // 停止线程
    if (machine->is_running) {
        machine_stop(machine);
    }

    free(machine);
    printf("DEBUG: Machine %u destroyed\n", machine->id);
}

// 启动Machine线程
bool machine_start(Machine* machine) {
    if (!machine || machine->is_running) return false;

    printf("DEBUG: Starting Machine %u thread\n", machine->id);

    int result = pthread_create(&machine->thread, NULL, machine_thread_entry, machine);
    if (result != 0) {
        fprintf(stderr, "ERROR: Failed to create thread for Machine %u: %d\n", machine->id, result);
        return false;
    }

    machine->is_running = true;
    printf("DEBUG: Machine %u thread started successfully\n", machine->id);
    return true;
}

// 停止Machine线程
void machine_stop(Machine* machine) {
    if (!machine || !machine->is_running) return;

    printf("DEBUG: Stopping Machine %u thread\n", machine->id);

    machine->should_stop = true;

    // 等待线程结束
    pthread_join(machine->thread, NULL);

    machine->is_running = false;
    printf("DEBUG: Machine %u thread stopped\n", machine->id);
}
