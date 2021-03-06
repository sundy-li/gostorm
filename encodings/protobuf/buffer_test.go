package protobuf

import (
	"runtime"
	"testing"
)

func Assert(test *testing.T, expression bool) {
	if !expression {
		_, filename, linenumber, ok := runtime.Caller(1)
		if !ok {
			panic("could not get call stack")
		}
		test.Fatalf("%v, line %v", filename, linenumber)
	}
}

func testallocator(t *testing.T, allocator Allocator) {
	for k := 0; k < (256 << 10); k++ {
		buffer := allocator.New(k + 1)
		Assert(t, buffer != nil)
		Assert(t, len(buffer) == (k+1))
		allocator.Dispose(buffer)
	}
}

func TestHeapAllocator(t *testing.T) {
	testallocator(t, NewAllocatorHeap())
}

func TestMappedAllocator(t *testing.T) {
	testallocator(t, NewAllocatorMapped())
}

func benchallocator(b *testing.B, allocator Allocator, size int) {
	b.ResetTimer()
	b.StartTimer()
	for k := 0; k < b.N; k++ {
		buffer := allocator.New(size)
		allocator.Dispose(buffer)
	}
	b.StopTimer()
	b.SetBytes(int64(size))
}

func BenchmarkHeapAllocator1K(b *testing.B) {
	benchallocator(b, NewAllocatorHeap(), 1<<10)
}

func BenchmarkHeapAllocator4K(b *testing.B) {
	benchallocator(b, NewAllocatorHeap(), 4<<10)
}

func BenchmarkHeapAllocator128K(b *testing.B) {
	benchallocator(b, NewAllocatorHeap(), 128<<10)
}

func BenchmarkMappedAllocator1K(b *testing.B) {
	benchallocator(b, NewAllocatorMapped(), 1<<10)
}

func BenchmarkMappedAllocator4K(b *testing.B) {
	benchallocator(b, NewAllocatorMapped(), 4<<10)
}

func BenchmarkMappedAllocator128K(b *testing.B) {
	benchallocator(b, NewAllocatorMapped(), 128<<10)
}

func TestSinglePool(t *testing.T) {
	const size = 4 << 10
	pool := NewBufferPoolSingle(NewAllocatorMapped())
	Assert(t, pool != nil)
	Assert(t, pool.(*singlePool).cached == nil)
	buffer := pool.New(size)
	Assert(t, len(buffer) == size)
	Assert(t, pool.(*singlePool).cached == nil)
	pool.Dispose(buffer)
	Assert(t, len(pool.(*singlePool).cached) == len(buffer))
	buffer2 := pool.New(size)
	Assert(t, len(buffer2) == size)
	Assert(t, pool.(*singlePool).cached == nil)
	pool.Dispose(buffer2)
	Assert(t, len(pool.(*singlePool).cached) == len(buffer2))
	buffer = pool.New(size + 1)
	Assert(t, len(buffer) == (size+1))
	Assert(t, pool.(*singlePool).cached == nil)
	pool.Dispose(buffer)
	Assert(t, len(pool.(*singlePool).cached) == len(buffer))
	pool.Close()
	Assert(t, pool.(*singlePool).cached == nil)
}

func min(a, b int) int {
	if a < b {
		return a
	}
	return b
}

func max(a, b int) int {
	if a > b {
		return a
	}
	return b
}

func TestFixedPool(t *testing.T) {
	const (
		size     = 4 << 10
		capacity = 32
	)
	pool := NewBufferPoolFixed(NewAllocatorHeap(), size, capacity)
	Assert(t, pool != nil)
	Assert(t, len(pool.(*fixedPool).free) == 0)
	Assert(t, cap(pool.(*fixedPool).free) == capacity)
	var buffers [][]byte
	for k := 0; k < 2*capacity; k++ {
		buffer := pool.New(size)
		Assert(t, len(buffer) == size)
		Assert(t, len(pool.(*fixedPool).free) == 0)
		Assert(t, cap(pool.(*fixedPool).free) == capacity)
		buffers = append(buffers, buffer)
	}
	for k, b := range buffers {
		pool.Dispose(b)
		Assert(t, len(pool.(*fixedPool).free) == min(k+1, capacity))
		Assert(t, cap(pool.(*fixedPool).free) == capacity)
	}
	buffers = nil
	for k := 1; k <= 2*capacity; k++ {
		buffer := pool.New(size)
		Assert(t, len(buffer) == size)
		Assert(t, len(pool.(*fixedPool).free) == max(0, capacity-k))
		Assert(t, cap(pool.(*fixedPool).free) == capacity)
		buffers = append(buffers, buffer)
	}
	for k, b := range buffers {
		pool.Dispose(b)
		Assert(t, len(pool.(*fixedPool).free) == min(k+1, capacity))
		Assert(t, cap(pool.(*fixedPool).free) == capacity)
	}
	pool.Close()
	Assert(t, len(pool.(*fixedPool).free) == 0)
	Assert(t, cap(pool.(*fixedPool).free) == capacity)
}

func benchbufferpool(b *testing.B, pool BufferPool, size int) {
	b.ResetTimer()
	b.StartTimer()
	for k := 0; k < b.N; k++ {
		buffer := pool.New(size)
		pool.Dispose(buffer)
	}
	b.StopTimer()
	b.SetBytes(int64(size))
}

func BenchmarkSinglePool128K(b *testing.B) {
	pool := NewBufferPoolSingle(NewAllocatorMapped())
	benchbufferpool(b, pool, 128<<10)
	pool.Close()
}

func BenchmarkFixedPool128K(b *testing.B) {
	const size = 128 << 10
	pool := NewBufferPoolFixed(NewAllocatorMapped(), size, 32)
	benchbufferpool(b, pool, size)
	pool.Close()
}
