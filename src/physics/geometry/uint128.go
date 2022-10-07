package geometry

import (
	"fmt"
	"math"
	"math/big"
	"math/bits"
	"strconv"
)

type Uint128 struct {
	hi, lo uint64
}

// Uint128FromRaw is the complement to Uint128.Raw(); it creates an Uint128 from two
// uint64s representing the hi and lo bits.
func Uint128FromRaw(hi, lo uint64) Uint128 { return Uint128{hi: hi, lo: lo} }

func Uint128From64(v uint64) Uint128 { return Uint128{lo: v} }
func Uint128From32(v uint32) Uint128 { return Uint128{lo: uint64(v)} }
func Uint128From16(v uint16) Uint128 { return Uint128{lo: uint64(v)} }
func Uint128From8(v uint8) Uint128   { return Uint128{lo: uint64(v)} }
func Uint128FromUint(v uint) Uint128 { return Uint128{lo: uint64(v)} }

// Uint128FromI64 creates a Uint128 from an int64 if the conversion is possible, and
// sets inRange to false if not.
func Uint128FromI64(v int64) (out Uint128, inRange bool) {
	if v < 0 {
		return zeroUint128, false
	}
	return Uint128{lo: uint64(v)}, true
}

func MustUint128FromI64(v int64) (out Uint128) {
	out, inRange := Uint128FromI64(v)
	if !inRange {
		panic(fmt.Errorf("num: int64 %d was not in valid Uint128 range", v))
	}
	return out
}

// Uint128FromString creates a Uint128 from a string. Overflow truncates to MaxUint128
// and sets inRange to 'false'. Only decimal strings are currently supported.
func Uint128FromString(s string) (out Uint128, inRange bool, err error) {
	// This deliberately limits the scope of what we accept as input just in case
	// we decide to hand-roll our own fast decimal-only parser:
	b, ok := new(big.Int).SetString(s, 10)
	if !ok {
		return out, false, fmt.Errorf("num: u128 string %q invalid", s)
	}
	out, inRange = Uint128FromBigInt(b)
	return out, inRange, nil
}

func MustUint128FromString(s string) Uint128 {
	out, inRange, err := Uint128FromString(s)
	if err != nil {
		panic(err)
	}
	if !inRange {
		panic(fmt.Errorf("num: string %q was not in valid Uint128 range", s))
	}
	return out
}

// Uint128FromBigInt creates a Uint128 from a big.Int. Overflow truncates to MaxUint128
// and sets inRange to 'false'.
func Uint128FromBigInt(v *big.Int) (out Uint128, inRange bool) {
	if v.Sign() < 0 {
		return out, false
	}

	words := v.Bits()

	switch intSize {
	case 64:
		lw := len(words)
		switch lw {
		case 0:
			return Uint128{}, true
		case 1:
			return Uint128{lo: uint64(words[0])}, true
		case 2:
			return Uint128{hi: uint64(words[1]), lo: uint64(words[0])}, true
		default:
			return MaxUint128, false
		}

	case 32:
		lw := len(words)
		switch lw {
		case 0:
			return Uint128{}, true
		case 1:
			return Uint128{lo: uint64(words[0])}, true
		case 2:
			return Uint128{lo: (uint64(words[1]) << 32) | (uint64(words[0]))}, true
		case 3:
			return Uint128{hi: uint64(words[2]), lo: (uint64(words[1]) << 32) | (uint64(words[0]))}, true
		case 4:
			return Uint128{
				hi: (uint64(words[3]) << 32) | (uint64(words[2])),
				lo: (uint64(words[1]) << 32) | (uint64(words[0])),
			}, true
		default:
			return MaxUint128, false
		}

	default:
		panic("num: unsupported bit size")
	}
}

func MustUint128FromBigInt(b *big.Int) Uint128 {
	out, inRange := Uint128FromBigInt(b)
	if !inRange {
		panic(fmt.Errorf("num: big.Int %d was not in valid Uint128 range", b))
	}
	return out
}

func Uint128FromFloat32(f float32) (out Uint128, inRange bool) {
	return Uint128FromFloat64(float64(f))
}

func MustUint128FromFloat32(f float32) Uint128 {
	out, inRange := Uint128FromFloat32(f)
	if !inRange {
		panic(fmt.Errorf("num: float32 %f was not in valid Uint128 range", f))
	}
	return out
}

