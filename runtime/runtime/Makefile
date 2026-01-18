# Echo Language Makefile

# Build directories
BUILD_DIR := build
ECHO_COMPILER := $(BUILD_DIR)/echoc
RUNTIME_LIB := $(BUILD_DIR)/libcoroutine.a
ASM_RUNTIME_LIB := $(BUILD_DIR)/libcoroutine_asm.a

# Compiler settings
GO := go
CLANG := clang -Iruntime/include -Iruntime/include/echo -Iruntime/domain/coroutine -Iruntime/domain/scheduler -Iruntime/domain/task -Iruntime/domain/future -Iruntime/domain/channel -Iruntime/domain

# Runtime library
RUNTIME_SRC := runtime/domain/runtime/echo_runtime.c runtime/domain/coroutine/coroutine_runtime.c
RUNTIME_SRC += runtime/domain/coroutine/context.c runtime/domain/coroutine/switcher.c runtime/domain/task/task.c runtime/domain/future/future.c runtime/domain/coroutine/coroutine.c runtime/domain/scheduler/processor.c runtime/domain/scheduler/scheduler.c runtime/domain/channel/channel.c runtime/domain/scheduler/machine.c

RUNTIME_OBJ := $(BUILD_DIR)/echo_runtime.o $(BUILD_DIR)/coroutine_runtime.o
RUNTIME_OBJ += $(BUILD_DIR)/context.o $(BUILD_DIR)/switcher.o
RUNTIME_OBJ += $(BUILD_DIR)/task.o $(BUILD_DIR)/future.o $(BUILD_DIR)/coroutine.o $(BUILD_DIR)/processor.o $(BUILD_DIR)/scheduler.o $(BUILD_DIR)/channel.o $(BUILD_DIR)/machine.o

.PHONY: all
all: build compile-examples

.PHONY: build-compiler
build-compiler:
	@echo "Building Echo compiler..."
	$(GO) build -o $(ECHO_COMPILER) ./cmd/main.go
	@echo "Echo compiler built successfully"

.PHONY: build-runtime
build-runtime: build/$(notdir $(RUNTIME_LIB))
	@echo "Runtime library built successfully"

$(BUILD_DIR)/libcoroutine.a: $(RUNTIME_OBJ)
	ar rcs $@ $^

$(BUILD_DIR)/%.o: runtime/%.c
	$(CLANG) -c $< -o $@ -lpthread

$(BUILD_DIR)/echo_runtime.o: runtime/domain/runtime/echo_runtime.c
	$(CLANG) -c $< -o $@ -lpthread

$(BUILD_DIR)/coroutine_runtime.o: runtime/domain/coroutine/coroutine_runtime.c
	$(CLANG) -c $< -o $@ -lpthread

$(BUILD_DIR)/context.o: runtime/domain/coroutine/context.c
	$(CLANG) -c $< -o $@ -lpthread

$(BUILD_DIR)/switcher.o: runtime/domain/coroutine/switcher.c
	$(CLANG) -c $< -o $@ -lpthread

$(BUILD_DIR)/task.o: runtime/domain/task/task.c
	$(CLANG) -c $< -o $@ -lpthread

$(BUILD_DIR)/future.o: runtime/domain/future/future.c
	$(CLANG) -c $< -o $@ -lpthread

$(BUILD_DIR)/coroutine.o: runtime/domain/coroutine/coroutine.c
	$(CLANG) -c $< -o $@ -lpthread

$(BUILD_DIR)/processor.o: runtime/domain/scheduler/processor.c
	$(CLANG) -c $< -o $@ -lpthread

$(BUILD_DIR)/scheduler.o: runtime/domain/scheduler/scheduler.c
	$(CLANG) -c $< -o $@ -lpthread

$(BUILD_DIR)/channel.o: runtime/domain/channel/channel.c
	$(CLANG) -c $< -o $@ -lpthread

$(BUILD_DIR)/machine.o: runtime/domain/scheduler/machine.c
	$(CLANG) -c $< -o $@ -lpthread

.PHONY: build
build: build-compiler

.PHONY: compile-file
compile-file: build-compiler build-runtime
	@if [ -z "$(FILE)" ]; then \
		echo "Usage: make compile-file FILE=path/to/file.eo"; \
		exit 1; \
	fi
	@echo "Compiling $(FILE)..."
	$(ECHO_COMPILER) build $(FILE)
	$(CLANG) $(BUILD_DIR)/$(basename $(notdir $(FILE))).o $(RUNTIME_LIB) -o $(BUILD_DIR)/$(basename $(notdir $(FILE)))_binary -lpthread
	@echo "Compilation successful: $(BUILD_DIR)/$(basename $(notdir $(FILE)))_binary"

.PHONY: run-async
run-async: build-compiler
	@echo "Running async test..."
	$(ECHO_COMPILER) run test_async_minimal.eo

.PHONY: clean
clean:
	rm -rf $(BUILD_DIR)

.PHONY: help
help:
	@echo "Available targets: build-compiler, build-runtime, compile-file FILE=file.eo, run-async, clean"
