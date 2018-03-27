package main

import (
	"fmt"
	"log"
	"os"

	"github.com/kaczmarj/gonifti/nifti1"
	"github.com/kaczmarj/gonifti/util"
)

func main() {

	if len(os.Args) < 2 {
		log.Fatal("nifti filename must be provided")
	}
	filepath := os.Args[1]

	// TODO(kaczmarj): move reading to nifti1.go
	allBytes := util.ReadBytesParallel(filepath, 8192)

	header, byteOrder := nifti1.ReadHeader(allBytes)
	fmt.Println(header)

	data := nifti1.ReadData(allBytes, header, byteOrder)

	fmt.Println(len(data))
}
