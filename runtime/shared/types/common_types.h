#ifndef COMMON_TYPES_H
#define COMMON_TYPES_H

#include <stdint.h>
#include <stdbool.h>
#include <stddef.h>

// ============================================================================
// 通用基础类型
// ============================================================================

// 时间相关类型
typedef uint64_t timestamp_t;        // 时间戳类型（毫秒）
typedef uint64_t duration_t;         // 时长类型（毫秒）
typedef struct {
    timestamp_t start;
    timestamp_t end;
} time_range_t;

// 标识符类型
typedef uint64_t entity_id_t;        // 实体ID
typedef uint32_t entity_version_t;   // 实体版本
typedef char echo_uuid_t[37];         // UUID字符串

// 字符串类型
typedef struct {
    char* data;
    size_t length;
    size_t capacity;
} string_t;

// 字节数组类型
typedef struct {
    uint8_t* data;
    size_t length;
    size_t capacity;
} byte_array_t;

// ============================================================================
// 通用状态枚举
// ============================================================================

// 通用状态
typedef enum {
    STATUS_UNKNOWN = 0,
    STATUS_INITIALIZING,
    STATUS_READY,
    STATUS_RUNNING,
    STATUS_PAUSED,
    STATUS_STOPPED,
    STATUS_ERROR,
    STATUS_DESTROYED
} common_status_t;

// 结果状态
typedef enum {
    RESULT_SUCCESS = 0,
    RESULT_PENDING,
    RESULT_FAILURE,
    RESULT_CANCELLED,
    RESULT_TIMEOUT
} result_status_t;

// 优先级枚举
typedef enum {
    PRIORITY_LOWEST = 0,
    PRIORITY_LOW = 25,
    PRIORITY_NORMAL = 50,
    PRIORITY_HIGH = 75,
    PRIORITY_HIGHEST = 100,
    PRIORITY_CRITICAL = 255
} priority_t;

// ============================================================================
// 通用数据结构
// ============================================================================

// 键值对
typedef struct {
    string_t key;
    string_t value;
} key_value_pair_t;

// 动态数组
typedef struct {
    void** data;
    size_t length;
    size_t capacity;
    size_t element_size;
} dynamic_array_t;

// Map键类型枚举（用于支持多种Map类型）
typedef enum {
    MAP_KEY_TYPE_STRING = 0,  // 字符串键
    MAP_KEY_TYPE_INT32,       // 32位整数键
    MAP_KEY_TYPE_INT64,       // 64位整数键
    MAP_KEY_TYPE_FLOAT,       // 浮点数键（double/f64）
    MAP_KEY_TYPE_BOOL,        // 布尔键（较少使用）
    MAP_KEY_TYPE_STRUCT,      // ✅ 新增：结构体键（需要Hash和Eq trait）
} map_key_type_t;

// Map值类型枚举（用于支持多种Map类型）
typedef enum {
    MAP_VALUE_TYPE_STRING = 0,  // 字符串值
    MAP_VALUE_TYPE_INT32,       // 32位整数值
    MAP_VALUE_TYPE_INT64,       // 64位整数值
    MAP_VALUE_TYPE_FLOAT,       // 浮点数值（double/f64）
    MAP_VALUE_TYPE_BOOL,        // 布尔值
    MAP_VALUE_TYPE_SLICE,       // ✅ 新增：切片/数组值（void*指针）
    MAP_VALUE_TYPE_MAP,         // ✅ 新增：嵌套Map值（void*指针，指向另一个Map）
    MAP_VALUE_TYPE_STRUCT,      // ✅ 新增：结构体值（void*指针，指向结构体实例）
} map_value_type_t;

// 哈希表条目（原有实现，用于map[string]string，保持向后兼容）
typedef struct hash_entry {
    string_t key;
    void* value;
    struct hash_entry* next;
} hash_entry_t;

// 哈希表（原有实现，用于map[string]string，保持向后兼容）
typedef struct {
    hash_entry_t** buckets;
    size_t bucket_count;
    size_t entry_count;
    uint32_t (*hash_function)(const char* key);
} hash_table_t;

// 哈希函数指针类型（用于结构体键）
typedef int64_t (*map_key_hash_func_t)(const void* key);

// 比较函数指针类型（用于结构体键）
typedef int (*map_key_equals_func_t)(const void* key1, const void* key2);

// 通用哈希表条目（支持多种键值类型）
typedef struct hash_entry_generic {
    void* key;                  // 键（通用指针）
    map_key_type_t key_type;    // 键类型标识
    size_t key_size;            // 键大小（用于内存管理）
    
    // ✅ 新增：函数指针（用于结构体键）
    map_key_hash_func_t key_hash_func;      // 哈希函数指针（结构体键使用）
    map_key_equals_func_t key_equals_func;  // 比较函数指针（结构体键使用）
    
    void* value;                // 值（通用指针）
    map_value_type_t value_type;// 值类型标识
    size_t value_size;          // 值大小（用于内存管理）
    
    struct hash_entry_generic* next;  // 链表指针（处理哈希冲突）
} hash_entry_generic_t;

// 通用哈希表（支持多种键值类型）
typedef struct {
    hash_entry_generic_t** buckets;   // 桶数组
    size_t bucket_count;               // 桶数量
    size_t entry_count;                // 条目数量
    map_key_type_t key_type;          // 键类型（所有条目共享）
    map_value_type_t value_type;      // 值类型（所有条目共享）
} hash_table_generic_t;

// 队列节点
typedef struct queue_node {
    void* data;
    struct queue_node* next;
    struct queue_node* prev;
} queue_node_t;

// 队列
typedef struct {
    queue_node_t* head;
    queue_node_t* tail;
    size_t length;
    size_t max_length;
} queue_t;

