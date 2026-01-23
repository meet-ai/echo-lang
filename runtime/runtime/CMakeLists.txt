cmake_minimum_required(VERSION 3.10)
project(echo_runtime C)

# 设置C标准
set(CMAKE_C_STANDARD 11)
set(CMAKE_C_STANDARD_REQUIRED ON)

# 包含目录
include_directories(include)

# 编译选项
if(MSVC)
    add_compile_options(/W4)
else()
    add_compile_options(-Wall -Wextra -Wpedantic)
endif()

# 设置汇编文件编译
enable_language(ASM)
set(CMAKE_ASM_FLAGS "${CMAKE_ASM_FLAGS} -x assembler-with-cpp")

# 为不同架构设置汇编标志
if(CMAKE_SYSTEM_PROCESSOR MATCHES "x86_64|amd64")
    set(CMAKE_ASM_FLAGS "${CMAKE_ASM_FLAGS} -m64")
elseif(CMAKE_SYSTEM_PROCESSOR MATCHES "aarch64|arm64")
    set(CMAKE_ASM_FLAGS "${CMAKE_ASM_FLAGS} -march=armv8-a")
endif()

# 调试模式
if(CMAKE_BUILD_TYPE STREQUAL "Debug")
    add_compile_definitions(DEBUG)
endif()

# ============================================================================
# 核心运行时库
# ============================================================================

# 收集源文件
set(RUNTIME_SOURCES
    # 应用层
    infrastructure/core/runtime.c
    infrastructure/core/task.c
    infrastructure/core/future.c
    infrastructure/core/memory_pool.c
    # infrastructure/core/advanced_memory_pool.c  # 暂时注释掉有问题的文件
    # infrastructure/core/cache_utils.c          # 暂时注释掉有问题的文件

    # 工业级内存池系统
    infrastructure/core/industrial_memory_pool.c
    infrastructure/core/async_memory_pool.c

    # ============================================================================
    # GC 系统
    # ============================================================================

    # 核心GC上下文
    domain/gc/core-gc/core_gc.c
    domain/gc/core-gc/domain/aggregates/garbage_collector.c
    domain/gc/core-gc/domain/aggregates/heap.c
    domain/gc/core-gc/domain/services/mark_service.c
    domain/gc/core-gc/domain/services/sweep_service.c
    domain/gc/core-gc/application/commands/trigger_gc_command.c
    domain/gc/core-gc/application/services/gc_scheduler.c

    # 领域层 - 暂时简化，只包含必要的
    # domain/executor/executor.c
    # domain/reactor/reactor.c
    # domain/reactor/event_loop.c  # 有编译错误，暂时注释
    # domain/reactor/timer_future.c
    # domain/coroutine/coroutine.c
    # domain/coroutine/context.c
    # domain/coroutine/switcher.c
    # domain/coroutine/yield.c
    # domain/scheduler/scheduler.c
    # domain/scheduler/processor.c
    # domain/scheduler/machine.c
    # domain/channel/channel.c
    # domain/channel/select.c

    # 基础设施层 - 暂时简化
    # infrastructure/platform/context/context_platform.c
    # infrastructure/platform/event/platform_detector.c
    # $<$<PLATFORM_ID:Linux>:infrastructure/platform/event/posix/linux/epoll.c>
    # $<$<PLATFORM_ID:Linux>:infrastructure/platform/event/posix/linux/io_uring.c>
    # $<$<PLATFORM_ID:Windows>:infrastructure/platform/event/windows/iocp.c>
    # $<$<OR:$<PLATFORM_ID:Linux>,$<PLATFORM_ID:Darwin>,$<PLATFORM_ID:Windows>>:infrastructure/platform/event/posix/select.c>
)

