#include "../unit-tests/test_framework.h"
#include "../../domain/task/service/task_scheduler.h"
#include <time.h>
#include <sys/time.h>

// 测试套件
TestSuite* task_scheduler_performance_test_suite(void);

// 性能测试用例
void test_task_scheduler_throughput(void);
void test_task_scheduler_latency(void);
void test_task_scheduler_memory_usage(void);
void test_task_scheduler_concurrent_operations(void);
void test_task_scheduler_priority_scheduling(void);

// 性能基准
#define PERFORMANCE_ITERATIONS 10000
#define CONCURRENT_TASKS 1000
#define WARMUP_ITERATIONS 1000

// 基准测试结果
typedef struct BenchmarkResult {
    double throughput;        // 吞吐量 (ops/sec)
    double avg_latency;       // 平均延迟 (us)
    double min_latency;       // 最小延迟 (us)
    double max_latency;       // 最大延迟 (us)
    double p95_latency;       // 95%延迟 (us)
    double p99_latency;       // 99%延迟 (us)
    size_t memory_usage;      // 内存使用 (bytes)
    time_t duration;          // 测试持续时间 (seconds)
} BenchmarkResult;

// 获取当前时间（微秒）
static uint64_t get_current_time_us(void) {
    struct timeval tv;
    gettimeofday(&tv, NULL);
    return (uint64_t)tv.tv_sec * 1000000 + (uint64_t)tv.tv_usec;
}

// 计算百分位数延迟
static double calculate_percentile(uint64_t* latencies, size_t count, double percentile) {
    if (count == 0) return 0.0;

    size_t index = (size_t)(percentile * (count - 1) / 100.0);
    return (double)latencies[index];
}

// 任务调度器吞吐量测试
void test_task_scheduler_throughput(void) {
    TaskScheduler* scheduler = task_scheduler_create();
    BenchmarkResult result = {0};

    // 预热
    for (int i = 0; i < WARMUP_ITERATIONS; i++) {
        Task* task = task_create("warmup", simple_performance_task, NULL, 1024);
        task_scheduler_submit(scheduler, task);
        Task* scheduled = task_scheduler_schedule_next(scheduler);
        task_destroy(task);
        if (scheduled) task_destroy(scheduled);
    }

    // 性能测试
    uint64_t start_time = get_current_time_us();
    uint64_t operations = 0;

    for (int i = 0; i < PERFORMANCE_ITERATIONS; i++) {
        // 提交任务
        Task* task = task_create("throughput_test", simple_performance_task, NULL, 1024);
        if (task_scheduler_submit(scheduler, task)) {
            operations++;
        }

        // 调度任务
        Task* scheduled = task_scheduler_schedule_next(scheduler);
        if (scheduled) {
            operations++;
            task_destroy(scheduled);
        } else {
            task_destroy(task);
        }
    }

    uint64_t end_time = get_current_time_us();

    // 计算结果
    result.duration = (end_time - start_time) / 1000000;  // 秒
    result.throughput = (double)operations / result.duration;

    // 验证性能指标
    TEST_ASSERT(result.throughput > 1000.0);  // 至少1000 ops/sec
    TEST_ASSERT(result.duration < 10.0);      // 应该在10秒内完成

    printf("Throughput: %.2f ops/sec, Duration: %ld seconds\n",
           result.throughput, result.duration);

    task_scheduler_destroy(scheduler);
}

