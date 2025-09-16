package compiler

import (
	"context"
	"os"
	"reflect"
	"testing"

	"github.com/govm-net/vm/config"
	"github.com/govm-net/vm/core"
)

func TestDefaultCompiler_CompileContract(t *testing.T) {
	old := config.ContractPackageBase
	allowLen := len(config.AllowImports)
	config.ContractPackageBase = "github.com/govm-net/vm/compiler/" + config.RootDirOfContracts
	config.AllowImports = append(config.AllowImports, config.ContractPackageBase)
	defer func() {
		config.ContractPackageBase = old
		config.AllowImports = config.AllowImports[:allowLen]
		os.RemoveAll(config.RootDirOfContracts)
		os.RemoveAll("temp")
		os.RemoveAll("internal")
	}()

	type args struct {
		ctx  context.Context
		code string
	}
	tests := []struct {
		name         string
		allowImports []string
		args         args
		wantErr      bool
	}{
		{
			name: "test",
			args: args{
				ctx: context.Background(),
				code: `package contract

				import "github.com/govm-net/vm/core"
				
				func Run() {
					core.Log("Hello, World!")
				}`,
			},
		},
		{
			name: "test2",
			args: args{
				ctx: context.Background(),
				code: `package contract

				import "github.com/govm-net/vm/core"
				
				func Run(name string) {
					core.Log("Hello",name)
				}`,
			},
		},
		{
			name: "test3",
			args: args{
				ctx: context.Background(),
				code: `package contract

				func Run(name string) {
					print("Hello",name)
				}`,
			},
		},
		// wrong import
		{
			name: "test_wrong_import",
			args: args{
				ctx: context.Background(),
				code: `package contract

				import "github.com/govm-net/vm/test"
				
				func Run() {
					test.Log("Hello, World!")
				}`,
			},
			wantErr: true,
		},
		// go:build
		{
			name: "test_go_build",
			args: args{
				ctx: context.Background(),
				code: `package contract

				import "github.com/govm-net/vm/core"
				
				//cgo
				//go:build cgo
				//+build x y
				func Run() {
					core.Log("Hello, World!")
				}`,
			},
			wantErr: true,
		},
		// go:ignore
		{
			name: "test_go_ignore",
			args: args{
				ctx: context.Background(),
				code: `package contract

				import "github.com/govm-net/vm/core"
				
				//go:build ignore
				func Run() {
					core.Log("Hello, World!")
				}`,
			},
			wantErr: true,
		},
		// limit keyword
		{
			name: "test_limit_keyword",
			args: args{
				ctx: context.Background(),
				code: `package contract

				import "github.com/govm-net/vm/core"
				
				func Run() {
					go core.Log("Hello, World!")
				}`,
			},
			wantErr: true,
		},
		{
			name: "test_import",
			args: args{
				ctx: context.Background(),
				code: `package contract

				import base "github.com/govm-net/vm/compiler/internal/contracts/66a10e1af9de8c76f3048392dc72647a401d7040"
				
				func Run() {
					base.Run("test input")
				}`,
			},
		},
	}
	runCase := ""
	for _, tt := range tests {
		if len(runCase) > 0 && tt.name != runCase {
			continue
		}
		t.Run(tt.name, func(t *testing.T) {
			c := NewDefaultCompiler()
			if tt.allowImports != nil {
				c.allowImports = tt.allowImports
			}
			got, err := c.CompileContract(tt.args.ctx, []byte(tt.args.code))
			if (err != nil) != tt.wantErr {
				t.Errorf("DefaultCompiler.CompileContract() error = %v, wantErr %v", err, tt.wantErr)
				return
			}
			if err == nil && reflect.DeepEqual(got, core.ZeroAddress) {
				t.Errorf("DefaultCompiler.CompileContract() = %v", got)
			}
		})
	}
}