// Uint128FromFloat64 creates a Uint128 from a float64.
//
// Any fractional portion will be truncated towards zero.
//
// Floats outside the bounds of a Uint128 may be discarded or clamped and inRange
// will be set to false.
//
// NaN is treated as 0, inRange is set to false. This may change to a panic
// at some point.
func Uint128FromFloat64(f float64) (out Uint128, inRange bool) {
	// WARNING: casts from float64 to uint64 have some astonishing properties:
	// https://github.com/golang/go/issues/29463
	if f == 0 {
		return Uint128{}, true

	} else if f < 0 {
		return Uint128{}, false

	} else if f <= maxRepresentableUint64Float {
		return Uint128{lo: uint64(f)}, true

	} else if f <= maxRepresentableUint128Float {
		lo := math.Mod(f, wrapUint64Float) // f is guaranteed to be > 0 here.
		return Uint128{hi: uint64(f / wrapUint64Float), lo: uint64(lo)}, true

	} else if f != f { // (f != f) == NaN
		return Uint128{}, false

	} else {
		return MaxUint128, false
	}
}

func MustUint128FromFloat64(f float64) Uint128 {
	out, inRange := Uint128FromFloat64(f)
	if !inRange {
		panic(fmt.Errorf("num: float64 %f was not in valid Uint128 range", f))
	}
	return out
}

func (u Uint128) IsZero() bool { return u.lo == 0 && u.hi == 0 }

// Raw returns access to the Uint128 as a pair of uint64s. See Uint128FromRaw() for
// the counterpart.
func (u Uint128) Raw() (hi, lo uint64) { return u.hi, u.lo }

func (u Uint128) String() string {
	// FIXME: This is good enough for now, but not forever.
	if u.lo == 0 && u.hi == 0 {
		return "0"
	}
	if u.hi == 0 {
		return strconv.FormatUint(u.lo, 10)
	}
	v := u.AsBigInt()
	return v.String()
}

func (u Uint128) Format(s fmt.State, c rune) {
	// FIXME: This is good enough for now, but not forever.
	u.AsBigInt().Format(s, c)
}

func (u *Uint128) Scan(state fmt.ScanState, verb rune) error {
	t, err := state.Token(true, nil)
	if err != nil {
		return err
	}
	ts := string(t)

	v, inRange, err := Uint128FromString(ts)
	if err != nil {
		return err
	} else if !inRange {
		return fmt.Errorf("num: u128 value %q is not in range", ts)
	}
	*u = v

	return nil
}

func (u Uint128) IntoBigInt(b *big.Int) {
	switch intSize {
	case 64:
		bits := b.Bits()
		ln := len(bits)
		if len(bits) < 2 {
			bits = append(bits, make([]big.Word, 2-ln)...)
		}
		bits = bits[:2]
		bits[0] = big.Word(u.lo)
		bits[1] = big.Word(u.hi)
		b.SetBits(bits)

	case 32:
		bits := b.Bits()
		ln := len(bits)
		if len(bits) < 4 {
			bits = append(bits, make([]big.Word, 4-ln)...)
		}
		bits = bits[:4]
		bits[0] = big.Word(u.lo & 0xFFFFFFFF)
		bits[1] = big.Word(u.lo >> 32)
		bits[2] = big.Word(u.hi & 0xFFFFFFFF)
		bits[3] = big.Word(u.hi >> 32)
		b.SetBits(bits)

	default:
		if u.hi > 0 {
			b.SetUint64(u.hi)
			b.Lsh(b, 64)
		}
		var lo big.Int
		lo.SetUint64(u.lo)
		b.Add(b, &lo)
	}
}

// AsBigInt returns the Uint128 as a big.Int. This will allocate memory. If
// performance is a concern and you are able to re-use memory, use
// Uint128.IntoBigInt().
func (u Uint128) AsBigInt() (b *big.Int) {
	var v big.Int
	u.IntoBigInt(&v)
	return &v
}

func (u Uint128) AsBigFloat() (b *big.Float) {
	return new(big.Float).SetInt(u.AsBigInt())
}

func (u Uint128) AsFloat64() float64 {
	if u.hi == 0 && u.lo == 0 {
		return 0
	} else if u.hi == 0 {
		return float64(u.lo)
	} else {
		return (float64(u.hi) * wrapUint64Float) + float64(u.lo)
	}
}

