// Copyright 2015 Cocoon Alarm Ltd.
//
// See LICENSE file for terms and conditions.

// Package libflac provides Go bindings to the libFLAC codec library.
package libflac

import (
	"errors"
	"fmt"
	"io"
	"reflect"
	"runtime"
	"unsafe"
)

/*
#cgo pkg-config: flac
#include <stdlib.h>

#include "FLAC/stream_decoder.h"
#include "FLAC/stream_encoder.h"

extern void
decoderErrorCallback_cgo(const FLAC__StreamDecoder *,
		         FLAC__StreamDecoderErrorStatus,
		         void *);


extern void
decoderMetadataCallback_cgo(const FLAC__StreamDecoder *,
			    const FLAC__StreamMetadata *,
			    void *);

extern FLAC__StreamDecoderWriteStatus
decoderWriteCallback_cgo(const FLAC__StreamDecoder *,
		         const FLAC__Frame *,
		         const FLAC__int32 **,
		         void *);

FLAC__StreamDecoderReadStatus
decoderReadCallback_cgo(const FLAC__StreamDecoder *,
		        const FLAC__byte *,
			size_t *,
		        void *);

FLAC__StreamEncoderWriteStatus
encoderWriteCallback_cgo(const FLAC__StreamEncoder *,
			 const FLAC__byte *,
			 size_t, unsigned,
			 unsigned,
		         void *);

FLAC__StreamEncoderSeekStatus
encoderSeekCallback_cgo(const FLAC__StreamEncoder *,
			FLAC__uint64,
		        void *);

FLAC__StreamEncoderTellStatus
encoderTellCallback_cgo(const FLAC__StreamEncoder *,
			FLAC__uint64 *,
		        void *);

extern const char *
get_decoder_error_str(FLAC__StreamDecoderErrorStatus status);

extern int
get_decoder_channels(FLAC__StreamMetadata *metadata);

extern int
get_decoder_depth(FLAC__StreamMetadata *metadata);

extern int
get_decoder_rate(FLAC__StreamMetadata *metadata);

extern void
get_audio_samples(int32_t *output, const FLAC__int32 **input,
                  unsigned int blocksize, unsigned int channels);

*/
import "C"

type FlacWriter interface {
	io.Writer
	io.Closer
	io.Seeker
}

// Frame is an interleaved buffer of audio data with the specified parameters.
type Frame struct {
	Channels int
	Depth    int
	Rate     int
	Buffer   []int32
}

// Decoder is a FLAC decoder.
type Decoder struct {
	d        *C.FLAC__StreamDecoder
	reader   io.ReadCloser
	Channels int
	Depth    int
	Rate     int
	error    bool
	errorStr string
	frame    *Frame
}

// Encoder is a FLAC encoder.
type Encoder struct {
	e        *C.FLAC__StreamEncoder
	writer   FlacWriter
	Channels int
	Depth    int
	Rate     int
}

//export decoderErrorCallback
func decoderErrorCallback(d *C.FLAC__StreamDecoder, status C.FLAC__StreamDecoderErrorStatus, data unsafe.Pointer) {
	decoder := (*Decoder)(data)
	decoder.error = true
	decoder.errorStr = C.GoString(C.get_decoder_error_str(status))
}

//export decoderMetadataCallback
func decoderMetadataCallback(d *C.FLAC__StreamDecoder, metadata *C.FLAC__StreamMetadata, data unsafe.Pointer) {
	decoder := (*Decoder)(data)
	if metadata._type == C.FLAC__METADATA_TYPE_STREAMINFO {
		decoder.Channels = int(C.get_decoder_channels(metadata))
		decoder.Depth = int(C.get_decoder_depth(metadata))
		decoder.Rate = int(C.get_decoder_rate(metadata))
	}
}

//export decoderWriteCallback
func decoderWriteCallback(d *C.FLAC__StreamDecoder, frame *C.FLAC__Frame, buffer **C.FLAC__int32, data unsafe.Pointer) C.FLAC__StreamDecoderWriteStatus {
	decoder := (*Decoder)(data)
	blocksize := int(frame.header.blocksize)
	decoder.frame = new(Frame)
	f := decoder.frame
	f.Channels = decoder.Channels
	f.Depth = decoder.Depth
	f.Rate = decoder.Rate
	f.Buffer = make([]int32, blocksize*decoder.Channels)
	C.get_audio_samples((*C.int32_t)(&f.Buffer[0]), buffer, C.uint(blocksize), C.uint(decoder.Channels))
	return C.FLAC__STREAM_DECODER_WRITE_STATUS_CONTINUE
}

