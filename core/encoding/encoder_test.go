package encoding_test

import (
	"crypto/rand"
	"log"
	"runtime"
	"testing"

	"github.com/0glabs/0g-data-avail/core"
	"github.com/0glabs/0g-data-avail/core/encoding"
	"github.com/0glabs/0g-data-avail/pkg/encoding/kzgEncoder"
	"github.com/stretchr/testify/assert"
)

var (
	enc core.Encoder

	gettysburgAddressBytes = []byte("Fourscore and seven years ago our fathers brought forth, on this continent, a new nation, conceived in liberty, and dedicated to the proposition that all men are created equal. Now we are engaged in a great civil war, testing whether that nation, or any nation so conceived, and so dedicated, can long endure. We are met on a great battle-field of that war. We have come to dedicate a portion of that field, as a final resting-place for those who here gave their lives, that that nation might live. It is altogether fitting and proper that we should do this. But, in a larger sense, we cannot dedicate, we cannot consecrate—we cannot hallow—this ground. The brave men, living and dead, who struggled here, have consecrated it far above our poor power to add or detract. The world will little note, nor long remember what we say here, but it can never forget what they did here. It is for us the living, rather, to be dedicated here to the unfinished work which they who fought here have thus far so nobly advanced. It is rather for us to be here dedicated to the great task remaining before us—that from these honored dead we take increased devotion to that cause for which they here gave the last full measure of devotion—that we here highly resolve that these dead shall not have died in vain—that this nation, under God, shall have a new birth of freedom, and that government of the people, by the people, for the people, shall not perish from the earth.")
)

func init() {
	var err error
	enc, err = makeTestEncoder()
	if err != nil {
		log.Fatal(err)
	}
}

// makeTestEncoder makes an encoder currently using the only supported backend.
func makeTestEncoder() (core.Encoder, error) {
	config := kzgEncoder.KzgConfig{
		G1Path:    "../../inabox/resources/kzg/g1.point.300000",
		G2Path:    "../../inabox/resources/kzg/g2.point.300000",
		CacheDir:  "../../inabox/resources/kzg/SRSTables",
		SRSOrder:  300000,
		NumWorker: uint64(runtime.GOMAXPROCS(0)),
	}

	return encoding.NewEncoder(encoding.EncoderConfig{KzgConfig: config})
}

func TestEncoder(t *testing.T) {
	params := core.EncodingParams{
		ChunkLength: 5,
		NumChunks:   5,
	}
	commitments, chunks, err := enc.Encode(gettysburgAddressBytes, params)
	assert.NoError(t, err)

	indices := []core.ChunkNumber{
		0, 1, 2, 3, 4, 5, 6, 7,
	}
	err = enc.VerifyChunks(chunks, indices, commitments, params)
	assert.NoError(t, err)
	err = enc.VerifyChunks(chunks, []core.ChunkNumber{
		7, 6, 5, 4, 3, 2, 1, 0,
	}, commitments, params)
	assert.Error(t, err)

	maxInputSize := uint64(len(gettysburgAddressBytes))
	decoded, err := enc.Decode(chunks, indices, params, maxInputSize)
	assert.NoError(t, err)
	assert.Equal(t, gettysburgAddressBytes, decoded)

	// shuffle chunks
	tmp := chunks[2]
	chunks[2] = chunks[5]
	chunks[5] = tmp
	indices = []core.ChunkNumber{
		0, 1, 5, 3, 4, 2, 6, 7,
	}

	err = enc.VerifyChunks(chunks, indices, commitments, params)
	assert.NoError(t, err)

	decoded, err = enc.Decode(chunks, indices, params, maxInputSize)
	assert.NoError(t, err)
	assert.Equal(t, gettysburgAddressBytes, decoded)
}

// Ballpark number for 400KiB blob encoding
//
// goos: darwin
// goarch: arm64
// pkg: github.com/0glabs/0g-data-avail/core/encoding
// BenchmarkEncode-12    	       1	2421900583 ns/op
func BenchmarkEncode(b *testing.B) {
	params := core.EncodingParams{
		ChunkLength: 512,
		NumChunks:   256,
	}
	blobSize := 400 * 1024
	numSamples := 30
	blobs := make([][]byte, numSamples)
	for i := 0; i < numSamples; i++ {
		blob := make([]byte, blobSize)
		_, _ = rand.Read(blob)
		blobs[i] = blob
	}

	// Warm up the encoder: ensures that all SRS tables are loaded so these aren't included in the benchmark.
	_, _, _ = enc.Encode(blobs[0], params)
	b.ResetTimer()

	for i := 0; i < b.N; i++ {
		_, _, _ = enc.Encode(blobs[i%numSamples], params)
	}
}
