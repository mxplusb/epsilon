package geometry

import (
	"fmt"
	"math"
	"math/big"
)

const (
	int128SignBit = 0x8000000000000000
)

type Int128 struct {
	hi Uint64
	lo Uint64
}

// Int128FromRaw is the complement to Int128.Raw(); it creates an Int128 from two
// Uint64s representing the hi and lo 
func Int128FromRaw(hi, lo Uint64) Int128 { return Int128{hi: hi, lo: lo} }

func Int128FromInt64(v Int64) (out Int128) {
	// There's a no-branch way of calculating this:
	//   out.lo = Uint64(v)
	//   out.hi = ^((out.lo >> 63) + maxUint64)
	//
	// There may be a better one than that, but that's the one I found. Bogus
	// microbenchmarks on an i7-3820 and an i7-6770HQ showed it may possibly be
	// slightly faster, but at huge cost to the inliner. The no-branch
	// version eats 20 more points out of Go 1.12's inlining budget of 80 than
	// the 'if v < 0' verson, which is probably worse overall.

	var hi Uint64
	if v < 0 {
		hi = maxUint64
	}
	return Int128{hi: hi, lo: Uint64(v)}
}

func Int128FromInt32(v Int32) Int128 { return Int128FromInt64(Int64(v)) }
func Int128FromInt16(v Int16) Int128 { return Int128FromInt64(Int64(v)) }
func Int128FromInt8(v Int8) Int128   { return Int128FromInt64(Int64(v)) }
func Int128FromInt(v int) Int128     { return Int128FromInt64(Int64(v)) }
func Int128FromUint64(v Uint64) Int128 { return Int128{lo: v} }

// Int128FromString creates a Int128 from a string. Overflow truncates to
// MaxInt128/MinInt128 and sets accurate to 'false'. Only decimal strings are
// currently supported.
func Int128FromString(s string) (out Int128, accurate bool, err error) {
	// This deliberately limits the scope of what we accept as input just in case
	// we decide to hand-roll our own fast decimal-only parser:
	b, ok := new(big.Int).SetString(s, 10)
	if !ok {
		return out, false, fmt.Errorf("num: Int128 string %q invalid", s)
	}
	out, accurate = Int128FromBigInt(b)
	return out, accurate, nil
}

func MustInt128FromString(s string) Int128 {
	out, inRange, err := Int128FromString(s)
	if err != nil {
		panic(err)
	}
	if !inRange {
		panic(fmt.Errorf("num: string %q was not in valid Int128 range", s))
	}
	return out
}

var (
	minInt128AsAbsUint128 = Uint128{hi: 0x8000000000000000, lo: 0}
	maxInt128AsUint128    = Uint128{hi: 0x7FFFFFFFFFFFFFFF, lo: 0xFFFFFFFFFFFFFFFF}
)

func Int128FromBigInt(v *big.Int) (out Int128, accurate bool) {
	neg := v.Sign() < 0

	words := v.Bits()

	var u Uint128
	accurate = true

	switch intSize {
	case 64:
		lw := len(words)
		switch lw {
		case 0:
		case 1:
			u.lo = Uint64(words[0])
		case 2:
			u.hi = Uint64(words[1])
			u.lo = Uint64(words[0])
		default:
			u, accurate = MaxUint128, false
		}

	case 32:
		lw := len(words)
		switch lw {
		case 0:
		case 1:
			u.lo = Uint64(words[0])
		case 2:
			u.lo = (Uint64(words[1]) << 32) | (Uint64(words[0]))
		case 3:
			u.hi = Uint64(words[2])
			u.lo = (Uint64(words[1]) << 32) | (Uint64(words[0]))
		case 4:
			u.hi = (Uint64(words[3]) << 32) | (Uint64(words[2]))
			u.lo = (Uint64(words[1]) << 32) | (Uint64(words[0]))
		default:
			u, accurate = MaxUint128, false
		}

	default:
		panic("num: unsupported bit size")
	}

	if !neg {
		if cmp := u.Cmp(maxInt128AsUint128); cmp == 0 {
			out = MaxInt128
		} else if cmp > 0 {
			out, accurate = MaxInt128, false
		} else {
			out = u.AsInt128()
		}

	} else {
		if cmp := u.Cmp(minInt128AsAbsUint128); cmp == 0 {
			out = MinInt128
		} else if cmp > 0 {
			out, accurate = MinInt128, false
		} else {
			out = u.AsInt128().Neg()
		}
	}

	return out, accurate
}