// AsInt128 performs a direct cast of a Uint128 to an Int128, which will interpret it
// as a two's complement value.
func (u Uint128) AsInt128() Int128 {
	return Int128{lo: u.lo, hi: u.hi}
}

// IsInt128 reports whether i can be represented in an Int128.
func (u Uint128) IsInt128() bool {
	return u.hi& int128SignBit == 0
}

// AsUint64 truncates the Uint128 to fit in a uint64. Values outside the range
// will over/underflow. See IsUint64() if you want to check before you convert.
func (u Uint128) AsUint64() uint64 {
	return u.lo
}

// IsUint64 reports whether u can be represented as a uint64.
func (u Uint128) IsUint64() bool {
	return u.hi == 0
}

// MustUint64 converts i to an unsigned 64-bit integer if the conversion would succeed,
// and panics if it would not.
func (u Uint128) MustUint64() uint64 {
	if u.hi != 0 {
		panic(fmt.Errorf("Uint128 %v is not representable as a uint64", u))
	}
	return u.lo
}

func (u Uint128) Inc() (v Uint128) {
	var carry uint64
	v.lo, carry = bits.Add64(u.lo, 1, 0)
	v.hi = u.hi + carry
	return v
}

func (u Uint128) Dec() (v Uint128) {
	var borrowed uint64
	v.lo, borrowed = bits.Sub64(u.lo, 1, 0)
	v.hi = u.hi - borrowed
	return v
}

func (u Uint128) Add(n Uint128) (v Uint128) {
	var carry uint64
	v.lo, carry = bits.Add64(u.lo, n.lo, 0)
	v.hi, _ = bits.Add64(u.hi, n.hi, carry)
	return v
}

func (u Uint128) Add64(n uint64) (v Uint128) {
	var carry uint64
	v.lo, carry = bits.Add64(u.lo, n, 0)
	v.hi = u.hi + carry
	return v
}

func (u Uint128) Sub(n Uint128) (v Uint128) {
	var borrowed uint64
	v.lo, borrowed = bits.Sub64(u.lo, n.lo, 0)
	v.hi, _ = bits.Sub64(u.hi, n.hi, borrowed)
	return v
}

func (u Uint128) Sub64(n uint64) (v Uint128) {
	var borrowed uint64
	v.lo, borrowed = bits.Sub64(u.lo, n, 0)
	v.hi = u.hi - borrowed
	return v
}

// Cmp compares 'u' to 'n' and returns:
//
//	< 0 if u <  n
//	  0 if u == n
//	> 0 if u >  n
//
// The specific value returned by Cmp is undefined, but it is guaranteed to
// satisfy the above constraints.
//
func (u Uint128) Cmp(n Uint128) int {
	if u.hi == n.hi {
		if u.lo > n.lo {
			return 1
		} else if u.lo < n.lo {
			return -1
		}
	} else {
		if u.hi > n.hi {
			return 1
		} else if u.hi < n.hi {
			return -1
		}
	}
	return 0
}

func (u Uint128) Cmp64(n uint64) int {
	if u.hi > 0 || u.lo > n {
		return 1
	} else if u.lo < n {
		return -1
	}
	return 0
}

func (u Uint128) Equal(n Uint128) bool {
	return u.hi == n.hi && u.lo == n.lo
}

func (u Uint128) Equal64(n uint64) bool {
	return u.hi == 0 && u.lo == n
}

func (u Uint128) GreaterThan(n Uint128) bool {
	return u.hi > n.hi || (u.hi == n.hi && u.lo > n.lo)
}

func (u Uint128) GreaterThan64(n uint64) bool {
	return u.hi > 0 || u.lo > n
}

func (u Uint128) GreaterOrEqualTo(n Uint128) bool {
	return u.hi > n.hi || (u.hi == n.hi && u.lo >= n.lo)
}

func (u Uint128) GreaterOrEqualTo64(n uint64) bool {
	return u.hi > 0 || u.lo >= n
}

func (u Uint128) LessThan(n Uint128) bool {
	return u.hi < n.hi || (u.hi == n.hi && u.lo < n.lo)
}

func (u Uint128) LessThan64(n uint64) bool {
	return u.hi == 0 && u.lo < n
}

func (u Uint128) LessOrEqualTo(n Uint128) bool {
	return u.hi < n.hi || (u.hi == n.hi && u.lo <= n.lo)
}

