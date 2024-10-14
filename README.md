# CosmWasmGo

A project to create a CosmWasm compatible x/wasm without the Rust/Cgo complications.

## Why 
Developers intending to build a CosmWasm compatible chain will no longer be forced to use multiple toolchains to perform their work, and a consistent simple programming language can be used end to end. 

This means lower costs for projects due to the easier readability, access to greater numbers of developers, shorter time to effective competence for new developers, faster cycle times during developer workflows and integrations, and a lower performance baseline for hardware required by developers to do their work.

## Introduction

Cosmos, and the Tendermint engine it is based on, were originally written in pure Go, and it was intended that multiple language support be possible primarily via a socket/RPC API so that applications (smart contracts) could be implemented in any language supporting the API interface (gRPC/Protobuf).

Enabling chains to dynamically load new contracts was an obvious deficiency of the design, when compared to Ethereum, and the other protocols like Solana, Avalanche, IOTA, and others, and so a project to create a virtual machine based runtime for Cosmos chains was started.

At the time, the best compiler for producing webassembly binaries and VM for executing them was written in Rust, and so this project chose this, creating complications in the layout and extra complexity due to needing to interface the Go code, forming the basis of the Tendermint Consensus engine, and the various Cosmos components like the bank, the mint, and the main innovation of Cosmos, the IBC cross chain communication protocol IBC.

The Go compilation target of wasm, specifically the embedded Go variant, "TinyGo", is now equally robust as the Rust one and there is now also a pure Go WASM VM engine, the one we are proposing to use is https://github.com/tetratelabs/wazero, and along with this, supporting the use of the TinyGo compiler to create the WASM binaries to use as contracts.

## Architecture of CosmWasm/wasmd

In order to implement this, one needs to understand the current architecture of the cosmwasm stack.

### x/wasmd

x/wasmd is a module that interfaces the Tendermint/Cosmos stack with the Wasmer based execution engine. This part of the system is pure Go, and is essentially the same as a basic Cosmos application with this unit in place of the baseapp module.

### wasmvm

wasmvm is a Go interface using Cgo to embed an instance of Wasmer, and provides the glue between the Go side of wasmd and the two CosmWasm components.

### CosmWasm

CosmWasm itself in as far as concerns building a Cosmos chain is primarily two components:

cosmwasm-std - which is a library for building wasm smart contracts in Rust, it provides contract management, access control and IBC interface to allow reading the current state of other chains and vice-versa.

cosmwasm-vm - this component provides the interface to control the WASM VM engine, and in the current release includes contract validation, memory-pinning and runtime metrics information. It is written in Rust.

### Wasmer

Wasmer is the back-end, WASM VM engine that can execute wasm binaries and return the results of that execution. It is written in Rust and wrapped by cosmwasm-vm.

## Moving The Stack to Go

There are 4 primary tasks involved in achieving the goal of making a CosmWasm compatible framework that moves everything to Go:

### wazero

The WASM VM engine we will use is a minimal dependency, pure Go implementation, https://github.com/tetratelabs/wazero. This engine is compatible with version 2 of the WASM specification.

### wasmvm

wasmvm currently is mostly a glue that plugs in the x/wasmd to an interface for Wasmer. It doesn't have a very large API, and for the basic features, execution, storage of code, IBC sockets and channels, there is very little on the Rust side that is complicated.

### Metrics interface

The main substance of moving to a pure Go implementation is ensuring cosmwasm-vm and cosmwasm-std both build and run and interface properly with the contracts that are to be deployed to it - and once these two are integrated, in their current form as Rust language implementations, the wasmvm component is done to the point of MVP. Completing the metrics is an additional task, important, but secondary.

The amount of work involved is probably something like 300 hours for a senior Go dev.


### cosmwasm-vm

This component is the simpler of the two, and the implementation language would be TinyGo. This can also be ported to TinyGo and simplify the building of Go based smart contracts by eliminating the need for bindings and the need for the developer to have a Rust toolchain.

The amount of work involved is probably something like 150 hours for a senior GO dev.

### cosmwasm-std

cosmwasm-std is a substantially larger API than cosmwasm-vm and will take a little longer to move to Go. Its main functions are related to the standard namespace layout of a CosmWasm contract, and is the primary contact point of smart contracts. Most of the cosmwasm-vm components are encapsulated within it.

The amount of work involved is probably something like 200 hours for a senior GO dev.

