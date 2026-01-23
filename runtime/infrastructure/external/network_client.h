#ifndef NETWORK_CLIENT_H
#define NETWORK_CLIENT_H

#include <stdint.h>
#include <stdbool.h>
#include <time.h>

// 前向声明
struct NetworkClient;
struct NetworkRequest;
struct NetworkResponse;
struct NetworkConfig;
struct ConnectionPool;
struct ConnectionStats;

// 网络请求
typedef struct NetworkRequest {
    char method[16];             // 请求方法 ("GET", "POST", etc.)
    char url[2048];              // 请求URL
    char* headers;               // 请求头 (JSON格式)
    void* body;                  // 请求体
    size_t body_size;            // 请求体大小
    uint32_t timeout_ms;         // 请求超时时间
    uint32_t connect_timeout_ms; // 连接超时时间
    uint32_t read_timeout_ms;    // 读取超时时间
    bool follow_redirects;       // 是否跟随重定向
    uint32_t max_redirects;      // 最大重定向次数
    char* proxy_url;             // 代理URL
    void* user_data;             // 用户数据
} NetworkRequest;

// 网络响应
typedef struct NetworkResponse {
    int status_code;             // HTTP状态码
    char* headers;               // 响应头 (JSON格式)
    void* body;                  // 响应体
    size_t body_size;            // 响应体大小
    char content_type[256];      // 内容类型
    uint64_t content_length;     // 内容长度
    char* redirect_url;          // 重定向URL
    time_t response_time;        // 响应时间
    uint64_t total_time_ms;      // 总耗时
    char error_message[1024];    // 错误信息
} NetworkResponse;

// 网络配置
typedef struct NetworkConfig {
    uint32_t max_connections;    // 最大连接数
    uint32_t max_connections_per_host; // 每主机最大连接数
    uint32_t connection_timeout_ms; // 连接超时
    uint32_t request_timeout_ms; // 请求超时
    uint32_t keep_alive_timeout_ms; // 保活超时
    bool enable_http2;           // 是否启用HTTP/2
    bool enable_tls;             // 是否启用TLS
    char* tls_cert_file;         // TLS证书文件
    char* tls_key_file;          // TLS密钥文件
    char* ca_cert_file;          // CA证书文件
    bool skip_tls_verify;        // 是否跳过TLS验证
    char* proxy_url;             // 代理URL
    bool enable_metrics;         // 是否启用指标收集
    uint32_t buffer_size;        // 缓冲区大小
} NetworkConfig;

// 连接统计
typedef struct ConnectionStats {
    uint64_t total_requests;     // 总请求数
    uint64_t successful_requests; // 成功请求数
    uint64_t failed_requests;    // 失败请求数
    uint64_t timeout_requests;   // 超时请求数
    uint64_t active_connections; // 活跃连接数
    uint64_t idle_connections;   // 空闲连接数
    uint64_t total_connections_created; // 总创建连接数
    uint64_t bytes_sent;         // 发送字节数
    uint64_t bytes_received;     // 接收字节数
    double average_response_time_ms; // 平均响应时间
    time_t last_request_at;      // 最后请求时间
} ConnectionStats;

