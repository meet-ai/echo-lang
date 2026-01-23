#ifndef TEST_FRAMEWORK_H
#define TEST_FRAMEWORK_H

#include <stdint.h>
#include <stdbool.h>
#include <time.h>
#include <setjmp.h>

// 前向声明
struct TestSuite;
struct TestCase;
struct TestResult;
struct TestRunner;
struct TestConfig;
struct TestStats;

// 测试结果状态
typedef enum {
    TEST_PASS,                  // 测试通过
    TEST_FAIL,                  // 测试失败
    TEST_SKIP,                  // 测试跳过
    TEST_ERROR,                 // 测试错误
    TEST_TIMEOUT,               // 测试超时
    TEST_CRASH                  // 测试崩溃
} TestStatus;

// 测试用例函数类型
typedef void (*TestFunction)(void);

// 测试套件
typedef struct TestSuite {
    char name[256];             // 套件名称
    char description[1024];     // 套件描述
    struct TestCase* tests;     // 测试用例数组
    size_t test_count;          // 测试用例数量
    size_t allocated_count;     // 已分配空间

    // 套件级设置
    void (*setup)(void);        // 套件初始化函数
    void (*teardown)(void);     // 套件清理函数
    uint32_t timeout_ms;        // 套件默认超时时间

    // 统计信息
    struct TestStats* stats;    // 套件统计
} TestSuite;

// 测试用例
typedef struct TestCase {
    char name[256];             // 测试用例名称
    char description[512];      // 测试用例描述
    TestFunction function;      // 测试函数
    uint32_t timeout_ms;        // 超时时间
    bool enabled;               // 是否启用
    char* tags;                 // 标签（逗号分隔）
    void* user_data;            // 用户数据

    // 测试结果
    struct TestResult* result;  // 测试结果
} TestCase;

// 测试结果
typedef struct TestResult {
    TestStatus status;          // 测试状态
    char message[2048];         // 结果消息
    time_t start_time;          // 开始时间
    time_t end_time;            // 结束时间
    uint64_t duration_ns;       // 执行时长（纳秒）
    char* output;               // 测试输出
    size_t output_size;         // 输出大小
    bool memory_leak_detected;  // 是否检测到内存泄漏
    uint64_t memory_used;       // 内存使用量
    char* stack_trace;          // 栈跟踪（失败时）
} TestResult;

// 测试统计
typedef struct TestStats {
    uint32_t total_tests;       // 总测试数
    uint32_t passed_tests;      // 通过测试数
    uint32_t failed_tests;      // 失败测试数
    uint32_t skipped_tests;     // 跳过测试数
    uint32_t error_tests;       // 错误测试数
    uint32_t timeout_tests;     // 超时测试数
    uint32_t crashed_tests;     // 崩溃测试数

    uint64_t total_duration_ns; // 总执行时长
    double average_duration_ns; // 平均执行时长
    uint64_t min_duration_ns;   // 最短执行时长
    uint64_t max_duration_ns;   // 最长执行时长

    uint64_t total_memory_used; // 总内存使用量
    uint64_t peak_memory_used;  // 峰值内存使用量

    time_t start_time;          // 开始时间
    time_t end_time;            // 结束时间
} TestStats;

// 测试配置
typedef struct TestConfig {
    bool verbose;               // 详细输出
    bool colored_output;        // 彩色输出
    char log_level[16];         // 日志级别
    char output_format[32];     // 输出格式 ("text", "json", "xml", "junit")
    char output_file[512];      // 输出文件
    bool parallel_execution;    // 并行执行
    uint32_t max_parallel_tests; // 最大并行测试数
    uint32_t timeout_ms;        // 默认超时时间
    bool memory_checking;       // 内存检查
    bool leak_detection;        // 泄漏检测
    char* include_patterns;     // 包含模式
    char* exclude_patterns;     // 排除模式
    char* include_tags;         // 包含标签
    char* exclude_tags;        // 排除标签
} TestConfig;

// 测试运行器
typedef struct TestRunner {
    struct TestConfig config;   // 运行器配置
    struct TestSuite** suites;  // 测试套件数组
    size_t suite_count;         // 套件数量

    // 运行状态
    bool running;               // 是否正在运行
    jmp_buf* jump_buffer;       // 异常处理跳转缓冲区
    void* current_test;         // 当前测试

    // 全局统计
    struct TestStats global_stats; // 全局统计

    // 回调函数
    void (*on_test_start)(const struct TestCase* test);
    void (*on_test_end)(const struct TestCase* test, const struct TestResult* result);
    void (*on_suite_start)(const struct TestSuite* suite);
    void (*on_suite_end)(const struct TestSuite* suite, const struct TestStats* stats);
} TestRunner;