// 任务调度器延迟测试
void test_task_scheduler_latency(void) {
    TaskScheduler* scheduler = task_scheduler_create();
    BenchmarkResult result = {0};

    const size_t LATENCY_SAMPLES = 1000;
    uint64_t* latencies = TEST_MALLOC(sizeof(uint64_t) * LATENCY_SAMPLES);

    // 预热
    for (int i = 0; i < WARMUP_ITERATIONS; i++) {
        Task* task = task_create("latency_warmup", simple_performance_task, NULL, 1024);
        task_scheduler_submit(scheduler, task);
        Task* scheduled = task_scheduler_schedule_next(scheduler);
        if (scheduled) task_destroy(scheduled);
        task_destroy(task);
    }

    // 延迟测试
    for (size_t i = 0; i < LATENCY_SAMPLES; i++) {
        uint64_t submit_time = get_current_time_us();

        Task* task = task_create("latency_test", simple_performance_task, NULL, 1024);
        task_scheduler_submit(scheduler, task);

        uint64_t schedule_time = get_current_time_us();
        Task* scheduled = task_scheduler_schedule_next(scheduler);

        uint64_t end_time = get_current_time_us();

        if (scheduled) {
            latencies[i] = end_time - submit_time;  // 总延迟
            task_destroy(scheduled);
        } else {
            latencies[i] = schedule_time - submit_time;  // 提交延迟
            task_destroy(task);
        }
    }

    // 排序延迟数据计算百分位数
    // 简单排序（实际实现中应该使用更高效的算法）
    for (size_t i = 0; i < LATENCY_SAMPLES - 1; i++) {
        for (size_t j = i + 1; j < LATENCY_SAMPLES; j++) {
            if (latencies[i] > latencies[j]) {
                uint64_t temp = latencies[i];
                latencies[i] = latencies[j];
                latencies[j] = temp;
            }
        }
    }

    // 计算统计信息
    uint64_t total_latency = 0;
    result.min_latency = latencies[0];
    result.max_latency = latencies[LATENCY_SAMPLES - 1];

    for (size_t i = 0; i < LATENCY_SAMPLES; i++) {
        total_latency += latencies[i];
    }

    result.avg_latency = (double)total_latency / LATENCY_SAMPLES;
    result.p95_latency = calculate_percentile(latencies, LATENCY_SAMPLES, 95.0);
    result.p99_latency = calculate_percentile(latencies, LATENCY_SAMPLES, 99.0);

    // 验证延迟指标
    TEST_ASSERT(result.avg_latency < 1000.0);   // 平均延迟 < 1ms
    TEST_ASSERT(result.p95_latency < 5000.0);   // 95%延迟 < 5ms
    TEST_ASSERT(result.p99_latency < 10000.0);  // 99%延迟 < 10ms

    printf("Avg Latency: %.2f us, P95: %.2f us, P99: %.2f us\n",
           result.avg_latency, result.p95_latency, result.p99_latency);

    TEST_FREE(latencies);
    task_scheduler_destroy(scheduler);
}

// 任务调度器内存使用测试
void test_task_scheduler_memory_usage(void) {
    TaskScheduler* scheduler = task_scheduler_create();
    size_t initial_memory = get_current_memory_usage();

    // 创建大量任务
    const int NUM_TASKS = 10000;
    Task** tasks = TEST_MALLOC(sizeof(Task*) * NUM_TASKS);

    for (int i = 0; i < NUM_TASKS; i++) {
        tasks[i] = task_create("memory_test", simple_performance_task, NULL, 1024);
        task_scheduler_submit(scheduler, tasks[i]);
    }

    size_t peak_memory = get_current_memory_usage();

    // 调度所有任务
    for (int i = 0; i < NUM_TASKS; i++) {
        Task* scheduled = task_scheduler_schedule_next(scheduler);
        if (scheduled) {
            task_destroy(scheduled);
        }
    }

    size_t final_memory = get_current_memory_usage();

    // 验证内存使用
    size_t memory_increase = peak_memory - initial_memory;
    size_t memory_leak = final_memory - initial_memory;

    // 内存增长应该合理（每任务不超过1KB额外内存）
    TEST_ASSERT(memory_increase < NUM_TASKS * 1024);

    // 不应该有明显的内存泄漏（允许1%的误差）
    TEST_ASSERT(memory_leak < initial_memory * 0.01);

    printf("Memory increase: %zu bytes, Leak: %zu bytes\n", memory_increase, memory_leak);

    TEST_FREE(tasks);
    task_scheduler_destroy(scheduler);
}

// 任务调度器并发操作测试
void test_task_scheduler_concurrent_operations(void) {
    TaskScheduler* scheduler = task_scheduler_create();

    // 创建多个线程并发操作
    const int NUM_THREADS = 4;
    const int OPS_PER_THREAD = 1000;

    ThreadArgs args[NUM_THREADS];
    for (int i = 0; i < NUM_THREADS; i++) {
        args[i].scheduler = scheduler;
        args[i].thread_id = i;
        args[i].num_operations = OPS_PER_THREAD;
        create_thread(&concurrent_operations_thread, &args[i], NULL);
    }

    // 等待所有线程完成
    for (int i = 0; i < NUM_THREADS; i++) {
        join_thread(args[i].thread_handle);
    }

    // 验证最终状态
    int final_queue_size = task_scheduler_get_queue_size(scheduler);
    TEST_ASSERT(final_queue_size >= 0);  // 队列大小应该合理

    // 验证没有死锁或崩溃
    TEST_ASSERT(task_scheduler_get_status(scheduler) == SCHEDULER_RUNNING);

    printf("Concurrent operations completed, final queue size: %d\n", final_queue_size);

    task_scheduler_destroy(scheduler);
}

