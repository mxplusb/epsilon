package geometry

import (
	"crypto/rand"
	"encoding/binary"
	"encoding/json"
	"fmt"
	"math"
	"math/big"
	"regexp"
	"strings"
	"testing"

	"github.com/stretchr/testify/require"
)

var u64 = Uint128From64

var (
	benchBigFloatResult *big.Float
	benchBigIntResult *big.Int
	benchBoolResult  bool
	benchFloatResult  float64
	benchIntResult     int
	benchStringResult  string
	benchUint128Result Uint128
	benchUint64Result  uint64

	benchUint641, benchUint642 uint64 = 12093749018, 18927348917
)

func bigU64(u uint64) *big.Int { return new(big.Int).SetUint64(u) }

func u128s(s string) Uint128 {
	s = strings.Replace(s, " ", "", -1)
	b, ok := new(big.Int).SetString(s, 0)
	if !ok {
		panic(fmt.Errorf("num: u128 string %q invalid", s))
	}
	out, acc := Uint128FromBigInt(b)
	if !acc {
		panic(fmt.Errorf("num: inaccurate u128 %s", s))
	}
	return out
}

func randUint128(scratch []byte) Uint128 {
	rand.Read(scratch)
	u := Uint128{}
	u.lo = binary.LittleEndian.Uint64(scratch)

	if scratch[0]%2 == 1 {
		// if we always generate hi bits, the universe will die before we
		// test a number < maxInt64
		u.hi = binary.LittleEndian.Uint64(scratch[8:])
	}
	return u
}

func TestLargerSmallerUint128(t *testing.T) {
	for idx, tc := range []struct {
		a, b        Uint128
		firstLarger bool
	}{
		{u64(0), u64(1), false},
		{MaxUint128, u64(1), true},
		{u64(1), u64(1), false},
		{u64(2), u64(1), true},
		{u128s("0xFFFFFFFF FFFFFFFF"), u128s("0x1 00000000 00000000"), false},
		{u128s("0x1 00000000 00000000"), u128s("0xFFFFFFFF FFFFFFFF"), true},
	} {
		t.Run(fmt.Sprintf("%d", idx), func(t *testing.T) {
			if tc.firstLarger {
				require.Equal(t, tc.a, LargerUint128(tc.a, tc.b))
				require.Equal(t, tc.a, LargerUint128(tc.a, tc.b))
				require.Equal(t, tc.b, SmallerUint128(tc.a, tc.b))
			} else {
				require.Equal(t, tc.b, LargerUint128(tc.a, tc.b))
				require.Equal(t, tc.a, SmallerUint128(tc.a, tc.b))
			}
		})
	}
}

func TestMustUint128FromI64(t *testing.T) {
	assert := func(ok bool, expected Uint128, v int64) {
		if !ok {
			require.Panics(t, func() {
				MustUint128FromI64(v)
			})
			return
		}
		require.Equal(t, expected, MustUint128FromI64(v))
	}

	assert(true, u128s("1234"), 1234)
	assert(false, u64(0), -1234)
}

func TestMustUint128FromString(t *testing.T) {
	assert := func(ok bool, expected Uint128, s string) {
		if !ok {
			require.Panics(t, func() {
				MustUint128FromString(s)
			})
			return
		}
		require.Equal(t, expected, MustUint128FromString(s))
	}

	assert(true, u128s("1234"), "1234")
	assert(false, u64(0), "quack")
	assert(false, u64(0), "120481092481092840918209481092380192830912830918230918")
}

func TestUint128Add(t *testing.T) {
	for _, tc := range []struct {
		a, b, c Uint128
	}{
		{u64(1), u64(2), u64(3)},
		{u64(10), u64(3), u64(13)},
		{MaxUint128, u64(1), u64(0)},                            // Overflow wraps
		{u64(maxUint64), u64(1), u128s("18446744073709551616")}, // lo carries to hi
		{u128s("18446744073709551615"), u128s("18446744073709551615"), u128s("36893488147419103230")},
	} {
		t.Run(fmt.Sprintf("%s+%s=%s", tc.a, tc.b, tc.c), func(t *testing.T) {
			require.True(t, tc.c.Equal(tc.a.Add(tc.b)))
		})
	}
}

func TestUint128Add64(t *testing.T) {
	for _, tc := range []struct {
		a Uint128
		b uint64
		c Uint128
	}{
		{u64(1), 2, u64(3)},
		{u64(10), 3, u64(13)},
		{MaxUint128, 1, u64(0)}, // Overflow wraps
	} {
		t.Run(fmt.Sprintf("%s+%d=%s", tc.a, tc.b, tc.c), func(t *testing.T) {
			require.True(t, tc.c.Equal(tc.a.Add64(tc.b)))
		})
	}
}

func TestUint128AsBigInt(t *testing.T) {
	for idx, tc := range []struct {
		a Uint128
		b *big.Int
	}{
		{Uint128{0, 2}, bigU64(2)},
		{Uint128{0xFFFFFFFFFFFFFFFF, 0xFFFFFFFFFFFFFFFE}, bigs("0xFFFFFFFFFFFFFFFF FFFFFFFFFFFFFFFE")},
		{Uint128{0x1, 0x0}, bigs("18446744073709551616")},
		{Uint128{0x1, 0xFFFFFFFFFFFFFFFF}, bigs("36893488147419103231")}, // (1<<65) - 1
		{Uint128{0x1, 0x8AC7230489E7FFFF}, bigs("28446744073709551615")},
		{Uint128{0x7FFFFFFFFFFFFFFF, 0xFFFFFFFFFFFFFFFF}, bigs("170141183460469231731687303715884105727")},
		{Uint128{0xFFFFFFFFFFFFFFFF, 0xFFFFFFFFFFFFFFFF}, bigs("0x FFFFFFFFFFFFFFFF FFFFFFFFFFFFFFFF")},
		{Uint128{0x8000000000000000, 0}, bigs("0x 8000000000000000 0000000000000000")},
	} {
		t.Run(fmt.Sprintf("%d/%d,%d=%s", idx, tc.a.hi, tc.a.lo, tc.b), func(t *testing.T) {
			v := tc.a.AsBigInt()
			require.True(t, tc.b.Cmp(v) == 0, "found: %s", v)
		})
	}
}