func MustInt128FromBigInt(b *big.Int) Int128 {
	out, inRange := Int128FromBigInt(b)
	if !inRange {
		panic(fmt.Errorf("num: big.Int %d was not in valid Int128 range", b))
	}
	return out
}

func Int128FromFloat32(f float32) (out Int128, inRange bool) {
	return Int128FromFloat64(float64(f))
}

func MustInt128FromFloat32(f float32) Int128 {
	out, inRange := Int128FromFloat32(f)
	if !inRange {
		panic(fmt.Errorf("num: float32 %f was not in valid Int128 range", f))
	}
	return out
}

// Int128FromFloat64 creates a Int128 from a float64.
//
// Any fractional portion will be truncated towards zero.
//
// Floats outside the bounds of a Int128 may be discarded or clamped and inRange
// will be set to false.
//
// NaN is treated as 0, inRange is set to false. This may change to a panic
// at some point.
func Int128FromFloat64(f float64) (out Int128, inRange bool) {
	const spillPos = float64(maxUint64) // (1<<64) - 1
	const spillNeg = -float64(maxUint64) - 1

	if f == 0 {
		return out, true

	} else if f != f { // f != f == isnan
		return out, false

	} else if f < 0 {
		if f >= spillNeg {
			return Int128{hi: maxUint64, lo: Uint64(f)}, true
		} else if f >= minInt128Float {
			f = -f
			lo := math.Mod(f, wrapUint64Float) // f is guaranteed to be < 0 here.
			return Int128{hi: ^Uint64(f / wrapUint64Float), lo: ^Uint64(lo)}, true
		} else {
			return MinInt128, false
		}

	} else {
		if f <= spillPos {
			return Int128{lo: Uint64(f)}, true
		} else if f <= maxInt128Float {
			lo := math.Mod(f, wrapUint64Float) // f is guaranteed to be > 0 here.
			return Int128{hi: Uint64(f / wrapUint64Float), lo: Uint64(lo)}, true
		} else {
			return MaxInt128, false
		}
	}
}

func MustInt128FromFloat64(f float64) Int128 {
	out, inRange := Int128FromFloat64(f)
	if !inRange {
		panic(fmt.Errorf("num: float64 %f was not in valid Int128 range", f))
	}
	return out
}

func (i Int128) IsZero() bool { return i.lo == 0 && i.hi == 0 }

// Raw returns access to the Int128 as a pair of Uint64s. See Int128FromRaw() for
// the counterpart.
func (i Int128) Raw() (hi Uint64, lo Uint64) { return i.hi, i.lo }

func (i Int128) String() string {
	// FIXME: This is good enough for now, but not forever.
	v := i.AsBigInt()
	return v.String()
}

func (i *Int128) Scan(state fmt.ScanState, verb rune) error {
	t, err := state.Token(true, nil)
	if err != nil {
		return err
	}
	ts := string(t)

	v, inRange, err := Int128FromString(ts)
	if err != nil {
		return err
	} else if !inRange {
		return fmt.Errorf("num: Int128 value %q is not in range", ts)
	}
	*i = v

	return nil
}

func (i Int128) Format(s fmt.State, c rune) {
	// FIXME: This is good enough for now, but not forever.
	i.AsBigInt().Format(s, c)
}

// IntoBigInt copies this Int128 into a big.Int, allowing you to retain and
// recycle memory.
func (i Int128) IntoBigInt(b *big.Int) {
	neg := i.hi&int128SignBit != 0
	if i.hi > 0 {
		b.SetUint64(uint64(i.hi))
		b.Lsh(b, 64)
	}
	var lo big.Int
	lo.SetUint64(uint64(i.lo))
	b.Add(b, &lo)

	if neg {
		b.Xor(b, maxBigUint128).Add(b, big1).Neg(b)
	}
}

// AsBigInt allocates a new big.Int and copies this Int128 into it.
func (i Int128) AsBigInt() (b *big.Int) {
	b = new(big.Int)
	neg := i.hi&int128SignBit != 0
	if i.hi > 0 {
		b.SetUint64(uint64(i.hi))
		b.Lsh(b, 64)
	}
	var lo big.Int
	lo.SetUint64(uint64(i.lo))
	b.Add(b, &lo)

	if neg {
		b.Xor(b, maxBigUint128).Add(b, big1).Neg(b)
	}

	return b
}

