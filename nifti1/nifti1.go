// nifti1 contains methods to read nifti1 files.
//
// Based on the official definition of the nifti1 header,
// https://nifti.nimh.nih.gov/pub/dist/src/niftilib/nifti1.h

package nifti1

// We use the official nifti1 header to access certain variables (e.g., datatype codes).

// #include "nifti1.h"
import "C"
import (
	"bytes"
	"encoding/binary"
	"fmt"
	"reflect"
	"strings"
)

// Header defines the structure of the Nifti1 header.
//
// Type translation from nifti1 C header to golang:
//
// C     Go
// -------------
// int   int32
// float float32
// short int16
// char  int8
type Header struct {
	SizeOfHdr          int32    // Must be 348
	UnusedDataType     [10]int8 // Unused
	UnusedDbName       [18]int8 // Unused
	UnusedExtents      int32    // Unused
	UnusedSessionError int16    // Unused
	UnusedRegular      int8     // Unused
	DimInfo            int8     // MRI slice ordering

	Dim           [8]int16   // Data array dimenions
	IntentP1      float32    // 1st intent parameter
	IntentP2      float32    // 2nd intent parameter
	IntentP3      float32    // 3rd intent parameter
	IntentCode    int16      // NIFTI_INTENT_* code
	DataType      int16      // Defines data type
	BitPix        int16      // Number bits/voxel
	SliceStart    int16      // First slice index
	PixDim        [8]float32 // Grid spacing
	VoxOffset     float32    // Offset into .nii file
	SclSlope      float32    // Data scaling: slope
	SclInter      float32    // Data scaling: offset
	SliceEnd      int16      // Last slice index
	SliceCode     int8       // Slice timing order
	XYZTUnits     int8       // Units of pixdim[1..4]
	CalMax        float32    // Max display intensity
	CalMin        float32    // Min display intensity
	SliceDuration float32    // Time for 1 slice
	TOffset       float32    // Time axis shift
	UnusedGlmax   int32      // Unused
	UnusedGlmin   int32      // Unused

	Descrip [80]int8 // Any text you like
	AuxFile [24]int8 // Auxiliary filename

	QFormCode int16 // NIFTI_XFORM_* code
	SFormCode int16 // NIFTI_XFORM_* code

	QuaternB float32 // Quaternion b params
	QuaternC float32 // Quaternion c params
	QuaternD float32 // Quaternion d params
	QOffsetX float32 // Quaternion x shift
	QOffsetY float32 // Quaternion y shift
	QOffsetZ float32 // Quaternion z shift

	SRowX [4]float32 // 1st row affine transform
	SRowY [4]float32 // 2nd row affine transform
	SRowZ [4]float32 // 3rd row affine transform

	IntentName [16]int8 // 'name' or meaning of data

	Magic [4]int8 // Must be "ni1\0" or "n+1\0"
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

const headerSize = 352
const minHeaderSize = 348

type mat44 struct {
	m [4][4]float32
}

type mat33 struct {
	m [3][3]float32
}

// Image is a high level image storage struct.
type Image struct {
	NDim     int16    // last dimension greater than 1 (1..7)
	Nx       int16    // dimensions of grid array
	Ny       int16    // dimensions of grid array
	Nz       int16    // dimensions of grid array
	Nt       int16    // dimensions of grid array
	Nu       int16    // dimensions of grid array
	Nv       int16    // dimensions of grid array
	Nw       int16    // dimensions of grid array
	Dim      [8]int16 // dim[0] = ndim, dim[1] = nx, etc
	NVox     int32    // number of voxels = nx*ny*nz*...*nw
	NByPer   int32    // bytes per voxel, matches datatype
	DataType int32    // type of data in voxels: DT_* code

	Dx     float32    // grid spacings
	Dy     float32    // grid spacings
	Dz     float32    // grid spacings
	Dt     float32    // grid spacings
	Du     float32    // grid spacings
	Dv     float32    // grid spacings
	Dw     float32    // grid spacings
	PixDim [8]float32 // pixdim[1]=dx, etc

	SclSlope float32 // scaling parameter: slope
	SclInter float32 // scaling parameter: intercept

	CalMin float32 // calibration parameter: minimum
	CalMax float32 // calbiration parameter: maximum

	QFormCode int32 // codes for (x,y,z) space meaning
	SFormCode int32 // codes for (x,y,z) space meaning

	FreqDim  int32 // indeces (1,2,3, or 0) for MRI
	PhaseDim int32 // directions in dim[]/pixdim[]
	SliceDim int32 // directions in dim[]/pixdim[]

	SliceCode     int32   // code for slice timing pattern
	SliceStart    int32   // index for start of slices
	SliceEnd      int32   // index for end of slices
	SliceDuration float32 // time between individual slices

	// Quaternion transform parameters
	// [when writing a dataset, these are used for qform, NOT qto_xyz]
	QuaternB, QuaternC, QuaternD, QOffsetX, QOffsetY, QOffsetZ, QFac float32

	QtoXYZ mat44 // qform: transform (i,j,k) to (x,y,z)
	QtoIJK mat44 // qform: transform (x,y,z) to (i,j,k)

	StoXYZ mat44 // sform: transform (i,j,k) to (x,y,z)
	StoIJK mat44 // sform: transform (x,y,z) to (i,j,k)

	TOffset float32 // time coordinate offset

	XYZUnits  int32 // dx,dy,dz units: NIFTI_UNITS_* code
	TimeUnits int32 // dt units: NIFTI_UNITS_* code

	// 0==Analyze, 1==NIFTI-1 (file), 2==NIFTI-1 (2 files), 3==NIFTI-ASCII (1 file)
	NiftiType int32

	IntentCode int // statistic type (or something)

	IntentP1   float32   // intent parameters
	IntentP2   float32   // intent parameters
	IntentP3   float32   // intent parameters
	IntentName [16]int16 // optional description of intent data

	Descrip [80]int8 // optional text to describe dataset
	AuxFile [24]int8 // auxiliary filename

	FName       *int8 // header filename
	IName       *int8 // image filename
	INameOffset int32 // offset into IName where data start
	SwapSize    int32 // swap unit in image data (might be 0)
	ByteOrder   int32 // byte order on disk (MSB_ or LSB_FIRST)

	data []uint8 // slice of data: nbyper*nvox bytes

	NumExt int32 // number of extensions in extList

	// TODO(kaczmarj): Add extensions list struct
	// ommitting analyze75_orient
}

func check(e error) {
	if e != nil {
		panic(e)
	}
}

// ReadHeader reads a header and returns the byteorder of the file.
// Refer to this link for C implementation
// https://github.com/afni/afni/blob/master/src/nifti/niftilib/nifti1_io.c#L3948-L4042
func ReadHeader(b []byte) (Header, binary.ByteOrder) {
	h := Header{}
	var order binary.ByteOrder = binary.LittleEndian

	buf := bytes.NewReader(b)
	err := binary.Read(buf, order, &h)
	check(err)

	if (h.Dim[0] <= 0) && (h.Dim[0] > 7) {
		h = Header{}
		order = binary.BigEndian
		err = binary.Read(buf, order, &h)
		check(err)
	}

	if (h.Dim[0] <= 0) && (h.Dim[0] > 7) {
		panic("Cannot infer byte order of file based on Dim[0]: not in range [1, 7]")
	}

	validateHeader(h)

	return h, order
}

// Check https://github.com/afni/afni/blob/master/src/nifti/niftilib/nifti1_io.c#L4045-L4104
func validateHeader(h Header) {
	switch {
	case h.SizeOfHdr != minHeaderSize:
		panic("invalid header size for nifti-1")
	// Assert that file magic is 'n+1', meaning that the header and data are in
	// the same file.
	case h.Magic != [4]int8{110, 43, 49, 0}:
		panic("invalid file magic. data must be stored in same file as header")
	case h.DataType == C.DT_BINARY || h.DataType == C.DT_UNKNOWN:
		panic("bad datatype")
	}
}

// Refer to this on how to create an Image struct.
// https://github.com/afni/afni/blob/master/src/nifti/niftilib/nifti1_io.c#L5377-L5420
func convertHeaderToImage(h Header, order binary.ByteOrder) Image {
	img := Image{
		NDim: h.Dim[0],
		Nx:   h.Dim[1],
		Ny:   h.Dim[2],
		Nz:   h.Dim[3],
		Nt:   h.Dim[4],
		Nu:   h.Dim[5],
		Nv:   h.Dim[6],
		Nw:   h.Dim[7],
		Dim:  h.Dim,
	}

	return img
}

// ReadData reads data.
// TODO(kaczmarj): refer to this link for implementation details.
// https://github.com/afni/afni/blob/master/src/nifti/niftilib/nifti1_io.c#L3712-L3899
// Total number of bytes in the image is dim[dim[0]] * bitpix / 8
// This must correspond with the datatype field.
func ReadData(b []byte, h Header, order binary.ByteOrder) []int16 {
	var offset int

	if h.VoxOffset < headerSize {
		offset = headerSize
	} else {
		offset = int(h.VoxOffset)
	}

	// QUESTION: is there a better way to multiply int16 ?
	dataSize := int(h.Dim[1]) * int(h.Dim[2]) * int(h.Dim[3])

	data := make([]int16, dataSize)
	bSlice := b[offset:]

	fmt.Println("offset", offset)
	fmt.Println("dataSize", dataSize)
	fmt.Println("len slice", len(bSlice))

	buf := bytes.NewReader(bSlice)
	err := binary.Read(buf, order, &data) // TODO: this is still the bottleneck.
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
