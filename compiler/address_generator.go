package compiler

import (
	"context"

	"github.com/govm-net/vm/core"
)

type AddressGenerator struct{}

func (ag AddressGenerator) Generate(ctx context.Context, code []byte) core.Address {
	hash := core.GetHash(code)
	var out core.Address
	copy(out[:], hash[:])
	return out
}
