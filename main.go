package main

import (
	"os"

	"github.com/kaczmarj/gonifti/nifti1"
	"github.com/kaczmarj/gonifti/util"
	log "github.com/sirupsen/logrus"
)

func main() {

	log.SetLevel(log.DebugLevel)

	if len(os.Args) < 2 {
		log.Fatal("nifti filename must be provided")
	}
	filename := os.Args[1]

	// TODO(kaczmarj): move reading to nifti1.go
	allBytes, err := util.ReadBytes(filename)
	if err != nil {
		log.Fatal(err)
	}

	header, byteOrder := nifti1.ReadHeader(allBytes)

	image := nifti1.ConvertHeaderToImage(header, byteOrder)
	image.SetData(allBytes, header)

	log.WithFields(log.Fields{
		"dataLen": len(image.Data),
	}).Info("Length of byte data in volume")

}
