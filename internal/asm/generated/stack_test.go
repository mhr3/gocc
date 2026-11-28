package generated_test

import (
	"bytes"
	"encoding/binary"
	"fmt"
	"testing"
	"unsafe"

	"github.com/mhr3/gocc/internal/asm/generated"
)

// The functions of testdata/test_stack.c each force one use of the stack: a
// vector spilled with an aligned move, a vector operand read from the stack,
// a pointer into the frame handed to a helper, a frame the compiler aligned
// itself, a frame of several pages. gocc has to fit all of that into a Go
// frame, which is only 8-byte aligned on amd64, sits behind a stack check
// and moves when the stack grows.
//
// The tests below call them at the bottom of a chain of Go frames on a
// goroutine of its own. Sweeping the sizes and the number of those frames
// runs the functions with the stack pointer at every alignment and next to
// the end of the stack, where entering them makes the stack grow and move.

// stackVariantID names a translation of test_stack.c, which is to say an
// instruction set. The architecture files define the ones they have.
type stackVariantID int

// stackVariant is one translation of test_stack.c.
type stackVariant struct {
	id        stackVariantID
	name      string
	width     int  // bytes in a vector
	available bool // whether this machine can run it
}

// stackFixtureID names a function of test_stack.c.
type stackFixtureID int

const (
	fixtureSpill stackFixtureID = iota
	fixtureOperand
	fixtureSelect
	fixtureConsts
	fixturePointer
	fixtureWalk
	fixtureOveraligned
	fixtureFrame
)

// stackFixture is one function of test_stack.c with its Go equivalent.
type stackFixture struct {
	id   stackFixtureID
	name string
	ref  func(data []byte, width int) uint64
	// grows: the function checks the stack on every instruction set, so
	// entering it near the end of the stack moves the stack. The others are
	// leaves with a frame small enough for the Go assembler to drop the check.
	grows bool
}

var stackFixtures = []stackFixture{
	{fixtureSpill, "spill", refSpill, false},
	{fixtureOperand, "operand", refOperand, false},
	{fixtureSelect, "select", refSelect, false},
	{fixtureConsts, "consts", refConsts, false},
	{fixturePointer, "pointer", refPointer, true},
	{fixtureWalk, "walk", refWalk, true},
	{fixtureOveraligned, "overaligned", refOveraligned, true},
	{fixtureFrame, "frame", refFrame, true},
}

var stackK = [8]uint64{
	0x9e3779b97f4a7c15, 0xbf58476d1ce4e5b9, 0x94d049bb133111eb, 0x2545f4914f6cdd1d,
	0xd6e8feb86659fd93, 0xa0761d6478bd642f, 0xe7037ed1a0b428db, 0x8ebc6af09c88c6e3,
}

// lanes is a vector of 64-bit lanes; only the first width/8 are in use.
type lanes [8]uint64

func loadLanes(data []byte, width int) (v lanes) {
	for i := range width / 8 {
		v[i] = binary.LittleEndian.Uint64(data[8*i:])
	}
	return v
}

func (v lanes) fold(width int) (r uint64) {
	for i := range width / 8 {
		r ^= v[i] * uint64(2*i+1)
	}
	return r
}

func (v lanes) add(w lanes) lanes {
	for i := range v {
		v[i] += w[i]
	}
	return v
}

func (v lanes) xor(w lanes) lanes {
	for i := range v {
		v[i] ^= w[i]
	}
	return v
}

func refSpill(data []byte, width int) uint64 {
	var acc lanes
	for i := 0; i+width <= len(data); i += width {
		acc = acc.add(loadLanes(data[i:], width))
	}
	return acc.fold(width)
}

func refOperand(data []byte, width int) uint64 {
	var acc, mask lanes
	if len(data) >= width {
		mask = loadLanes(data, width)
	}
	for i := 0; i+width <= len(data); i += width {
		acc = acc.add(loadLanes(data[i:], width)).xor(mask)
	}
	return acc.fold(width)
}

func refSelect(data []byte, width int) uint64 {
	limit := make([]byte, width)
	other := make([]byte, width)
	picked := make([]byte, width)
	for i := range width / 8 {
		binary.LittleEndian.PutUint64(other[8*i:], stackK[i])
	}
	if len(data) >= width {
		copy(limit, data)
	}

	var acc lanes
	for i := 0; i+width <= len(data); i += width {
		for j, b := range data[i : i+width] {
			picked[j] = other[j]
			if b > limit[j] {
				picked[j] = b
			}
		}
		acc = acc.add(loadLanes(picked, width))
	}
	return acc.fold(width)
}

