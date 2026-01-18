# Echo Language Compiler Makefile

# Compiler settings
GO := go
ECHO_COMPILER := ./echoc
SOURCE_DIR := examples
BUILD_DIR := build
OCAML_DIR := $(BUILD_DIR)/ocaml

# Backend options: llvm-ir (only)
BACKEND ?= llvm-ir

# LLVM tools
CLANG := clang
LLC := /opt/homebrew/opt/llvm@19/bin/llc
OPT := /opt/homebrew/opt/llvm@19/bin/opt

# Runtime library
RUNTIME_SRC := runtime/echo_runtime.c runtime/coroutine_runtime.c
# Context switching implementations (platform-specific)
RUNTIME_SRC += runtime/context_x86_64.c runtime/context_aarch64.c runtime/task.c runtime/future.c runtime/coroutine.c runtime/processor.c runtime/scheduler.c runtime/machine.c
RUNTIME_OBJ := $(BUILD_DIR)/echo_runtime.o $(BUILD_DIR)/coroutine_runtime.o
RUNTIME_OBJ += $(BUILD_DIR)/context_x86_64.o $(BUILD_DIR)/context_aarch64.o
RUNTIME_OBJ += $(BUILD_DIR)/task.o $(BUILD_DIR)/future.o $(BUILD_DIR)/coroutine.o $(BUILD_DIR)/processor.o $(BUILD_DIR)/scheduler.o $(BUILD_DIR)/channel.o $(BUILD_DIR)/machine.o
RUNTIME_LIB := $(BUILD_DIR)/libcoroutine.a

# Assembly-based runtime (alternative implementation)
ASM_RUNTIME_SRC := runtime/echo_runtime.c runtime/coroutine_runtime_asm.c
ASM_RUNTIME_OBJ := $(BUILD_DIR)/echo_runtime.o $(BUILD_DIR)/coroutine_runtime_asm.o
ASM_RUNTIME_LIB := $(BUILD_DIR)/libcoroutine_asm.a

# Default target
.PHONY: all
all: build compile-examples

# Build the Echo compiler
.PHONY: build-compiler
build-compiler:
	@echo "Building Echo compiler..."
	$(GO) build -o $(ECHO_COMPILER) ./cmd/main.go
	@echo "Echo compiler built successfully"

# Build runtime library
.PHONY: build-runtime
build-runtime: build/$(notdir $(RUNTIME_LIB))
	@echo "Runtime library built successfully"

# Build assembly-based runtime library
.PHONY: build-runtime-asm
build-runtime-asm: build/$(notdir $(ASM_RUNTIME_LIB))
	@echo "Assembly-based runtime library built successfully"

# Compile individual runtime objects
$(BUILD_DIR)/%.o: runtime/%.c
	$(CLANG) -c $< -o $@ -lpthread

$(BUILD_DIR)/%_asm.o: runtime/%.s
	$(CLANG) -c $< -o $@

# Compile assembly-based runtime objects
$(BUILD_DIR)/coroutine_runtime_asm.o: runtime/coroutine_runtime_asm.c
	$(CLANG) -c $< -o $@ -lpthread

# Create static library from runtime objects
$(BUILD_DIR)/libcoroutine.a: $(RUNTIME_OBJ)
	ar rcs $@ $^

# Create assembly-based static library
$(BUILD_DIR)/libcoroutine_asm.a: $(ASM_RUNTIME_OBJ)
	ar rcs $@ $^

# Build target: build compiler or compile .eo file to binary
.PHONY: build
build: build-compiler
	@if [ -n "$(filter-out build, $(MAKECMDGOALS))" ]; then \
		for target in $(filter-out build, $(MAKECMDGOALS)); do \
			if echo "$$target" | grep -q "\.eo$$"; then \
				$(MAKE) build-file FILE=$$target; \
				exit $$?; \
			fi; \
		done; \
	fi

# Build echoc compiler binary (user requested: make build -f)
.PHONY: build-echoc
build-echoc: build-compiler
	@echo "Echo compiler 'echoc' built successfully"
	@echo "Usage: ./echoc -backend=llvm-ir -target=ir <file.eo>"
	@echo "       ./echoc -backend=llvm-ir -target=executable <file.eo>"

