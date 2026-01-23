#ifndef ATOMIC_H
#define ATOMIC_H

#include <stdint.h>
#include <stdbool.h>
#include <stdatomic.h>

// 原子整数类型
typedef struct {
    atomic_int value;
} AtomicInt;

typedef struct {
    atomic_long value;
} AtomicLong;

// 原子操作函数

// AtomicInt 操作
AtomicInt* atomic_int_create(int initial_value);
void atomic_int_destroy(AtomicInt* atomic);

int atomic_int_load(const AtomicInt* atomic);
void atomic_int_store(AtomicInt* atomic, int value);
int atomic_int_exchange(AtomicInt* atomic, int new_value);
bool atomic_int_compare_exchange(AtomicInt* atomic, int expected, int desired);

int atomic_int_fetch_add(AtomicInt* atomic, int delta);
int atomic_int_fetch_sub(AtomicInt* atomic, int delta);
int atomic_int_fetch_and(AtomicInt* atomic, int value);
int atomic_int_fetch_or(AtomicInt* atomic, int value);
int atomic_int_fetch_xor(AtomicInt* atomic, int value);

// AtomicLong 操作
AtomicLong* atomic_long_create(long initial_value);
void atomic_long_destroy(AtomicLong* atomic);

long atomic_long_load(const AtomicLong* atomic);
void atomic_long_store(AtomicLong* atomic, long value);
long atomic_long_exchange(AtomicLong* atomic, long new_value);
bool atomic_long_compare_exchange(AtomicLong* atomic, long expected, long desired);

long atomic_long_fetch_add(AtomicLong* atomic, long delta);
long atomic_long_fetch_sub(AtomicLong* atomic, long delta);
long atomic_long_fetch_and(AtomicLong* atomic, long value);
long atomic_long_fetch_or(AtomicLong* atomic, long value);
long atomic_long_fetch_xor(AtomicLong* atomic, long value);

// 原子操作的内存序
typedef enum {
    ATOMIC_MEMORY_ORDER_RELAXED,
    ATOMIC_MEMORY_ORDER_CONSUME,
    ATOMIC_MEMORY_ORDER_ACQUIRE,
    ATOMIC_MEMORY_ORDER_RELEASE,
    ATOMIC_MEMORY_ORDER_ACQ_REL,
    ATOMIC_MEMORY_ORDER_SEQ_CST
} AtomicMemoryOrder;

// 高级原子操作
bool atomic_spin_lock(AtomicInt* lock);
void atomic_spin_unlock(AtomicInt* lock);

// 原子标志
typedef AtomicInt AtomicFlag;

#define ATOMIC_FLAG_INIT {0}

void atomic_flag_clear(AtomicFlag* flag);
bool atomic_flag_test_and_set(AtomicFlag* flag);

#endif // ATOMIC_H
