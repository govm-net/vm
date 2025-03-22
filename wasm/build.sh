#! /bin/bash

tinygo build -o contract.wasm -target wasi -gc=leaking ./contract.go
