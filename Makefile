.PHONY: build run-counter clean test

# 设置Go编译器
GO=go

# 项目路径
PROJECT_ROOT=$(shell pwd)
WASM_DIR=$(PROJECT_ROOT)/wasm
VM_DIR=$(PROJECT_ROOT)/vm
RUNNER_DIR=$(VM_DIR)/runner

# 构建目标
RUNNER_BIN=$(RUNNER_DIR)/vm_runner

# 默认目标
all: build

# 构建VM runner工具
build: $(RUNNER_BIN)

# 构建runner可执行文件
$(RUNNER_BIN):
	@echo "Building VM runner..."
	@cd $(RUNNER_DIR) && $(GO) build -o vm_runner

# 列出计数器合约的可用函数
list-counter-funcs: build
	@echo "列出计数器合约的可用函数..."
	@$(RUNNER_BIN) -list

# 初始化计数器合约
init-counter: build
	@echo "初始化计数器合约..."
	@$(RUNNER_BIN) -func Initialize -params '{"value": 10}'

# 增加计数器值
increment-counter: build
	@echo "增加计数器值..."
	@$(RUNNER_BIN) -func Increment

# 获取计数器当前值
get-counter: build
	@echo "获取计数器当前值..."
	@$(RUNNER_BIN) -func GetCounter

# 重置计数器值
reset-counter: build
	@echo "重置计数器值..."
	@$(RUNNER_BIN) -func Reset -params '{"value": 0}'

# 完整测试计数器合约流程
test-counter: build
	@echo "测试计数器合约完整流程..."
	@echo "\n1. 初始化计数器为10"
	@$(RUNNER_BIN) -func Initialize -params '{"value": 10}'
	@echo "\n2. 获取初始计数器值"
	@$(RUNNER_BIN) -func GetCounter
	@echo "\n3. 增加计数器值"
	@$(RUNNER_BIN) -func Increment
	@echo "\n4. 再次获取计数器值"
	@$(RUNNER_BIN) -func GetCounter
	@echo "\n5. 重置计数器为0"
	@$(RUNNER_BIN) -func Reset -params '{"value": 0}'
	@echo "\n6. 获取重置后的计数器值"
	@$(RUNNER_BIN) -func GetCounter

# 清理构建产物
clean:
	@echo "清理构建产物..."
	@rm -f $(RUNNER_BIN)

# 帮助信息
help:
	@echo "可用的构建目标:"
	@echo "  build               - 构建VM runner"
	@echo "  list-counter-funcs  - 列出计数器合约的可用函数"
	@echo "  init-counter        - 初始化计数器合约"
	@echo "  increment-counter   - 增加计数器值"
	@echo "  get-counter         - 获取计数器当前值"
	@echo "  reset-counter       - 重置计数器值"
	@echo "  test-counter        - 测试计数器合约完整流程"
	@echo "  clean               - 清理构建产物" 