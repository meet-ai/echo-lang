#include "network_client.h"
#include "../../shared/types/common_types.h"
#include <stdlib.h>
#include <stdio.h>
#include <string.h>
#include <unistd.h>
#include <sys/socket.h>
#include <netinet/in.h>
#include <arpa/inet.h>
#include <netdb.h>
#include <fcntl.h>
#include <errno.h>
#include <time.h>
#include <pthread.h>

// 内部实现结构体
typedef struct NetworkClientImpl {
    NetworkClient base;

    // 配置
    NetworkConfig config;

    // 连接池
    ConnectionPool* connection_pool;

    // 统计信息
    NetworkClientStats stats;

    // 同步原语
    pthread_mutex_t mutex;
    bool initialized;
} NetworkClientImpl;

// 连接结构体
typedef struct Connection {
    int sockfd;
    char host[256];
    int port;
    time_t created_at;
    time_t last_used_at;
    bool is_connected;
    uint64_t total_requests;
    uint64_t total_bytes_sent;
    uint64_t total_bytes_received;
} Connection;

// 连接池结构体
struct ConnectionPool {
    Connection** connections;
    size_t capacity;
    size_t size;
    pthread_mutex_t mutex;
};

// HTTP解析帮助函数
static int parse_http_response(const char* response, size_t response_len, NetworkResponse* network_response) {
    // 简单的HTTP响应解析
    const char* status_line_end = strstr(response, "\r\n");
    if (!status_line_end) return -1;

    char status_line[1024];
    size_t status_len = status_line_end - response;
    if (status_len >= sizeof(status_line)) return -1;

    memcpy(status_line, response, status_len);
    status_line[status_len] = '\0';

    // 解析状态码
    if (sscanf(status_line, "HTTP/%*s %d", &network_response->status_code) != 1) {
        return -1;
    }

    // 查找响应体开始位置
    const char* body_start = strstr(response, "\r\n\r\n");
    if (body_start) {
        body_start += 4;
        network_response->body_size = response_len - (body_start - response);
        network_response->body = malloc(network_response->body_size + 1);
        if (network_response->body) {
            memcpy(network_response->body, body_start, network_response->body_size);
            ((char*)network_response->body)[network_response->body_size] = '\0';
        }
    }

    return 0;
}

// DNS解析
static int resolve_hostname(const char* hostname, struct sockaddr_in* addr) {
    struct hostent* host = gethostbyname(hostname);
    if (!host) return -1;

    memset(addr, 0, sizeof(*addr));
    addr->sin_family = AF_INET;
    addr->sin_port = htons(80); // 默认HTTP端口
    memcpy(&addr->sin_addr, host->h_addr, host->h_length);

    return 0;
}

// 创建TCP连接
static int create_tcp_connection(const char* host, int port, int timeout_ms) {
    int sockfd = socket(AF_INET, SOCK_STREAM, 0);
    if (sockfd < 0) return -1;

    // 设置非阻塞
    int flags = fcntl(sockfd, F_GETFL, 0);
    fcntl(sockfd, F_SETFL, flags | O_NONBLOCK);

    struct sockaddr_in server_addr;
    memset(&server_addr, 0, sizeof(server_addr));
    server_addr.sin_family = AF_INET;
    server_addr.sin_port = htons(port);

    if (inet_pton(AF_INET, host, &server_addr.sin_addr) <= 0) {
        // 如果不是IP地址，尝试DNS解析
        if (resolve_hostname(host, &server_addr) != 0) {
            close(sockfd);
            return -1;
        }
    }

    // 尝试连接
    int result = connect(sockfd, (struct sockaddr*)&server_addr, sizeof(server_addr));
    if (result < 0 && errno != EINPROGRESS) {
        close(sockfd);
        return -1;
    }

    // 如果是异步连接，等待完成
    if (errno == EINPROGRESS) {
        fd_set writefds;
        struct timeval tv;
        FD_ZERO(&writefds);
        FD_SET(sockfd, &writefds);
        tv.tv_sec = timeout_ms / 1000;
        tv.tv_usec = (timeout_ms % 1000) * 1000;

        result = select(sockfd + 1, NULL, &writefds, NULL, &tv);
        if (result <= 0) {
            close(sockfd);
            return -1;
        }

        // 检查连接是否成功
        int error;
        socklen_t len = sizeof(error);
        getsockopt(sockfd, SOL_SOCKET, SO_ERROR, &error, &len);
        if (error != 0) {
            close(sockfd);
            return -1;
        }
    }

    // 设置回阻塞模式
    fcntl(sockfd, F_SETFL, flags);

    return sockfd;
}

