#!/bin/bash

tinygo build -o contract.wasm -target wasi ./
#GOMAXPROCS=1 TINYGO_DISABLE_SIGNAL_STACK=1 tinygo build -o contract.wasm -target wasi ./