// AsUint128 performs a direct cast of an Int128 to a Uint128. Negative numbers
// become values > math.MaxInt128.
func (i Int128) AsUint128() Uint128 {
	return Uint128{lo: i.lo, hi: i.hi}
}

// IsUint128 reports wehether i can be represented in a Uint128.
func (i Int128) IsUint128() bool {
	return i.hi&int128SignBit == 0
}

func (i Int128) AsBigFloat() (b *big.Float) {
	return new(big.Float).SetInt(i.AsBigInt())
}

func (i Int128) AsFloat64() float64 {
	if i.hi == 0 {
		if i.lo == 0 {
			return 0
		} else {
			return float64(i.lo)
		}
	} else if i.hi == maxUint64 {
		return -float64((^i.lo) + 1)
	} else if i.hi&int128SignBit == 0 {
		return (float64(i.hi) * maxUint64Float) + float64(i.lo)
	} else {
		return (-float64(^i.hi) * maxUint64Float) + -float64(^i.lo)
	}
}

// AsInt64 truncates the Int128 to fit in a int64. Values outside the range will
// over/underflow. See IsInt64() if you want to check before you convert.
func (i Int128) AsInt64() int64 {
	if i.hi&int128SignBit != 0 {
		return -int64(^(i.lo - 1))
	} else {
		return int64(i.lo)
	}
}

// IsInt64 reports whether i can be represented as a int64.
func (i Int128) IsInt64() bool {
	if i.hi&int128SignBit != 0 {
		return i.hi == maxUint64 && i.lo >= 0x8000000000000000
	} else {
		return i.hi == 0 && i.lo <= maxInt64
	}
}

// MustInt64 converts i to a signed 64-bit integer if the conversion would succeed, and
// panics if it would not.
func (i Int128) MustInt64() Int64 {
	if i.hi&int128SignBit != 0 {
		if i.hi == maxUint64 && i.lo >= 0x8000000000000000 {
			return -Int64(^(i.lo - 1))
		}
	} else {
		if i.hi == 0 && i.lo <= maxInt64 {
			return Int64(i.lo)
		}
	}
	panic(fmt.Errorf("Int128 %v is not representable as an int64", i))
}

// AsUint64 truncates the Int128 to fit in a Uint64. Values outside the range will
// over/underflow. Signedness is discarded, as with the following conversion:
//
//	var i int64 = -3
//	var u = uint32(i)
//	fmt.Printf("%x", u)
//	// fffffffd
//
// See IsUint64() if you want to check before you convert.
func (i Int128) AsUint64() Uint64 {
	return i.lo
}

// AsUint64 truncates the Int128 to fit in a Uint64. Values outside the range will
// over/underflow. See IsUint64() if you want to check before you convert.
func (i Int128) IsUint64() bool {
	return i.hi == 0
}

// MustUint64 converts i to an unsigned 64-bit integer if the conversion would succeed,
// and panics if it would not.
func (i Int128) MustUint64() Uint64 {
	if i.hi != 0 {
		panic(fmt.Errorf("Int128 %v is not representable as a Uint64", i))
	}
	return i.lo
}

func (i Int128) Sign() int {
	if i == zeroInt128 {
		return 0
	} else if i.hi&int128SignBit == 0 {
		return 1
	}
	return -1
}

func (i Int128) Inc() (v Int128) {
	v.lo = i.lo + 1
	v.hi = i.hi
	if i.lo > v.lo {
		v.hi++
	}
	return v
}

func (i Int128) Dec() (v Int128) {
	v.lo = i.lo - 1
	v.hi = i.hi
	if i.lo < v.lo {
		v.hi--
	}
	return v
}

func (i Int128) Add(n Int128) (v Int128) {
	var carry Uint64
	v.lo, carry = Add64(i.lo, n.lo, 0)
	v.hi, _ = Add64(i.hi, n.hi, carry)
	return v
}