func (u Uint128) LessOrEqualTo64(n uint64) bool {
	return u.hi == 0 && u.lo <= n
}

func (u Uint128) And(n Uint128) Uint128 {
	u.hi = u.hi & n.hi
	u.lo = u.lo & n.lo
	return u
}

func (u Uint128) And64(n uint64) Uint128 {
	return Uint128{lo: u.lo & n}
}

func (u Uint128) AndNot(n Uint128) Uint128 {
	u.hi = u.hi &^ n.hi
	u.lo = u.lo &^ n.lo
	return u
}

func (u Uint128) Not() (out Uint128) {
	out.hi = ^u.hi
	out.lo = ^u.lo
	return out
}

func (u Uint128) Or(n Uint128) (out Uint128) {
	out.hi = u.hi | n.hi
	out.lo = u.lo | n.lo
	return out
}

func (u Uint128) Or64(n uint64) Uint128 {
	u.lo = u.lo | n
	return u
}

func (u Uint128) Xor(v Uint128) Uint128 {
	u.hi = u.hi ^ v.hi
	u.lo = u.lo ^ v.lo
	return u
}

func (u Uint128) Xor64(v uint64) Uint128 {
	u.hi = u.hi ^ 0
	u.lo = u.lo ^ v
	return u
}

// BitLen returns the length of the absolute value of u in bits. The bit length of 0 is 0.
func (u Uint128) BitLen() int {
	if u.hi > 0 {
		return bits.Len64(u.hi) + 64
	}
	return bits.Len64(u.lo)
}

// OnesCount returns the number of one bits ("population count") in u.
func (u Uint128) OnesCount() int {
	if u.hi > 0 {
		return bits.OnesCount64(u.hi) + 64
	}
	return bits.OnesCount64(u.lo)
}

// Bit returns the value of the i'th bit of x. That is, it returns (x>>i)&1.
// The bit index i must be 0 <= i < 128
func (u Uint128) Bit(i int) uint {
	if i < 0 || i >= 128 {
		panic("num: bit out of range")
	}
	if i >= 64 {
		return uint((u.hi >> uint(i-64)) & 1)
	} else {
		return uint((u.lo >> uint(i)) & 1)
	}
}

// SetBit returns a Uint128 with u's i'th bit set to b (0 or 1).
// If b is not 0 or 1, SetBit will panic. If i < 0, SetBit will panic.
func (u Uint128) SetBit(i int, b uint) (out Uint128) {
	if i < 0 || i >= 128 {
		panic("num: bit out of range")
	}
	if b == 0 {
		if i >= 64 {
			u.hi = u.hi &^ (1 << uint(i-64))
		} else {
			u.lo = u.lo &^ (1 << uint(i))
		}
	} else if b == 1 {
		if i >= 64 {
			u.hi = u.hi | (1 << uint(i-64))
		} else {
			u.lo = u.lo | (1 << uint(i))
		}
	} else {
		panic("num: bit value not 0 or 1")
	}
	return u
}

func (u Uint128) Lsh(n uint) (v Uint128) {
	if n == 0 {
		return u
	} else if n > 64 {
		v.hi = u.lo << (n - 64)
		v.lo = 0
	} else if n < 64 {
		v.hi = (u.hi << n) | (u.lo >> (64 - n))
		v.lo = u.lo << n
	} else if n == 64 {
		v.hi = u.lo
		v.lo = 0
	}
	return v
}

func (u Uint128) Rsh(n uint) (v Uint128) {
	if n == 0 {
		return u
	} else if n > 64 {
		v.lo = u.hi >> (n - 64)
		v.hi = 0
	} else if n < 64 {
		v.lo = (u.lo >> n) | (u.hi << (64 - n))
		v.hi = u.hi >> n
	} else if n == 64 {
		v.lo = u.hi
		v.hi = 0
	}

	return v
}

func (u Uint128) Mul(n Uint128) Uint128 {
	hi, lo := bits.Mul64(u.lo, n.lo)
	hi += u.hi*n.lo + u.lo*n.hi
	return Uint128{hi, lo}
}

func (u Uint128) Mul64(n uint64) (dest Uint128) {
	dest.hi, dest.lo = bits.Mul64(u.lo, n)
	dest.hi += u.hi * n
	return dest
}