# Compile default file (examples/hello.eo) to binary
.PHONY: compile
compile: build-compiler build-runtime
	@echo "Compiling $(SOURCE_DIR)/hello.eo to binary..."
	$(ECHO_COMPILER) -backend=llvm-ir -target=ir $(SOURCE_DIR)/hello.eo > $(BUILD_DIR)/hello.ll
	$(LLC) -filetype=obj $(BUILD_DIR)/hello.ll -o $(BUILD_DIR)/hello.o
	$(CLANG) $(BUILD_DIR)/hello.o $(RUNTIME_LIB) -o $(BUILD_DIR)/hello_binary -lpthread
	@echo "Generated $(BUILD_DIR)/hello_binary"

# Compile a specific file to binary
# Usage: make compile-file FILE=path/to/file.eo
.PHONY: compile-file
compile-file: build-compiler build-runtime
	@if [ -z "$(FILE)" ]; then \
		echo "Usage: make compile-file FILE=<filename.eo>"; \
		echo "Example: make compile-file FILE=examples/demo.eo"; \
		exit 1; \
	fi
	@echo "Compiling $(FILE) to binary..."
	@if [ ! -f "$(FILE)" ]; then \
		echo "File $(FILE) not found"; \
		exit 1; \
	fi
	$(ECHO_COMPILER) -backend=llvm-ir -target=ir $(FILE) > $(BUILD_DIR)/$(basename $(notdir $(FILE))).ll
	$(LLC) -filetype=obj $(BUILD_DIR)/$(basename $(notdir $(FILE))).ll -o $(BUILD_DIR)/$(basename $(notdir $(FILE))).o
	$(CLANG) $(BUILD_DIR)/$(basename $(notdir $(FILE))).o $(RUNTIME_LIB) -o $(BUILD_DIR)/$(basename $(notdir $(FILE)))_binary -lpthread
	@echo "Generated $(BUILD_DIR)/$(basename $(notdir $(FILE)))_binary"

# Build a specific .eo file to binary
# Usage: make build-file FILE=path/to/file.eo
.PHONY: build-file
build-file: build-compiler
	@if [ -z "$(FILE)" ]; then \
		echo "Usage: make build-file FILE=<filename.eo>"; \
		echo "Example: make build-file FILE=examples/demo.eo"; \
		exit 1; \
	fi
	@echo "Building $(FILE) to binary..."
	@if [ ! -f "$(FILE)" ]; then \
		echo "File $(FILE) not found"; \
		exit 1; \
	fi
	$(ECHO_COMPILER) -backend=llvm-ir -target=ir $(FILE) > $(BUILD_DIR)/$(basename $(notdir $(FILE))).ll
	$(LLC) -filetype=obj $(BUILD_DIR)/$(basename $(notdir $(FILE))).ll -o $(BUILD_DIR)/$(basename $(notdir $(FILE))).o
	$(CLANG) $(BUILD_DIR)/$(basename $(notdir $(FILE))).o -o $(BUILD_DIR)/$(basename $(notdir $(FILE)))_binary
	@echo "Generated $(BUILD_DIR)/$(basename $(notdir $(FILE)))_binary"

# Clean build artifacts
.PHONY: clean
clean:
	@echo "Cleaning build artifacts..."
	rm -rf $(BUILD_DIR) $(ECHO_COMPILER)
	@echo "Clean completed"

# Create build directories
$(BUILD_DIR):
	mkdir -p $(BUILD_DIR)

$(OCAML_DIR): $(BUILD_DIR)
	mkdir -p $(OCAML_DIR)

# Compile all examples
.PHONY: compile-examples
compile-examples: build $(OCAML_DIR)
	@echo "Compiling examples..."
	$(MAKE) compile-hello
	$(MAKE) compile-hello-exec
	@echo "All examples compiled"

