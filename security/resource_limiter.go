// Package security 提供VM安全性和资源限制相关功能
package security

// ResourceLimiter 用于限制合约执行的资源使用
type ResourceLimiter struct {
	maxMemoryMB      uint64
	maxExecutionTime uint64
	maxInstructions  uint64
}

// ResourceMonitor 监控资源使用情况
type ResourceMonitor struct {
	limiter *ResourceLimiter
}

// NewResourceLimiter 创建资源限制器
func NewResourceLimiter(maxMemoryMB, maxExecutionTime, maxInstructions uint64) *ResourceLimiter {
	return &ResourceLimiter{
		maxMemoryMB:      maxMemoryMB,
		maxExecutionTime: maxExecutionTime,
		maxInstructions:  maxInstructions,
	}
}

// StartMonitoring 开始监控资源使用
func (r *ResourceLimiter) StartMonitoring() *ResourceMonitor {
	return &ResourceMonitor{
		limiter: r,
	}
}

// Stop 停止监控
func (m *ResourceMonitor) Stop() {
	// 实际实现中会停止监控并进行资源清理
}

// CallTracer 用于追踪合约调用链
type CallTracer struct {
	callStack []CallFrame
}

// CallFrame 表示一个调用栈帧
type CallFrame struct {
	Sender    interface{}
	Contract  interface{}
	Function  string
	StartTime int64
}

// NewCallTracer 创建调用追踪器
func NewCallTracer() *CallTracer {
	return &CallTracer{
		callStack: make([]CallFrame, 0),
	}
}

// BeginCall 记录调用开始
func (t *CallTracer) BeginCall(sender, contract interface{}, function string) {
	frame := CallFrame{
		Sender:    sender,
		Contract:  contract,
		Function:  function,
		StartTime: 0, // 实际使用时应记录当前时间戳
	}
	t.callStack = append(t.callStack, frame)
}

// EndCall 记录调用结束
func (t *CallTracer) EndCall() {
	if len(t.callStack) > 0 {
		t.callStack = t.callStack[:len(t.callStack)-1]
	}
}
