CC = gcc
NVCC = nvcc
CFLAGS = -O3 -march=native -mavx512f -flto -fopenmp -Wall -Wextra -Wno-unused-parameter -std=c23 -D_GNU_SOURCE
CUFLAGS = -O3 -arch=sm_89 -Isrc -Iinclude --compiler-options -Wall -DHAS_CUDA
LDFLAGS = -lm -lyaml
INCLUDES = -Iinclude -Isrc

HAVE_CUDA := $(shell which nvcc >/dev/null 2>&1 && echo 1 || echo 0)

C_SRC = $(wildcard src/*.c)
C_OBJ = $(C_SRC:.c=.o)

ifeq ($(HAVE_CUDA),1)
  CU_OBJ = src/render_cuda_gpu.o
  CFLAGS += -DHAS_CUDA
  LDFLAGS += -L/opt/cuda/lib64 -lcudart
  LINK = $(NVCC) -arch=sm_89 -Xcompiler -flto
else
  CU_OBJ =
  LINK = $(CC) $(CFLAGS)
endif

BIN = crelray
OBJ = $(C_OBJ) $(CU_OBJ)

all: $(BIN)

$(BIN): $(OBJ)
	$(LINK) -o $@ $^ $(LDFLAGS)

src/render_cuda_gpu.o: src/render_cuda.cu
	$(NVCC) $(CUFLAGS) -c -o $@ $<

src/%.o: src/%.c
	$(CC) $(CFLAGS) $(INCLUDES) -c -o $@ $<

install: $(BIN)
	install -m 755 $(BIN) $(DESTDIR)$(PREFIX)/bin/$(BIN)

clean:
	rm -f src/*.o $(BIN)

PREFIX ?= /usr/local
.PHONY: all clean install
