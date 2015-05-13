// Copyright 2015 Cocoon Alarm Ltd.
//
// See LICENSE file for terms and conditions.

package libflac

/*

#include "FLAC/stream_decoder.h"
#include "FLAC/stream_encoder.h"

void
decoderErrorCallback_cgo(const FLAC__StreamDecoder *decoder,
		         FLAC__StreamDecoderErrorStatus status,
		         void *data)
{
     decoderErrorCallback(decoder, status, data);
}

void
decoderMetadataCallback_cgo(const FLAC__StreamDecoder *decoder,
			    const FLAC__StreamMetadata *metadata,
			    void *data)
{
    decoderMetadataCallback(decoder, metadata, data);
}

FLAC__StreamDecoderWriteStatus
decoderWriteCallback_cgo(const FLAC__StreamDecoder *decoder,
		         const FLAC__Frame *frame,
		         const FLAC__int32 **buffer,
		         void *data)
{
    return decoderWriteCallback(decoder, frame, buffer, data);
}

FLAC__StreamDecoderReadStatus
decoderReadCallback_cgo(const FLAC__StreamDecoder *decoder,
		        const FLAC__byte buffer[],
			size_t *bytes,
		        void *data)
{
    return decoderReadCallback(decoder, buffer, bytes, data);
}

FLAC__StreamEncoderWriteStatus
encoderWriteCallback_cgo(const FLAC__StreamEncoder *encoder,
			 const FLAC__byte buffer[],
			 size_t bytes, unsigned samples,
			 unsigned current_frame,
		         void *data)
{
    return encoderWriteCallback(encoder, buffer, bytes, samples, current_frame,
				data);
}

FLAC__StreamEncoderSeekStatus
encoderSeekCallback_cgo(const FLAC__StreamEncoder *encoder,
			FLAC__uint64 absolute_byte_offset,
		        void *data)
{
    return encoderSeekCallback(encoder, absolute_byte_offset, data);
}

FLAC__StreamEncoderTellStatus
encoderTellCallback_cgo(const FLAC__StreamEncoder *encoder,
			FLAC__uint64 *absolute_byte_offset,
		        void *data)
{
    return encoderTellCallback(encoder, absolute_byte_offset, data);
}

extern const char *
get_decoder_error_str(FLAC__StreamDecoderErrorStatus status)
{
     return FLAC__StreamDecoderErrorStatusString[status];
}

extern int
get_decoder_channels(FLAC__StreamMetadata *metadata)
{
     return metadata->data.stream_info.channels;
}

extern int
get_decoder_depth(FLAC__StreamMetadata *metadata)
{
     return metadata->data.stream_info.bits_per_sample;
}

extern int
get_decoder_rate(FLAC__StreamMetadata *metadata)
{
     return metadata->data.stream_info.sample_rate;
}

extern void
get_audio_samples(int32_t *output, const FLAC__int32 **input,
                  unsigned int blocksize, unsigned int channels)
{
    unsigned int i, j, samples = blocksize * channels;
    for (i = 0; i < blocksize; i++)
        for (j = 0; j < channels; j++)
            output[i * channels + j] = input[j][i];
}

*/
import "C"