// See BenchmarkUint128QuoRemTZ for the test that helps determine this magic number:
const divAlgoLeading0Spill = 16

// Quo returns the quotient x/y for y != 0. If y == 0, a division-by-zero
// run-time panic occurs. Quo implements truncated division (like Go); see
// QuoRem for more details.
func (u Uint128) Quo(by Uint128) (q Uint128) {
	if by.lo == 0 && by.hi == 0 {
		panic("u128: division by zero")
	}

	if u.hi|by.hi == 0 {
		q.lo = u.lo / by.lo // FIXME: div/0 risk?
		return q
	}

	var byLoLeading0, byHiLeading0, byLeading0 uint
	if by.hi == 0 {
		byLoLeading0, byHiLeading0 = uint(bits.LeadingZeros64(by.lo)), 64
		byLeading0 = byLoLeading0 + 64
	} else {
		byHiLeading0 = uint(bits.LeadingZeros64(by.hi))
		byLeading0 = byHiLeading0
	}

	if byLeading0 == 127 {
		return u
	}

	byTrailing0 := by.TrailingZeros()
	if (byLeading0 + byTrailing0) == 127 {
		return u.Rsh(byTrailing0)
	}

	if cmp := u.Cmp(by); cmp < 0 {
		return q // it's 100% remainder
	} else if cmp == 0 {
		q.lo = 1 // dividend and divisor are the same
		return q
	}

	uLeading0 := u.LeadingZeros()
	if byLeading0-uLeading0 > divAlgoLeading0Spill {
		q, _ = quorem128by128(u, by, byHiLeading0, byLoLeading0)
		return q
	} else {
		return quo128bin(u, by, uLeading0, byLeading0)
	}
}

func (u Uint128) Quo64(by uint64) (q Uint128) {
	if u.hi < by {
		q.lo, _ = bits.Div64(u.hi, u.lo, by)
	} else {
		q.hi = u.hi / by
		q.lo, _ = bits.Div64(u.hi%by, u.lo, by)
	}
	return q
}

// QuoRem returns the quotient q and remainder r for y != 0. If y == 0, a
// division-by-zero run-time panic occurs.
//
// QuoRem implements T-division and modulus (like Go):
//
//	q = x/y      with the result truncated to zero
//	r = x - y*q
//
// Uint128 does not support big.Int.DivMod()-style Euclidean division.
//
func (u Uint128) QuoRem(by Uint128) (q, r Uint128) {
	if by.lo == 0 && by.hi == 0 {
		panic("u128: division by zero")
	}

	if u.hi|by.hi == 0 {
		// protected from div/0 because by.lo is guaranteed to be set if by.hi is 0:
		q.lo = u.lo / by.lo
		r.lo = u.lo % by.lo
		return q, r
	}

	var byLoLeading0, byHiLeading0, byLeading0 uint
	if by.hi == 0 {
		byLoLeading0, byHiLeading0 = uint(bits.LeadingZeros64(by.lo)), 64
		byLeading0 = byLoLeading0 + 64
	} else {
		byHiLeading0 = uint(bits.LeadingZeros64(by.hi))
		byLeading0 = byHiLeading0
	}

	if byLeading0 == 127 {
		return u, r
	}

	byTrailing0 := by.TrailingZeros()
	if (byLeading0 + byTrailing0) == 127 {
		q = u.Rsh(byTrailing0)
		by = by.Dec()
		r = by.And(u)
		return
	}

	if cmp := u.Cmp(by); cmp < 0 {
		return q, u // it's 100% remainder

	} else if cmp == 0 {
		q.lo = 1 // dividend and divisor are the same
		return q, r
	}

	uLeading0 := u.LeadingZeros()
	if byLeading0-uLeading0 > divAlgoLeading0Spill {
		return quorem128by128(u, by, byHiLeading0, byLoLeading0)
	} else {
		return quorem128bin(u, by, uLeading0, byLeading0)
	}
}

func (u Uint128) QuoRem64(by uint64) (q, r Uint128) {
	if u.hi < by {
		q.lo, r.lo = bits.Div64(u.hi, u.lo, by)
	} else {
		q.hi, r.lo = bits.Div64(0, u.hi, by)
		q.lo, r.lo = bits.Div64(r.lo, u.lo, by)
	}
	return q, r
}