func (i Int128) Add64(n int64) (v Int128) {
	var carry Uint64
	v.lo, carry = Add64(i.lo, Uint64(n), 0)
	if n < 0 {
		v.hi = i.hi + maxUint64 + carry
	} else {
		v.hi = i.hi + carry
	}
	return v
}

func (i Int128) Sub(n Int128) (v Int128) {
	var borrowed Uint64
	v.lo, borrowed = Sub64(i.lo, n.lo, 0)
	v.hi, _ = Sub64(i.hi, n.hi, borrowed)
	return v
}

func (i Int128) Sub64(n int64) (v Int128) {
	var borrowed Uint64
	if n < 0 {
		v.lo, borrowed = Sub64(i.lo, Uint64(n), 0)
		v.hi = i.hi - maxUint64 - borrowed
	} else {
		v.lo, borrowed = Sub64(i.lo, Uint64(n), 0)
		v.hi = i.hi - borrowed
	}
	return v
}

func (i Int128) Neg() (v Int128) {
	if i.hi == 0 && i.lo == 0 {
		return v
	}

	if i == MinInt128 {
		// Overflow case: -MinInt128 == MinInt128
		return i

	} else if i.hi&int128SignBit != 0 {
		v.hi = ^i.hi
		v.lo = ^(i.lo - 1)
	} else {
		v.hi = ^i.hi
		v.lo = (^i.lo) + 1
	}
	if v.lo == 0 { // handle overflow
		v.hi++
	}
	return v
}

// Abs returns the absolute value of i as a signed integer.
//
// If i == MinInt128, overflow occurs such that Abs(i) == MinInt128.
// If this is not desired, use AbsUint128.
//
func (i Int128) Abs() Int128 {
	if i.hi&int128SignBit != 0 {
		i.hi = ^i.hi
		i.lo = ^(i.lo - 1)
		if i.lo == 0 { // handle carry
			i.hi++
		}
	}
	return i
}

// AbsUint128 returns the absolute value of i as an unsigned integer. All
// values of i are representable using this function, but the type is
// changed.
//
func (i Int128) AbsUint128() Uint128 {
	if i == MinInt128 {
		return minInt128AsUint128
	}
	if i.hi&int128SignBit != 0 {
		i.hi = ^i.hi
		i.lo = ^(i.lo - 1)
		if i.lo == 0 { // handle carry
			i.hi++
		}
	}
	return Uint128{hi: i.hi, lo: i.lo}
}

// Cmp compares i to n and returns:
//
//	< 0 if i <  n
//	  0 if i == n
//	> 0 if i >  n
//
// The specific value returned by Cmp is undefined, but it is guaranteed to
// satisfy the above constraints.
//
func (i Int128) Cmp(n Int128) int {
	if i.hi == n.hi && i.lo == n.lo {
		return 0
	} else if i.hi&int128SignBit == n.hi&int128SignBit {
		if i.hi > n.hi || (i.hi == n.hi && i.lo > n.lo) {
			return 1
		}
	} else if i.hi&int128SignBit == 0 {
		return 1
	}
	return -1
}

// Cmp64 compares 'i' to 64-bit int 'n' and returns:
//
//	< 0 if i <  n
//	  0 if i == n
//	> 0 if i >  n
//
// The specific value returned by Cmp is undefined, but it is guaranteed to
// satisfy the above constraints.
//
func (i Int128) Cmp64(n int64) int {
	var nhi Uint64
	var nlo = Uint64(n)
	if n < 0 {
		nhi = maxUint64
	}
	if i.hi == nhi && i.lo == nlo {
		return 0
	} else if i.hi&int128SignBit == nhi&int128SignBit {
		if i.hi > nhi || (i.hi == nhi && i.lo > nlo) {
			return 1
		}
	} else if i.hi&int128SignBit == 0 {
		return 1
	}
	return -1
}

func (i Int128) Equal(n Int128) bool {
	return i.hi == n.hi && i.lo == n.lo
}

func (i Int128) Equal64(n int64) bool {
	var nhi Uint64
	var nlo = Uint64(n)
	if n < 0 {
		nhi = maxUint64
	}
	return i.hi == nhi && i.lo == nlo
}

func (i Int128) GreaterThan(n Int128) bool {
	if i.hi&int128SignBit == n.hi&int128SignBit {
		return i.hi > n.hi || (i.hi == n.hi && i.lo > n.lo)
	} else if i.hi&int128SignBit == 0 {
		return true
	}
	return false
}