# 平台特定源文件
if(APPLE)
    list(APPEND RUNTIME_SOURCES
        infrastructure/platform/event/posix/macos/kqueue.c
    )
    # macOS 通常是x86_64或aarch64
    if(CMAKE_SYSTEM_PROCESSOR MATCHES "x86_64|amd64")
        list(APPEND RUNTIME_SOURCES
            infrastructure/platform/context/x86_64/context_switch.S
            infrastructure/platform/context/x86_64/makecontext.c
        )
    elseif(CMAKE_SYSTEM_PROCESSOR MATCHES "aarch64|arm64")
        list(APPEND RUNTIME_SOURCES
            infrastructure/platform/context/aarch64/context_switch.S
            infrastructure/platform/context/aarch64/makecontext.c
        )
    endif()
elseif(UNIX AND NOT APPLE)
    # Linux
    if(CMAKE_SYSTEM_PROCESSOR MATCHES "x86_64|amd64")
        list(APPEND RUNTIME_SOURCES
            infrastructure/platform/context/x86_64/context_switch.S
            infrastructure/platform/context/x86_64/makecontext.c
        )
    elseif(CMAKE_SYSTEM_PROCESSOR MATCHES "aarch64|arm64")
        list(APPEND RUNTIME_SOURCES
            infrastructure/platform/context/aarch64/context_switch.S
            infrastructure/platform/context/aarch64/makecontext.c
        )
    endif()
elseif(WIN32)
    # Windows x86_64
    if(CMAKE_SYSTEM_PROCESSOR MATCHES "x86_64|amd64")
        list(APPEND RUNTIME_SOURCES
            infrastructure/platform/context/x86_64/context_switch.S
            infrastructure/platform/context/x86_64/makecontext.c
        )
    endif()
else()
    # 未知平台，使用setjmp/longjmp后备
    message(WARNING "Unknown platform, using setjmp/longjmp fallback for context switching")
endif()

# 创建运行时库
add_library(echo_runtime ${RUNTIME_SOURCES})

# 链接依赖
if(UNIX)
    target_link_libraries(echo_runtime pthread)
endif()

# ============================================================================
# 示例程序
# ============================================================================

# 异步编程示例
add_executable(async_await_example examples/async_await.c)
target_link_libraries(async_await_example echo_runtime)

# 工业级内存池基本使用示例
add_executable(basic_usage_example examples/basic_usage.c)
target_link_libraries(basic_usage_example echo_runtime)

# 性能对比示例
add_executable(performance_comparison_example examples/performance_comparison.c)
target_link_libraries(performance_comparison_example echo_runtime)

# ============================================================================
# 测试
# ============================================================================

# 启用测试
enable_testing()

# 简单的运行时测试
add_executable(runtime_test tests/runtime_test.c)
target_link_libraries(runtime_test echo_runtime)
add_test(NAME runtime_test COMMAND runtime_test)

# 高级内存池测试
add_executable(advanced_memory_pool_test tests/advanced_memory_pool_test.c)
target_link_libraries(advanced_memory_pool_test echo_runtime)
add_test(NAME advanced_memory_pool_test COMMAND advanced_memory_pool_test)

# 工业级内存池测试
add_executable(industrial_memory_pool_test tests/industrial_memory_pool_test.c)
target_link_libraries(industrial_memory_pool_test echo_runtime)
add_test(NAME industrial_memory_pool_test COMMAND industrial_memory_pool_test)

# 异步内存池测试
add_executable(async_memory_pool_test tests/async_memory_pool_test.c)
target_link_libraries(async_memory_pool_test echo_runtime)
add_test(NAME async_memory_pool_test COMMAND async_memory_pool_test)

# GC测试
add_executable(gc_test tests/gc_test.c)
target_link_libraries(gc_test echo_runtime)
add_test(NAME gc_test COMMAND gc_test)

# ============================================================================
# 安装
# ============================================================================

# 安装头文件
install(DIRECTORY include/ DESTINATION include)

# 安装库
install(TARGETS echo_runtime
    LIBRARY DESTINATION lib
    ARCHIVE DESTINATION lib)

# 安装示例
install(TARGETS async_await_example basic_usage_example performance_comparison_example
    RUNTIME DESTINATION bin)