//export decoderReadCallback
func decoderReadCallback(d *C.FLAC__StreamDecoder, buffer *C.FLAC__byte, bytes *C.size_t, data unsafe.Pointer) C.FLAC__StreamDecoderReadStatus {
	decoder := (*Decoder)(data)
	numBytes := int(*bytes)
	if numBytes <= 0 {
		return C.FLAC__STREAM_DECODER_READ_STATUS_ABORT
	}
	hdr := reflect.SliceHeader{
		Data: uintptr(unsafe.Pointer(buffer)),
		Len:  numBytes,
		Cap:  numBytes,
	}
	buf := *(*[]byte)(unsafe.Pointer(&hdr))
	n, err := decoder.reader.Read(buf)
	*bytes = C.size_t(n)
	if err == io.EOF && n == 0 {
		return C.FLAC__STREAM_DECODER_READ_STATUS_END_OF_STREAM
	} else if err != nil && err != io.EOF {
		return C.FLAC__STREAM_DECODER_READ_STATUS_ABORT
	}
	return C.FLAC__STREAM_DECODER_READ_STATUS_CONTINUE
}

// NewDecoder creates a new Decoder object.
func NewDecoder(name string) (d *Decoder, err error) {
	d = new(Decoder)
	d.d = C.FLAC__stream_decoder_new()
	if d.d == nil {
		return nil, errors.New("failed to create decoder")
	}
	c := C.CString(name)
	defer C.free(unsafe.Pointer(c))
	runtime.SetFinalizer(d, (*Decoder).Close)
	status := C.FLAC__stream_decoder_init_file(d.d, c,
		(C.FLAC__StreamDecoderWriteCallback)(unsafe.Pointer(C.decoderWriteCallback_cgo)),
		(C.FLAC__StreamDecoderMetadataCallback)(unsafe.Pointer(C.decoderMetadataCallback_cgo)),
		(C.FLAC__StreamDecoderErrorCallback)(unsafe.Pointer(C.decoderErrorCallback_cgo)),
		unsafe.Pointer(d))
	if status != C.FLAC__STREAM_DECODER_INIT_STATUS_OK {
		return nil, errors.New("failed to open file")
	}
	ret := C.FLAC__stream_decoder_process_until_end_of_metadata(d.d)
	if ret == 0 || d.error == true || d.Channels == 0 {
		return nil, fmt.Errorf("failed to process metadata %s", d.errorStr)
	}
	return
}

// NewDecoderReader creates a new Decoder object from a Reader.
func NewDecoderReader(reader io.ReadCloser) (d *Decoder, err error) {
	d = new(Decoder)
	d.d = C.FLAC__stream_decoder_new()
	if d.d == nil {
		return nil, errors.New("failed to create decoder")
	}
	d.reader = reader
	runtime.SetFinalizer(d, (*Decoder).Close)
	status := C.FLAC__stream_decoder_init_stream(d.d,
		(C.FLAC__StreamDecoderReadCallback)(unsafe.Pointer(C.decoderReadCallback_cgo)),
		nil, nil, nil, nil,
		(C.FLAC__StreamDecoderWriteCallback)(unsafe.Pointer(C.decoderWriteCallback_cgo)),
		(C.FLAC__StreamDecoderMetadataCallback)(unsafe.Pointer(C.decoderMetadataCallback_cgo)),
		(C.FLAC__StreamDecoderErrorCallback)(unsafe.Pointer(C.decoderErrorCallback_cgo)),
		unsafe.Pointer(d))
	if status != C.FLAC__STREAM_DECODER_INIT_STATUS_OK {
		return nil, errors.New("failed to open stream")
	}
	ret := C.FLAC__stream_decoder_process_until_end_of_metadata(d.d)
	if ret == 0 || d.error == true || d.Channels == 0 {
		return nil, fmt.Errorf("failed to process metadata %s", d.errorStr)
	}
	return
}

// Close closes a decoder and frees the resources.
func (d *Decoder) Close() {
	if d.d != nil {
		C.FLAC__stream_decoder_delete(d.d)
		d.d = nil
	}
	if d.reader != nil {
		d.reader.Close()
	}
	runtime.SetFinalizer(d, nil)
}

// ReadFrame reads a frame of audio data from the decoder.
func (d *Decoder) ReadFrame() (f *Frame, err error) {
	ret := C.FLAC__stream_decoder_process_single(d.d)
	if ret == 0 || d.error == true {
		return nil, errors.New("error reading frame")
	}
	state := C.FLAC__stream_decoder_get_state(d.d)
	if state == C.FLAC__STREAM_DECODER_END_OF_STREAM {
		err = io.EOF
	}
	f = d.frame
	d.frame = nil
	return
}

// NewEncoder creates a new Encoder object.
func NewEncoder(name string, channels int, depth int, rate int) (e *Encoder, err error) {
	if channels == 0 {
		return nil, errors.New("channels must be greater than 0")
	}
	if !(depth == 16 || depth == 24) {
		return nil, errors.New("depth must be 16 or 24")
	}
	e = new(Encoder)
	e.e = C.FLAC__stream_encoder_new()
	if e.e == nil {
		return nil, errors.New("failed to create decoder")
	}
	c := C.CString(name)
	defer C.free(unsafe.Pointer(c))
	runtime.SetFinalizer(e, (*Encoder).Close)
	C.FLAC__stream_encoder_set_channels(e.e, C.uint(channels))
	C.FLAC__stream_encoder_set_bits_per_sample(e.e, C.uint(depth))
	C.FLAC__stream_encoder_set_sample_rate(e.e, C.uint(rate))
	status := C.FLAC__stream_encoder_init_file(e.e, c, nil, nil)
	if status != C.FLAC__STREAM_ENCODER_INIT_STATUS_OK {
		return nil, errors.New("failed to open file")
	}
	e.Channels = channels
	e.Depth = depth
	e.Rate = rate
	return
}

