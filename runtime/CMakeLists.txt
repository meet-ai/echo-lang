# Echo Runtime - CMake Build System
cmake_minimum_required(VERSION 3.20)
project(echo-runtime C)

# Set C standard
set(CMAKE_C_STANDARD 11)
set(CMAKE_C_STANDARD_REQUIRED ON)

# Global build options
option(RUNTIME_ENABLE_GC "Enable garbage collection support" ON)
option(RUNTIME_ENABLE_COROUTINE "Enable coroutine support" ON)
option(RUNTIME_ENABLE_NETWORK "Enable network communication support" ON)
option(RUNTIME_ENABLE_FILESYSTEM "Enable filesystem support" ON)
option(RUNTIME_ENABLE_DEBUG "Enable debug and monitoring support" ON)
option(RUNTIME_BUILD_TESTS "Build test executables" ON)
option(RUNTIME_BUILD_EXAMPLES "Build example programs" ON)
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
include_directories(${CMAKE_CURRENT_SOURCE_DIR}/platform/interface)

# Platform-specific includes
if(RUNTIME_PLATFORM STREQUAL "linux" OR RUNTIME_PLATFORM STREQUAL "macos")
    include_directories(${CMAKE_CURRENT_SOURCE_DIR}/platform/posix)
endif()

# Find required packages
find_package(Threads REQUIRED)

# Core runtime (always included)
add_subdirectory(core)

# API layer
add_subdirectory(api)

# Bounded contexts (conditionally included)
if(RUNTIME_ENABLE_NETWORK)
    add_subdirectory(bounded-contexts/network)
endif()

if(RUNTIME_ENABLE_FILESYSTEM)
    add_subdirectory(bounded-contexts/filesystem)
endif()

add_subdirectory(bounded-contexts/channel)

if(RUNTIME_ENABLE_DEBUG)
    add_subdirectory(bounded-contexts/debug)
endif()

# Platform abstraction
add_subdirectory(platform)

# Tests
if(RUNTIME_BUILD_TESTS)
    enable_testing()
    add_subdirectory(tests)
endif()

# Examples
if(RUNTIME_BUILD_EXAMPLES)
    add_subdirectory(examples)
endif()

# Installation
install(DIRECTORY api/echo/
        DESTINATION include/echo
        FILES_MATCHING PATTERN "*.h")

# Generate compile_commands.json for IDE support
set(CMAKE_EXPORT_COMPILE_COMMANDS ON)