// Rem returns the remainder of x%y for y != 0. If y == 0, a division-by-zero
// run-time panic occurs. Rem implements truncated modulus (like Go); see
// QuoRem for more details.
func (u Uint128) Rem(by Uint128) (r Uint128) {
	// FIXME: inline only the needed bits
	_, r = u.QuoRem(by)
	return r
}

func (u Uint128) Rem64(by uint64) (r Uint128) {
	// XXX: bits.Rem64 (added in 1.14) shows no noticeable improvement on my 8th-gen i7
	// (though it sounds like it isn't necessarily meant to):
	// https://github.com/golang/go/issues/28970
	// if u.hi < by {
	//     _, r.lo = bits.Rem64(u.hi, u.lo, by)
	// } else {
	//     _, r.lo = bits.Rem64(bits.Rem64(0, u.hi, by), u.lo, by)
	// }

	if u.hi < by {
		_, r.lo = bits.Div64(u.hi, u.lo, by)
	} else {
		_, r.lo = bits.Div64(0, u.hi, by)
		_, r.lo = bits.Div64(r.lo, u.lo, by)
	}
	return r
}

func (u Uint128) Reverse() Uint128 {
	return Uint128{hi: bits.Reverse64(u.lo), lo: bits.Reverse64(u.hi)}
}

func (u Uint128) ReverseBytes() Uint128 {
	return Uint128{hi: bits.ReverseBytes64(u.lo), lo: bits.ReverseBytes64(u.hi)}
}

// To rotate u right by k bits, call u.RotateLeft(-k).
func (u Uint128) RotateLeft(k int) Uint128 {
	s := uint(k) & (127)
	if s > 64 {
		s = 128 - s
		l := 64 - s
		return Uint128{
			hi: u.hi>>s | u.lo<<l,
			lo: u.lo>>s | u.hi<<l,
		}
	} else {
		l := 64 - s
		return Uint128{
			hi: u.hi<<s | u.lo>>l,
			lo: u.lo<<s | u.hi>>l,
		}
	}
}

func (u Uint128) LeadingZeros() uint {
	if u.hi == 0 {
		return uint(bits.LeadingZeros64(u.lo)) + 64
	} else {
		return uint(bits.LeadingZeros64(u.hi))
	}
}

func (u Uint128) TrailingZeros() uint {
	if u.lo == 0 {
		return uint(bits.TrailingZeros64(u.hi)) + 64
	} else {
		return uint(bits.TrailingZeros64(u.lo))
	}
}

// Hacker's delight 9-4, divlu:
func quo128by64(u1, u0, v uint64, vLeading0 uint) (q uint64) {
	var b uint64 = 1 << 32
	var un1, un0, vn1, vn0, q1, q0, un32, un21, un10, rhat, vs, left, right uint64

	vs = v << vLeading0

	vn1 = vs >> 32
	vn0 = vs & 0xffffffff

	if vLeading0 > 0 {
		un32 = (u1 << vLeading0) | (u0 >> (64 - vLeading0))
		un10 = u0 << vLeading0
	} else {
		un32 = u1
		un10 = u0
	}

	un1 = un10 >> 32
	un0 = un10 & 0xffffffff

	q1 = un32 / vn1
	rhat = un32 % vn1

	left = q1 * vn0
	right = (rhat << 32) | un1

again1:
	if (q1 >= b) || (left > right) {
		q1--
		rhat += vn1
		if rhat < b {
			left -= vn0
			right = (rhat << 32) | un1
			goto again1
		}
	}

	un21 = (un32 << 32) + (un1 - (q1 * vs))

	q0 = un21 / vn1
	rhat = un21 % vn1

	left = q0 * vn0
	right = (rhat << 32) | un0

again2:
	if (q0 >= b) || (left > right) {
		q0--
		rhat += vn1
		if rhat < b {
			left -= vn0
			right = (rhat << 32) | un0
			goto again2
		}
	}

	return (q1 << 32) | q0
}