func TestUint128AsFloat64Random(t *testing.T) {

	bts := make([]byte, 16)

	for i := 0; i < 10000; i++ {
		rand.Read(bts)

		num := Uint128{}
		num.lo = binary.LittleEndian.Uint64(bts)
		num.hi = binary.LittleEndian.Uint64(bts[8:])

		af := num.AsFloat64()
		bf := new(big.Float).SetFloat64(af)
		rf := num.AsBigFloat()

		diff := new(big.Float).Sub(rf, bf)
		pct := new(big.Float).Quo(diff, rf)
		require.True(t, pct.Cmp(floatDiffLimit) < 0, "%s: %.20f > %.20f", num, diff, floatDiffLimit)
	}
}

func TestUint128AsFloat64Direct(t *testing.T) {
	for _, tc := range []struct {
		a   Uint128
		out string
	}{
		{u128s("2384067163226812360730"), "2384067163226812448768"},
	} {
		t.Run(fmt.Sprintf("float64(%s)=%s", tc.a, tc.out), func(t *testing.T) {
			require.Equal(t, tc.out, cleanFloatStr(fmt.Sprintf("%f", tc.a.AsFloat64())))
		})
	}
}

func TestUint128AsFloat64Epsilon(t *testing.T) {
	for _, tc := range []struct {
		a Uint128
	}{
		{u128s("120")},
		{u128s("12034267329883109062163657840918528")},
		{MaxUint128},
	} {
		t.Run(fmt.Sprintf("float64(%s)", tc.a), func(t *testing.T) {

			af := tc.a.AsFloat64()
			bf := new(big.Float).SetFloat64(af)
			rf := tc.a.AsBigFloat()

			diff := new(big.Float).Sub(rf, bf)
			pct := new(big.Float).Quo(diff, rf)
			require.True(t, pct.Cmp(floatDiffLimit) < 0, fmt.Sprintf("%s: %.20f > %.20f", tc.a, diff, floatDiffLimit))
		})
	}
}

func TestUint128Dec(t *testing.T) {
	for _, tc := range []struct {
		a, b Uint128
	}{
		{u64(1), u64(0)},
		{u64(10), u64(9)},
		{u64(maxUint64), u128s("18446744073709551614")},
		{u64(0), MaxUint128},
		{u64(maxUint64).Add(u64(1)), u64(maxUint64)},
	} {
		t.Run(fmt.Sprintf("%s-1=%s", tc.a, tc.b), func(t *testing.T) {
			dec := tc.a.Dec()
			require.True(t, tc.b.Equal(dec), "%s - 1 != %s, found %s", tc.a, tc.b, dec)
		})
	}
}

func TestUint128Format(t *testing.T) {
	for idx, tc := range []struct {
		v   Uint128
		fmt string
		out string
	}{
		{u64(1), "%d", "1"},
		{u64(1), "%s", "1"},
		{u64(1), "%v", "1"},
		{MaxUint128, "%d", "340282366920938463463374607431768211455"},
		{MaxUint128, "%#d", "340282366920938463463374607431768211455"},
		{MaxUint128, "%o", "3777777777777777777777777777777777777777777"},
		{MaxUint128, "%b", strings.Repeat("1", 128)},
		{MaxUint128, "%#o", "03777777777777777777777777777777777777777777"},
		{MaxUint128, "%#x", "0xffffffffffffffffffffffffffffffff"},
		{MaxUint128, "%#X", "0XFFFFFFFFFFFFFFFFFFFFFFFFFFFFFFFF"},

		// No idea why big.Int doesn't support this:
		// {MaxUint128, "%#b", "0b" + strings.Repeat("1", 128)},
	} {
		t.Run(fmt.Sprintf("%d/%s/%s", idx, tc.fmt, tc.v), func(t *testing.T) {

			result := fmt.Sprintf(tc.fmt, tc.v)
			require.Equal(t, tc.out, result)
		})
	}
}

func TestUint128FromBigInt(t *testing.T) {
	for idx, tc := range []struct {
		a   *big.Int
		b   Uint128
		acc bool
	}{
		{bigU64(2), u64(2), true},
		{bigs("18446744073709551616"), Uint128{hi: 0x1, lo: 0x0}, true},                // 1 << 64
		{bigs("36893488147419103231"), Uint128{hi: 0x1, lo: 0xFFFFFFFFFFFFFFFF}, true}, // (1<<65) - 1
		{bigs("28446744073709551615"), u128s("28446744073709551615"), true},
		{bigs("170141183460469231731687303715884105727"), u128s("170141183460469231731687303715884105727"), true},
		{bigs("0x FFFFFFFFFFFFFFFF FFFFFFFFFFFFFFFF"), Uint128{0xFFFFFFFFFFFFFFFF, 0xFFFFFFFFFFFFFFFF}, true},
		{bigs("0x 1 0000000000000000 00000000000000000"), MaxUint128, false},
		{bigs("0x FFFFFFFFFFFFFFFF FFFFFFFFFFFFFFFF FFFFFFFFFFFFFFFFF"), MaxUint128, false},
	} {
		t.Run(fmt.Sprintf("%d/%s=%d,%d", idx, tc.a, tc.b.lo, tc.b.hi), func(t *testing.T) {

			v, acc := Uint128FromBigInt(tc.a)
			require.Equal(t, acc, tc.acc)
			require.True(t, tc.b.Cmp(v) == 0, "found: (%d, %d), expected (%d, %d)", v.hi, v.lo, tc.b.hi, tc.b.lo)
		})
	}
}