// 栈
typedef struct {
    void** data;
    size_t top;
    size_t capacity;
} echo_stack_t;

// ============================================================================
// 资源管理类型
// ============================================================================

// 资源统计
typedef struct {
    size_t total_allocated;
    size_t current_used;
    size_t peak_used;
    uint64_t allocation_count;
    uint64_t deallocation_count;
} resource_stats_t;

// 内存块
typedef struct {
    void* data;
    size_t size;
    bool is_free;
    const char* allocation_source;
    timestamp_t allocation_time;
} memory_block_t;

// ============================================================================
// 事件系统类型
// ============================================================================

// 事件类型枚举（基础事件）
typedef enum {
    EVENT_NONE = 0,
    EVENT_SYSTEM_START,
    EVENT_SYSTEM_STOP,
    EVENT_ERROR_OCCURRED,
    EVENT_WARNING_RAISED,
    EVENT_INFO_MESSAGE,
    EVENT_DEBUG_MESSAGE
} system_event_type_t;

// 事件数据
typedef struct {
    system_event_type_t type;
    timestamp_t timestamp;
    entity_id_t source_id;
    string_t message;
    void* context_data;
    size_t context_size;
} event_data_t;

// ============================================================================
// 错误处理类型
// ============================================================================

// 错误信息
typedef struct {
    int error_code;
    string_t error_message;
    string_t error_source;
    string_t stack_trace;
    timestamp_t timestamp;
} error_info_t;

// 错误上下文
typedef struct {
    error_info_t* errors;
    size_t error_count;
    size_t max_errors;
    bool propagate_errors;
} error_context_t;

// ============================================================================
// 配置相关类型
// ============================================================================

// 配置值类型
typedef enum {
    CONFIG_TYPE_BOOL,
    CONFIG_TYPE_INT32,
    CONFIG_TYPE_INT64,
    CONFIG_TYPE_UINT32,
    CONFIG_TYPE_UINT64,
    CONFIG_TYPE_FLOAT,
    CONFIG_TYPE_DOUBLE,
    CONFIG_TYPE_STRING,
    CONFIG_TYPE_ARRAY,
    CONFIG_TYPE_OBJECT
} config_value_type_t;

// 配置值
typedef union {
    bool bool_value;
    int32_t int32_value;
    int64_t int64_value;
    uint32_t uint32_value;
    uint64_t uint64_value;
    float float_value;
    double double_value;
    string_t string_value;
    dynamic_array_t array_value;
    hash_table_t object_value;
} config_value_t;

// 配置条目
typedef struct {
    string_t key;
    config_value_type_t type;
    config_value_t value;
    string_t description;
    bool is_required;
    bool is_readonly;
} config_entry_t;

// 配置集合
typedef struct {
    config_entry_t* entries;
    size_t entry_count;
    size_t max_entries;
    hash_table_t lookup_table;
} config_collection_t;

// ============================================================================
// 并发和同步类型
// ============================================================================

// 原子整数类型
typedef struct {
    volatile int32_t value;
} atomic_int32_t;

typedef struct {
    volatile int64_t value;
} atomic_int64_t;

// 自旋锁
typedef struct {
    volatile int32_t locked;
} spin_lock_t;

// ============================================================================
// 工具宏
// ============================================================================

// 数组大小
#define ARRAY_SIZE(arr) (sizeof(arr) / sizeof((arr)[0]))

// 容器初始化
#define DYNAMIC_ARRAY_INIT { NULL, 0, 0, 0 }
#define HASH_TABLE_INIT { NULL, 0, 0, NULL }
#define QUEUE_INIT { NULL, NULL, 0, 0 }
#define STACK_INIT { NULL, 0, 0 }

// 字符串操作
#define STRING_INIT { NULL, 0, 0 }
#define STRING_LITERAL(str) { (char*)str, strlen(str), strlen(str) + 1 }

// 内存操作宏
#define ZERO_STRUCT(ptr) memset(ptr, 0, sizeof(*(ptr)))
#define COPY_STRUCT(dst, src) memcpy(dst, src, sizeof(*(dst)))

// 错误处理宏
#define RETURN_IF_ERROR(expr) do { \
    int _err = (expr); \
    if (_err != RESULT_SUCCESS) { \
        return _err; \
    } \
} while(0)

#define RETURN_NULL_IF_ERROR(expr) do { \
    int _err = (expr); \
    if (_err != RESULT_SUCCESS) { \
        return NULL; \
    } \
} while(0)

#define RETURN_FALSE_IF_ERROR(expr) do { \
    int _err = (expr); \
    if (_err != RESULT_SUCCESS) { \
        return false; \
    } \
} while(0)

// ============================================================================
// 函数指针类型
// ============================================================================

typedef void (*cleanup_function_t)(void* data);
typedef int (*compare_function_t)(const void* a, const void* b);
typedef uint32_t (*hash_function_t)(const char* key);
typedef void (*event_handler_t)(const event_data_t* event, void* user_data);
typedef void (*error_handler_t)(const error_info_t* error, void* user_data);
typedef bool (*validator_function_t)(const void* data, void* context);

// ============================================================================
// 常量定义
// ============================================================================

#define MAX_STRING_LENGTH (1024 * 1024)    // 最大字符串长度（1MB）
#define MAX_ARRAY_SIZE (1024 * 1024)       // 最大数组大小
#define DEFAULT_HASH_BUCKET_COUNT 1024     // 默认哈希桶数量
#define DEFAULT_QUEUE_MAX_LENGTH 10000     // 默认队列最大长度
#define DEFAULT_STACK_CAPACITY 1024        // 默认栈容量
#define DEFAULT_ERROR_BUFFER_SIZE 256      // 默认错误缓冲区大小

#endif // COMMON_TYPES_H
