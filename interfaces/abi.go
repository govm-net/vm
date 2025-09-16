package interfaces

// ABIExtractor 定义了 ABI 提取器的接口
type ABIExtractor interface {
	// Extract 从源码中提取 ABI 信息
	Extract(code []byte) (*ABI, error)
}

// ABI 表示合约的应用程序二进制接口
type ABI struct {
	PackageName string     `json:"package_name,omitempty"`
	Imports     []Import   `json:"imports,omitempty"`
	Functions   []Function `json:"functions,omitempty"`
	Events      []Event    `json:"events,omitempty"`
}

// Import represents an import statement in the contract
type Import struct {
	Path string `json:"path,omitempty"` // 导入路径
	Name string `json:"name,omitempty"` // 别名（如果有）
}

// Function 表示合约中的函数
type Function struct {
	Name    string      `json:"name,omitempty"`
	Inputs  []Parameter `json:"inputs,omitempty"`
	Outputs []Parameter `json:"outputs,omitempty"`
}

// Event 表示合约中的事件
type Event struct {
	Name   string      `json:"name,omitempty"`
	Params []Parameter `json:"params,omitempty"`
}

// Parameter 表示函数参数或事件字段
type Parameter struct {
	Name string `json:"name,omitempty"`
	Type string `json:"type,omitempty"`
}