func (i Int128) GreaterThan64(n int64) bool {
	var nhi Uint64
	var nlo = Uint64(n)
	if n < 0 {
		nhi = maxUint64
	}

	if i.hi&int128SignBit == nhi&int128SignBit {
		return i.hi > nhi || (i.hi == nhi && i.lo > nlo)
	} else if i.hi&int128SignBit == 0 {
		return true
	}
	return false
}

func (i Int128) GreaterOrEqualTo(n Int128) bool {
	if i.hi == n.hi && i.lo == n.lo {
		return true
	}
	if i.hi&int128SignBit == n.hi&int128SignBit {
		return i.hi > n.hi || (i.hi == n.hi && i.lo > n.lo)
	} else if i.hi&int128SignBit == 0 {
		return true
	}
	return false
}

func (i Int128) GreaterOrEqualTo64(n int64) bool {
	var nhi Uint64
	var nlo = Uint64(n)
	if n < 0 {
		nhi = maxUint64
	}

	if i.hi == nhi && i.lo == nlo {
		return true
	}
	if i.hi&int128SignBit == nhi&int128SignBit {
		return i.hi > nhi || (i.hi == nhi && i.lo > nlo)
	} else if i.hi&int128SignBit == 0 {
		return true
	}
	return false
}

func (i Int128) LessThan(n Int128) bool {
	if i.hi&int128SignBit == n.hi&int128SignBit {
		return i.hi < n.hi || (i.hi == n.hi && i.lo < n.lo)
	} else if i.hi&int128SignBit != 0 {
		return true
	}
	return false
}

func (i Int128) LessThan64(n int64) bool {
	var nhi Uint64
	var nlo = Uint64(n)
	if n < 0 {
		nhi = maxUint64
	}

	if i.hi&int128SignBit == nhi&int128SignBit {
		return i.hi < nhi || (i.hi == nhi && i.lo < nlo)
	} else if i.hi&int128SignBit != 0 {
		return true
	}
	return false
}

func (i Int128) LessOrEqualTo(n Int128) bool {
	if i.hi == n.hi && i.lo == n.lo {
		return true
	}
	if i.hi&int128SignBit == n.hi&int128SignBit {
		return i.hi < n.hi || (i.hi == n.hi && i.lo < n.lo)
	} else if i.hi&int128SignBit != 0 {
		return true
	}
	return false
}

func (i Int128) LessOrEqualTo64(n int64) bool {
	var nhi Uint64
	var nlo = Uint64(n)
	if n < 0 {
		nhi = maxUint64
	}

	if i.hi == nhi && i.lo == nlo {
		return true
	}
	if i.hi&int128SignBit == nhi&int128SignBit {
		return i.hi < nhi || (i.hi == nhi && i.lo < nlo)
	} else if i.hi&int128SignBit != 0 {
		return true
	}
	return false
}

// Mul returns the product of two Int128s.
//
// Overflow should wrap around, as per the Go spec.
//
func (i Int128) Mul(n Int128) (dest Int128) {
	hi, lo := Mul64(i.lo, n.lo)
	hi += i.hi*n.lo + i.lo*n.hi
	return Int128{hi, lo}
}