func refConsts(data []byte, width int) uint64 {
	table := [2]lanes{
		{stackK[7], stackK[6], stackK[5], stackK[4], stackK[3], stackK[2], stackK[1], stackK[0]},
		{stackK[0], stackK[2], stackK[4], stackK[6], stackK[1], stackK[3], stackK[5], stackK[7]},
	}

	var acc lanes
	for i := 0; i+width <= len(data); i += width {
		acc = acc.add(loadLanes(data[i:], width)).xor(lanes(stackK)).xor(table[i/width%2])
	}
	return acc.fold(width)
}

func refPointer(data []byte, width int) uint64 {
	var s0 lanes
	s1 := lanes(stackK)
	for i := 0; i+width <= len(data); i += width {
		s0 = s0.add(loadLanes(data[i:], width))
		s1 = s1.xor(s0)
	}
	return s0.fold(width) + s1.fold(width)
}

func refWalk(data []byte, width int) uint64 {
	var ring [8]lanes
	for i := range ring {
		for j := range ring[i] {
			ring[i][j] = stackK[j] * uint64(i+1)
		}
	}
	for i := 0; i+width <= len(data); i += width {
		slot := (int(data[i]) ^ i/width) & 7
		ring[slot] = ring[slot].add(loadLanes(data[i:], width))
	}

	var acc lanes
	for i := range ring {
		acc = acc.xor(ring[i])
	}
	return acc.fold(width)
}

func refOveraligned(data []byte, width int) uint64 {
	b0 := lanes(stackK)
	var b1 lanes
	for i := 0; i+2*width <= len(data); i += 2 * width {
		b0 = b0.add(loadLanes(data[i:], width))
		b1 = b1.xor(b0).xor(loadLanes(data[i+width:], width))
	}
	return b0.fold(width) ^ b1.fold(width)
}

func refFrame(data []byte, _ int) uint64 {
	var buf [6000]byte
	for i := range buf {
		buf[i] = byte(i * 7)
	}
	for i, b := range data {
		buf[i%6000] ^= b
		buf[5999-i%6000] += b
	}

	scramble := func(a, b uint64) uint64 {
		t := a * stackK[1]
		return (t ^ t>>29) + b
	}
	a := scramble(uint64(buf[0]), uint64(len(data)))
	b := scramble(uint64(buf[5999]), a)
	c := scramble(uint64(buf[1]), b)
	d := scramble(uint64(buf[5998]), c)
	e := scramble(uint64(buf[2]), d)
	f := scramble(uint64(buf[5997]), e)
	g := scramble(uint64(buf[3]), f)
	h := scramble(uint64(buf[5996]), g)
	k := scramble(uint64(buf[4]), h)

	sum := a ^ b*3 ^ c*5 ^ d*7 ^ e*11 ^ f*13 ^ g*17 ^ h*19 ^ k*23
	for _, b := range buf {
		sum = sum*31 + uint64(b)
	}
	return sum
}

// stackInputMax is the longest input the calling frame can hold.
const stackInputMax = 320

// stackCall is one call of a fixture at the bottom of a chain of frames.
type stackCall struct {
	variant stackVariantID
	fixture stackFixtureID
	data    []byte
	// onStack has the input copied into the frame that calls the fixture,
	// which leaves the runtime to relocate its argument when the stack moves.
	onStack bool

	got     uint64
	align   uintptr // the place of the calling frame, modulo 64
	moved   bool    // the stack moved under the fixture
	corrupt bool    // a frame above the fixture was written to
	// lostFramePointer: the frame pointer that the Go function calling the
	// fixture saved on the stack came back as something else
	lostFramePointer bool
}

// stackFrameKinds is the number of frame sizes stackDescend picks from.
const stackFrameKinds = 8

// stackDescend adds the frames picked by layout, a byte each, and calls the
// fixture below them.
func stackDescend(c *stackCall, layout []byte) {
	if len(layout) == 0 {
		stackLeaf(c)
		return
	}

	rest := layout[1:]
	switch layout[0] % stackFrameKinds {
	case 0:
		stackFrame[[8]byte](c, rest)
	case 1:
		stackFrame[[16]byte](c, rest)
	case 2:
		stackFrame[[24]byte](c, rest)
	case 3:
		stackFrame[[40]byte](c, rest)
	case 4:
		stackFrame[[56]byte](c, rest)
	case 5:
		stackFrame[[72]byte](c, rest)
	case 6:
		stackFrame[[200]byte](c, rest)
	case 7:
		stackFrame[[1000]byte](c, rest)
	}
}

