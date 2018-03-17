// Methods to read Nifti-1 files.
//
// Based on the official definition of the nifti1 header,
// https://nifti.nimh.nih.gov/pub/dist/src/niftilib/nifti1.h

package main

import (
	"encoding/binary"
	"fmt"
	"os"
	"reflect"
	"strings"
)

// Header defines the structure of the Nifti1 header.
type Header struct {
	SizeofHdr          int32      // Must be 348
	UnusedDataType     [10]int8   // Unused
	UnusedDbName       [18]int8   // Unused
	UnusedExtents      int32      // Unused
	UnusedSessionError int16      // Unused
	UnusedRegular      int8       // Unused
	DimInfo            int8       // MRI slice ordering
	Dim                [8]int16   // Data array dimenions
	IntentP1           float32    // 1st intent parameter
	IntentP2           float32    // 2nd intent parameter
	IntentP3           float32    // 3rd intent parameter
	IntentCode         int16      // NIFTI_INTENT_* code
	Datatype           int16      // Defines data type
	Bitpix             int16      // Number bits/voxel
	SliceStart         int16      // First slice index
	Pixdim             [8]float32 // Grid spacing
	VoxOffset          float32    // Offset into .nii file
	SclSlope           float32    // Data scaling: slope
	SclInter           float32    // Data scaling: offset
	SliceEnd           int16      // Last slice index
	SliceCode          int8       // Slice timing order
	XyztUnits          int8       // Units of pixdim[1..4]
	CalMax             float32    // Max display intensity
	CalMin             float32    // Min display intensity
	SliceDuration      float32    // Time for 1 slice
	Toffset            float32    // Time axis shift
	UnusedGlmax        int32      // Unused
	UnusedGlmin        int32      // Unused
	Descrip            [80]int8   // Any text you like
	AuxFile            [24]int8   // Auxiliary filename
	QformCode          int16      // NIFTI_XFORM_* code
	SformCode          int16      // NIFTI_XFORM_* code
	QuaternB           float32    // Quaternion b params
	QuaternC           float32    // Quaternion c params
	QuaternD           float32    // Quaternion d params
	QoffsetX           float32    // Quaternion x shift
	QoffsetY           float32    // Quaternion y shift
	QoffsetZ           float32    // Quaternion z shift
	SrowX              [4]float32 // 1st row affine transform
	SrowY              [4]float32 // 2nd row affine transform
	SrowZ              [4]float32 // 3rd row affine transform
	IntentName         [16]int8   // 'name' or meaning of data
	Magic              [4]int8    // Must be "ni1\0" or "n+1\0"
}

const headerSize = 352
const minHeaderSize = 348

func check(e error) {
	if e != nil {
		panic(e)
	}
}

// Print Header information.
func (h Header) String() string {
	s := reflect.ValueOf(&h).Elem()
	typeOfT := s.Type()
	nField := s.NumField()
	strs := make([]string, nField)
	for i := 0; i < s.NumField(); i++ {
		f := s.Field(i)
		strs[i] = fmt.Sprintf("%d: %s %s = %v", i,
			typeOfT.Field(i).Name, f.Type(), f.Interface())
	}
	return strings.Join(strs[:], "\n")
}

// readHeader reads a header from filepath.
func readHeader(filepath string) (Header, binary.ByteOrder) {
	f, err := os.Open(filepath)
	check(err)
	h := Header{}
	var order binary.ByteOrder = binary.LittleEndian

	err = binary.Read(f, order, &h)
	check(err)

	if (0 >= h.Dim[0]) && (7 <= h.Dim[0]) {
		f.Seek(0, 0)
		h = Header{}
		order = binary.BigEndian
		err = binary.Read(f, order, &h)
		check(err)
	}
	if (0 >= h.Dim[0]) && (7 <= h.Dim[0]) {
		panic("Dim[0] is not in range [1, 7]")
	}
	return h, order
}

func validateHeader(h Header) {
	switch {
	case h.SizeofHdr != minHeaderSize:
		panic("invalid header size for nifti-1")
	case h.Magic != [4]int8{110, 43, 49, 0}:
		panic("invalid file magic. data must be stored in same file as header")
	}
}

func readData(filepath string, h Header, order binary.ByteOrder) []int16 {
	var offset float32

	if h.VoxOffset < headerSize {
		offset = headerSize
	} else {
		offset = h.VoxOffset
	}

	// QUESTION: is there a better way to multiply int16 ?
	dataSize := int(h.Dim[1]) * int(h.Dim[2]) * int(h.Dim[3])

	data := make([]int16, dataSize)

	f, err := os.Open(filepath)
	check(err)
	o, err := f.Seek(int64(offset), 0)
	if float32(o) < offset {
		panic("file has fewer bytes than offset requires")
	}
	check(err)
	err = binary.Read(f, order, &data) // TODO: this is the bottleneck.
	check(err)

	return data
}

func scaleData(data []int16, m float32, b float32) []float32 {
	dataScaled := make([]float32, len(data))
	for i, d := range data {
		dataScaled[i] = m*float32(d) + b
	}
	return dataScaled
}

func main() {
	if len(os.Args) < 2 {
		panic("nifti filename must be provided")
	}
	filepath := os.Args[1]

	header, byteorder := readHeader(filepath)
	validateHeader(header)

	data := readData(filepath, header, byteorder)

	// Scale data if necessary.
	// if header.SclSlope > 0 {
	// 	data = scaleData(data, header.SclSlope, header.SclInter)
	// }

	fmt.Println(header)
	fmt.Println(cap(data))
}