// Hacker's delight 9-4, divlu:
func quorem128by64(u1, u0, v uint64, vLeading0 uint) (q, r uint64) {
	var b uint64 = 1 << 32
	var un1, un0, vn1, vn0, q1, q0, un32, un21, un10, rhat, left, right uint64

	v <<= vLeading0

	vn1 = v >> 32
	vn0 = v & 0xffffffff

	if vLeading0 > 0 {
		un32 = (u1 << vLeading0) | (u0 >> (64 - vLeading0))
		un10 = u0 << vLeading0
	} else {
		un32 = u1
		un10 = u0
	}

	un1 = un10 >> 32
	un0 = un10 & 0xffffffff

	q1 = un32 / vn1
	rhat = un32 % vn1

	left = q1 * vn0
	right = (rhat << 32) + un1

again1:
	if (q1 >= b) || (left > right) {
		q1--
		rhat += vn1
		if rhat < b {
			left -= vn0
			right = (rhat << 32) | un1
			goto again1
		}
	}

	un21 = (un32 << 32) + (un1 - (q1 * v))

	q0 = un21 / vn1
	rhat = un21 % vn1

	left = q0 * vn0
	right = (rhat << 32) | un0

again2:
	if (q0 >= b) || (left > right) {
		q0--
		rhat += vn1
		if rhat < b {
			left -= vn0
			right = (rhat << 32) | un0
			goto again2
		}
	}

	return (q1 << 32) | q0, ((un21 << 32) + (un0 - (q0 * v))) >> vLeading0
}

func quorem128by128(m, v Uint128, vHiLeading0, vLoLeading0 uint) (q, r Uint128) {
	if v.hi == 0 {
		if m.hi < v.lo {
			q.lo, r.lo = quorem128by64(m.hi, m.lo, v.lo, vLoLeading0)
			return q, r

		} else {
			q.hi = m.hi / v.lo
			r.hi = m.hi % v.lo
			q.lo, r.lo = quorem128by64(r.hi, m.lo, v.lo, vLoLeading0)
			r.hi = 0
			return q, r
		}

	} else {
		v1 := v.Lsh(vHiLeading0)
		u1 := m.Rsh(1)

		var q1 Uint128
		q1.lo = quo128by64(u1.hi, u1.lo, v1.hi, vLoLeading0)
		q1 = q1.Rsh(63 - vHiLeading0)

		if q1.hi|q1.lo != 0 {
			q1 = q1.Dec()
		}
		q = q1
		q1 = q1.Mul(v)
		r = m.Sub(q1)

		if r.Cmp(v) >= 0 {
			q = q.Inc()
			r = r.Sub(v)
		}

		return q, r
	}
}

func quorem128bin(u, by Uint128, uLeading0, byLeading0 uint) (q, r Uint128) {
	shift := int(byLeading0 - uLeading0)
	by = by.Lsh(uint(shift))

	for {
		// q << 1
		q.hi = (q.hi << 1) | (q.lo >> 63)
		q.lo = q.lo << 1

		// performance tweak: simulate greater than or equal by hand-inlining "not less than".
		if u.hi > by.hi || (u.hi == by.hi && u.lo >= by.lo) {
			u = u.Sub(by)
			q.lo |= 1
		}

		// by >> 1
		by.lo = (by.lo >> 1) | (by.hi << 63)
		by.hi = by.hi >> 1

		if shift <= 0 {
			break
		}
		shift--
	}

	r = u
	return q, r
}

func quo128bin(u, by Uint128, uLeading0, byLeading0 uint) (q Uint128) {
	shift := int(byLeading0 - uLeading0)
	by = by.Lsh(uint(shift))

	for {
		// q << 1
		q.hi = (q.hi << 1) | (q.lo >> 63)
		q.lo = q.lo << 1

		// u >= by
		if u.hi > by.hi || (u.hi == by.hi && u.lo >= by.lo) {
			u = u.Sub(by)
			q.lo |= 1
		}

		// q >> 1
		by.lo = (by.lo >> 1) | (by.hi << 63)
		by.hi = by.hi >> 1

		if shift <= 0 {
			break
		}
		shift--
	}

	return q
}

func (u Uint128) MarshalText() ([]byte, error) {
	return []byte(u.String()), nil
}

func (u *Uint128) UnmarshalText(bts []byte) (err error) {
	v, _, err := Uint128FromString(string(bts))
	if err != nil {
		return err
	}
	*u = v
	return nil
}

func (u Uint128) MarshalJSON() ([]byte, error) {
	return []byte(`"` + u.String() + `"`), nil
}