func TestUint128FromFloat64Random(t *testing.T) {

	bts := make([]byte, 16)

	for i := 0; i < 10000; i++ {
		rand.Read(bts)

		num := Uint128{}
		num.lo = binary.LittleEndian.Uint64(bts)
		num.hi = binary.LittleEndian.Uint64(bts[8:])
		rbf := num.AsBigFloat()

		rf, _ := rbf.Float64()
		rn, inRange := Uint128FromFloat64(rf)
		require.True(t, inRange)

		diff := DifferenceUint128(num, rn)

		ibig, diffBig := num.AsBigFloat(), diff.AsBigFloat()
		pct := new(big.Float).Quo(diffBig, ibig)
		require.True(t, pct.Cmp(floatDiffLimit) < 0, "%s: %.20f > %.20f", num, pct, floatDiffLimit)
	}
}

func TestUint128FromFloat64(t *testing.T) {
	for idx, tc := range []struct {
		f       float64
		out     Uint128
		inRange bool
	}{
		{math.NaN(), u128s("0"), false},
		{math.Inf(0), MaxUint128, false},
		{math.Inf(-1), u128s("0"), false},
		{1.0, u64(1), true},

		// {{{ Explore weird corner cases around uint64(float64(math.MaxUint64)) nonsense.
		// 1 greater than maxUint64 because maxUint64 is not representable in a float64 exactly:
		{maxUint64Float, u128s("18446744073709551616"), true},

		// Largest whole number representable in a float64 without exceeding the size of a uint64:
		{maxRepresentableUint64Float, u128s("18446744073709549568"), true},

		// Largest whole number representable in a float64 without exceeding the size of a Uint128:
		{maxRepresentableUint128Float, u128s("340282366920938425684442744474606501888"), true},

		// Not inRange because maxUint128Float is not representable in a float64 exactly:
		{maxUint128Float, MaxUint128, false},
		// }}}
	} {
		t.Run(fmt.Sprintf("%d/fromfloat64(%f)==%s", idx, tc.f, tc.out), func(t *testing.T) {

			rn, inRange := Uint128FromFloat64(tc.f)
			require.Equal(t, tc.inRange, inRange)
			require.Equal(t, tc.out, rn)

			diff := DifferenceUint128(tc.out, rn)

			ibig, diffBig := tc.out.AsBigFloat(), diff.AsBigFloat()
			pct := new(big.Float)
			if diff != zeroUint128 {
				pct.Quo(diffBig, ibig)
			}
			pct.Abs(pct)
			require.True(t, pct.Cmp(floatDiffLimit) < 0, "%.20f -> %s: %.20f > %.20f", tc.f, tc.out, pct, floatDiffLimit)
		})
	}
}

func TestUint128FromI64(t *testing.T) {
	for idx, tc := range []struct {
		in      int64
		out     Uint128
		inRange bool
	}{
		{0, zeroUint128, true},
		{-1, zeroUint128, false},
		{minInt64, zeroUint128, false},
		{maxInt64, u64(0x7fffffffffffffff), true},
	} {
		t.Run(fmt.Sprintf("%d/fromint64(%d)==%s", idx, tc.in, tc.out), func(t *testing.T) {

			rn, inRange := Uint128FromI64(tc.in)
			require.True(t, rn.Equal(tc.out))
			require.Equal(t, tc.inRange, inRange)
		})
	}
}

func TestUint128FromSize(t *testing.T) {

	assertInRange := func(expected Uint128) func(v Uint128, inRange bool) {
		return func(v Uint128, inRange bool) {
			require.Equal(t, expected, v)
			require.True(t, inRange)
		}
	}

	require.Equal(t, Uint128From8(255), u128s("255"))
	require.Equal(t, Uint128From16(65535), u128s("65535"))
	require.Equal(t, Uint128From32(4294967295), u128s("4294967295"))

	assertInRange(u128s("12345"))(Uint128FromFloat32(12345))
	assertInRange(u128s("12345"))(Uint128FromFloat32(12345.6))
	assertInRange(u128s("12345"))(Uint128FromFloat64(12345))
	assertInRange(u128s("12345"))(Uint128FromFloat64(12345.6))
}

func TestUint128Inc(t *testing.T) {
	for _, tc := range []struct {
		a, b Uint128
	}{
		{u64(1), u64(2)},
		{u64(10), u64(11)},
		{u64(maxUint64), u128s("18446744073709551616")},
		{u64(maxUint64), u64(maxUint64).Add(u64(1))},
		{MaxUint128, u64(0)},
	} {
		t.Run(fmt.Sprintf("%s+1=%s", tc.a, tc.b), func(t *testing.T) {

			inc := tc.a.Inc()
			require.True(t, tc.b.Equal(inc), "%s + 1 != %s, found %s", tc.a, tc.b, inc)
		})
	}
}

func TestUint128Lsh(t *testing.T) {
	for idx, tc := range []struct {
		u  Uint128
		by uint
		r  Uint128
	}{
		{u: u64(2), by: 1, r: u64(4)},
		{u: u64(1), by: 2, r: u64(4)},
		{u: u128s("18446744073709551615"), by: 1, r: u128s("36893488147419103230")}, // (1<<64) - 1

		// These cases were found by the fuzzer:
		{u: u128s("5080864651895"), by: 57, r: u128s("732229764895815899943471677440")},
		{u: u128s("63669103"), by: 85, r: u128s("2463079120908903847397520463364096")},
		{u: u128s("0x1f1ecfd29cb51500c1a0699657"), by: 104, r: u128s("0x69965700000000000000000000000000")},
		{u: u128s("0x4ff0d215cf8c26f26344"), by: 58, r: u128s("0xc348573e309bc98d1000000000000000")},
		{u: u128s("0x6b5823decd7ef067f78e8cc3d8"), by: 74, r: u128s("0xc19fde3a330f60000000000000000000")},
		{u: u128s("0x8b93924e1f7b6ac551d66f18ab520a2"), by: 50, r: u128s("0xdab154759bc62ad48288000000000000")},
		{u: u128s("173760885"), by: 68, r: u128s("51285161209860430747989442560")},
		{u: u128s("213"), by: 65, r: u128s("7858312975400268988416")},
		{u: u128s("0x2203b9f3dbe0afa82d80d998641aa0"), by: 75, r: u128s("0x6c06ccc320d500000000000000000000")},
		{u: u128s("40625"), by: 55, r: u128s("1463669878895411200000")},
	} {
		t.Run(fmt.Sprintf("%d/%s<<%d=%s", idx, tc.u, tc.by, tc.r), func(t *testing.T) {

			ub := tc.u.AsBigInt()
			ub.Lsh(ub, tc.by).And(ub, maxBigUint128)

			ru := tc.u.Lsh(tc.by)
			require.Equal(t, tc.r.String(), ru.String(), "%s != %s; big: %s", tc.r, ru, ub)
			require.Equal(t, ub.String(), ru.String())
		})
	}
}