// 发送HTTP请求
static int send_http_request(int sockfd, const NetworkRequest* request, const char* host) {
    char buffer[8192];
    int offset = 0;

    // 构建请求行
    offset += snprintf(buffer + offset, sizeof(buffer) - offset,
                      "%s %s HTTP/1.1\r\n", request->method, request->url);

    // Host头
    offset += snprintf(buffer + offset, sizeof(buffer) - offset,
                      "Host: %s\r\n", host);

    // User-Agent
    offset += snprintf(buffer + offset, sizeof(buffer) - offset,
                      "User-Agent: EchoRuntime/1.0\r\n");

    // Connection
    offset += snprintf(buffer + offset, sizeof(buffer) - offset,
                      "Connection: close\r\n");

    // Content-Length (如果有请求体)
    if (request->body && request->body_size > 0) {
        offset += snprintf(buffer + offset, sizeof(buffer) - offset,
                          "Content-Length: %zu\r\n", request->body_size);
    }

    // 结束头部
    offset += snprintf(buffer + offset, sizeof(buffer) - offset, "\r\n");

    // 发送请求头
    if (send(sockfd, buffer, offset, 0) < 0) {
        return -1;
    }

    // 发送请求体
    if (request->body && request->body_size > 0) {
        if (send(sockfd, request->body, request->body_size, 0) < 0) {
            return -1;
        }
    }

    return 0;
}

// 接收HTTP响应
static int receive_http_response(int sockfd, NetworkResponse* response, int timeout_ms) {
    char buffer[8192];
    size_t total_received = 0;
    time_t start_time = time(NULL);

    while (total_received < sizeof(buffer) - 1) {
        // 检查超时
        if (timeout_ms > 0) {
            time_t current_time = time(NULL);
            if ((current_time - start_time) * 1000 >= timeout_ms) {
                break;
            }
        }

        ssize_t received = recv(sockfd, buffer + total_received,
                               sizeof(buffer) - total_received - 1, 0);
        if (received < 0) {
            if (errno == EWOULDBLOCK || errno == EAGAIN) {
                // 暂时没有数据，继续等待
                usleep(1000); // 1ms
                continue;
            }
            return -1;
        } else if (received == 0) {
            // 连接关闭
            break;
        }

        total_received += received;

        // 检查是否收到完整的响应
        if (strstr(buffer, "\r\n\r\n")) {
            break;
        }
    }

    buffer[total_received] = '\0';

    // 解析响应
    if (parse_http_response(buffer, total_received, response) != 0) {
        return -1;
    }

    response->response_time = time(NULL);
    response->total_time_ms = (time_t)(time(NULL) - start_time) * 1000;

    return 0;
}

// 创建网络客户端
NetworkClient* network_client_create(const NetworkConfig* config) {
    NetworkClientImpl* impl = (NetworkClientImpl*)malloc(sizeof(NetworkClientImpl));
    if (!impl) {
        fprintf(stderr, "ERROR: Failed to allocate memory for network client\n");
        return NULL;
    }

    memset(impl, 0, sizeof(NetworkClientImpl));

    // 设置默认配置
    if (config) {
        memcpy(&impl->config, config, sizeof(NetworkConfig));
    } else {
        impl->config.max_connections = 100;
        impl->config.max_connections_per_host = 10;
        impl->config.connection_timeout_ms = 5000;
        impl->config.read_timeout_ms = 10000;
        impl->config.write_timeout_ms = 5000;
        impl->config.keep_alive = true;
        impl->config.keep_alive_timeout_ms = 30000;
        impl->config.max_retries = 3;
        impl->config.retry_delay_ms = 1000;
        impl->config.enable_ssl = false;
        impl->config.enable_compression = true;
        impl->config.enable_metrics = true;
    }

    // 初始化连接池
    impl->connection_pool = (ConnectionPool*)malloc(sizeof(ConnectionPool));
    if (impl->connection_pool) {
        impl->connection_pool->capacity = impl->config.max_connections;
        impl->connection_pool->connections = (Connection**)calloc(impl->connection_pool->capacity, sizeof(Connection*));
        impl->connection_pool->size = 0;
        pthread_mutex_init(&impl->connection_pool->mutex, NULL);
    }

    // 初始化统计信息
    memset(&impl->stats, 0, sizeof(NetworkClientStats));
    impl->stats.created_at = time(NULL);

    // 初始化同步原语
    pthread_mutex_init(&impl->mutex, NULL);

    impl->initialized = true;

    return (NetworkClient*)impl;
}