func (u *Uint128) UnmarshalJSON(bts []byte) (err error) {
	if bts[0] == '"' {
		ln := len(bts)
		if bts[ln-1] != '"' {
			return fmt.Errorf("num: u128 invalid JSON %q", string(bts))
		}
		bts = bts[1 : ln-1]
	}

	v, _, err := Uint128FromString(string(bts))
	if err != nil {
		return err
	}
	*u = v
	return nil
}

// Put big-endian encoded bytes representing this Uint128 into byte slice b.
// len(b) must be >= 16.
func (u Uint128) PutBigEndian(b []byte) {
	_ = b[15] // BCE
	b[0], b[1], b[2], b[3], b[4], b[5], b[6], b[7] = byte(u.hi>>56), byte(u.hi>>48), byte(u.hi>>40), byte(u.hi>>32), byte(u.hi>>24), byte(u.hi>>16), byte(u.hi>>8), byte(u.hi)
	b[8], b[9], b[10], b[11], b[12], b[13], b[14], b[15] = byte(u.lo>>56), byte(u.lo>>48), byte(u.lo>>40), byte(u.lo>>32), byte(u.lo>>24), byte(u.lo>>16), byte(u.lo>>8), byte(u.lo)
}

// Decode 16 bytes as a big-endian Uint128. Panics if len(b) < 16.
func MustUint128FromBigEndian(b []byte) Uint128 {
	_ = b[15] // BCE
	return Uint128{
		lo: uint64(b[15]) | uint64(b[14])<<8 | uint64(b[13])<<16 | uint64(b[12])<<24 |
			uint64(b[11])<<32 | uint64(b[10])<<40 | uint64(b[9])<<48 | uint64(b[8])<<56,
		hi: uint64(b[7]) | uint64(b[6])<<8 | uint64(b[5])<<16 | uint64(b[4])<<24 |
			uint64(b[3])<<32 | uint64(b[2])<<40 | uint64(b[1])<<48 | uint64(b[0])<<56,
	}
}

// Put little-endian encoded bytes representing this Uint128 into byte slice b.
// len(b) must be >= 16.
func (u Uint128) PutLittleEndian(b []byte) {
	_ = b[15] // BCE
	b[0], b[1], b[2], b[3], b[4], b[5], b[6], b[7] = byte(u.lo), byte(u.lo>>8), byte(u.lo>>16), byte(u.lo>>24), byte(u.lo>>32), byte(u.lo>>40), byte(u.lo>>48), byte(u.lo>>56)
	b[8], b[9], b[10], b[11], b[12], b[13], b[14], b[15] = byte(u.hi), byte(u.hi>>8), byte(u.hi>>16), byte(u.hi>>24), byte(u.hi>>32), byte(u.hi>>40), byte(u.hi>>48), byte(u.hi>>56)
}

// Decode 16 bytes as a little-endian Uint128. Panics if len(b) < 16.
func MustUint128FromLittleEndian(b []byte) Uint128 {
	_ = b[15] // BCE
	return Uint128{
		lo: uint64(b[0]) | uint64(b[1])<<8 | uint64(b[2])<<16 | uint64(b[3])<<24 |
			uint64(b[4])<<32 | uint64(b[5])<<40 | uint64(b[6])<<48 | uint64(b[7])<<56,
		hi: uint64(b[8]) | uint64(b[9])<<8 | uint64(b[10])<<16 | uint64(b[11])<<24 |
			uint64(b[12])<<32 | uint64(b[13])<<40 | uint64(b[14])<<48 | uint64(b[15])<<56,
	}
}

// DifferenceUint128 subtracts the smaller of a and b from the larger.
func DifferenceUint128(a, b Uint128) Uint128 {
	if a.hi > b.hi {
		return a.Sub(b)
	} else if a.hi < b.hi {
		return b.Sub(a)
	} else if a.lo > b.lo {
		return a.Sub(b)
	} else if a.lo < b.lo {
		return b.Sub(a)
	}
	return Uint128{}
}

func LargerUint128(a, b Uint128) Uint128 {
	if a.hi > b.hi {
		return a
	} else if a.hi < b.hi {
		return b
	} else if a.lo > b.lo {
		return a
	} else if a.lo < b.lo {
		return b
	}
	return a
}

func SmallerUint128(a, b Uint128) Uint128 {
	if a.hi < b.hi {
		return a
	} else if a.hi > b.hi {
		return b
	} else if a.lo < b.lo {
		return a
	} else if a.lo > b.lo {
		return b
	}
	return a
}