func TestUint128MarshalJSON(t *testing.T) {

	bts := make([]byte, 16)

	for i := 0; i < 5000; i++ {
		u := randUint128(bts)

		bts, err := json.Marshal(u)
		require.NoError(t, err)

		var result Uint128
		require.NoError(t, json.Unmarshal(bts, &result))
		require.True(t, result.Equal(u))
	}
}

func TestUint128Mul(t *testing.T) {

	u := Uint128From64(maxUint64)
	v := u.Mul(Uint128From64(maxUint64))

	var v1, v2 big.Int
	v1.SetUint64(maxUint64)
	v2.SetUint64(maxUint64)
	require.Equal(t, v.String(), v1.Mul(&v1, &v2).String())
}

func TestUint128MustUint64(t *testing.T) {
	for _, tc := range []struct {
		a  Uint128
		ok bool
	}{
		{u64(0), true},
		{u64(1), true},
		{u64(maxInt64), true},
		{u64(maxUint64), true},
		{Uint128FromRaw(1, 0), false},
		{MaxUint128, false},
	} {
		t.Run(fmt.Sprintf("(%s).64?==%v", tc.a, tc.ok), func(t *testing.T) {

			defer func() {
				require.True(t, (recover() == nil) == tc.ok)
			}()

			require.Equal(t, tc.a, Uint128From64(tc.a.MustUint64()))
		})
	}
}

func TestUint128Not(t *testing.T) {
	for idx, tc := range []struct {
		a, b Uint128
	}{
		{u64(0), MaxUint128},
		{u64(1), u128s("340282366920938463463374607431768211454")},
		{u64(2), u128s("340282366920938463463374607431768211453")},
		{u64(maxUint64), u128s("340282366920938463444927863358058659840")},
	} {
		t.Run(fmt.Sprintf("%d/%s=^%s", idx, tc.a, tc.b), func(t *testing.T) {

			out := tc.a.Not()
			require.True(t, tc.b.Equal(out), "^%s != %s, found %s", tc.a, tc.b, out)

			back := out.Not()
			require.True(t, tc.a.Equal(back), "^%s != %s, found %s", out, tc.a, back)
		})
	}
}

func TestUint128QuoRem(t *testing.T) {
	for idx, tc := range []struct {
		u, by, q, r Uint128
	}{
		{u: u64(1), by: u64(2), q: u64(0), r: u64(1)},
		{u: u64(10), by: u64(3), q: u64(3), r: u64(1)},

		// Investigate possible div/0 where lo of divisor is 0:
		{u: Uint128{hi: 0, lo: 1}, by: Uint128{hi: 1, lo: 0}, q: u64(0), r: u64(1)},

		// 128-bit 'cmp == 0' shortcut branch:
		{u128s("0x1234567890123456"), u128s("0x1234567890123456"), u64(1), u64(0)},

		// 128-bit 'cmp < 0' shortcut branch:
		{u128s("0x123456789012345678901234"), u128s("0x222222229012345678901234"), u64(0), u128s("0x123456789012345678901234")},

		// 128-bit 'cmp == 0' shortcut branch:
		{u128s("0x123456789012345678901234"), u128s("0x123456789012345678901234"), u64(1), u64(0)},

		// These test cases were found by the fuzzer and exposed a bug in the 128-bit divisor
		// branch of divmod128by128:
		// 3289699161974853443944280720275488 / 9261249991223143249760: u128(48100516172305203) != big(355211139435)
		// 51044189592896282646990963682604803 / 15356086376658915618524: u128(16290274193854465) != big(3324036368438)
		// 555579170280843546177 / 21475569273528505412: u128(12) != big(25)
	} {
		t.Run(fmt.Sprintf("%d/%s÷%s=%s,%s", idx, tc.u, tc.by, tc.q, tc.r), func(t *testing.T) {

			q, r := tc.u.QuoRem(tc.by)
			require.Equal(t, tc.q.String(), q.String())
			require.Equal(t, tc.r.String(), r.String())

			uBig := tc.u.AsBigInt()
			byBig := tc.by.AsBigInt()

			qBig, rBig := new(big.Int).Set(uBig), new(big.Int).Set(uBig)
			qBig = qBig.Quo(qBig, byBig)
			rBig = rBig.Rem(rBig, byBig)

			require.Equal(t, tc.q.String(), qBig.String())
			require.Equal(t, tc.r.String(), rBig.String())
		})
	}
}

func TestUint128ReverseBytes(t *testing.T) {
	for _, tc := range []struct {
		u Uint128
		r Uint128
	}{
		{
			u: u128s("0x_00_11_22_33_44_55_66_77_88_99_AA_BB_CC_DD_EE_FF"),
			r: u128s("0x_FF_EE_DD_CC_BB_AA_99_88_77_66_55_44_33_22_11_00")},
		{
			u: u128s("0x_00_00_00_00_00_00_00_00_11_22_33_44_55_66_77_88"),
			r: u128s("0x_88_77_66_55_44_33_22_11_00_00_00_00_00_00_00_00")},
	} {
		t.Run(fmt.Sprintf("revbytes-%s=%s", tc.u, tc.r), func(t *testing.T) {

			ru := tc.u.ReverseBytes()
			require.Equal(t, tc.r.String(), ru.String(), "%s != %s", tc.r, ru)
		})
	}
}

