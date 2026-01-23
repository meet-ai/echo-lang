#ifndef EXTERNAL_SERVICE_ADAPTER_H
#define EXTERNAL_SERVICE_ADAPTER_H

#include <stdint.h>
#include <stdbool.h>

// 前向声明
struct ExternalServiceAdapter;
struct HttpRequest;
struct HttpResponse;
struct DatabaseConnection;
struct MessageQueue;

// HTTP请求结构
typedef struct HttpRequest {
    char method[16];              // 请求方法 (GET, POST, etc.)
    char url[2048];              // 请求URL
    char* headers;               // 请求头 (JSON格式)
    void* body;                  // 请求体
    size_t body_size;            // 请求体大小
    uint32_t timeout_ms;         // 超时时间
} HttpRequest;

// HTTP响应结构
typedef struct HttpResponse {
    int status_code;             // 状态码
    char* headers;               // 响应头 (JSON格式)
    void* body;                  // 响应体
    size_t body_size;            // 响应体大小
    char error_message[512];     // 错误信息
} HttpResponse;

// 数据库连接配置
typedef struct DatabaseConfig {
    char host[256];              // 主机地址
    uint16_t port;               // 端口号
    char database[128];          // 数据库名
    char username[128];          // 用户名
    char password[128];          // 密码
    uint32_t connection_timeout_ms; // 连接超时
    uint32_t query_timeout_ms;   // 查询超时
    uint32_t max_connections;    // 最大连接数
} DatabaseConfig;

// 消息队列配置
typedef struct MessageQueueConfig {
    char broker_url[512];        // 消息代理URL
    char topic[256];             // 主题/队列名
    char client_id[128];         // 客户端ID
    uint32_t connection_timeout_ms; // 连接超时
    bool persistent;             // 是否持久化
    uint32_t ack_timeout_ms;     // 确认超时
} MessageQueueConfig;

