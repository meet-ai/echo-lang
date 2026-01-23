#ifndef ECHO_CORE_GC_H
#define ECHO_CORE_GC_H

#include "echo/gc.h"

// 端口接口声明
typedef void (*echo_gc_event_publisher_t)(const char* event_type, const void* event_data);

// 根对象扫描函数
void echo_gc_scan_stack_roots(void* context);
void echo_gc_scan_global_roots(void* context);
void echo_gc_scan_thread_roots(void* context);

#endif // ECHO_CORE_GC_H