// stackFrame is a frame holding a canary the size of T, which has to survive
// whatever runs below it.
//
//go:noinline
func stackFrame[T any](c *stackCall, layout []byte) {
	var pad T
	canary := unsafe.Slice((*byte)(unsafe.Pointer(&pad)), unsafe.Sizeof(pad))
	seed := byte(len(layout))

	stackCanaryFill(canary, seed)
	stackDescend(c, layout)
	if !stackCanaryIntact(canary, seed) {
		c.corrupt = true
	}
}

//go:noinline
func stackCanaryFill(canary []byte, seed byte) {
	for i := range canary {
		canary[i] = seed + byte(i)
	}
}

//go:noinline
func stackCanaryIntact(canary []byte, seed byte) bool {
	for i := range canary {
		if canary[i] != seed+byte(i) {
			return false
		}
	}
	return true
}

//go:noinline
func stackLeaf(c *stackCall) {
	var buf [stackInputMax]byte

	data := c.data
	if c.onStack {
		data = buf[:copy(buf[:], data)]
	}

	// the frame is somewhere else once the stack has moved, and its frame
	// pointer with it
	frame := uintptr(unsafe.Pointer(&buf))
	c.align = frame % 64
	framePointer := generated.FramePointer() - frame

	c.got = stackKernel(c.variant, c.fixture, data)

	c.moved = uintptr(unsafe.Pointer(&buf)) != frame
	c.lostFramePointer = generated.FramePointer()-uintptr(unsafe.Pointer(&buf)) != framePointer
}

// run calls the fixture below the frames of layout on a new goroutine, which
// starts with a small stack.
func (c *stackCall) run(layout []byte) {
	done := make(chan struct{})
	go func() {
		defer close(done)
		stackDescend(c, layout)
	}()
	<-done
}

func stackInputs(width int) [][]byte {
	pattern := func(n int) []byte {
		data := make([]byte, n)
		x := uint32(n)*2654435761 + 1
		for i := range data {
			x = x*1664525 + 1013904223
			data[i] = byte(x >> 24)
		}
		return data
	}
	return [][]byte{
		pattern(0),
		pattern(width - 1),
		pattern(2 * width),
		pattern(4*width + width/2 + 3),
	}
}

func TestStackFixtures(t *testing.T) {
	for _, variant := range stackVariants {
		t.Run(variant.name, func(t *testing.T) {
			if !variant.available {
				t.Skipf("this machine cannot run %s code", variant.name)
			}
			inputs := stackInputs(variant.width)

			for _, fixture := range stackFixtures {
				t.Run(fixture.name, func(t *testing.T) {
					var calls, moves int
					aligns := map[uintptr]bool{}

					for kind := range byte(stackFrameKinds) {
						// frames of one size, one more at a time: this walks
						// the fixture towards the end of the stack in steps
						// smaller than its own frame
						for depth := 0; depth <= 96; depth++ {
							layout := bytes.Repeat([]byte{kind}, depth)

							for _, data := range inputs {
								for _, onStack := range []bool{false, true} {
									c := stackCall{variant: variant.id, fixture: fixture.id, data: data, onStack: onStack}
									c.run(layout)

									where := fmt.Sprintf("%d bytes, on stack: %v, below %d frames of kind %d", len(data), onStack, depth, kind)
									if c.corrupt {
										t.Fatalf("a frame above the function was written to (%s)", where)
									}
									if c.lostFramePointer {
										t.Fatalf("the frame pointer of the calling function was written over (%s)", where)
									}
									if want := fixture.ref(data, variant.width); c.got != want {
										t.Fatalf("got %#x, want %#x (%s)", c.got, want, where)
									}

									calls++
									if c.moved {
										moves++
									}
									aligns[c.align] = true
								}
							}
						}
					}

					t.Logf("%d calls, the stack moved under the function in %d, %d placements of the frame modulo 64", calls, moves, len(aligns))
					if fixture.grows && moves == 0 {
						t.Errorf("the stack never moved under the function: the sweep does not reach its stack check")
					}
					if len(aligns) < stackPlacements {
						t.Errorf("the frame was placed at %d addresses modulo 64, want %d", len(aligns), stackPlacements)
					}
				})
			}
		})
	}
}
