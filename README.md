# CosmWasGo

A project to create a CosmWasm-compatible `x/wasm` module without the Rust/Cgo complications.

## Why?

Developers intending to build a CosmWasm-compatible chain will no longer be forced to use multiple toolchains. A consistent and simple programming language can be used end-to-end.

This translates to lower costs for projects due to easier readability, access to a greater number of developers, shorter time to effective competence for new developers, faster cycle times during developer workflows and integrations, and a lower performance baseline for hardware required by developers to do their work.

## Plan and progress

The first apparent blocker was the fact that the go compiler kept looking for the wrong shared object files.  After some work, we realized that if it works on one platform, that is good enough for our purposes.  So, as of today, mac seems working on the old Go-C-Rust codebase.  That'll do.  Our CI system will be "god" here.

- [x] make gas errors return integers for easy fixes instead of hex
- [x] bring all libraries up to date with current cosmos releases
- [ ] fully lint wasmd and wasmvm
- [ ] implement wasmer at the wasmvm level
- [ ] reimplement cosmwasm-vm in TinyGo
- [ ] reimplement cosmwasm-std in TinyGo





## Introduction

Cosmos, and the Tendermint engine it is based on, were originally written in pure Go. It was intended that multiple language support be possible primarily via a socket/RPC API so that applications (smart contracts) could be implemented in any language supporting the API interface (gRPC/Protobuf).

However, enabling chains to dynamically load new contracts was an obvious deficiency of the design when compared to Ethereum and other protocols like Solana, Avalanche, and IOTA. Thus, a project to create a virtual machine-based runtime for Cosmos chains was started.

At the time, the best compiler for producing WebAssembly (WASM) binaries and the VM for executing them was written in Rust. Therefore, this project chose Rust, creating complications in the layout and extra complexity due to needing to interface the Go code forming the basis of the Tendermint consensus engine and the various Cosmos components like the bank, the mint, and the main innovation of Cosmos, the Inter-Blockchain Communication (IBC) protocol.

The Go compilation target of WASM, specifically the embedded Go variant, **TinyGo**, is now equally robust as the Rust one. There is now also a pure Go WASM VM engine—[Wazero](https://github.com/tetratelabs/wazero)—which we propose to use, along with supporting the use of the TinyGo compiler to create the WASM binaries to use as contracts.

## Goals

* full compatiblity with the existing set of Rust smart contracts for Cosmos.
* improved performance and stability compared to `wasmd` using Rust
* same integration path
* same useage of state
* same storage of contracts

## Architecture of CosmWasm/wasmd

To implement this, one needs to understand the current architecture of the CosmWasm stack.

### x/wasmd

`x/wasmd` is a module that interfaces the Tendermint/Cosmos stack with the Wasmer-based execution engine. This part of the system is pure Go and is essentially the same as a basic Cosmos application with this unit in place of the `baseapp` module.

### wasmvm

`wasmvm` is a Go interface using Cgo to embed an instance of Wasmer. It provides the glue between the Go side of `wasmd` and the two CosmWasm components.

### CosmWasm

CosmWasm itself, as far as building a Cosmos chain is concerned, consists primarily of two components:

* **cosmwasm-std**: A library for building WASM smart contracts in Rust. It provides contract management, access control, and IBC interface to allow reading the current state of other chains and vice versa.

* **cosmwasm-vm**: This component provides the interface to control the WASM VM engine, and in the current release includes contract validation, memory-pinning, and runtime metrics information. It is written in Rust.

### Wasmer

Wasmer is the backend WASM VM engine that can execute WASM binaries and return the results of that execution. It is written in Rust and wrapped by `cosmwasm-vm`.

## Moving the Stack to Go

There are four primary tasks involved in achieving the goal of making a CosmWasm-compatible framework that moves everything to Go:

### Wazero

The WASM VM engine we will use is a minimal dependency, pure Go implementation: [Wazero](https://github.com/tetratelabs/wazero). This engine is compatible with version 2 of the WASM specification.

### wasmvm

Currently, `wasmvm` is mostly glue that plugs `x/wasmd` into an interface for Wasmer. It doesn't have a very large API, and for the basic features—execution, storage of code, IBC sockets, and channels—there is very little on the Rust side that is complicated.

### cosmwasm-vm

This component is the simpler of the two, and the implementation language would be TinyGo. This can also be ported to TinyGo and simplify the building of Go-based smart contracts by eliminating the need for bindings and the need for the developer to have a Rust toolchain.

The amount of work involved is probably something like **150 hours** for a senior Go developer.

### cosmwasm-std

`cosmwasm-std` is a substantially larger API than `cosmwasm-vm` and will take a little longer to move to Go. Its main functions are related to the standard namespace layout of a CosmWasm contract and is the primary contact point of smart contracts. Most of the `cosmwasm-vm` components are encapsulated within it.

The amount of work involved is probably something like **200 hours** for a senior Go developer.

### Metrics Interface

The main substance of moving to a pure Go implementation is ensuring `cosmwasm-vm` and `cosmwasm-std` both build and run and interface properly with the contracts that are to be deployed to it. Once these two are integrated—in their current form as Rust language implementations—the `wasmvm` component is done to the point of MVP. Completing the metrics is an additional, important, but secondary task.

The amount of work involved is probably something like **300 hours** for a senior Go developer.

## Conclusion

By moving the CosmWasm stack to pure Go, we simplify the development process for building Cosmos chains with smart contract capabilities. Developers can work within a consistent language ecosystem, reducing complexity and increasing productivity.
