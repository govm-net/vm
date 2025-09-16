package config

var RootDirOfContracts = "internal/contracts"
var ContractPackageBase = "github.com/govm-net/" + RootDirOfContracts
var AllowImports = []string{
	"github.com/govm-net/vm/core",
	ContractPackageBase,
}
var GasLimitPackage = `import _cover_atomic_ "github.com/govm-net/vm/resource"`

type CtxKeyTxHash struct{}
type CtxKeyBlockHeight struct{}
