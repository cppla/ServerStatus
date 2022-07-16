OUT = sergate

#CC = clang
CC = gcc
CFLAGS = -Wall -O2

#CXX = clang++
CXX = g++
CXXFLAGS = -Wall -O2 -std=c++11

ODIR = obj
SDIR = src
LIBS = -pthread -lm
INC = -Iinclude

C_SRCS := $(wildcard $(SDIR)/*.c)
CXX_SRCS := $(wildcard $(SDIR)/*.cpp)
C_OBJS := $(patsubst $(SDIR)/%.c,$(ODIR)/%.o,$(C_SRCS))
CXX_OBJS := $(patsubst $(SDIR)/%.cpp,$(ODIR)/%.o,$(CXX_SRCS))
OBJS := $(C_OBJS) $(CXX_OBJS)

$(ODIR)/%.o: $(SDIR)/%.c
	$(CC) -c $(INC) $(CFLAGS) $< -o $@

$(ODIR)/%.o: $(SDIR)/%.cpp
	$(CXX) -c $(INC) $(CXXFLAGS) $< -o $@

$(OUT): $(OBJS)
	$(CXX) $(LIBS) $^ -o $(OUT) -lcurl

.PHONY: clean

clean:
	rm -f $(ODIR)/*.o $(OUT)