// 销毁网络客户端
void network_client_destroy(NetworkClient* client) {
    if (!client) return;

    NetworkClientImpl* impl = (NetworkClientImpl*)client;

    if (impl->initialized) {
        // 清理连接池
        if (impl->connection_pool) {
            pthread_mutex_lock(&impl->connection_pool->mutex);
            for (size_t i = 0; i < impl->connection_pool->size; i++) {
                Connection* conn = impl->connection_pool->connections[i];
                if (conn && conn->is_connected) {
                    close(conn->sockfd);
                }
                free(conn);
            }
            free(impl->connection_pool->connections);
            pthread_mutex_unlock(&impl->connection_pool->mutex);
            pthread_mutex_destroy(&impl->connection_pool->mutex);
            free(impl->connection_pool);
        }

        pthread_mutex_destroy(&impl->mutex);
    }

    free(impl);
}

// 发送HTTP请求
NetworkResponse* network_client_send_request(NetworkClient* client, const NetworkRequest* request) {
    if (!client || !request || !((NetworkClientImpl*)client)->initialized) {
        return NULL;
    }

    NetworkClientImpl* impl = (NetworkClientImpl*)client;

    NetworkResponse* response = (NetworkResponse*)malloc(sizeof(NetworkResponse));
    if (!response) {
        return NULL;
    }

    memset(response, 0, sizeof(NetworkResponse));

    // 解析URL
    char host[256] = {0};
    char path[2048] = {0};
    int port = 80;

    // 简单的URL解析 (只支持http://)
    if (sscanf(request->url, "http://%255[^:/]:%d/%2047s", host, &port, path) < 2) {
        if (sscanf(request->url, "http://%255[^/]/%2047s", host, path) < 1) {
            strcpy(host, "localhost");
            strcpy(path, "/");
        }
    }

    // 创建连接
    int sockfd = create_tcp_connection(host, port, impl->config.connection_timeout_ms);
    if (sockfd < 0) {
        strcpy(response->error_message, "Failed to create connection");
        return response;
    }

    // 发送请求
    if (send_http_request(sockfd, request, host) != 0) {
        strcpy(response->error_message, "Failed to send request");
        close(sockfd);
        return response;
    }

    // 接收响应
    if (receive_http_response(sockfd, response, impl->config.read_timeout_ms) != 0) {
        strcpy(response->error_message, "Failed to receive response");
    }

    // 关闭连接
    close(sockfd);

    // 更新统计信息
    pthread_mutex_lock(&impl->mutex);
    impl->stats.total_requests++;
    if (response->status_code >= 200 && response->status_code < 300) {
        impl->stats.successful_requests++;
    } else {
        impl->stats.failed_requests++;
    }
    impl->stats.total_response_time_ms += response->total_time_ms;
    if (response->body_size > 0) {
        impl->stats.total_bytes_received += response->body_size;
    }
    pthread_mutex_unlock(&impl->mutex);

    return response;
}

// 异步发送HTTP请求
void network_client_send_request_async(NetworkClient* client, const NetworkRequest* request,
                                      void (*callback)(NetworkResponse*, void*), void* user_data) {
    // 异步实现（暂时使用同步方式）
    NetworkResponse* response = network_client_send_request(client, request);
    if (callback) {
        callback(response, user_data);
    }
    network_response_destroy(response);
}