// 测试断言宏
#define TEST_ASSERT(condition) \
    do { \
        if (!(condition)) { \
            test_framework_fail(__FILE__, __LINE__, "Assertion failed: " #condition); \
        } \
    } while (0)

#define TEST_ASSERT_EQUAL(expected, actual) \
    do { \
        if ((expected) != (actual)) { \
            test_framework_fail(__FILE__, __LINE__, \
                "Assertion failed: expected %d, but got %d", (expected), (actual)); \
        } \
    } while (0)

#define TEST_ASSERT_STR_EQUAL(expected, actual) \
    do { \
        if (strcmp((expected), (actual)) != 0) { \
            test_framework_fail(__FILE__, __LINE__, \
                "Assertion failed: expected '%s', but got '%s'", (expected), (actual)); \
        } \
    } while (0)

#define TEST_ASSERT_NULL(ptr) \
    do { \
        if ((ptr) != NULL) { \
            test_framework_fail(__FILE__, __LINE__, "Assertion failed: expected NULL, but got %p", (ptr)); \
        } \
    } while (0)

#define TEST_ASSERT_NOT_NULL(ptr) \
    do { \
        if ((ptr) == NULL) { \
            test_framework_fail(__FILE__, __LINE__, "Assertion failed: expected not NULL, but got NULL"); \
        } \
    } while (0)

// 测试框架函数
void test_framework_fail(const char* file, int line, const char* format, ...);

// 测试套件管理
TestSuite* test_suite_create(const char* name, const char* description);
void test_suite_destroy(TestSuite* suite);
bool test_suite_add_test(TestSuite* suite, const char* name, const char* description, TestFunction function);
bool test_suite_set_setup_teardown(TestSuite* suite, void (*setup)(void), void (*teardown)(void));

// 测试用例管理
TestCase* test_case_create(const char* name, const char* description, TestFunction function);
void test_case_destroy(TestCase* test_case);
bool test_case_set_timeout(TestCase* test_case, uint32_t timeout_ms);
bool test_case_add_tag(TestCase* test_case, const char* tag);

// 测试运行器管理
TestRunner* test_runner_create(const struct TestConfig* config);
void test_runner_destroy(TestRunner* runner);
bool test_runner_add_suite(TestRunner* runner, TestSuite* suite);
bool test_runner_run_all(TestRunner* runner);
bool test_runner_run_suite(TestRunner* runner, const char* suite_name);
bool test_runner_run_test(TestRunner* runner, const char* suite_name, const char* test_name);

// 配置管理
struct TestConfig* test_config_create(void);
void test_config_destroy(struct TestConfig* config);
struct TestConfig* test_config_create_default(void);

// 便利函数
static inline TestSuite* TEST_SUITE(const char* name, const char* description) {
    return test_suite_create(name, description);
}

static inline bool TEST_ADD(TestSuite* suite, const char* name, const char* description, TestFunction function) {
    return test_suite_add_test(suite, name, description, function);
}

static inline TestRunner* TEST_RUNNER(const struct TestConfig* config) {
    return test_runner_create(config);
}

static inline bool TEST_RUN(TestRunner* runner) {
    return test_runner_run_all(runner);
}

// 内存泄漏检测宏（如果启用）
#ifdef TEST_MEMORY_CHECKING
#define TEST_MALLOC(size) test_framework_malloc(size, __FILE__, __LINE__)
#define TEST_FREE(ptr) test_framework_free(ptr, __FILE__, __LINE__)
#define TEST_CALLOC(nmemb, size) test_framework_calloc(nmemb, size, __FILE__, __LINE__)
#define TEST_REALLOC(ptr, size) test_framework_realloc(ptr, size, __FILE__, __LINE__)
#else
#define TEST_MALLOC(size) malloc(size)
#define TEST_FREE(ptr) free(ptr)
#define TEST_CALLOC(nmemb, size) calloc(nmemb, size)
#define TEST_REALLOC(ptr, size) realloc(ptr, size)
#endif

// 内存检查函数（可选）
void* test_framework_malloc(size_t size, const char* file, int line);
void test_framework_free(void* ptr, const char* file, int line);
void* test_framework_calloc(size_t nmemb, size_t size, const char* file, int line);
void* test_framework_realloc(void* ptr, size_t size, const char* file, int line);

#endif // TEST_FRAMEWORK_H
