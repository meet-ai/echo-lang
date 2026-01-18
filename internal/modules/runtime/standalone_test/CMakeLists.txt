cmake_minimum_required(VERSION 3.10)
project(industrial_memory_pool_standalone C)

# 设置C标准
set(CMAKE_C_STANDARD 11)
set(CMAKE_C_STANDARD_REQUIRED ON)

# 包含目录
include_directories(${CMAKE_SOURCE_DIR}/../include)
include_directories(${CMAKE_SOURCE_DIR}/../src/core)

# 编译选项
if(MSVC)
    add_compile_options(/W4)
else()
    add_compile_options(-Wall -Wextra -Wpedantic)
endif()

# 注意：原子操作使用stdatomic.h，编译器内置支持

# 内存池源文件
set(MEMORY_POOL_SOURCES
    ../src/core/industrial_memory_pool.c
    ../src/core/async_memory_pool.c
)

# 独立测试程序
add_executable(standalone_memory_pool_test
    ../tests/standalone_memory_pool_test.c
    ${MEMORY_POOL_SOURCES}
)

# 链接依赖
if(UNIX)
    target_link_libraries(standalone_memory_pool_test pthread)
endif()

# 启用测试
enable_testing()
add_test(NAME standalone_memory_pool_test COMMAND standalone_memory_pool_test)
