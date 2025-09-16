package interfaces

// ResourceManager 定义了资源管理器的接口
type ResourceManager interface {
	// Consume 消耗指定数量的资源
	Consume(amount int64) error

	// GetRemaining 获取剩余资源
	GetRemaining() int64
}

// OutOfResourceError 表示资源不足的错误
type OutOfResourceError struct {
	Required  int64
	Available int64
}

func (e *OutOfResourceError) Error() string {
	return "out of resource"
}