// 外部服务适配器接口 - 统一外部系统访问
typedef struct {
    // HTTP客户端适配
    bool (*http_request)(struct ExternalServiceAdapter* adapter,
                        const HttpRequest* request,
                        HttpResponse* response);

    // 数据库适配
    bool (*db_connect)(struct ExternalServiceAdapter* adapter,
                      const DatabaseConfig* config,
                      void** connection_handle);
    bool (*db_disconnect)(struct ExternalServiceAdapter* adapter, void* connection_handle);
    bool (*db_execute_query)(struct ExternalServiceAdapter* adapter,
                           void* connection_handle,
                           const char* query,
                           void** result_handle);
    bool (*db_execute_update)(struct ExternalServiceAdapter* adapter,
                            void* connection_handle,
                            const char* query,
                            uint64_t* affected_rows);
    bool (*db_begin_transaction)(struct ExternalServiceAdapter* adapter, void* connection_handle);
    bool (*db_commit_transaction)(struct ExternalServiceAdapter* adapter, void* connection_handle);
    bool (*db_rollback_transaction)(struct ExternalServiceAdapter* adapter, void* connection_handle);

    // 消息队列适配
    bool (*mq_connect)(struct ExternalServiceAdapter* adapter,
                      const MessageQueueConfig* config,
                      void** connection_handle);
    bool (*mq_disconnect)(struct ExternalServiceAdapter* adapter, void* connection_handle);
    bool (*mq_publish)(struct ExternalServiceAdapter* adapter,
                      void* connection_handle,
                      const char* message,
                      size_t message_size,
                      const char* routing_key);
    bool (*mq_subscribe)(struct ExternalServiceAdapter* adapter,
                        void* connection_handle,
                        void (*message_handler)(const char* message, size_t size, void* user_data),
                        void* user_data);

    // 文件存储适配 (S3, GCS, etc.)
    bool (*file_upload)(struct ExternalServiceAdapter* adapter,
                       const char* bucket_name,
                       const char* key,
                       const void* data,
                       size_t data_size,
                       char* result_url,
                       size_t url_size);
    bool (*file_download)(struct ExternalServiceAdapter* adapter,
                         const char* bucket_name,
                         const char* key,
                         void** data,
                         size_t* data_size);
    bool (*file_delete)(struct ExternalServiceAdapter* adapter,
                       const char* bucket_name,
                       const char* key);
    bool (*file_list)(struct ExternalServiceAdapter* adapter,
                     const char* bucket_name,
                     const char* prefix,
                     char*** keys,
                     size_t* count);

    // 缓存适配 (Redis, Memcached, etc.)
    bool (*cache_connect)(struct ExternalServiceAdapter* adapter,
                         const char* connection_string,
                         void** connection_handle);
    bool (*cache_disconnect)(struct ExternalServiceAdapter* adapter, void* connection_handle);
    bool (*cache_set)(struct ExternalServiceAdapter* adapter,
                     void* connection_handle,
                     const char* key,
                     const void* value,
                     size_t value_size,
                     uint32_t ttl_seconds);
    bool (*cache_get)(struct ExternalServiceAdapter* adapter,
                     void* connection_handle,
                     const char* key,
                     void** value,
                     size_t* value_size);
    bool (*cache_delete)(struct ExternalServiceAdapter* adapter,
                        void* connection_handle,
                        const char* key);
    bool (*cache_exists)(struct ExternalServiceAdapter* adapter,
                        void* connection_handle,
                        const char* key);

    // 监控和日志适配
    bool (*log_send)(struct ExternalServiceAdapter* adapter,
                    const char* level,
                    const char* message,
                    const char* component,
                    const char* metadata_json);
    bool (*metrics_send)(struct ExternalServiceAdapter* adapter,
                        const char* metric_name,
                        double value,
                        const char* tags_json,
                        uint64_t timestamp);

    // 配置服务适配
    bool (*config_get)(struct ExternalServiceAdapter* adapter,
                      const char* key,
                      char* value,
                      size_t value_size);
    bool (*config_set)(struct ExternalServiceAdapter* adapter,
                      const char* key,
                      const char* value);
    bool (*config_watch)(struct ExternalServiceAdapter* adapter,
                        const char* key,
                        void (*callback)(const char* key, const char* value, void* user_data),
                        void* user_data);
} ExternalServiceAdapterInterface;

// 外部服务适配器基类
typedef struct ExternalServiceAdapter {
    ExternalServiceAdapterInterface* vtable;  // 虚函数表

    // 适配器状态
    bool initialized;
    char adapter_name[128];
    char adapter_version[32];

    // 连接池和资源管理
    void* connection_pool;
    uint32_t active_connections;
    uint32_t max_connections;

    // 错误处理
    char last_error[1024];
    int last_error_code;
} ExternalServiceAdapter;

// 外部服务适配器工厂函数
ExternalServiceAdapter* create_external_service_adapter(void);
void destroy_external_service_adapter(ExternalServiceAdapter* adapter);

// 便捷构造函数
HttpRequest* http_request_create(const char* method, const char* url, const void* body, size_t body_size);
void http_request_destroy(HttpRequest* request);

HttpResponse* http_response_create(void);
void http_response_destroy(HttpResponse* response);

// 便捷函数（直接调用虚函数表）
static inline bool external_service_adapter_http_request(
    ExternalServiceAdapter* adapter, const HttpRequest* request, HttpResponse* response) {
    return adapter->vtable->http_request(adapter, request, response);
}

static inline bool external_service_adapter_db_connect(
    ExternalServiceAdapter* adapter, const DatabaseConfig* config, void** connection_handle) {
    return adapter->vtable->db_connect(adapter, config, connection_handle);
}

static inline bool external_service_adapter_mq_publish(
    ExternalServiceAdapter* adapter, void* connection_handle,
    const char* message, size_t message_size, const char* routing_key) {
    return adapter->vtable->mq_publish(adapter, connection_handle, message, message_size, routing_key);
}

#endif // EXTERNAL_SERVICE_ADAPTER_H
