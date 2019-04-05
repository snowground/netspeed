package protocol

import (
	"errors"
	"unsafe"
)

const (
	HEADER_SIG        = 0x11223311
	HEADER_FUNC_READ  = 0x00000001
	HEADER_FUNC_WRITE = 0x00000002
)

type Header struct {
	Sig     uint32
	Func    uint32
	DataLen uint32
}
type SliceMock struct {
	Addr uintptr
	Len  int
	Cap  int
}

func Data2header(data []byte, data_len int) (err error, header *Header) {
	if int(unsafe.Sizeof(*header)) != data_len {
		return errors.New("data too small"), nil
	}
	header = *(**Header)(unsafe.Pointer(&data))
	if header.Sig != HEADER_SIG {
		return errors.New("header.Sig"), nil
	}
	return nil, header
}
func Header2Data(header *Header) (data []byte) {
	Len := unsafe.Sizeof(*header)
	tmpBytes := &SliceMock{
		Addr: uintptr(unsafe.Pointer(header)),
		Cap:  int(Len),
		Len:  int(Len),
	}
	data = *(*[]byte)(unsafe.Pointer(tmpBytes))
	return data
}