func TestUint128Reverse(t *testing.T) {
	for _, tc := range []struct {
		u Uint128
		r Uint128
	}{
		{
			u: u128s("0b_11111111_11111110_11111100_11111000_11110000_11100000_11000000_10000000_11111111_11111110_11111100_11111000_11110000_11100000_11000000_10000000"),
			/*    */ r: u128s("0b_00000001_00000011_00000111_00001111_00011111_00111111_01111111_11111111_00000001_00000011_00000111_00001111_00011111_00111111_01111111_11111111")},
	} {
		t.Run(fmt.Sprintf("revbytes-%s=%s", tc.u, tc.r), func(t *testing.T) {

			ru := tc.u.Reverse()
			require.Equal(t, tc.r.String(), ru.String(), "%s != %s", tc.r, ru)
		})
	}
}

func TestUint128RotateLeft(t *testing.T) {
	for _, tc := range []struct {
		u  Uint128
		by int
		r  Uint128
	}{
		{u: u64(1), by: 1, r: u64(2)},
		{u: u64(1), by: 2, r: u64(4)},
		{u: u128s("0x0000_0000_0000_0000_8000_0000_0000_0000"), by: 1, r: u128s("0x0000_0000_0000_0001_0000_0000_0000_0000")},
		{u: u128s("0x8000_0000_0000_0000_0000_0000_0000_0000"), by: 1, r: u64(1)},
		{u: u128s("0xF000_0000_0000_0000_0000_0000_0000_0000"), by: 4, r: u64(0xF)},

		{by: 1,
			u: u128s("0b_00000000_00000000_00000000_00000000_00000000_00000000_00000000_00000000_00000000_00000000_00000000_00000000_00000000_00000000_00000000_00000001"),
			r: u128s("0b_00000000_00000000_00000000_00000000_00000000_00000000_00000000_00000000_00000000_00000000_00000000_00000000_00000000_00000000_00000000_00000010")},
		{by: 1,
			u: u128s("0b_00000000_00000000_00000000_00000000_00000000_00000000_00000000_00000000_10000000_00000000_00000000_00000000_00000000_00000000_00000000_00000000"),
			r: u128s("0b_00000000_00000000_00000000_00000000_00000000_00000000_00000000_00000001_00000000_00000000_00000000_00000000_00000000_00000000_00000000_00000000")},
		{by: 1,
			u: u128s("0b_10000000_00000000_00000000_00000000_00000000_00000000_00000000_00000000_00000000_00000000_00000000_00000000_00000000_00000000_00000000_00000000"),
			r: u128s("0b_00000000_00000000_00000000_00000000_00000000_00000000_00000000_00000000_00000000_00000000_00000000_00000000_00000000_00000000_00000000_00000001")},
		{by: 64,
			u: u128s("0b_10000000_00000000_00000000_00000000_00000000_00000000_00000000_00000000_00000000_00000000_00000000_00000000_00000000_00000000_00000000_00000000"),
			r: u128s("0b_00000000_00000000_00000000_00000000_00000000_00000000_00000000_00000000_10000000_00000000_00000000_00000000_00000000_00000000_00000000_00000000")},
		{by: 127,
			u: u128s("0b_10000000_00000000_00000000_00000000_00000000_00000000_00000000_00000000_00000000_00000000_00000000_00000000_00000000_00000000_00000000_00000000"),
			r: u128s("0b_01000000_00000000_00000000_00000000_00000000_00000000_00000000_00000000_00000000_00000000_00000000_00000000_00000000_00000000_00000000_00000000")},
		{by: -1,
			u: u128s("0b_10000000_00000000_00000000_00000000_00000000_00000000_00000000_00000000_00000000_00000000_00000000_00000000_00000000_00000000_00000000_00000000"),
			r: u128s("0b_01000000_00000000_00000000_00000000_00000000_00000000_00000000_00000000_00000000_00000000_00000000_00000000_00000000_00000000_00000000_00000000")},
	} {
		t.Run(fmt.Sprintf("%s rotl %d=%s", tc.u, tc.by, tc.r), func(t *testing.T) {

			ru := tc.u.RotateLeft(tc.by)
			require.Equal(t, tc.r.String(), ru.String(), "%s != %s", tc.r, ru)
		})
	}
}

func TestUint128Rsh(t *testing.T) {
	for _, tc := range []struct {
		u  Uint128
		by uint
		r  Uint128
	}{
		{u: u64(2), by: 1, r: u64(1)},
		{u: u64(1), by: 2, r: u64(0)},
		{u: u128s("36893488147419103232"), by: 1, r: u128s("18446744073709551616")}, // (1<<65) - 1

		// These test cases were found by the fuzzer:
		{u: u128s("2465608830469196860151950841431"), by: 104, r: u64(0)},
		{u: u128s("377509308958315595850564"), by: 58, r: u64(1309748)},
		{u: u128s("8504691434450337657905929307096"), by: 74, r: u128s("450234615")},
		{u: u128s("11595557904603123290159404941902684322"), by: 50, r: u128s("10298924295251697538375")},
		{u: u128s("176613673099733424757078556036831904"), by: 75, r: u128s("4674925001596")},
		{u: u128s("3731491383344351937489898072501894878"), by: 112, r: u64(718)},
	} {
		t.Run(fmt.Sprintf("%s>>%d=%s", tc.u, tc.by, tc.r), func(t *testing.T) {

			ub := tc.u.AsBigInt()
			ub.Rsh(ub, tc.by).And(ub, maxBigUint128)

			ru := tc.u.Rsh(tc.by)
			require.Equal(t, tc.r.String(), ru.String(), "%s != %s; big: %s", tc.r, ru, ub)
			require.Equal(t, ub.String(), ru.String())
		})
	}
}

