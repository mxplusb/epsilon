package geometry

import (
	"math"
	"math/big"
)

const (
	Infinity = math.MaxFloat64
	Epsilon = 1.19209e-07 // defined by clang for x86
)

const (
	maxUint64 = 1<<64 - 1
	maxInt64  = 1<<63 - 1
	minInt64  = -1 << 63

	minInt64Float = float64(minInt64) // -(1<<63)
	maxInt64Float = float64(maxInt64) // (1<<63) - 1

	// WARNING: this can not be represented accurately as a float; attempting to
	// convert it to uint64 will overflow and cause weird truncation issues that
	// violate the principle of least astonishment.
	maxUint64Float = float64(maxUint64) // (1<<64) - 1

	wrapUint64Float = float64(maxUint64) + 1 // 1 << 64

	maxUint128Float = float64(340282366920938463463374607431768211455)  // (1<<128) - 1
	maxInt128Float = float64(170141183460469231731687303715884105727)  // (1<<127) - 1
	minInt128Float = float64(-170141183460469231731687303715884105728) // -(1<<127)

	intSize Uint64 = 32 << (^uint(0) >> 63)
)

var (
	MaxInt128 = Int128{hi: 0x7FFFFFFFFFFFFFFF, lo: 0xFFFFFFFFFFFFFFFF}
	MinInt128 = Int128{hi: 0x8000000000000000, lo: 0}
	MaxUint128 = Uint128{hi: maxUint64, lo: maxUint64}

	zeroInt128 Int128
	zeroUint128 Uint128

	minusOne = Int128{hi: 0xFFFFFFFFFFFFFFFF, lo: 0xFFFFFFFFFFFFFFFF}

	big0 = new(big.Int).SetInt64(0)
	big1 = new(big.Int).SetInt64(1)

	maxBigUint64  = new(big.Int).SetUint64(maxUint64)
	maxBigUint128, _ = new(big.Int).SetString("340282366920938463463374607431768211455", 10)
	maxBigInt64   = new(big.Int).SetUint64(maxInt64)
	minBigInt64   = new(big.Int).SetInt64(minInt64)

	minBigInt128, _ = new(big.Int).SetString("-0x80000000000000000000000000000000", 0)
	maxBigInt128, _ = new(big.Int).SetString("0x7FFFFFFFFFFFFFFFFFFFFFFFFFFFFFFF", 0)

	// wrapBigUint128 is 1 << 128, used to simulate over/underflow:
	wrapBigUint128, _ = new(big.Int).SetString("340282366920938463463374607431768211456", 10)

	// wrapBigU64 is 1 << 64:
	wrapBigU64, _ = new(big.Int).SetString("18446744073709551616", 10)

	// wrapOverBigInt128 is 1 << 127, used to simulate over/underflow:
	wrapOverBigInt128, _ = new(big.Int).SetString("0x80000000000000000000000000000000", 0)

	// wrapUnderBigInt128 is -(1 << 127) - 1, used to simulate over/underflow:
	wrapUnderBigInt128, _ = new(big.Int).SetString("-170141183460469231731687303715884105729", 0)

	// minInt128AsUint128 is used for the Int128.AbsUint128() overflow case where the
	// Int128 == MinInt128.
	minInt128AsUint128 = Uint128{hi: 0x8000000000000000, lo: 0x0}

	// This specifies the maximum error allowed between the float64 version of
	// a 128-bit int/uint and the result of the same operation performed by
	// big.Float.
	//
	// Calculate like so:
	//	return math.Nextafter(1.0, 2.0) - 1.0
	//
	floatDiffLimit, _ = new(big.Float).SetString("2.220446049250313080847263336181640625e-16")

	maxRepresentableUint64Float  = math.Nextafter(maxUint64Float, 0)           // < (1<<64)
	wrapRepresentableUint64Float = math.Nextafter(maxUint64Float, math.Inf(1)) // >= (1<<64)

	maxRepresentableUint128Float = math.Nextafter(float64(340282366920938463463374607431768211455), 0) // < (1<<128)
)