package resource

var gasLimit uint64
var gasUsed uint64

func ConsumeGas(delta uint32) {
	gasUsed += uint64(delta)
	if gasUsed > gasLimit {
		panic("out of gas")
	}
}
func GetGasUsed() uint64 {
	return gasUsed
}
func ResetGasUsed() {
	gasUsed = 0
}
func SetGasLimit(limit uint64) {
	gasLimit = limit
}
