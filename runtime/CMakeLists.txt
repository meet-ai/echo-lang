# Echo Runtime - CMake Build System
cmake_minimum_required(VERSION 3.20)
project(echo-runtime C)

# Set C standard
set(CMAKE_C_STANDARD 11)
set(CMAKE_C_STANDARD_REQUIRED ON)

# Global build options
option(RUNTIME_ENABLE_GC "Enable garbage collection support" ON)
option(RUNTIME_ENABLE_COROUTINE "Enable coroutine support" ON)
option(RUNTIME_ENABLE_NETWORK "Enable network communication support" OFF)
option(RUNTIME_ENABLE_FILESYSTEM "Enable filesystem support" ON)
option(RUNTIME_ENABLE_DEBUG "Enable debug and monitoring support" ON)
option(RUNTIME_BUILD_TESTS "Build test executables" OFF)
option(RUNTIME_BUILD_EXAMPLES "Build example programs" OFF)
option(RUNTIME_BUILD_SHARED "Build shared libraries" OFF)

# Platform detection
if(CMAKE_SYSTEM_NAME STREQUAL "Linux")
    set(RUNTIME_PLATFORM "linux")
elseif(CMAKE_SYSTEM_NAME STREQUAL "Darwin")
    set(RUNTIME_PLATFORM "macos")
elseif(CMAKE_SYSTEM_NAME STREQUAL "Windows")
    set(RUNTIME_PLATFORM "windows")
else()
    set(RUNTIME_PLATFORM "unix")
endif()

message(STATUS "Building Echo Runtime for platform: ${RUNTIME_PLATFORM}")

# Global compile definitions
add_compile_definitions(
    RUNTIME_PLATFORM_${RUNTIME_PLATFORM}=1
)

if(RUNTIME_ENABLE_GC)
    add_compile_definitions(RUNTIME_ENABLE_GC=1)
endif()

if(RUNTIME_ENABLE_COROUTINE)
    add_compile_definitions(RUNTIME_ENABLE_COROUTINE=1)
endif()

# Include directories
include_directories(${CMAKE_CURRENT_SOURCE_DIR}/api)
include_directories(${CMAKE_CURRENT_SOURCE_DIR}/include)
include_directories(${CMAKE_CURRENT_SOURCE_DIR}/platform/interface)

# Platform-specific includes
if(RUNTIME_PLATFORM STREQUAL "linux" OR RUNTIME_PLATFORM STREQUAL "macos")
    include_directories(${CMAKE_CURRENT_SOURCE_DIR}/platform/posix)
endif()

# Find required packages
find_package(Threads REQUIRED)

# Collect all source files
file(GLOB_RECURSE RUNTIME_SOURCES
    "domain/**/*.c"
    "infrastructure/**/*.c"
    "shared/**/*.c"
    "api/*.c"
    "platform/**/*.c"
)

# Exclude incomplete modules for now
list(FILTER RUNTIME_SOURCES EXCLUDE REGEX ".*/communication/.*")
list(FILTER RUNTIME_SOURCES EXCLUDE REGEX ".*/concurrency/.*")
list(FILTER RUNTIME_SOURCES EXCLUDE REGEX ".*/coroutine/context\.c$") # 暂时排除有问题的context.c
list(FILTER RUNTIME_SOURCES EXCLUDE REGEX ".*/event/.*")
list(FILTER RUNTIME_SOURCES EXCLUDE REGEX ".*/executor/.*")
list(FILTER RUNTIME_SOURCES EXCLUDE REGEX ".*/future/.*")
list(FILTER RUNTIME_SOURCES EXCLUDE REGEX ".*/gc/.*")
list(FILTER RUNTIME_SOURCES EXCLUDE REGEX ".*/reactor/.*")
list(FILTER RUNTIME_SOURCES EXCLUDE REGEX ".*/scheduler/.*")
# list(FILTER RUNTIME_SOURCES EXCLUDE REGEX ".*/task/.*") # 已修复，重新包含
list(FILTER RUNTIME_SOURCES EXCLUDE REGEX ".*/infrastructure/.*")

# Create the main library
add_library(echo_runtime ${RUNTIME_SOURCES})

# Include directories
target_include_directories(echo_runtime PUBLIC
    ${CMAKE_CURRENT_SOURCE_DIR}/api
    ${CMAKE_CURRENT_SOURCE_DIR}/include
    ${CMAKE_CURRENT_SOURCE_DIR}/shared/types
    ${CMAKE_CURRENT_SOURCE_DIR}/shared/events
    ${CMAKE_CURRENT_SOURCE_DIR}/shared/exceptions
    ${CMAKE_CURRENT_SOURCE_DIR}/shared/monitoring
    ${CMAKE_CURRENT_SOURCE_DIR}/domain
    ${CMAKE_CURRENT_SOURCE_DIR}/infrastructure
    ${CMAKE_CURRENT_SOURCE_DIR}/application
    ${CMAKE_CURRENT_SOURCE_DIR}/platform/interface
)

# Link libraries
target_link_libraries(echo_runtime PUBLIC Threads::Threads)

# Tests
if(RUNTIME_BUILD_TESTS)
    enable_testing()

    # Build test executables directly
    file(GLOB TEST_SOURCES "tests/*.c")
    foreach(TEST_SRC ${TEST_SOURCES})
        get_filename_component(TEST_NAME ${TEST_SRC} NAME_WE)
        add_executable(${TEST_NAME} ${TEST_SRC})
        target_link_libraries(${TEST_NAME} echo_runtime)
        add_test(NAME ${TEST_NAME} COMMAND ${TEST_NAME})
    endforeach()

    # Build integration tests
    file(GLOB INTEGRATION_TEST_SOURCES "tests/integration-tests/*.c")
    foreach(TEST_SRC ${INTEGRATION_TEST_SOURCES})
        get_filename_component(TEST_NAME ${TEST_SRC} NAME_WE)
        add_executable(${TEST_NAME} ${TEST_SRC})
        target_link_libraries(${TEST_NAME} echo_runtime)
        add_test(NAME ${TEST_NAME} COMMAND ${TEST_NAME})
    endforeach()

    # Build performance tests
    file(GLOB PERF_TEST_SOURCES "tests/performance-tests/*.c")
    foreach(TEST_SRC ${PERF_TEST_SOURCES})
        get_filename_component(TEST_NAME ${TEST_SRC} NAME_WE)
        add_executable(${TEST_NAME} ${TEST_SRC})
        target_link_libraries(${TEST_NAME} echo_runtime)
        add_test(NAME ${TEST_NAME} COMMAND ${TEST_NAME})
    endforeach()
else()
    message(STATUS "Tests disabled")
endif()

# Examples
if(RUNTIME_BUILD_EXAMPLES)
    # Build example executables directly
    file(GLOB EXAMPLE_SOURCES "examples/*.c")
    foreach(EXAMPLE_SRC ${EXAMPLE_SOURCES})
        get_filename_component(EXAMPLE_NAME ${EXAMPLE_SRC} NAME_WE)
        add_executable(${EXAMPLE_NAME} ${EXAMPLE_SRC})
        target_link_libraries(${EXAMPLE_NAME} echo_runtime)
    endforeach()
else()
    message(STATUS "Examples disabled")
endif()

# Installation
install(DIRECTORY api/echo/
        DESTINATION include/echo
        FILES_MATCHING PATTERN "*.h")

# Generate compile_commands.json for IDE support
set(CMAKE_EXPORT_COMPILE_COMMANDS ON)