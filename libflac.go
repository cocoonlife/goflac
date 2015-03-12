// Copyright 2015 Cocoon Alarm Ltd.
//
// See LICENSE file for terms and conditions.

// Package libflac provides Go bindings to the libFLAC codec library.
package libflac

import (
	"errors"
	"fmt"
	"io"
	"runtime"
	"unsafe"
)

/*
#cgo LDFLAGS: -lFLAC
#include <stdlib.h>

#include "FLAC/stream_decoder.h"
#include "FLAC/stream_encoder.h"

extern void
decoderErrorCallback_cgo(const FLAC__StreamDecoder *,
		         FLAC__StreamDecoderErrorStatus,
		         void *);
typedef void (*decoderErrorCallbackFn)(const FLAC__StreamDecoder *,
		                       FLAC__StreamDecoderErrorStatus,
		                       void *);


extern void
decoderMetadataCallback_cgo(const FLAC__StreamDecoder *,
			    const FLAC__StreamMetadata *,
			    void *);
typedef void (*decoderMetadataCallbackFn)(const FLAC__StreamDecoder *,
		                          FLAC__StreamMetadata *,
		                          void *);

extern FLAC__StreamDecoderWriteStatus
decoderWriteCallback_cgo(const FLAC__StreamDecoder *,
		         const FLAC__Frame *,
		         const FLAC__int32 **,
		         void *);
typedef FLAC__StreamDecoderWriteStatus (*decoderWriteCallbackFn)(const FLAC__StreamDecoder *,
                                        const FLAC__Frame *,
		                        const FLAC__int32 **,
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
		(C.decoderWriteCallbackFn)(unsafe.Pointer(C.decoderWriteCallback_cgo)),
		(C.decoderMetadataCallbackFn)(unsafe.Pointer(C.decoderMetadataCallback_cgo)),
		(C.decoderErrorCallbackFn)(unsafe.Pointer(C.decoderErrorCallback_cgo)),
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

// Close closes a decoder and frees the resources.
func (d *Decoder) Close() {
	if d.d != nil {
		C.FLAC__stream_decoder_delete(d.d)
		d.d = nil
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

// WriteFrame writes a frame of audio data to the encoder.
func (e *Encoder) WriteFrame(f Frame) (err error) {
	if f.Channels != e.Channels || f.Depth != e.Depth || f.Rate != e.Rate {
		return errors.New("frame type does not match encoder")
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
	runtime.SetFinalizer(e, nil)
}
