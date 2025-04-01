package mock

import (
	"testing"
)

func TestGas(t *testing.T) {
	// 测试初始化
	InitGas(1000)
	if GetGas() != 1000 {
		t.Errorf("expected gas=1000, got=%d", GetGas())
	}
	if GetUsedGas() != 0 {
		t.Errorf("expected used=0, got=%d", GetUsedGas())
	}

	// 测试消耗gas
	ConsumeGas(500)
	if GetGas() != 500 {
		t.Errorf("expected gas=500, got=%d", GetGas())
	}
	if GetUsedGas() != 500 {
		t.Errorf("expected used=500, got=%d", GetUsedGas())
	}

	// 测试退还gas
	RefundGas(200)
	if GetGas() != 700 {
		t.Errorf("expected gas=700, got=%d", GetGas())
	}
	if GetUsedGas() != 300 {
		t.Errorf("expected used=300, got=%d", GetUsedGas())
	}

	// 测试重置gas
	ResetGas(2000)
	if GetGas() != 2000 {
		t.Errorf("expected gas=2000, got=%d", GetGas())
	}
	if GetUsedGas() != 0 {
		t.Errorf("expected used=0, got=%d", GetUsedGas())
	}
}

func TestGasPanic(t *testing.T) {
	InitGas(100)

	// 测试gas不足
	defer func() {
		if r := recover(); r == nil {
			t.Error("expected panic for out of gas")
		}
	}()
	ConsumeGas(200)

	// 测试无效退还
	InitGas(100)
	ConsumeGas(50)
	defer func() {
		if r := recover(); r == nil {
			t.Error("expected panic for invalid refund")
		}
	}()
	RefundGas(100)
}
