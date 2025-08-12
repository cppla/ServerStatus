OUT = sergate

# Build mode: make (默认 release) 或 make BUILD=debug
BUILD ?= release

# 自动检测 ccache
CCACHE_BIN:=$(shell which ccache 2>/dev/null)
ifeq ($(CCACHE_BIN),)
	CC := gcc
	CXX := g++
else
	CC := ccache gcc
	CXX := ccache g++
endif

COMMON_WARN=-Wall
COMMON_INC=-Iinclude
COMMON_PIPE=-pipe

ifeq ($(BUILD),debug)
	CFLAGS   = $(COMMON_WARN) -O0 -g $(COMMON_PIPE)
	CXXFLAGS = $(COMMON_WARN) -O0 -g -std=c++11 $(COMMON_PIPE)
else
	CFLAGS   = $(COMMON_WARN) -O2 $(COMMON_PIPE)
	CXXFLAGS = $(COMMON_WARN) -O2 -std=c++11 $(COMMON_PIPE)
endif

ODIR = obj
SDIR = src
LIBS = -pthread -lm -lcurl
INC = $(COMMON_INC) -Isrc

# 预编译头（主要加速包含 exprtk.hpp 的 C++ 编译）
PCH_HDR = $(SDIR)/pch.hpp
PCH = $(ODIR)/pch.hpp.gch

C_SRCS := $(wildcard $(SDIR)/*.c)
CXX_SRCS := $(wildcard $(SDIR)/*.cpp)
C_OBJS := $(patsubst $(SDIR)/%.c,$(ODIR)/%.o,$(C_SRCS))
CXX_OBJS := $(patsubst $(SDIR)/%.cpp,$(ODIR)/%.o,$(CXX_SRCS))
OBJS := $(C_OBJS) $(CXX_OBJS)


all: $(OUT)

$(ODIR):
	mkdir -p $(ODIR)

$(PCH): $(PCH_HDR) | $(ODIR)
	$(CXX) $(INC) $(CXXFLAGS) -MMD -MP -x c++-header $< -o $@

$(ODIR)/%.o: $(SDIR)/%.cpp $(PCH) | $(ODIR)
	$(CXX) -c $(INC) $(CXXFLAGS) -MMD -MP -include src/pch.hpp $< -o $@

$(ODIR)/%.o: $(SDIR)/%.c | $(ODIR)
	$(CC) -c $(INC) $(CFLAGS) -MMD -MP $< -o $@

$(OUT): $(OBJS)
	$(CXX) $(CXXFLAGS) $^ -o $(OUT) $(LIBS)

.PHONY: clean all

clean:
	rm -f $(ODIR)/*.o $(ODIR)/*.d $(OUT) $(PCH)

.PHONY: debug release
debug:
	$(MAKE) BUILD=debug
release:
	$(MAKE) BUILD=release

-include $(ODIR)/*.d
