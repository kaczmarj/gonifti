package util

import (
	"bytes"
	"compress/gzip"
	"io/ioutil"
	"net/http"

	log "github.com/sirupsen/logrus"
)

// ReadBytes returns the contents of a file as an array of bytes. It accepts
// files compressed with gzip and uncompressed files.
func ReadBytes(filename string) ([]byte, error) {
	content, err := ioutil.ReadFile(filename)
	if err != nil {
		log.Fatal(err)
	}

	// This function usees at most 512 bytes.
	mime := http.DetectContentType(content[:512])

	log.WithFields(log.Fields{
		"mimeType": mime,
	}).Debug("Found mime type")

	// TODO(kaczmarj): Decompression seems to be the bottleneck for large files.
	// Inflate if file is gzipped.
	if mime == "application/x-gzip" {
		log.WithFields(log.Fields{
			"decompression": "gzip",
		}).Debug("Decompressing ...")
		// Overwrite array of compressed bytes with array of inflated bytes.
		content, err = inflateGzip(content)
		if err != nil {
			log.Fatal(err)
		}
	}

	return content, nil
}

// inflateGzip inflates a gzip compressed array of bytes.
func inflateGzip(b []byte) ([]byte, error) {
	br := bytes.NewReader(b)
	g, err := gzip.NewReader(br)
	if err != nil {
		return nil, err
	}
	defer g.Close()

	p, err := ioutil.ReadAll(g)
	if err != nil {
		return nil, err
	}

	return p, nil
}