func TestUint128Scan(t *testing.T) {
	for idx, tc := range []struct {
		in  string
		out Uint128
		ok  bool
	}{
		{"1", u64(1), true},
		{"0xFF", zeroUint128, false},
		{"-1", zeroUint128, false},
		{"340282366920938463463374607431768211456", zeroUint128, false},
	} {
		t.Run(fmt.Sprintf("%d/%s==%d", idx, tc.in, tc.out), func(t *testing.T) {

			var result Uint128
			n, err := fmt.Sscan(tc.in, &result)
			require.Equal(t, tc.ok, err == nil, "%v", err)
			if err == nil {
				require.Equal(t, 1, n)
			} else {
				require.Equal(t, 0, n)
			}
			require.Equal(t, tc.out, result)
		})
	}

	for idx, ws := range []string{" ", "\n", "   ", " \t "} {
		t.Run(fmt.Sprintf("scan/3/%d", idx), func(t *testing.T) {

			var a, b, c Uint128
			n, err := fmt.Sscan(strings.Join([]string{"123", "456", "789"}, ws), &a, &b, &c)
			require.Equal(t, 3, n)
			require.NoError(t, err)
			require.Equal(t, "123", a.String())
			require.Equal(t, "456", b.String())
			require.Equal(t, "789", c.String())
		})
	}
}

func TestSetBit(t *testing.T) {
	for i := 0; i < 128; i++ {
		t.Run(fmt.Sprintf("setcheck/%d", i), func(t *testing.T) {

			var u Uint128
			require.Equal(t, uint(0), u.Bit(i))
			u = u.SetBit(i, 1)
			require.Equal(t, uint(1), u.Bit(i))
			u = u.SetBit(i, 0)
			require.Equal(t, uint(0), u.Bit(i))
		})
	}

	for idx, tc := range []struct {
		in  Uint128
		out Uint128
		i   int
		b   uint
	}{
		{in: u64(0), out: u128s("0b 1"), i: 0, b: 1},
		{in: u64(0), out: u128s("0b 10"), i: 1, b: 1},
		{in: u64(0), out: u128s("0x 8000000000000000"), i: 63, b: 1},
		{in: u64(0), out: u128s("0x 10000000000000000"), i: 64, b: 1},
		{in: u64(0), out: u128s("0x 20000000000000000"), i: 65, b: 1},
	} {
		t.Run(fmt.Sprintf("%d/%s/%d/%d", idx, tc.in, tc.i, tc.b), func(t *testing.T) {

			out := tc.in.SetBit(tc.i, tc.b)
			require.Equal(t, tc.out, out)
		})
	}

	for idx, tc := range []struct {
		i int
		b uint
	}{
		{i: -1, b: 0},
		{i: 128, b: 0},
		{i: 0, b: 2},
	} {
		t.Run(fmt.Sprintf("failures/%d/%d/%d", idx, tc.i, tc.b), func(t *testing.T) {
			defer func() {
				if v := recover(); v == nil {
					t.Fatal()
				}
			}()
			var u Uint128
			u.SetBit(tc.i, tc.b)
		})
	}
}

func BenchmarkUint128Add(b *testing.B) {
	for idx, tc := range []struct {
		a, b Uint128
		name string
	}{
		{zeroUint128, zeroUint128, "0+0"},
		{MaxUint128, MaxUint128, "max+max"},
		{u128s("0x7FFFFFFFFFFFFFFF"), u128s("0x7FFFFFFFFFFFFFFF"), "lo-only"},
		{u128s("0xFFFFFFFFFFFFFFFF"), u128s("0x7FFFFFFFFFFFFFFF"), "carry"},
	} {
		b.Run(fmt.Sprintf("%d/%s", idx, tc.name), func(b *testing.B) {
			for i := 0; i < b.N; i++ {
				benchUint128Result = tc.a.Add(tc.b)
			}
		})
	}
}

func BenchmarkUint128Add64(b *testing.B) {
	for idx, tc := range []struct {
		a    Uint128
		b    uint64
		name string
	}{
		{zeroUint128, 0, "0+0"},
		{MaxUint128, maxUint64, "max+max"},
		{u128s("0xFFFFFFFFFFFFFFFF"), 1, "carry"},
	} {
		b.Run(fmt.Sprintf("%d/%s", idx, tc.name), func(b *testing.B) {
			for i := 0; i < b.N; i++ {
				benchUint128Result = tc.a.Add64(tc.b)
			}
		})
	}
}

func BenchmarkUint128AsBigFloat(b *testing.B) {
	n := u128s("36893488147419103230")
	for i := 0; i < b.N; i++ {
		benchBigFloatResult = n.AsBigFloat()
	}
}

func BenchmarkUint128AsBigInt(b *testing.B) {
	u := Uint128{lo: 0xFEDCBA9876543210, hi: 0xFEDCBA9876543210}
	benchBigIntResult = new(big.Int)

	for i := uint(0); i <= 128; i += 32 {
		v := u.Rsh(128 - i)
		b.Run(fmt.Sprintf("%x,%x", v.hi, v.lo), func(b *testing.B) {
			for i := 0; i < b.N; i++ {
				benchBigIntResult = v.AsBigInt()
			}
		})
	}
}

func BenchmarkUint128AsFloat(b *testing.B) {
	n := u128s("36893488147419103230")
	for i := 0; i < b.N; i++ {
		benchFloatResult = n.AsFloat64()
	}
}

var benchUint128CmpCases = []struct {
	a, b Uint128
	name string
}{
	{Uint128From64(maxUint64), Uint128From64(maxUint64), "equal64"},
	{MaxUint128, MaxUint128, "equal128"},
	{u128s("0xFFFFFFFFFFFFFFFF FFFFFFFFFFFFFFFF"), u128s("0xEFFFFFFFFFFFFFFF FFFFFFFFFFFFFFFF"), "lesshi"},
	{u128s("0xEFFFFFFFFFFFFFFF"), u128s("0xFFFFFFFFFFFFFFFF"), "lesslo"},
	{u128s("0xFFFFFFFFFFFFFFFF FFFFFFFFFFFFFFFF"), u128s("0xEFFFFFFFFFFFFFFF FFFFFFFFFFFFFFFF"), "greaterhi"},
	{u128s("0xFFFFFFFFFFFFFFFF"), u128s("0xEFFFFFFFFFFFFFFF"), "greaterlo"},
}