# Compile hello.eo example
.PHONY: compile-hello
compile-hello: build $(OCAML_DIR)
	@echo "Compiling hello.eo..."
	$(ECHO_COMPILER) $(SOURCE_DIR)/hello.eo | awk '/=== Generated OCaml Code ===/{flag=1; next} /=== Executable OCaml Program ===/{flag=0} flag' > $(OCAML_DIR)/hello.ml
	@echo "Generated $(OCAML_DIR)/hello.ml"

# Generate executable OCaml program
.PHONY: compile-hello-exec
compile-hello-exec: build $(OCAML_DIR)
	@echo "Generating executable OCaml program..."
	$(ECHO_COMPILER) $(SOURCE_DIR)/hello.eo | awk '/=== Executable OCaml Program ===/{flag=1; next} flag' > $(OCAML_DIR)/hello_exec.ml
	@echo "Generated $(OCAML_DIR)/hello_exec.ml"

# Compile OCaml to native executable (requires OCaml)
.PHONY: compile-ocaml
compile-ocaml: compile-hello-exec
	@echo "Compiling OCaml to native executable..."
	@if command -v ocamlc >/dev/null 2>&1; then \
		ocamlc -o $(BUILD_DIR)/hello $(OCAML_DIR)/hello_exec.ml; \
		echo "Generated $(BUILD_DIR)/hello executable"; \
	else \
		echo "OCaml compiler not found. Install with: brew install ocaml"; \
		exit 1; \
	fi


# Compile Echo to LLVM IR
.PHONY: compile-ir
compile-ir: build
	@echo "Compiling $(SOURCE_DIR)/hello.eo to LLVM IR using $(BACKEND) backend..."
	@if [ "$(BACKEND)" = "llvm-ir" ]; then \
		$(ECHO_COMPILER) -backend=llvm-ir -target=ir $(SOURCE_DIR)/hello.eo > $(BUILD_DIR)/hello.ll; \
		echo "Generated $(BUILD_DIR)/hello.ll LLVM IR file from LLVM IR backend"; \
	elif command -v $(CLANG) >/dev/null 2>&1; then \
		$(ECHO_COMPILER) -backend=clang $(SOURCE_DIR)/hello.eo | $(CLANG) -x c - -S -emit-llvm -o $(BUILD_DIR)/hello.ll; \
		echo "Generated $(BUILD_DIR)/hello.ll LLVM IR file from C backend"; \
	else \
		echo "No suitable backend found."; \
		exit 1; \
	fi

# Compile LLVM IR to binary
.PHONY: compile-ir-to-binary
compile-ir-to-binary: compile-ir
	@echo "Compiling LLVM IR to binary..."
	@if command -v $(LLC) >/dev/null 2>&1 && command -v $(CLANG) >/dev/null 2>&1; then \
		$(LLC) -filetype=obj $(BUILD_DIR)/hello.ll -o $(BUILD_DIR)/hello_ir.o; \
		$(CLANG) $(BUILD_DIR)/hello_ir.o -o $(BUILD_DIR)/hello_ir_binary; \
		echo "Generated $(BUILD_DIR)/hello_ir_binary from LLVM IR"; \
	else \
		echo "LLC ($(LLC)) or CLANG ($(CLANG)) not found."; \
		exit 1; \
	fi

# Test the compiler with hello.eo
.PHONY: test
test: build
	@echo "Testing compiler with hello.eo..."
	$(ECHO_COMPILER) $(SOURCE_DIR)/hello.eo

# Format Go code
.PHONY: fmt
fmt:
	@echo "Formatting Go code..."
	$(GO) fmt ./...

# Run Go tests
.PHONY: test-go
test-go:
	@echo "Running Go tests..."
	$(GO) test ./...

# Lint Go code
.PHONY: lint
lint:
	@echo "Linting Go code..."
	$(GO) vet ./...

# Development setup
.PHONY: dev-setup
dev-setup:
	@echo "Setting up development environment..."
	$(GO) mod tidy
	$(GO) mod download

# Install OCaml (optional)
.PHONY: install-ocaml
install-ocaml:
	@echo "Installing OCaml (requires Homebrew)..."
	brew install ocaml