//export encoderWriteCallback
func encoderWriteCallback(e *C.FLAC__StreamEncoder, buffer *C.FLAC__byte, bytes C.size_t, samples, current_frame C.unsigned, data unsafe.Pointer) C.FLAC__StreamEncoderWriteStatus {
	encoder := (*Encoder)(data)
	numBytes := int(bytes)
	if numBytes <= 0 {
		return C.FLAC__STREAM_DECODER_READ_STATUS_ABORT
	}
	hdr := reflect.SliceHeader{
		Data: uintptr(unsafe.Pointer(buffer)),
		Len:  numBytes,
		Cap:  numBytes,
	}
	buf := *(*[]byte)(unsafe.Pointer(&hdr))
	_, err := encoder.writer.Write(buf)
	if err != nil {
		return C.FLAC__STREAM_ENCODER_WRITE_STATUS_FATAL_ERROR
	}
	return C.FLAC__STREAM_ENCODER_WRITE_STATUS_OK
}

//export encoderSeekCallback
func encoderSeekCallback(e *C.FLAC__StreamEncoder, absPos C.FLAC__uint64, data unsafe.Pointer) C.FLAC__StreamEncoderWriteStatus {
	encoder := (*Encoder)(data)
	_, err := encoder.writer.Seek(int64(absPos), 0)
	if err != nil {
		return C.FLAC__STREAM_ENCODER_SEEK_STATUS_ERROR
	}
	return C.FLAC__STREAM_ENCODER_SEEK_STATUS_OK
}

//export encoderTellCallback
func encoderTellCallback(e *C.FLAC__StreamEncoder, absPos *C.FLAC__uint64, data unsafe.Pointer) C.FLAC__StreamEncoderWriteStatus {
	encoder := (*Encoder)(data)
	newPos, err := encoder.writer.Seek(0, 1)
	if err != nil {
		return C.FLAC__STREAM_ENCODER_TELL_STATUS_ERROR
	}
	*absPos = C.FLAC__uint64(newPos)
	return C.FLAC__STREAM_ENCODER_TELL_STATUS_OK
}

// NewEncoderWriter creates a new Encoder object from a FlacWriter.
func NewEncoderWriter(writer FlacWriter, channels int, depth int, rate int) (e *Encoder, err error) {
	if channels == 0 {
		return nil, errors.New("channels must be greater than 0")
	}
	if !(depth == 16 || depth == 24) {
		return nil, errors.New("depth must be 16 or 24")
	}
	e = new(Encoder)
	e.e = C.FLAC__stream_encoder_new()
	if e.e == nil {
		return nil, errors.New("failed to create decoder")
	}
	e.writer = writer
	runtime.SetFinalizer(e, (*Encoder).Close)
	C.FLAC__stream_encoder_set_channels(e.e, C.uint(channels))
	C.FLAC__stream_encoder_set_bits_per_sample(e.e, C.uint(depth))
	C.FLAC__stream_encoder_set_sample_rate(e.e, C.uint(rate))
	status := C.FLAC__stream_encoder_init_stream(e.e,
		(C.FLAC__StreamEncoderWriteCallback)(unsafe.Pointer(C.encoderWriteCallback_cgo)),
		(C.FLAC__StreamEncoderSeekCallback)(unsafe.Pointer(C.encoderSeekCallback_cgo)),
		(C.FLAC__StreamEncoderTellCallback)(unsafe.Pointer(C.encoderTellCallback_cgo)),
		nil, unsafe.Pointer(e))
	if status != C.FLAC__STREAM_ENCODER_INIT_STATUS_OK {
		return nil, errors.New("failed to open file")
	}
	e.Channels = channels
	e.Depth = depth
	e.Rate = rate
	return
}

// WriteFrame writes a frame of audio data to the encoder.
func (e *Encoder) WriteFrame(f Frame) (err error) {
	if f.Channels != e.Channels || f.Depth != e.Depth || f.Rate != e.Rate {
		return errors.New("frame type does not match encoder")
	}
	if len(f.Buffer) == 0 {
		return
	}
	ret := C.FLAC__stream_encoder_process_interleaved(e.e, (*C.FLAC__int32)(&f.Buffer[0]), C.uint(len(f.Buffer)/e.Channels))
	if ret == 0 {
		return errors.New("error encoding frame")
	}
	return
}

// Close closes an encoder and frees the resources.
func (e *Encoder) Close() {
	if e.e != nil {
		C.FLAC__stream_encoder_finish(e.e)
		e.e = nil
	}
	if e.writer != nil {
		e.writer.Close()
	}
	runtime.SetFinalizer(e, nil)
}