// 获取统计信息
bool network_client_get_stats(const NetworkClient* client, NetworkClientStats* stats) {
    if (!client || !stats) return false;

    const NetworkClientImpl* impl = (NetworkClientImpl*)client;

    pthread_mutex_lock(&impl->mutex);
    memcpy(stats, &impl->stats, sizeof(NetworkClientStats));
    pthread_mutex_unlock(&impl->mutex);

    return true;
}

// 清理连接池
void network_client_cleanup_connections(NetworkClient* client) {
    if (!client) return;

    NetworkClientImpl* impl = (NetworkClientImpl*)client;

    if (impl->connection_pool) {
        pthread_mutex_lock(&impl->connection_pool->mutex);
        time_t now = time(NULL);
        size_t i = 0;
        while (i < impl->connection_pool->size) {
            Connection* conn = impl->connection_pool->connections[i];
            if (conn && (now - conn->last_used_at) > (impl->config.keep_alive_timeout_ms / 1000)) {
                // 连接超时，关闭并移除
                close(conn->sockfd);
                free(conn);
                // 移动后面的连接
                for (size_t j = i; j < impl->connection_pool->size - 1; j++) {
                    impl->connection_pool->connections[j] = impl->connection_pool->connections[j + 1];
                }
                impl->connection_pool->size--;
            } else {
                i++;
            }
        }
        pthread_mutex_unlock(&impl->connection_pool->mutex);
    }
}

// 创建网络请求
NetworkRequest* network_request_create(void) {
    NetworkRequest* request = (NetworkRequest*)malloc(sizeof(NetworkRequest));
    if (request) {
        memset(request, 0, sizeof(NetworkRequest));
        strcpy(request->method, "GET");
        request->timeout_ms = 10000;
        request->connect_timeout_ms = 5000;
        request->read_timeout_ms = 10000;
        request->follow_redirects = true;
        request->max_redirects = 5;
    }
    return request;
}

// 销毁网络请求
void network_request_destroy(NetworkRequest* request) {
    if (request) {
        free(request->headers);
        free(request->body);
        free(request->proxy_url);
        free(request);
    }
}

// 创建网络响应
NetworkResponse* network_response_create(void) {
    NetworkResponse* response = (NetworkResponse*)malloc(sizeof(NetworkResponse));
    if (response) {
        memset(response, 0, sizeof(NetworkResponse));
        response->response_time = time(NULL);
    }
    return response;
}

// 销毁网络响应
void network_response_destroy(NetworkResponse* response) {
    if (response) {
        free(response->headers);
        free(response->body);
        free(response->redirect_url);
        free(response);
    }
}

// 便捷的GET请求
NetworkResponse* network_client_get(NetworkClient* client, const char* url) {
    NetworkRequest* request = network_request_create();
    strcpy(request->method, "GET");
    strncpy(request->url, url, sizeof(request->url) - 1);

    NetworkResponse* response = network_client_send_request(client, request);

    network_request_destroy(request);
    return response;
}

// 便捷的POST请求
NetworkResponse* network_client_post(NetworkClient* client, const char* url, const void* data, size_t data_size) {
    NetworkRequest* request = network_request_create();
    strcpy(request->method, "POST");
    strncpy(request->url, url, sizeof(request->url) - 1);

    if (data && data_size > 0) {
        request->body = malloc(data_size);
        if (request->body) {
            memcpy(request->body, data, data_size);
            request->body_size = data_size;
        }
    }

    NetworkResponse* response = network_client_send_request(client, request);

    network_request_destroy(request);
    return response;
}

// 设置请求头
void network_request_set_header(NetworkRequest* request, const char* key, const char* value) {
    if (!request || !key || !value) return;

    // 简单的实现：将headers存储为JSON格式的字符串
    // 实际实现应该使用更高效的数据结构
    char header[512];
    snprintf(header, sizeof(header), "\"%s\": \"%s\"", key, value);

    if (request->headers) {
        char* new_headers = realloc(request->headers, strlen(request->headers) + strlen(header) + 3);
        if (new_headers) {
            request->headers = new_headers;
            strcat(request->headers, ", ");
            strcat(request->headers, header);
        }
    } else {
        request->headers = malloc(strlen(header) + 3);
        if (request->headers) {
            strcpy(request->headers, "{ ");
            strcat(request->headers, header);
            strcat(request->headers, " }");
        }
    }
}