func BenchmarkUint128Cmp(b *testing.B) {
	for _, tc := range benchUint128CmpCases {
		b.Run(fmt.Sprintf("u128cmp/%s", tc.name), func(b *testing.B) {
			for i := 0; i < b.N; i++ {
				benchIntResult = tc.a.Cmp(tc.b)
			}
		})
	}
}

func BenchmarkUint128FromBigInt(b *testing.B) {
	for _, bi := range []*big.Int{
		bigs("0"),
		bigs("0xfedcba98"),
		bigs("0xfedcba9876543210"),
		bigs("0xfedcba9876543210fedcba98"),
		bigs("0xfedcba9876543210fedcba9876543210"),
	} {
		b.Run(fmt.Sprintf("%x", bi), func(b *testing.B) {
			for i := 0; i < b.N; i++ {
				benchUint128Result, _ = Uint128FromBigInt(bi)
			}
		})
	}
}

func BenchmarkUint128FromFloat(b *testing.B) {
	for _, pow := range []float64{1, 63, 64, 65, 127, 128} {
		b.Run(fmt.Sprintf("pow%d", int(pow)), func(b *testing.B) {
			f := math.Pow(2, pow)
			for i := 0; i < b.N; i++ {
				benchUint128Result, _ = Uint128FromFloat64(f)
			}
		})
	}
}

func BenchmarkUint128GreaterThan(b *testing.B) {
	for _, tc := range benchUint128CmpCases {
		b.Run(fmt.Sprintf("u128gt/%s", tc.name), func(b *testing.B) {
			for i := 0; i < b.N; i++ {
				benchBoolResult = tc.a.GreaterThan(tc.b)
			}
		})
	}
}

func BenchmarkUint128GreaterOrEqualTo(b *testing.B) {
	for _, tc := range benchUint128CmpCases {
		b.Run(fmt.Sprintf("u128gte/%s", tc.name), func(b *testing.B) {
			for i := 0; i < b.N; i++ {
				benchBoolResult = tc.a.GreaterOrEqualTo(tc.b)
			}
		})
	}
}

func BenchmarkUint128Inc(b *testing.B) {
	for idx, tc := range []struct {
		name string
		a    Uint128
	}{
		{"0", zeroUint128},
		{"max", MaxUint128},
		{"carry", u64(maxUint64)},
	} {
		b.Run(fmt.Sprintf("%d/%s", idx, tc.name), func(b *testing.B) {
			for i := 0; i < b.N; i++ {
				benchUint128Result = tc.a.Inc()
			}
		})
	}
}

func BenchmarkUint128IntoBigInt(b *testing.B) {
	u := Uint128{lo: 0xFEDCBA9876543210, hi: 0xFEDCBA9876543210}
	benchBigIntResult = new(big.Int)

	for i := uint(0); i <= 128; i += 32 {
		v := u.Rsh(128 - i)
		b.Run(fmt.Sprintf("%x,%x", v.hi, v.lo), func(b *testing.B) {
			for i := 0; i < b.N; i++ {
				v.IntoBigInt(benchBigIntResult)
			}
		})
	}
}

func BenchmarkUint128LessThan(b *testing.B) {
	for _, tc := range benchUint128CmpCases {
		b.Run(fmt.Sprintf("u128lt/%s", tc.name), func(b *testing.B) {
			for i := 0; i < b.N; i++ {
				benchBoolResult = tc.a.LessThan(tc.b)
			}
		})
	}
}

func BenchmarkUint128Lsh(b *testing.B) {
	for _, tc := range []struct {
		in Uint128
		sh uint
	}{
		{u64(maxUint64), 1},
		{u64(maxUint64), 2},
		{u64(maxUint64), 8},
		{u64(maxUint64), 64},
		{u64(maxUint64), 126},
		{u64(maxUint64), 127},
		{u64(maxUint64), 128},
		{MaxUint128, 1},
		{MaxUint128, 2},
		{MaxUint128, 8},
		{MaxUint128, 64},
		{MaxUint128, 126},
		{MaxUint128, 127},
		{MaxUint128, 128},
	} {
		b.Run(fmt.Sprintf("%s>>%d", tc.in, tc.sh), func(b *testing.B) {
			for i := 0; i < b.N; i++ {
				benchUint128Result = tc.in.Lsh(tc.sh)
			}
		})
	}
}

func BenchmarkUint128Mul(b *testing.B) {
	u := Uint128From64(maxUint64)
	for i := 0; i < b.N; i++ {
		benchUint128Result = u.Mul(u)
	}
}

func BenchmarkUint128Mul64(b *testing.B) {
	u := Uint128From64(maxUint64)
	lim := uint64(b.N)
	for i := uint64(0); i < lim; i++ {
		benchUint128Result = u.Mul64(i)
	}
}

var benchQuoCases = []struct {
	name     string
	dividend Uint128
	divisor  Uint128
}{
	// 128-bit divide by 1 branch:
	{"128bit/1", MaxUint128, u64(1)},

	// 128-bit divide by power of 2 branch:
	{"128bit/pow2", MaxUint128, u64(2)},

	// 64-bit divide by 1 branch:
	{"64-bit/1", u64(maxUint64), u64(1)},

	// 128-bit divisor lz+tz > threshold branch:
	{"128bit/lz+tz>thresh", u128s("0x123456789012345678901234567890"), u128s("0xFF0000000000000000000")},

	// 128-bit divisor lz+tz <= threshold branch:
	{"128bit/lz+tz<=thresh", u128s("0x12345678901234567890123456789012"), u128s("0x10000000000000000000000000000001")},

	// 128-bit 'cmp == 0' shortcut branch:
	{"128bit/samesies", u128s("0x1234567890123456"), u128s("0x1234567890123456")},
}

func BenchmarkUint128Quo(b *testing.B) {
	for idx, bc := range benchQuoCases {
		b.Run(fmt.Sprintf("%d/%s", idx, bc.name), func(b *testing.B) {
			for i := 0; i < b.N; i++ {
				benchUint128Result = bc.dividend.Quo(bc.divisor)
			}
		})
	}
}