# Run the compiled OCaml program (requires OCaml)
.PHONY: run
run: compile-ocaml
	@echo "Running the compiled OCaml program..."
	@if [ -f "$(BUILD_DIR)/hello" ]; then \
		$(BUILD_DIR)/hello; \
	else \
		echo "Executable not found. Run 'make compile-ocaml' first."; \
		exit 1; \
	fi

# Run the compiled binary program
.PHONY: run-binary
run-binary: compile
	@echo "Running the compiled binary program..."
	@if [ -f "$(BUILD_DIR)/hello_binary" ]; then \
		$(BUILD_DIR)/hello_binary; \
	else \
		echo "Binary executable not found. Run 'make compile' first."; \
		exit 1; \
	fi

# Run the compiled LLVM IR binary program
.PHONY: run-ir-binary
run-ir-binary: compile-ir-to-binary
	@echo "Running the compiled LLVM IR binary program..."
	@if [ -f "$(BUILD_DIR)/hello_ir_binary" ]; then \
		$(BUILD_DIR)/hello_ir_binary; \
	else \
		echo "IR binary executable not found. Run 'make compile-ir-to-binary' first."; \
		exit 1; \
	fi

# Run a specific .eo file directly (user requested: make run -f)
# Usage: make run FILE=examples/spawn_test.eo
.PHONY: run
run:
	@if [ -z "$(FILE)" ]; then \
		echo "Usage: make run FILE=<filename.eo>"; \
		echo "Example: make run FILE=examples/spawn_test.eo"; \
		exit 1; \
	fi
	@if [ ! -f "$(FILE)" ]; then \
		echo "File $(FILE) not found"; \
		exit 1; \
	fi
	@echo "Running $(FILE) with Echo compiler..."
	$(GO) run ./cmd/main.go -backend=llvm-ir $(FILE)

# Quick run targets for common test files
.PHONY: run-spawn
run-spawn: build-compiler
	@echo "Running spawn_test.eo..."
	$(GO) run ./cmd/main.go -backend=llvm-ir examples/spawn_test.eo

.PHONY: run-async
run-async: build-compiler
	@echo "Running async_await_test.eo..."
	$(GO) run ./cmd/main.go -backend=llvm-ir examples/async_await_test.eo

.PHONY: run-concurrent
run-concurrent: build-compiler
	@echo "Running concurrent_integration_test.eo..."
	$(GO) run ./cmd/main.go -backend=llvm-ir examples/concurrent_integration_test.eo

.PHONY: run-select
run-select: build-compiler
	@echo "Running select_detailed_test.eo..."
	$(GO) run ./cmd/main.go -backend=llvm-ir examples/select_detailed_test.eo

# Show build information
.PHONY: info
info:
	@echo "Echo Language Compiler Build Info"
	@echo "=================================="
	@echo "Compiler: $(ECHO_COMPILER)"
	@echo "Source dir: $(SOURCE_DIR)"
	@echo "Build dir: $(BUILD_DIR)"
	@echo "OCaml dir: $(OCAML_DIR)"
	@echo "Backend: $(BACKEND)"
	@echo "Target: $(TARGET)"
	@echo ""
	@echo "Generated files:"
	@find $(BUILD_DIR) -type f -exec echo "  {}" \; 2>/dev/null || echo "  (none)"
	@echo ""
	@echo "Available tools:"
	@echo "  OCaml: $(shell command -v ocamlc >/dev/null 2>&1 && echo 'Yes' || echo 'No')"
	@echo "  Clang: $(shell command -v clang >/dev/null 2>&1 && echo 'Yes' || echo 'No')"
	@echo "  LLC: $(shell command -v llc >/dev/null 2>&1 && echo 'Yes' || echo 'No')"
	@echo "  OPT: $(shell command -v opt >/dev/null 2>&1 && echo 'Yes' || echo 'No')" 

# Full development workflow
.PHONY: dev
dev: dev-setup fmt lint test build compile-examples
	@echo "Development workflow completed"