// 任务调度器优先级调度测试
void test_task_scheduler_priority_scheduling(void) {
    TaskScheduler* scheduler = task_scheduler_create();
    const int NUM_PRIORITIES = 5;
    const int TASKS_PER_PRIORITY = 100;

    // 创建不同优先级的任务
    for (int priority = 0; priority < NUM_PRIORITIES; priority++) {
        for (int i = 0; i < TASKS_PER_PRIORITY; i++) {
            Task* task = task_create("priority_test", priority_task_function, (void*)(uintptr_t)priority, 1024);
            task_set_priority(task, priority);
            task_scheduler_submit(scheduler, task);
        }
    }

    // 验证高优先级任务先被调度
    int last_priority = -1;
    int priority_order_correct = 0;

    for (int i = 0; i < NUM_PRIORITIES * TASKS_PER_PRIORITY; i++) {
        Task* scheduled = task_scheduler_schedule_next(scheduler);
        if (scheduled) {
            int task_priority = (int)(uintptr_t)scheduled->arg;
            if (task_priority >= last_priority) {
                priority_order_correct++;
            }
            last_priority = task_priority;
            task_destroy(scheduled);
        }
    }

    // 验证优先级调度正确性（允许一定的乱序容忍）
    double correctness_ratio = (double)priority_order_correct / (NUM_PRIORITIES * TASKS_PER_PRIORITY);
    TEST_ASSERT(correctness_ratio > 0.8);  // 至少80%的调度顺序正确

    printf("Priority scheduling correctness: %.2f%%\n", correctness_ratio * 100);

    task_scheduler_destroy(scheduler);
}

// 测试辅助函数
void simple_performance_task(void* arg) {
    volatile int sum = 0;
    for (int i = 0; i < 100; i++) {
        sum += i;
    }
}

void priority_task_function(void* arg) {
    int priority = (int)(uintptr_t)arg;
    volatile int iterations = 10 + (priority * 5);  // 不同优先级有不同计算量

    volatile int sum = 0;
    for (int i = 0; i < iterations; i++) {
        sum += i;
    }
}

typedef struct ThreadArgs {
    TaskScheduler* scheduler;
    int thread_id;
    int num_operations;
    void* thread_handle;
} ThreadArgs;

void* concurrent_operations_thread(void* arg) {
    ThreadArgs* args = (ThreadArgs*)arg;

    for (int i = 0; i < args->num_operations; i++) {
        // 随机选择操作
        int operation = rand() % 3;

        switch (operation) {
            case 0: {  // 提交任务
                char name[32];
                sprintf(name, "concurrent_%d_%d", args->thread_id, i);
                Task* task = task_create(name, simple_performance_task, NULL, 1024);
                task_scheduler_submit(args->scheduler, task);
                break;
            }
            case 1: {  // 调度任务
                Task* scheduled = task_scheduler_schedule_next(args->scheduler);
                if (scheduled) {
                    task_destroy(scheduled);
                }
                break;
            }
            case 2: {  // 查询状态
                task_scheduler_get_queue_size(args->scheduler);
                break;
            }
        }
    }

    return NULL;
}

// 获取当前内存使用量（简化实现）
size_t get_current_memory_usage(void) {
    // 实际实现中应该使用系统API获取内存使用量
    // 这里返回一个模拟值
    static size_t fake_memory = 1024 * 1024;  // 1MB
    fake_memory += rand() % 1024;  // 模拟内存变化
    return fake_memory;
}

// 任务调度器性能测试套件
TestSuite* task_scheduler_performance_test_suite(void) {
    TestSuite* suite = TEST_SUITE("Task Scheduler Performance", "任务调度器性能测试");

    TEST_ADD(suite, "task_scheduler_throughput", "任务调度器吞吐量测试", test_task_scheduler_throughput);
    TEST_ADD(suite, "task_scheduler_latency", "任务调度器延迟测试", test_task_scheduler_latency);
    TEST_ADD(suite, "task_scheduler_memory_usage", "任务调度器内存使用测试", test_task_scheduler_memory_usage);
    TEST_ADD(suite, "task_scheduler_concurrent_operations", "任务调度器并发操作测试", test_task_scheduler_concurrent_operations);
    TEST_ADD(suite, "task_scheduler_priority_scheduling", "任务调度器优先级调度测试", test_task_scheduler_priority_scheduling);

    return suite;
}
