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

	log "github.com/sirupsen/logrus"
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
	NDim     int    // last dimension greater than 1 (1..7)
	Nx       int    // dimensions of grid array
	Ny       int    // dimensions of grid array
	Nz       int    // dimensions of grid array
	Nt       int    // dimensions of grid array
	Nu       int    // dimensions of grid array
	Nv       int    // dimensions of grid array
	Nw       int    // dimensions of grid array
	Dim      [8]int // dim[0] = ndim, dim[1] = nx, etc
	NVox     int    // number of voxels = nx*ny*nz*...*nw
	NByPer   int    // bytes per voxel, matches datatype
	DataType int    // type of data in voxels: DT_* code

	Dx     float64    // grid spacings
	Dy     float64    // grid spacings
	Dz     float64    // grid spacings
	Dt     float64    // grid spacings
	Du     float64    // grid spacings
	Dv     float64    // grid spacings
	Dw     float64    // grid spacings
	PixDim [8]float64 // pixdim[1]=dx, etc

	SclSlope float64 // scaling parameter: slope
	SclInter float64 // scaling parameter: intercept

	CalMin float64 // calibration parameter: minimum
	CalMax float64 // calbiration parameter: maximum

	QFormCode int // codes for (x,y,z) space meaning
	SFormCode int // codes for (x,y,z) space meaning

	FreqDim  int // indeces (1,2,3, or 0) for MRI
	PhaseDim int // directions in dim[]/pixdim[]
	SliceDim int // directions in dim[]/pixdim[]

	SliceCode     int     // code for slice timing pattern
	SliceStart    int     // index for start of slices
	SliceEnd      int     // index for end of slices
	SliceDuration float64 // time between individual slices

	// Quaternion transform parameters
	// [when writing a dataset, these are used for qform, NOT qto_xyz]
	QuaternB, QuaternC, QuaternD, QOffsetX, QOffsetY, QOffsetZ, QFac float64

	QtoXYZ mat44 // qform: transform (i,j,k) to (x,y,z)
	QtoIJK mat44 // qform: transform (x,y,z) to (i,j,k)

	StoXYZ mat44 // sform: transform (i,j,k) to (x,y,z)
	StoIJK mat44 // sform: transform (x,y,z) to (i,j,k)

	TOffset float64 // time coordinate offset

	XYZUnits  int // dx,dy,dz units: NIFTI_UNITS_* code
	TimeUnits int // dt units: NIFTI_UNITS_* code

	// 0==Analyze, 1==NIFTI-1 (file), 2==NIFTI-1 (2 files), 3==NIFTI-ASCII (1 file)
	NiftiType int

	IntentCode int // statistic type (or something)

	IntentP1   float64 // intent parameters
	IntentP2   float64 // intent parameters
	IntentP3   float64 // intent parameters
	IntentName [16]int // optional description of intent data

	Descrip [80]int // optional text to describe dataset
	AuxFile [24]int // auxiliary filename

	FName       *int             // header filename
	IName       *int             // image filename
	INameOffset int              // offset into IName where data start
	SwapSize    int              // swap unit in image data (might be 0)
	ByteOrder   binary.ByteOrder // byte order on disk (MSB_ or LSB_FIRST)

	Data []byte // slice of data: nbyper*nvox bytes

	NumExt int // number of extensions in extList

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

	log.Debug("Reading header ...")
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

	log.WithFields(log.Fields{
		"byteOrder": order,
	}).Debug("Found byte order")

	return h, order
}

// Check https://github.com/afni/afni/blob/master/src/nifti/niftilib/nifti1_io.c#L4045-L4104
func validateHeader(h Header) {
	switch {

	case h.SizeOfHdr != minHeaderSize:
		log.WithFields(log.Fields{
			"cause":       "invalid header size",
			"headerSize":  h.SizeOfHdr,
			"headerValid": false,
		}).Fatal("Invalid header size for nifti1")

	// Assert that file magic is 'n+1', meaning that the header and data are in
	// the same file.
	case h.Magic != [4]int8{110, 43, 49, 0}:
		log.WithFields(log.Fields{
			"cause":       "invalid file magic",
			"headerValid": false,
		}).Fatal("Invalid file magic. Data must be stored in same file as header")

	case h.DataType == C.DT_BINARY || h.DataType == C.DT_UNKNOWN:
		log.WithFields(log.Fields{
			"cause":       "bad datatype",
			"headerValid": false,
			"dataType":    h.DataType,
		}).Fatal("Data type is invalid")
	}

	log.WithFields(log.Fields{
		"headerValid": true,
	}).Debug("Header is valid")
}

// ConvertHeaderToImage converts a header to an image.
// Refer to this on how to create an Image struct.
// https://github.com/afni/afni/blob/master/src/nifti/niftilib/nifti1_io.c#L5377-L5420
func ConvertHeaderToImage(h Header, order binary.ByteOrder) *Image {

	img := new(Image)

	img.NDim = int(h.Dim[0])
	img.Nx = int(h.Dim[1])
	img.Ny = int(h.Dim[2])
	img.Nz = int(h.Dim[3])
	img.Nt = int(h.Dim[4])
	img.Nu = int(h.Dim[5])
	img.Nv = int(h.Dim[6])
	img.Nw = int(h.Dim[7])
	img.ByteOrder = order

	for i := range img.Dim {
		img.Dim[i] = int(h.Dim[i])
	}

	return img
}

// SetData sets data into the Image struct. Operates in-place.
// TODO(kaczmarj): refer to this link for implementation details.
// https://github.com/afni/afni/blob/master/src/nifti/niftilib/nifti1_io.c#L3712-L3899
// Total number of bytes in the image is dim[dim[0]] * bitpix / 8
// This must correspond with the datatype field.
func (img *Image) SetData(b []byte, h Header) {

	timeDim := 1
	if img.Dim[4] > 0 {
		timeDim = img.Dim[4]
	}

	statDim := 1
	if img.Dim[5] > 0 {
		statDim = img.Dim[5]
	}

	var offset int
	if h.VoxOffset < headerSize {
		offset = headerSize
	} else {
		// It should be fine to cast from float32 to int16 here because the index
		// should always be integer-like.
		offset = int(h.VoxOffset)
	}

	dataSize := img.Dim[1] * img.Dim[2] * img.Dim[3] * timeDim * statDim * (int(h.BitPix) / 8)

	img.Data = b[offset : offset+dataSize]

}

// func scaleData(data []int16, m float32, b float32) []float32 {
// 	dataScaled := make([]float32, len(data))
// 	for i, d := range data {
// 		dataScaled[i] = m*float32(d) + b
// 	}
// 	return dataScaled
// }
