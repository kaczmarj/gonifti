package util

import (
	"fmt"
	"os"
	"sync"
)

// ReadBytesParallel returns the bytes of a file.
// Based on
// https://github.com/kgrz/reading-files-in-go/blob/master/reading-chunkwise-multiple.go
func ReadBytesParallel(filepath string, bufferSize int) []byte {
	file, err := os.Open(filepath)
	if err != nil {
		panic(err)
	}
	defer file.Close()

	fileinfo, err := file.Stat()
	if err != nil {
		panic(err)
	}

	type chunk struct {
		bufsize int
		offset  int64
	}

	filesize := int(fileinfo.Size())
	// Number of go routines we need to spawn.
	concurrency := filesize / bufferSize
	// buffer sizes that each of the go routine below should use. ReadAt
	// returns an error if the buffer size is larger than the bytes returned
	// from the file.
	chunksizes := make([]chunk, concurrency)

	// All buffer sizes are the same in the normal case. Offsets depend on the
	// index. Second go routine should start at 100, for example, given our
	// buffer size of 100.
	for i := 0; i < concurrency; i++ {
		chunksizes[i].bufsize = bufferSize
		chunksizes[i].offset = int64(bufferSize * i)
	}

	// check for any left over bytes. Add the residual number of bytes as the
	// the last chunk size.
	if remainder := filesize % bufferSize; remainder != 0 {
		c := chunk{bufsize: remainder, offset: int64(concurrency * bufferSize)}
		concurrency++
		chunksizes = append(chunksizes, c)
	}

	var wg sync.WaitGroup
	wg.Add(concurrency)

	// Allocate array of bytes to hold all of the bytes in the file. The function
	// returns this array.
	allBytes := make([]byte, filesize)

	for i := 0; i < concurrency; i++ {
		go func(chunksizes []chunk, i int) {
			defer wg.Done()

			chunk := chunksizes[i]
			start := chunk.offset
			end := int(start) + chunk.bufsize
			bytesRead, err := file.ReadAt(allBytes[start:end], chunk.offset)
			if bytesRead < 1 {
				panic("Did not read any bytes")
			}
			if err != nil {
				fmt.Println(err)
				return
			}
		}(chunksizes, i)
	}
	wg.Wait()
	return allBytes
}