# Show help
.PHONY: help
help:
	@echo "Echo Language Compiler Makefile"
	@echo ""
	@echo "Configuration options:"
	@echo "  BACKEND=clang     - Use clang backend (default)"
	@echo "  BACKEND=llvm-ir   - Use LLVM IR backend"
	@echo "  TARGET=binary     - Generate binary executable (default)"
	@echo "  TARGET=ir         - Generate LLVM IR only"
	@echo ""
	@echo "Available targets:"
	@echo "  all                 - Build compiler and compile examples (default)"
	@echo "  build               - Build the Echo compiler"
	@echo "  build-echoc         - Build echoc compiler binary (what you wanted: make build -f)"
	@echo "  compile             - Compile default file (examples/hello.eo) to binary"
	@echo "  build-file          - Build a specific .eo file to binary (use FILE=)"
	@echo "  compile-examples    - Compile all examples"
	@echo "  compile-hello       - Compile hello.eo to OCaml code"
	@echo "  compile-hello-exec  - Generate executable OCaml program"
	@echo "  compile-ocaml       - Compile OCaml to native executable"
	@echo "  compile-ir          - Generate LLVM IR from Echo code"
	@echo "  compile-ir-to-binary- Compile LLVM IR to binary"
	@echo "  run                 - Run a specific .eo file with compiler (use FILE=)"
	@echo "  run-spawn           - Run spawn_test.eo"
	@echo "  run-async           - Run async_await_test.eo"
	@echo "  run-concurrent      - Run concurrent_integration_test.eo"
	@echo "  run-select          - Run select_detailed_test.eo"
	@echo "  test                - Test compiler with hello.eo"
	@echo "  clean               - Clean build artifacts"
	@echo "  fmt                 - Format Go code"
	@echo "  test-go             - Run Go tests"
	@echo "  lint                - Lint Go code"
	@echo "  dev-setup           - Setup development environment"
	@echo "  run-binary          - Run the compiled binary program"
	@echo "  run-ir-binary       - Run the LLVM IR compiled binary"
	@echo "  info                - Show build information"
	@echo "  dev                 - Full development workflow"
	@echo "  help                - Show this help message"
	@echo ""
	@echo "Examples:"
	@echo "  make build                          # Build the Go compiler"
	@echo "  make compile                        # Compile default file to binary"
	@echo "  make build-file FILE=examples/demo.eo  # Build specific .eo file"
	@echo "  make compile-hello                  # Compile hello.eo to OCaml code"
	@echo "  make compile-ocaml                  # Compile OCaml to native executable"
	@echo "  make run                            # Compile and run OCaml program"
	@echo "  make compile-ir                     # Generate LLVM IR"
	@echo "  make compile-ir-to-binary           # Compile LLVM IR to binary"
	@echo "  make run-ir-binary                  # Run the LLVM IR compiled binary"
	@echo "  make run-binary                     # Run the compiled binary program"
	@echo "  make test                           # Test compiler with hello.eo"
	@echo "  make clean                          # Clean build artifacts"
	@echo "  make info                           # Show build information"
	@echo "  make dev                            # Full development workflow"
	@echo "  help                                # Show this help message"
	@echo ""
	@echo "Examples:"
	@echo "  make build                          # Build the Go compiler"
	@echo "  make build-echoc                    # Build echoc binary (what you wanted)"
	@echo "  make run FILE=examples/spawn_test.eo  # Run specific .eo file (what you wanted)"
	@echo "  make run-spawn                      # Quick run spawn_test.eo"
	@echo "  make run-async                      # Quick run async_await_test.eo"
	@echo "  make run-concurrent                 # Quick run concurrent_integration_test.eo"
	@echo "  make run-select                     # Quick run select_detailed_test.eo"
	@echo "  make compile                        # Compile default file to binary"
	@echo "  make build-file FILE=examples/demo.eo  # Build specific .eo file"
	@echo ""
	@echo "Workflow Options:"
	@echo "  1. OCaml workflow:"
	@echo "     make build && make compile-hello && make compile-ocaml && make run"
	@echo ""
	@echo "  2. LLVM IR workflow (recommended):"
	@echo "     make build && make compile && make run-binary"
	@echo ""
	@echo "  3. Quick development workflow:"
	@echo "     make build-echoc && make run-spawn"
