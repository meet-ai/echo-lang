#include "result.h"
#include <stdio.h>
#include <stdlib.h>
#include <string.h>

// ============================================================================
// Result 创建函数实现
// ============================================================================

void* result_ok(void* value, size_t value_size) {
    if (!value && value_size > 0) {
        return NULL; // 无效参数
    }

    Result* result = (Result*)malloc(sizeof(Result));
    if (!result) {
        return NULL; // 内存分配失败
    }

    result->is_ok = true;
    result->value_size = value_size;

    // 如果值不为空，复制值
    if (value && value_size > 0) {
        result->value = malloc(value_size);
        if (!result->value) {
            free(result);
            return NULL; // 内存分配失败
        }
        memcpy(result->value, value, value_size);
    } else {
        result->value = NULL;
    }

    result->error_type[0] = '\0';

    return (void*)result;
}

void* result_err(const char* error_message) {
    if (!error_message) {
        return NULL; // 无效参数
    }

    Result* result = (Result*)malloc(sizeof(Result));
    if (!result) {
        return NULL; // 内存分配失败
    }

    result->is_ok = false;
    result->value = NULL;
    result->value_size = 0;
    result->error_type[0] = '\0';

    // 将错误消息存储在 value 字段中（作为字符串）
    size_t msg_len = strlen(error_message);
    result->value = malloc(msg_len + 1);
    if (!result->value) {
        free(result);
        return NULL; // 内存分配失败
    }
    strcpy((char*)result->value, error_message);
    result->value_size = msg_len + 1;

    return (void*)result;
}

void* result_err_with_type(const char* error_type, const char* error_message) {
    if (!error_message) {
        return NULL; // 无效参数
    }

    Result* result = (Result*)malloc(sizeof(Result));
    if (!result) {
        return NULL; // 内存分配失败
    }

    result->is_ok = false;
    result->value = NULL;
    result->value_size = 0;

    // 存储错误类型
    if (error_type) {
        strncpy(result->error_type, error_type, sizeof(result->error_type) - 1);
        result->error_type[sizeof(result->error_type) - 1] = '\0';
    } else {
        result->error_type[0] = '\0';
    }

    // 将错误消息存储在 value 字段中（作为字符串）
    size_t msg_len = strlen(error_message);
    result->value = malloc(msg_len + 1);
    if (!result->value) {
        free(result);
        return NULL; // 内存分配失败
    }
    strcpy((char*)result->value, error_message);
    result->value_size = msg_len + 1;

    return (void*)result;
}

// ============================================================================
// Result 提取函数实现
// ============================================================================

bool result_is_ok(void* result) {
    if (!result) {
        return false;
    }

    Result* r = (Result*)result;
    return r->is_ok;
}

bool result_is_err(void* result) {
    if (!result) {
        return false;
    }

    Result* r = (Result*)result;
    return !r->is_ok;
}

bool result_unwrap_ok(void* result, void* value_buffer, size_t buffer_size, size_t* actual_size) {
    if (!result || !value_buffer || !actual_size) {
        return false;
    }

    Result* r = (Result*)result;
    if (!r->is_ok) {
        return false; // Result 是 Err
    }

    if (r->value_size > buffer_size) {
        return false; // 缓冲区太小
    }

    if (r->value && r->value_size > 0) {
        memcpy(value_buffer, r->value, r->value_size);
        *actual_size = r->value_size;
    } else {
        *actual_size = 0;
    }

    return true;
}

bool result_unwrap_err(void* result, char* error_buffer, size_t buffer_size) {
    if (!result || !error_buffer) {
        return false;
    }

    Result* r = (Result*)result;
    if (r->is_ok) {
        return false; // Result 是 Ok
    }

    if (r->value) {
        size_t msg_len = strlen((char*)r->value);
        if (msg_len >= buffer_size) {
            return false; // 缓冲区太小
        }
        strcpy(error_buffer, (char*)r->value);
    } else {
        if (buffer_size > 0) {
            error_buffer[0] = '\0';
        }
    }

    return true;
}

void result_free(void* result) {
    if (!result) {
        return;
    }

    Result* r = (Result*)result;
    if (r->value) {
        free(r->value);
    }
    free(r);
}

// ============================================================================
// Result 辅助函数实现
// ============================================================================

bool result_unwrap_or(void* result, void* default_value, size_t value_size, void* output_buffer) {
    if (!result || !output_buffer) {
        return false;
    }

    Result* r = (Result*)result;
    if (r->is_ok && r->value) {
        // 使用 Ok 值
        size_t copy_size = (r->value_size < value_size) ? r->value_size : value_size;
        memcpy(output_buffer, r->value, copy_size);
        return true;
    } else {
        // 使用默认值
        if (default_value) {
            memcpy(output_buffer, default_value, value_size);
        }
        return false;
    }
}

bool result_unwrap_or_else(void* result, void* value_buffer, size_t buffer_size, 
                           void (*error_callback)(const char*)) {
    if (!result || !value_buffer) {
        return false;
    }

    Result* r = (Result*)result;
    if (r->is_ok && r->value) {
        // 提取 Ok 值
        size_t copy_size = (r->value_size < buffer_size) ? r->value_size : buffer_size;
        memcpy(value_buffer, r->value, copy_size);
        return true;
    } else {
        // 调用错误回调
        if (error_callback && r->value) {
            error_callback((const char*)r->value);
        }
        return false;
    }
}
