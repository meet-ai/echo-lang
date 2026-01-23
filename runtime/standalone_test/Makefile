# Simple Makefile for standalone memory pool test

CC = clang
CFLAGS = -std=c11 -Wall -Wextra -I../include -I../src/core -O2
LDFLAGS = -pthread

TARGET = standalone_memory_pool_test
SOURCES = ../tests/standalone_memory_pool_test.c \
          ../src/core/industrial_memory_pool.c \
          ../src/core/async_memory_pool.c
HEADERS = ../include/echo/industrial_memory_pool.h

$(TARGET): $(SOURCES) $(HEADERS)
	$(CC) $(CFLAGS) $(SOURCES) -o $(TARGET) $(LDFLAGS)

clean:
	rm -f $(TARGET) *.o

test: $(TARGET)
	./$(TARGET)

.PHONY: clean test

