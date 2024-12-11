package api

import (
	"testing"

	"github.com/stretchr/testify/require"
)

// MockMemory is a mock implementation of the Memory interface for testing purposes.
type MockMemory struct {
	Data []byte
}

func (m *MockMemory) Read(offset, length uint32) ([]byte, bool) {
	end := offset + length
	if end > uint32(len(m.Data)) {
		return nil, false
	}
	return m.Data[offset:end], true
}

func (m *MockMemory) Write(offset uint32, data []byte) bool {
	end := offset + uint32(len(data))
	if end > uint32(len(m.Data)) {
		return false
	}
	copy(m.Data[offset:end], data)
	return true
}

func TestMakeView(t *testing.T) {
	// Test when offset and length are zero (IsNil should be true)
	view := makeView(0, 0)
	require.True(t, view.IsNil)
	require.Equal(t, uint32(0), view.Data)
	require.Equal(t, uint32(0), view.Len)

	// Test normal case (IsNil should be false)
	view = makeView(100, 50)
	require.False(t, view.IsNil)
	require.Equal(t, uint32(100), view.Data)
	require.Equal(t, uint32(50), view.Len)
}

func TestConstructUnmanagedVector(t *testing.T) {
	// Test when IsNone is true
	uv := constructUnmanagedVector(true, 0, 0, 0)
	require.True(t, uv.IsNone)
	require.Equal(t, uint32(0), uv.Data)
	require.Equal(t, uint32(0), uv.Len)
	require.Equal(t, uint32(0), uv.Cap)

	// Test normal case
	uv = constructUnmanagedVector(false, 200, 100, 150)
	require.False(t, uv.IsNone)
	require.Equal(t, uint32(200), uv.Data)
	require.Equal(t, uint32(100), uv.Len)
	require.Equal(t, uint32(150), uv.Cap)
}

func TestNewUnmanagedVector(t *testing.T) {
	// Test when dataOffset, length, and capacity are zero (IsNone should be true)
	uv := newUnmanagedVector(0, 0, 0)
	require.True(t, uv.IsNone)
	require.Equal(t, uint32(0), uv.Data)
	require.Equal(t, uint32(0), uv.Len)
	require.Equal(t, uint32(0), uv.Cap)

	// Test normal case
	uv = newUnmanagedVector(300, 80, 100)
	require.False(t, uv.IsNone)
	require.Equal(t, uint32(300), uv.Data)
	require.Equal(t, uint32(80), uv.Len)
	require.Equal(t, uint32(100), uv.Cap)
}

func TestCopyAndDestroyUnmanagedVector(t *testing.T) {
	// Prepare mock memory
	memData := make([]byte, 500)
	for i := range memData {
		memData[i] = byte(i % 256)
	}
	mem := &MockMemory{Data: memData}

	// Test when IsNone is true
	uv := UnmanagedVector{IsNone: true}
	data, err := copyAndDestroyUnmanagedVector(uv, mem)
	require.NoError(t, err)
	require.Nil(t, data)

	// Test when Len is zero
	uv = UnmanagedVector{IsNone: false, Data: 100, Len: 0}
	data, err = copyAndDestroyUnmanagedVector(uv, mem)
	require.NoError(t, err)
	require.Equal(t, []byte{}, data)

	// Test normal case
	uv = UnmanagedVector{IsNone: false, Data: 100, Len: 50}
	data, err = copyAndDestroyUnmanagedVector(uv, mem)
	require.NoError(t, err)
	require.Equal(t, memData[100:150], data)

	// Test out-of-bounds read
	uv = UnmanagedVector{IsNone: false, Data: 480, Len: 30} // 480 + 30 = 510 > 500
	data, err = copyAndDestroyUnmanagedVector(uv, mem)
	require.Error(t, err)
	require.Nil(t, data)
}

func TestOptionalU64ToPtr(t *testing.T) {
	// Test when IsSome is false
	opt := OptionalU64{IsSome: false}
	ptr := optionalU64ToPtr(opt)
	require.Nil(t, ptr)

	// Test when IsSome is true
	opt = OptionalU64{IsSome: true, Value: 123456789}
	ptr = optionalU64ToPtr(opt)
	require.NotNil(t, ptr)
	require.Equal(t, uint64(123456789), *ptr)
}

func TestCopyU8Slice(t *testing.T) {
	// Prepare mock memory
	memData := make([]byte, 500)
	for i := range memData {
		memData[i] = byte((i * 2) % 256)
	}
	mem := &MockMemory{Data: memData}

	// Test when IsNone is true
	view := U8SliceView{IsNone: true}
	data, err := copyU8Slice(view, mem)
	require.NoError(t, err)
	require.Nil(t, data)

	// Test when Len is zero
	view = U8SliceView{IsNone: false, Data: 200, Len: 0}
	data, err = copyU8Slice(view, mem)
	require.NoError(t, err)
	require.Equal(t, []byte{}, data)

	// Test normal case
	view = U8SliceView{IsNone: false, Data: 200, Len: 50}
	data, err = copyU8Slice(view, mem)
	require.NoError(t, err)
	require.Equal(t, memData[200:250], data)

	// Test out-of-bounds read
	view = U8SliceView{IsNone: false, Data: 480, Len: 30} // 480 + 30 = 510 > 500
	data, err = copyU8Slice(view, mem)
	require.Error(t, err)
	require.Nil(t, data)
}

func TestMemory_ReadWrite(t *testing.T) {
	mem := &MockMemory{Data: make([]byte, 100)}
	dataToWrite := []byte{1, 2, 3, 4, 5}

	// Test Write within bounds
	ok := mem.Write(10, dataToWrite)
	require.True(t, ok)

	// Test Read within bounds
	readData, ok := mem.Read(10, uint32(len(dataToWrite)))
	require.True(t, ok)
	require.Equal(t, dataToWrite, readData)

	// Test out-of-bounds Write
	ok = mem.Write(98, dataToWrite) // 98 + 5 = 103 > 100
	require.False(t, ok)

	// Test out-of-bounds Read
	readData, ok = mem.Read(98, uint32(len(dataToWrite))) // 98 + 5 = 103 > 100
	require.False(t, ok)
	require.Nil(t, readData)
}