func BenchmarkUint128QuoRem(b *testing.B) {
	for idx, bc := range benchQuoCases {
		b.Run(fmt.Sprintf("%d/%s", idx, bc.name), func(b *testing.B) {
			for i := 0; i < b.N; i++ {
				benchUint128Result, _ = bc.dividend.QuoRem(bc.divisor)
			}
		})
	}
}

func BenchmarkUint128QuoRemTZ(b *testing.B) {
	type tc struct {
		zeros  int
		useRem int
		da, db Uint128
	}
	var cases []tc

	// If there's a big jump in speed from one of these cases to the next, it
	// could be indicative that the algorithm selection spill point
	// (divAlgoLeading0Spill) needs to change.
	//
	// This could probably be automated a little better, and the result is also
	// likely platform and possibly CPU specific.
	for zeros := 0; zeros < 31; zeros++ {
		for useRem := 0; useRem < 2; useRem++ {
			bs := "0b"
			for j := 0; j < 128; j++ {
				if j >= zeros {
					bs += "1"
				} else {
					bs += "0"
				}
			}

			da := u128s("0x98765432109876543210987654321098")
			db := u128s(bs)
			cases = append(cases, tc{
				zeros:  zeros,
				useRem: useRem,
				da:     da,
				db:     db,
			})
		}
	}

	for _, tc := range cases {
		b.Run(fmt.Sprintf("z=%d/rem=%v", tc.zeros, tc.useRem == 1), func(b *testing.B) {
			for i := 0; i < b.N; i++ {
				if tc.useRem == 1 {
					benchUint128Result, _ = tc.da.QuoRem(tc.db)
				} else {
					benchUint128Result = tc.da.Quo(tc.db)
				}
			}
		})
	}
}

func BenchmarkUint128QuoRem64(b *testing.B) {
	// FIXME: benchmark numbers of various sizes
	u, v := u64(1234), uint64(56)
	for i := 0; i < b.N; i++ {
		benchUint128Result, _ = u.QuoRem64(v)
	}
}

func BenchmarkUint128QuoRem64TZ(b *testing.B) {
	type tc struct {
		zeros  int
		useRem int
		da     Uint128
		db     uint64
	}
	var cases []tc

	for zeros := 0; zeros < 31; zeros++ {
		for useRem := 0; useRem < 2; useRem++ {
			bs := "0b"
			for j := 0; j < 64; j++ {
				if j >= zeros {
					bs += "1"
				} else {
					bs += "0"
				}
			}

			da := u128s("0x98765432109876543210987654321098")
			db128 := u128s(bs)
			if !db128.IsUint64() {
				panic("oh dear!")
			}
			db := db128.AsUint64()

			cases = append(cases, tc{
				zeros:  zeros,
				useRem: useRem,
				da:     da,
				db:     db,
			})
		}
	}

	for _, tc := range cases {
		b.Run(fmt.Sprintf("z=%d/rem=%v", tc.zeros, tc.useRem == 1), func(b *testing.B) {
			for i := 0; i < b.N; i++ {
				if tc.useRem == 1 {
					benchUint128Result, _ = tc.da.QuoRem64(tc.db)
				} else {
					benchUint128Result = tc.da.Quo64(tc.db)
				}
			}
		})
	}
}

func BenchmarkUint128Rem64(b *testing.B) {
	b.Run("fast", func(b *testing.B) {
		u, v := Uint128{1, 0}, uint64(56) // u.hi < v
		for i := 0; i < b.N; i++ {
			benchUint128Result = u.Rem64(v)
		}
	})

	b.Run("slow", func(b *testing.B) {
		u, v := Uint128{100, 0}, uint64(56) // u.hi >= v
		for i := 0; i < b.N; i++ {
			benchUint128Result = u.Rem64(v)
		}
	})
}

func BenchmarkUint128String(b *testing.B) {
	for _, bi := range []Uint128{
		u128s("0"),
		u128s("0xfedcba98"),
		u128s("0xfedcba9876543210"),
		u128s("0xfedcba9876543210fedcba98"),
		u128s("0xfedcba9876543210fedcba9876543210"),
	} {
		b.Run(fmt.Sprintf("%x", bi.AsBigInt()), func(b *testing.B) {
			for i := 0; i < b.N; i++ {
				benchStringResult = bi.String()
			}
		})
	}
}

func BenchmarkUint128Sub(b *testing.B) {
	for idx, tc := range []struct {
		name string
		a, b Uint128
	}{
		{"0+0", zeroUint128, zeroUint128},
		{"0-max", zeroUint128, MaxUint128},
		{"lo-only", u128s("0x7FFFFFFFFFFFFFFF"), u128s("0x7FFFFFFFFFFFFFFF")},
		{"carry", MaxUint128, u64(maxUint64).Add64(1)},
	} {
		b.Run(fmt.Sprintf("%d/%s", idx, tc.name), func(b *testing.B) {
			for i := 0; i < b.N; i++ {
				benchUint128Result = tc.a.Sub(tc.b)
			}
		})
	}
}

func BenchmarkUint128Sub64(b *testing.B) {
	for idx, tc := range []struct {
		a    Uint128
		b    uint64
		name string
	}{
		{zeroUint128, 0, "0+0"},
		{MaxUint128, maxUint64, "max+max"},
		{u128s("0xFFFFFFFFFFFFFFFF"), 1, "carry"},
	} {
		b.Run(fmt.Sprintf("%d/%s", idx, tc.name), func(b *testing.B) {
			for i := 0; i < b.N; i++ {
				benchUint128Result = tc.a.Sub64(tc.b)
			}
		})
	}
}

func BenchmarkUint128MustUint128FromBigEndian(b *testing.B) {
	var bts = make([]byte, 16)
	rand.Read(bts)
	for i := 0; i < b.N; i++ {
		benchUint128Result = MustUint128FromBigEndian(bts)
	}
}

var trimFloatPattern = regexp.MustCompile(`(\.0+$|(\.\d+[1-9])\0+$)`)

func cleanFloatStr(str string) string {
	return trimFloatPattern.ReplaceAllString(str, "$2")
}
