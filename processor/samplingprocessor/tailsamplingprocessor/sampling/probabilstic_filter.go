package sampling

import (
	tracepb "github.com/census-instrumentation/opencensus-proto/gen-go/trace/v1"
	"go.uber.org/zap"
)

const(
	numHashBuckets        = 0x4000 // Using a power of 2 to avoid division.
	bitMaskHashBuckets    = numHashBuckets - 1
	percentageScaleFactor = numHashBuckets / 100.0

)
// just being lazy here, this is all copy pasted from sampling processor

type probabilisticFilter struct {
	logger             *zap.Logger
	seed               uint32
	scaledSamplingRate uint32
}

func (p probabilisticFilter) OnLateArrivingSpans(earlyDecision Decision, spans []*tracepb.Span) error {
	p.logger.Debug("Triggering action for late arriving spans in probabilistic filter")
	return nil
}

func (p probabilisticFilter) Evaluate(traceID []byte, trace *TraceData) (Decision, error) {
	sampled := hash(traceID, p.seed)&bitMaskHashBuckets < p.scaledSamplingRate
	if sampled {
		return Sampled, nil
	}
	return NotSampled, nil
}

func (p probabilisticFilter) OnDroppedSpans(traceID []byte, trace *TraceData) (Decision, error) {
	p.logger.Debug("Triggering action for dropped spans in probabilistic filter")
	return NotSampled, nil
}

func NewProbabilisticFilter(logger *zap.Logger, percentage float32, seed uint32) PolicyEvaluator {
	return probabilisticFilter{logger:  logger, scaledSamplingRate: uint32(percentage * percentageScaleFactor), seed: seed}
}
// hash is a murmur3 hash function, see http://en.wikipedia.org/wiki/MurmurHash.
func hash(key []byte, seed uint32) (hash uint32) {
	const (
		c1 = 0xcc9e2d51
		c2 = 0x1b873593
		c3 = 0x85ebca6b
		c4 = 0xc2b2ae35
		r1 = 15
		r2 = 13
		m  = 5
		n  = 0xe6546b64
	)

	hash = seed
	iByte := 0
	for ; iByte+4 <= len(key); iByte += 4 {
		k := uint32(key[iByte]) | uint32(key[iByte+1])<<8 | uint32(key[iByte+2])<<16 | uint32(key[iByte+3])<<24
		k *= c1
		k = (k << r1) | (k >> (32 - r1))
		k *= c2
		hash ^= k
		hash = (hash << r2) | (hash >> (32 - r2))
		hash = hash*m + n
	}

	// TraceId and SpanId have lengths that are multiple of 4 so the code below is never expected to
	// be hit when sampling traces. However, it is preserved here to keep it as a correct murmur3 implementation.
	// This is enforced via tests.
	var remainingBytes uint32
	switch len(key) - iByte {
	case 3:
		remainingBytes += uint32(key[iByte+2]) << 16
		fallthrough
	case 2:
		remainingBytes += uint32(key[iByte+1]) << 8
		fallthrough
	case 1:
		remainingBytes += uint32(key[iByte])
		remainingBytes *= c1
		remainingBytes = (remainingBytes << r1) | (remainingBytes >> (32 - r1))
		remainingBytes = remainingBytes * c2
		hash ^= remainingBytes
	}

	hash ^= uint32(len(key))
	hash ^= hash >> 16
	hash *= c3
	hash ^= hash >> 13
	hash *= c4
	hash ^= hash >> 16

	return
}