func (i Int128) Mul64(n Int64) Int128 {
	nlo := Uint64(n)
	var nhi Uint64
	if n < 0 {
		nhi = maxUint64
	}
	hi, lo := Mul64(i.lo, nlo)
	hi += i.hi*nlo + i.lo*nhi
	return Int128{hi, lo}
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
// Note: dividing MinInt128 by -1 will overflow, returning MinInt128, as
// per the Go spec (https://golang.org/ref/spec#Integer_operators):
//
//	The one exception to this rule is that if the dividend x is the most
//	negative value for the int type of x, the quotient q = x / -1 is equal to x
//	(and r = 0) due to two's-complement integer overflow.
//
func (i Int128) QuoRem(by Int128) (q, r Int128) {
	qSign, rSign := 1, 1
	if i.LessThan(zeroInt128) {
		qSign, rSign = -1, -1
		i = i.Neg()
	}
	if by.LessThan(zeroInt128) {
		qSign = -qSign
		by = by.Neg()
	}

	qu, ru := i.AsUint128().QuoRem(by.AsUint128())
	q, r = qu.AsInt128(), ru.AsInt128()
	if qSign < 0 {
		q = q.Neg()
	}
	if rSign < 0 {
		r = r.Neg()
	}
	return q, r
}

func (i Int128) QuoRem64(by int64) (q, r Int128) {
	ineg := i.hi&int128SignBit != 0
	if ineg {
		i = i.Neg()
	}
	byneg := by < 0
	if byneg {
		by = -by
	}

	n := Uint64(by)
	if i.hi < n {
		q.lo, r.lo = Div64(i.hi, i.lo, n)
	} else {
		q.hi, r.lo = Div64(0, i.hi, n)
		q.lo, r.lo = Div64(r.lo, i.lo, n)
	}
	if ineg != byneg {
		q = q.Neg()
	}
	if ineg {
		r = r.Neg()
	}
	return q, r
}

// Quo returns the quotient x/y for y != 0. If y == 0, a division-by-zero
// run-time panic occurs. Quo implements truncated division (like Go); see
// QuoRem for more details.
func (i Int128) Quo(by Int128) (q Int128) {
	qSign := 1
	if i.LessThan(zeroInt128) {
		qSign = -1
		i = i.Neg()
	}
	if by.LessThan(zeroInt128) {
		qSign = -qSign
		by = by.Neg()
	}

	qu := i.AsUint128().Quo(by.AsUint128())
	q = qu.AsInt128()
	if qSign < 0 {
		q = q.Neg()
	}
	return q
}

func (i Int128) Quo64(by int64) (q Int128) {
	ineg := i.hi&int128SignBit != 0
	if ineg {
		i = i.Neg()
	}
	byneg := by < 0
	if byneg {
		by = -by
	}

	n := Uint64(by)
	if i.hi < n {
		q.lo, _ = Div64(i.hi, i.lo, n)
	} else {
		var rlo Uint64
		q.hi, rlo = Div64(0, i.hi, n)
		q.lo, _ = Div64(rlo, i.lo, n)
	}
	if ineg != byneg {
		q = q.Neg()
	}
	return q
}

// Rem returns the remainder of x%y for y != 0. If y == 0, a division-by-zero
// run-time panic occurs. Rem implements truncated modulus (like Go); see
// QuoRem for more details.
func (i Int128) Rem(by Int128) (r Int128) {
	// FIXME: inline only the needed bits
	_, r = i.QuoRem(by)
	return r
}

func (i Int128) Rem64(by int64) (r Int128) {
	ineg := i.hi&int128SignBit != 0
	if ineg {
		i = i.Neg()
	}
	if by < 0 {
		by = -by
	}

	n := Uint64(by)
	if i.hi < n {
		_, r.lo = Div64(i.hi, i.lo, n)
	} else {
		_, r.lo = Div64(0, i.hi, n)
		_, r.lo = Div64(r.lo, i.lo, n)
	}
	if ineg {
		r = r.Neg()
	}
	return r
}

func (i Int128) MarshalText() ([]byte, error) {
	return []byte(i.String()), nil
}

func (i *Int128) UnmarshalText(bts []byte) (err error) {
	v, _, err := Int128FromString(string(bts))
	if err != nil {
		return err
	}
	*i = v
	return nil
}

func (i Int128) ToScalar() Scalar {
	return Scalar(i.AsFloat64())
}

func (i Int128) MarshalJSON() ([]byte, error) {
	return []byte(`"` + i.String() + `"`), nil
}

func (i *Int128) UnmarshalJSON(bts []byte) (err error) {
	if bts[0] == '"' {
		ln := len(bts)
		if bts[ln-1] != '"' {
			return fmt.Errorf("num: Int128 invalid JSON %q", string(bts))
		}
		bts = bts[1 : ln-1]
	}

	v, _, err := Int128FromString(string(bts))
	if err != nil {
		return err
	}
	*i = v
	return nil
}

// DifferenceInt128 subtracts the smaller of a and b from the larger.
func DifferenceInt128(a, b Int128) Int128 {
	if a.hi > b.hi {
		return a.Sub(b)
	} else if a.hi < b.hi {
		return b.Sub(a)
	} else if a.lo > b.lo {
		return a.Sub(b)
	} else if a.lo < b.lo {
		return b.Sub(a)
	}
	return Int128{}
}