// 网络客户端接口
typedef struct {
    // 基本HTTP请求
    bool (*get)(struct NetworkClient* client, const char* url, struct NetworkResponse* response);
    bool (*post)(struct NetworkClient* client, const char* url, const void* data, size_t data_size,
                struct NetworkResponse* response);
    bool (*put)(struct NetworkClient* client, const char* url, const void* data, size_t data_size,
               struct NetworkResponse* response);
    bool (*delete)(struct NetworkClient* client, const char* url, struct NetworkResponse* response);
    bool (*patch)(struct NetworkClient* client, const char* url, const void* data, size_t data_size,
                 struct NetworkResponse* response);

    // 高级请求
    bool (*request)(struct NetworkClient* client, const struct NetworkRequest* request,
                   struct NetworkResponse* response);
    bool (*head)(struct NetworkClient* client, const char* url, struct NetworkResponse* response);

    // WebSocket支持
    bool (*websocket_connect)(struct NetworkClient* client, const char* url,
                             void (*message_handler)(const char* message, size_t size, void* user_data),
                             void* user_data, void** websocket_handle);
    bool (*websocket_send)(struct NetworkClient* client, void* websocket_handle,
                          const char* message, size_t message_size);
    bool (*websocket_close)(struct NetworkClient* client, void* websocket_handle);

    // 连接管理
    bool (*close_idle_connections)(struct NetworkClient* client);
    bool (*close_all_connections)(struct NetworkClient* client);

    // 统计和监控
    bool (*get_stats)(const struct NetworkClient* client, struct ConnectionStats* stats);
    bool (*reset_stats)(struct NetworkClient* client);

    // 配置管理
    bool (*reconfigure)(struct NetworkClient* client, const struct NetworkConfig* new_config);
    bool (*get_config)(const struct NetworkClient* client, struct NetworkConfig* config);

    // 下载和上传
    bool (*download_file)(struct NetworkClient* client, const char* url, const char* local_path);
    bool (*upload_file)(struct NetworkClient* client, const char* url, const char* local_path,
                       const char* field_name);

    // 异步请求支持
    bool (*async_request)(struct NetworkClient* client, const struct NetworkRequest* request,
                         void (*callback)(const struct NetworkResponse* response, void* user_data),
                         void* user_data, void** async_handle);
    bool (*cancel_async_request)(struct NetworkClient* client, void* async_handle);
} NetworkClientInterface;

// 网络客户端实现
typedef struct NetworkClient {
    NetworkClientInterface* vtable;  // 虚函数表

    // 配置
    struct NetworkConfig config;

    // 状态
    bool initialized;
    time_t created_at;
    char client_name[128];

    // 内部数据结构
    struct ConnectionPool* connection_pool;
    void* dns_resolver;
    void* tls_context;
    void* metrics_collector;

    // 同步原语
    void* mutex;

    // 统计
    struct ConnectionStats stats;
} NetworkClient;

// 网络客户端工厂函数
NetworkClient* network_client_create(const struct NetworkConfig* config);
void network_client_destroy(NetworkClient* client);

// 便捷构造函数
struct NetworkConfig* network_config_create(void);
void network_config_destroy(struct NetworkConfig* config);
struct NetworkRequest* network_request_create(const char* method, const char* url);
void network_request_destroy(struct NetworkRequest* request);
struct NetworkResponse* network_response_create(void);
void network_response_destroy(struct NetworkResponse* response);

// 预定义配置
struct NetworkConfig* network_config_create_default(void);
struct NetworkConfig* network_config_create_high_performance(void);
struct NetworkConfig* network_config_create_secure(void);

// 便捷函数（直接调用虚函数表）
static inline bool network_client_get(NetworkClient* client, const char* url, struct NetworkResponse* response) {
    return client->vtable->get(client, url, response);
}

static inline bool network_client_post(NetworkClient* client, const char* url,
                                      const void* data, size_t data_size, struct NetworkResponse* response) {
    return client->vtable->post(client, url, data, data_size, response);
}

static inline bool network_client_request(NetworkClient* client, const struct NetworkRequest* request,
                                         struct NetworkResponse* response) {
    return client->vtable->request(client, request, response);
}

static inline bool network_client_get_stats(const NetworkClient* client, struct ConnectionStats* stats) {
    return client->vtable->get_stats(client, stats);
}

// 全局网络客户端实例（可选）
extern NetworkClient* g_global_network_client;

// 全局便捷函数
static inline bool network_client_get_global(const char* url, struct NetworkResponse* response) {
    if (g_global_network_client) {
        return network_client_get(g_global_network_client, url, response);
    }
    return false;
}

#endif // NETWORK_CLIENT_H
