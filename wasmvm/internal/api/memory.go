package api

import (
	"fmt"
)

// Memory interface represents the linear memory in wazero.
type Memory interface {
	Read(offset, length uint32) ([]byte, bool)
	Write(offset uint32, data []byte) bool
}

// ByteSliceView represents a view into a byte slice in linear memory.
type ByteSliceView struct {
	IsNil bool
	Data  uint32 // Offset in linear memory
	Len   uint32 // Length of the slice
}

// makeView creates a view into the given byte slice in linear memory.
func makeView(offset, length uint32) ByteSliceView {
	if offset == 0 && length == 0 {
		return ByteSliceView{IsNil: true, Data: 0, Len: 0}
	}
	return ByteSliceView{
		IsNil: false,
		Data:  offset,
		Len:   length,
	}
}

// UnmanagedVector represents a vector that is not managed by Go's garbage collector.
// In the context of wazero, this would be data allocated in wasm linear memory.
type UnmanagedVector struct {
	IsNone bool
	Data   uint32 // Offset in linear memory
	Len    uint32 // Length of the data
	Cap    uint32 // Capacity of the data
}

// constructUnmanagedVector creates an UnmanagedVector.
func constructUnmanagedVector(isNone bool, dataOffset, length, capacity uint32) UnmanagedVector {
	return UnmanagedVector{
		IsNone: isNone,
		Data:   dataOffset,
		Len:    length,
		Cap:    capacity,
	}
}

// newUnmanagedVector creates a new UnmanagedVector from data in linear memory.
func newUnmanagedVector(dataOffset, length, capacity uint32) UnmanagedVector {
	if dataOffset == 0 && length == 0 && capacity == 0 {
		return UnmanagedVector{IsNone: true, Data: 0, Len: 0, Cap: 0}
	}
	return UnmanagedVector{
		IsNone: false,
		Data:   dataOffset,
		Len:    length,
		Cap:    capacity,
	}
}

// copyAndDestroyUnmanagedVector copies the data from the UnmanagedVector and "destroys" it.
// In wazero, this would involve copying data from linear memory.
func copyAndDestroyUnmanagedVector(v UnmanagedVector, mem Memory) ([]byte, error) {
	if v.IsNone {
		return nil, nil
	}
	if v.Len == 0 {
		return []byte{}, nil
	}
	data, ok := mem.Read(v.Data, v.Len)
	if !ok {
		return nil, fmt.Errorf("failed to read memory at offset %d", v.Data)
	}
	// Simulate destruction if necessary (e.g., by zeroing memory)
	return data, nil
}

// OptionalU64 represents an optional uint64 value.
type OptionalU64 struct {
	IsSome bool
	Value  uint64
}

// optionalU64ToPtr converts an OptionalU64 to a *uint64.
func optionalU64ToPtr(val OptionalU64) *uint64 {
	if val.IsSome {
		return &val.Value
	}
	return nil
}

// U8SliceView represents a view into a slice of uint8 in linear memory.
type U8SliceView struct {
	IsNone bool
	Data   uint32 // Offset in linear memory
	Len    uint32 // Length of the data
}

// copyU8Slice copies the contents of an Option<&[u8]> that was allocated elsewhere.
// Returns nil if and only if the source is None.
func copyU8Slice(view U8SliceView, mem Memory) ([]byte, error) {
	if view.IsNone {
		return nil, nil
	}
	if view.Len == 0 {
		return []byte{}, nil
	}
	data, ok := mem.Read(view.Data, view.Len)
	if !ok {
		return nil, fmt.Errorf("failed to read memory at offset %d", view.Data)
	}
	return data, nil
}
