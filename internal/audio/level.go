package audio

import "math"

// Amplitude analysis for the status meter. None of this retains audio: a level
// is a single scalar derived from a buffer that is discarded immediately, and a
// short history of those scalars cannot reconstruct speech.

// Peak returns the loudest absolute amplitude in little-endian signed 16-bit
// PCM, normalised to 0..1. A trailing odd byte is ignored rather than treated
// as a sample.
//
// Peak rather than RMS: this drives a meter, and the Windows sound settings
// meter is a peak meter. The RMS of speech sits far below its peak, so an RMS
// meter reads much lower than the one the user is comparing it against.
func Peak(pcm []byte) float64 {
	samples := len(pcm) / 2
	loudest := 0.0
	for i := 0; i < samples; i++ {
		sample := float64(int16(uint16(pcm[i*2]) | uint16(pcm[i*2+1])<<8))
		if sample < 0 {
			sample = -sample
		}
		if sample > loudest {
			loudest = sample
		}
	}
	// 32768 rather than 32767: the negative extreme is what bounds the range.
	return math.Min(1, loudest/32768)
}

// LevelFloorDB is where the meter bottoms out. Room tone sits near it, so a
// quiet room shows an almost-empty meter rather than a dead one.
const LevelFloorDB = -48.0

// NormalizeLevel maps a 0..1 amplitude onto a 0..1 meter position through
// decibels, which is how loudness is perceived and how every audio meter
// behaves. Linear amplitude spends nearly all of speech in the bottom tenth of
// its range and so barely moves.
func NormalizeLevel(amplitude float64) float64 {
	if amplitude <= 0 {
		return 0
	}
	if amplitude > 1 {
		amplitude = 1
	}
	db := 20 * math.Log10(amplitude)
	if db <= LevelFloorDB {
		return 0
	}
	return math.Min(1, (db-LevelFloorDB)/-LevelFloorDB)
}

// Envelope smooths a noisy amplitude series. It rises almost immediately and
// falls slowly, which is what makes a meter read as a voice instead of as
// jitter: raw RMS twitches, and a twitching meter looks broken rather than
// responsive.
type Envelope struct {
	// Attack and Release are per-sample retention factors in 0..1. Lower
	// attack follows a rise faster; higher release decays more slowly.
	Attack  float64
	Release float64

	value float64
}

// Time constants for a speech meter, in seconds: near-instant onto a rise,
// and a fall slow enough to read as a voice trailing off rather than as a
// value being switched off.
const (
	envelopeAttackSeconds  = 0.02
	envelopeReleaseSeconds = 0.18
)

// NewEnvelopeAt returns an envelope for a meter refreshed at hz. Deriving the
// factors from the rate keeps the meter feeling identical whether it is fed 30
// times a second or 15: a fixed factor would decay twice as slowly when the
// caller halves its rate.
func NewEnvelopeAt(hz float64) Envelope {
	if hz <= 0 {
		return Envelope{}
	}
	return Envelope{
		Attack:  math.Exp(-1 / (hz * envelopeAttackSeconds)),
		Release: math.Exp(-1 / (hz * envelopeReleaseSeconds)),
	}
}

// NewEnvelope returns an envelope for the overlay's 30 Hz meter.
func NewEnvelope() Envelope { return NewEnvelopeAt(30) }

// Push folds one sample in and returns the smoothed level.
func (e *Envelope) Push(sample float64) float64 {
	if sample < 0 {
		sample = 0
	}
	factor := e.Release
	if sample > e.value {
		factor = e.Attack
	}
	if factor < 0 || factor >= 1 {
		factor = 0
	}
	e.value = e.value*factor + sample*(1-factor)
	return e.value
}

func (e *Envelope) Value() float64 { return e.value }

func (e *Envelope) Reset() { e.value = 0 }

// LevelRing keeps the most recent levels for a scrolling meter. It holds a
// fixed history so the meter shows the shape of the last moment of speech
// rather than a single instantaneous value.
type LevelRing struct {
	values []float64
	next   int
}

func NewLevelRing(size int) *LevelRing {
	if size < 1 {
		size = 1
	}
	return &LevelRing{values: make([]float64, size)}
}

func (r *LevelRing) Len() int { return len(r.values) }

func (r *LevelRing) Push(level float64) {
	if level < 0 {
		level = 0
	}
	if level > 1 {
		level = 1
	}
	r.values[r.next] = level
	r.next = (r.next + 1) % len(r.values)
}

// Snapshot copies the history oldest first into dst, which must hold Len
// entries. Returning the slice keeps the caller free of an allocation per
// frame.
func (r *LevelRing) Snapshot(dst []float64) []float64 {
	if len(dst) < len(r.values) {
		dst = make([]float64, len(r.values))
	}
	dst = dst[:len(r.values)]
	for i := range r.values {
		dst[i] = r.values[(r.next+i)%len(r.values)]
	}
	return dst
}

func (r *LevelRing) Reset() {
	for i := range r.values {
		r.values[i] = 0
	}
	r.next = 0
}
