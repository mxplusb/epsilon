package geometry

import (
	"crypto/rand"
	"encoding/binary"
	"encoding/json"
	"encoding/xml"
	"fmt"
	"math"
	"math/big"
	"strings"
	"testing"

	"github.com/stretchr/testify/require"
)

var i64 = Int128From64

func bigI64(i int64) *big.Int { return new(big.Int).SetInt64(i) }
func bigs(s string) *big.Int {
	v, _ := new(big.Int).SetString(strings.Replace(s, " ", "", -1), 0)
	return v
}

func i128s(s string) Int128 {
	s = strings.Replace(s, " ", "", -1)
	b, ok := new(big.Int).SetString(s, 0)
	if !ok {
		panic(s)
	}
	i, acc := Int128FromBigInt(b)
	if !acc {
		panic(fmt.Errorf("num: inaccurate i128 %s", s))
	}
	return i
}

func randInt128(scratch []byte) Int128 {
	rand.Read(scratch)
	i := Int128{}
	i.lo = binary.LittleEndian.Uint64(scratch)

	if scratch[0]%2 == 1 {
		// if we always generate hi bits, the universe will die before we
		// test a number < maxInt64
		i.hi = binary.LittleEndian.Uint64(scratch[8:])
	}
	if scratch[1]%2 == 1 {
		i = i.Neg()
	}
	return i
}

func TestInt128Abs(t *testing.T) {
	for idx, tc := range []struct {
		a, b Int128
	}{
		{i64(0), i64(0)},
		{i64(1), i64(1)},
		{Int128{lo: maxUint64}, Int128{lo: maxUint64}},
		{i64(-1), i64(1)},
		{Int128{hi: maxUint64}, Int128{hi: 1}},

		{MaxInt128, MaxInt128}, // Should work
		{MinInt128, MinInt128}, // Overflow
	} {
		t.Run(fmt.Sprintf("%d/|%s|=%s", idx, tc.a, tc.b), func(t *testing.T) {
			
			result := tc.a.Abs()
			require.Equal(t,tc.b, result)
		})
	}
}

func TestInt128AbsUint128(t *testing.T) {
	for idx, tc := range []struct {
		a Int128
		b Uint128
	}{
		{i64(0), u64(0)},
		{i64(1), u64(1)},
		{Int128{lo: maxUint64}, Uint128{lo: maxUint64}},
		{i64(-1), u64(1)},
		{Int128{hi: maxUint64}, Uint128{hi: 1}},

		{MinInt128, minInt128AsAbsUint128}, // Overflow does not affect this function
	} {
		t.Run(fmt.Sprintf("%d/|%s|=%s", idx, tc.a, tc.b), func(t *testing.T) {
			
			result := tc.a.AbsUint128()
			require.Equal(t,tc.b, result)
		})
	}
}

func TestInt128Add(t *testing.T) {
	for idx, tc := range []struct {
		a, b, c Int128
	}{
		{i64(-2), i64(-1), i64(-3)},
		{i64(-2), i64(1), i64(-1)},
		{i64(-1), i64(1), i64(0)},
		{i64(1), i64(2), i64(3)},
		{i64(10), i64(3), i64(13)},

		// Hi/lo carry:
		{Int128{lo: 0xFFFFFFFFFFFFFFFF}, i64(1), Int128{hi: 1, lo: 0}},
		{Int128{hi: 1, lo: 0}, i64(-1), Int128{lo: 0xFFFFFFFFFFFFFFFF}},

		// Overflow:
		{Int128{hi: 0xFFFFFFFFFFFFFFFF, lo: 0xFFFFFFFFFFFFFFFF}, i64(1), Int128{}},

		// Overflow wraps:
		{MaxInt128, i64(1), MinInt128},
	} {
		t.Run(fmt.Sprintf("%d/%s+%s=%s", idx, tc.a, tc.b, tc.c), func(t *testing.T) {
			
			require.True(t,tc.c.Equal(tc.a.Add(tc.b)))
		})
	}
}

func TestInt128Add64(t *testing.T) {
	for _, tc := range []struct {
		a Int128
		b int64
		c Int128
	}{
		{i64(1), 2, i64(3)},
		{i64(10), 3, i64(13)},
		{MaxInt128, 1, MinInt128}, // Overflow wraps
	} {
		t.Run(fmt.Sprintf("%s+%d=%s", tc.a, tc.b, tc.c), func(t *testing.T) {
			
			require.True(t,tc.c.Equal(tc.a.Add64(tc.b)))
		})
	}
}

func TestInt128AsBigIntAndIntoBigInt(t *testing.T) {
	for idx, tc := range []struct {
		a Int128
		b *big.Int
	}{
		{Int128{0, 2}, bigI64(2)},
		{Int128{0xFFFFFFFFFFFFFFFF, 0xFFFFFFFFFFFFFFFE}, bigI64(-2)},
		{Int128{0x1, 0x0}, bigs("18446744073709551616")},
		{Int128{0x1, 0xFFFFFFFFFFFFFFFF}, bigs("36893488147419103231")}, // (1<<65) - 1
		{Int128{0x1, 0x8AC7230489E7FFFF}, bigs("28446744073709551615")},
		{Int128{0x7FFFFFFFFFFFFFFF, 0xFFFFFFFFFFFFFFFF}, bigs("170141183460469231731687303715884105727")},
		{Int128{0xFFFFFFFFFFFFFFFF, 0xFFFFFFFFFFFFFFFF}, bigs("-1")},
		{Int128{0x8000000000000000, 0}, bigs("-170141183460469231731687303715884105728")},
	} {
		t.Run(fmt.Sprintf("%d/%d,%d=%s", idx, tc.a.hi, tc.a.lo, tc.b), func(t *testing.T) {
			
			v := tc.a.AsBigInt()
			require.True(t,tc.b.Cmp(v) == 0, "found: %s", v)

			var v2 big.Int
			tc.a.IntoBigInt(&v2)
			require.True(t,tc.b.Cmp(&v2) == 0, "found: %s", v2)
		})
	}
}

func TestInt128AsFloat64Random(t *testing.T) {
	

	bts := make([]byte, 16)

	for i := 0; i < 1000; i++ {
		for bits := uint(1); bits <= 127; bits++ {
			rand.Read(bts)

			var loMask, hiMask uint64
			var loSet, hiSet uint64
			if bits > 64 {
				loMask = maxUint64
				hiMask = (1 << (bits - 64)) - 1
				hiSet = 1 << (bits - 64 - 1)
			} else {
				loMask = (1 << bits) - 1
				loSet = 1 << (bits - 1)
			}

			num := Int128{}
			num.lo = (binary.LittleEndian.Uint64(bts) & loMask) | loSet
			num.hi = (binary.LittleEndian.Uint64(bts[8:]) & hiMask) | hiSet

			for neg := 0; neg <= 1; neg++ {
				if neg == 1 {
					num = num.Neg()
				}

				af := num.AsFloat64()
				bf := new(big.Float).SetFloat64(af)
				rf := num.AsBigFloat()

				diff := new(big.Float).Sub(rf, bf)
				pct := new(big.Float).Quo(diff, rf)
				require.True(t,pct.Cmp(floatDiffLimit) < 0, "%s: %.20f > %.20f", num, diff, floatDiffLimit)
			}
		}
	}
}

func TestInt128AsFloat64(t *testing.T) {
	for _, tc := range []struct {
		a Int128
	}{
		{i128s("-120")},
		{i128s("12034267329883109062163657840918528")},
		{MaxInt128},
	} {
		t.Run(fmt.Sprintf("float64(%s)", tc.a), func(t *testing.T) {
			

			af := tc.a.AsFloat64()
			bf := new(big.Float).SetFloat64(af)
			rf := tc.a.AsBigFloat()

			diff := new(big.Float).Sub(rf, bf)
			pct := new(big.Float).Quo(diff, rf)
			require.True(t,pct.Cmp(floatDiffLimit) < 0, "%s: %.20f > %.20f", tc.a, diff, floatDiffLimit)
		})
	}
}

func TestInt128AsInt64(t *testing.T) {
	for idx, tc := range []struct {
		a   Int128
		out int64
	}{
		{i64(-1), -1},
		{i64(minInt64), minInt64},
		{i64(maxInt64), maxInt64},
		{i128s("9223372036854775808"), minInt64},  // (maxInt64 + 1) overflows to min
		{i128s("-9223372036854775809"), maxInt64}, // (minInt64 - 1) underflows to max
	} {
		t.Run(fmt.Sprintf("%d/int64(%s)=%d", idx, tc.a, tc.out), func(t *testing.T) {
			
			iv := tc.a.AsInt64()
			require.Equal(t,tc.out, iv)
		})
	}
}

func TestInt128Cmp(t *testing.T) {
	for idx, tc := range []struct {
		a, b   Int128
		result int
	}{
		{i64(0), i64(0), 0},
		{i64(1), i64(0), 1},
		{i64(10), i64(9), 1},
		{i64(-1), i64(1), -1},
		{i64(1), i64(-1), 1},
		{MinInt128, MaxInt128, -1},
	} {
		t.Run(fmt.Sprintf("%d/%s-1=%s", idx, tc.a, tc.b), func(t *testing.T) {
			
			result := tc.a.Cmp(tc.b)
			require.Equal(t,tc.result, result)
		})
	}
}

func TestInt128Dec(t *testing.T) {
	for _, tc := range []struct {
		a, b Int128
	}{
		{i64(1), i64(0)},
		{i64(10), i64(9)},
		{MinInt128, MaxInt128}, // underflow
		{Int128{hi: 1}, Int128{lo: 0xFFFFFFFFFFFFFFFF}}, // carry
	} {
		t.Run(fmt.Sprintf("%s-1=%s", tc.a, tc.b), func(t *testing.T) {
			
			dec := tc.a.Dec()
			require.True(t,tc.b.Equal(dec), "%s - 1 != %s, found %s", tc.a, tc.b, dec)
		})
	}
}

func TestInt128Format(t *testing.T) {
	for _, tc := range []struct {
		in  Int128
		f   string
		out string
	}{
		{i64(123456789), "%d", "123456789"},
		{i64(12), "%2d", "12"},
		{i64(12), "%3d", " 12"},
		{i64(12), "%02d", "12"},
		{i64(12), "%03d", "012"},
		{i64(123456789), "%s", "123456789"},
	} {
		t.Run("", func(t *testing.T) {
			
			require.Equal(t,tc.out, fmt.Sprintf(tc.f, tc.in))
		})
	}
}

func TestInt128From64(t *testing.T) {
	for idx, tc := range []struct {
		in  int64
		out Int128
	}{
		{0, i64(0)},
		{maxInt64, i128s("0x7F FF FF FF FF FF FF FF")},
		{-1, i128s("-1")},
		{minInt64, i128s("-9223372036854775808")},
	} {
		t.Run(fmt.Sprintf("%d/%d=%s", idx, tc.in, tc.out), func(t *testing.T) {
			
			result := Int128From64(tc.in)
			require.Equal(t,tc.out, result, "found: (%d, %d), expected (%d, %d)", result.hi, result.lo, tc.out.hi, tc.out.lo)
		})
	}
}

func TestInt128FromBigInt(t *testing.T) {
	for idx, tc := range []struct {
		a *big.Int
		b Int128
	}{
		{bigI64(0), i64(0)},
		{bigI64(2), i64(2)},
		{bigI64(-2), i64(-2)},
		{bigs("18446744073709551616"), Int128{0x1, 0x0}}, // 1 << 64
		{bigs("36893488147419103231"), Int128{0x1, 0xFFFFFFFFFFFFFFFF}}, // (1<<65) - 1
		{bigs("28446744073709551615"), i128s("28446744073709551615")},
		{bigs("170141183460469231731687303715884105727"), i128s("170141183460469231731687303715884105727")},
		{bigs("-1"), Int128{0xFFFFFFFFFFFFFFFF, 0xFFFFFFFFFFFFFFFF}},
	} {
		t.Run(fmt.Sprintf("%d/%s=%d,%d", idx, tc.a, tc.b.lo, tc.b.hi), func(t *testing.T) {
			
			v := accInt128FromBigInt(tc.a)
			require.True(t,tc.b.Cmp(v) == 0, "found: (%d, %d), expected (%d, %d)", v.hi, v.lo, tc.b.hi, tc.b.lo)
		})
	}
}

func TestInt128FromFloat64(t *testing.T) {
	for idx, tc := range []struct {
		f       float64
		out     Int128
		inRange bool
	}{
		{math.NaN(), i128s("0"), false},
		{math.Inf(0), MaxInt128, false},
		{math.Inf(-1), MinInt128, false},
	} {
		t.Run(fmt.Sprintf("%d/fromfloat64(%f)==%s", idx, tc.f, tc.out), func(t *testing.T) {
			

			rn, inRange := Int128FromFloat64(tc.f)
			require.Equal(t,tc.inRange, inRange)
			diff := DifferenceInt128(tc.out, rn)

			ibig, diffBig := tc.out.AsBigFloat(), diff.AsBigFloat()
			pct := new(big.Float)
			if diff != zeroInt128 {
				pct.Quo(diffBig, ibig)
			}
			pct.Abs(pct)
			require.True(t,pct.Cmp(floatDiffLimit) < 0, "%s: %.20f > %.20f", tc.out, pct, floatDiffLimit)
		})
	}
}

func TestInt128FromFloat64Random(t *testing.T) {
	

	bts := make([]byte, 16)

	for i := 0; i < 100000; i++ {
		rand.Read(bts)

		num := Int128{}
		num.lo = binary.LittleEndian.Uint64(bts)
		num.hi = binary.LittleEndian.Uint64(bts[8:])
		rbf := num.AsBigFloat()

		rf, _ := rbf.Float64()
		rn, acc := Int128FromFloat64(rf)
		require.True(t,acc)
		diff := DifferenceInt128(num, rn)

		ibig, diffBig := num.AsBigFloat(), diff.AsBigFloat()
		pct := new(big.Float).Quo(diffBig, ibig)
		require.True(t,pct.Cmp(floatDiffLimit) < 0, "%s: %.20f > %.20f", num, pct, floatDiffLimit)
	}
}

func TestInt128FromSize(t *testing.T) {
	
	require.Equal(t,Int128From8(127), i128s("127"))
	require.Equal(t,Int128From8(-128), i128s("-128"))
	require.Equal(t,Int128From16(32767), i128s("32767"))
	require.Equal(t,Int128From16(-32768), i128s("-32768"))
	require.Equal(t,Int128From32(2147483647), i128s("2147483647"))
	require.Equal(t,Int128From32(-2147483648), i128s("-2147483648"))
}

func TestInt128Inc(t *testing.T) {
	for idx, tc := range []struct {
		a, b Int128
	}{
		{i64(-1), i64(0)},
		{i64(-2), i64(-1)},
		{i64(1), i64(2)},
		{i64(10), i64(11)},
		{i64(maxInt64), i128s("9223372036854775808")},
		{i128s("18446744073709551616"), i128s("18446744073709551617")},
		{i128s("-18446744073709551617"), i128s("-18446744073709551616")},
	} {
		t.Run(fmt.Sprintf("%d/%s+1=%s", idx, tc.a, tc.b), func(t *testing.T) {
			
			inc := tc.a.Inc()
			require.True(t,tc.b.Equal(inc), "%s + 1 != %s, found %s", tc.a, tc.b, inc)
		})
	}
}

func TestInt128IsInt64(t *testing.T) {
	for idx, tc := range []struct {
		a  Int128
		is bool
	}{
		{i64(-1), true},
		{i64(minInt64), true},
		{i64(maxInt64), true},
		{i128s("9223372036854775808"), false},  // (maxInt64 + 1)
		{i128s("-9223372036854775809"), false}, // (minInt64 - 1)
	} {
		t.Run(fmt.Sprintf("%d/isint64(%s)=%v", idx, tc.a, tc.is), func(t *testing.T) {
			
			iv := tc.a.IsInt64()
			require.Equal(t,tc.is, iv)
		})
	}
}

func TestInt128MarshalJSON(t *testing.T) {
	
	bts := make([]byte, 16)

	for i := 0; i < 5000; i++ {
		n := randInt128(bts)

		bts, err := json.Marshal(n)
		require.NoError(t,err)

		var result Int128
		require.NoError(t,json.Unmarshal(bts, &result))
		require.True(t,result.Equal(n))
	}
}

func TestInt128MarshalText(t *testing.T) {
	
	bts := make([]byte, 16)

	type Encoded struct {
		Num Int128
	}

	for i := 0; i < 5000; i++ {
		n := randInt128(bts)

		var v = Encoded{Num: n}

		out, err := xml.Marshal(&v)
		require.NoError(t,err)

		require.Equal(t,fmt.Sprintf("<Encoded><Num>%s</Num></Encoded>", n.String()), string(out))

		var v2 Encoded
		require.NoError(t,xml.Unmarshal(out, &v2))

		require.Equal(t,v2.Num, n)
	}
}

func TestInt128Mul(t *testing.T) {
	for _, tc := range []struct {
		a, b, out Int128
	}{
		{i64(1), i64(0), i64(0)},
		{i64(-2), i64(2), i64(-4)},
		{i64(-2), i64(-2), i64(4)},
		{i64(10), i64(9), i64(90)},
		{i64(maxInt64), i64(maxInt64), i128s("85070591730234615847396907784232501249")},
		{i64(minInt64), i64(minInt64), i128s("85070591730234615865843651857942052864")},
		{i64(minInt64), i64(maxInt64), i128s("-85070591730234615856620279821087277056")},
		{MaxInt128, i64(2), i128s("-2")}, // Overflow. "math.MaxInt64 * 2" produces the same result, "-2".
		{MaxInt128, MaxInt128, i128s("1")}, // Overflow
	} {
		t.Run(fmt.Sprintf("%s*%s=%s", tc.a, tc.b, tc.out), func(t *testing.T) {
			

			v := tc.a.Mul(tc.b)
			require.True(t,tc.out.Equal(v), "%s * %s != %s, found %s", tc.a, tc.b, tc.out, v)
		})
	}
}

func TestInt128MustInt64(t *testing.T) {
	for _, tc := range []struct {
		a  Int128
		ok bool
	}{
		{i64(0), true},
		{i64(1), true},
		{i64(maxInt64), true},
		{i128s("9223372036854775808"), false},
		{MaxInt128, false},

		{i64(-1), true},
		{i64(minInt64), true},
		{i128s("-9223372036854775809"), false},
		{MinInt128, false},
	} {
		t.Run(fmt.Sprintf("(%s).64?==%v", tc.a, tc.ok), func(t *testing.T) {
			
			defer func() {
				require.True(t,(recover() == nil) == tc.ok)
			}()

			require.Equal(t,tc.a, Int128From64(tc.a.MustInt64()))
		})
	}
}

func TestInt128Neg(t *testing.T) {
	for idx, tc := range []struct {
		a, b Int128
	}{
		{i64(0), i64(0)},
		{i64(-2), i64(2)},
		{i64(2), i64(-2)},

		// hi/lo carry:
		{Int128{lo: 0xFFFFFFFFFFFFFFFF}, Int128{hi: 0xFFFFFFFFFFFFFFFF, lo: 1}},
		{Int128{hi: 0xFFFFFFFFFFFFFFFF, lo: 1}, Int128{lo: 0xFFFFFFFFFFFFFFFF}},

		// These cases popped up as a weird regression when refactoring Int128FromBigInt:
		{i128s("18446744073709551616"), i128s("-18446744073709551616")},
		{i128s("-18446744073709551616"), i128s("18446744073709551616")},
		{i128s("-18446744073709551617"), i128s("18446744073709551617")},
		{Int128{hi: 1, lo: 0}, Int128{hi: 0xFFFFFFFFFFFFFFFF, lo: 0x0}},

		{i128s("28446744073709551615"), i128s("-28446744073709551615")},
		{i128s("-28446744073709551615"), i128s("28446744073709551615")},

		// Negating MaxInt128 should yield MinInt128 + 1:
		{Int128{hi: 0x7FFFFFFFFFFFFFFF, lo: 0xFFFFFFFFFFFFFFFF}, Int128{hi: 0x8000000000000000, lo: 1}},

		// Negating MinInt128 should yield MinInt128:
		{Int128{hi: 0x8000000000000000, lo: 0}, Int128{hi: 0x8000000000000000, lo: 0}},

		{i128s("-170141183460469231731687303715884105728"), i128s("-170141183460469231731687303715884105728")},
	} {
		t.Run(fmt.Sprintf("%d/-%s=%s", idx, tc.a, tc.b), func(t *testing.T) {
			
			result := tc.a.Neg()
			require.True(t,tc.b.Equal(result))
		})
	}
}

func TestInt128QuoRem(t *testing.T) {
	for _, tc := range []struct {
		i, by, q, r Int128
	}{
		{i: i64(1), by: i64(2), q: i64(0), r: i64(1)},
		{i: i64(10), by: i64(3), q: i64(3), r: i64(1)},
		{i: i64(10), by: i64(-3), q: i64(-3), r: i64(1)},
		{i: i64(10), by: i64(10), q: i64(1), r: i64(0)},

		// Hit the 128-bit division 'lz+tz == 127' branch:
		{i: i128s("0x10000000000000000"), by: i128s("0x10000000000000000"), q: i64(1), r: i64(0)},

		// Hit the 128-bit division 'cmp == 0' branch
		{i: i128s("0x12345678901234567"), by: i128s("0x12345678901234567"), q: i64(1), r: i64(0)},

		{i: MinInt128, by: i64(-1), q: MinInt128, r: zeroInt128},
	} {
		t.Run(fmt.Sprintf("%s÷%s=%s,%s", tc.i, tc.by, tc.q, tc.r), func(t *testing.T) {
			
			q, r := tc.i.QuoRem(tc.by)
			require.Equal(t,tc.q.String(), q.String())
			require.Equal(t,tc.r.String(), r.String())

			// Skip the weird overflow edge case where we divide MinInt128 by -1:
			// this effectively becomes a negation operation, which overflows:
			//
			//   -170141183460469231731687303715884105728 / -1 == -170141183460469231731687303715884105728
			//
			if tc.i != MinInt128 {
				iBig := tc.i.AsBigInt()
				byBig := tc.by.AsBigInt()

				qBig, rBig := new(big.Int).Set(iBig), new(big.Int).Set(iBig)
				qBig = qBig.Div(qBig, byBig)
				rBig = rBig.Mod(rBig, byBig)

				require.Equal(t,tc.q.String(), qBig.String())
				require.Equal(t,tc.r.String(), rBig.String())
			}
		})
	}
}

func TestInt128Scan(t *testing.T) {
	for idx, tc := range []struct {
		in  string
		out Int128
		ok  bool
	}{
		{"1", i64(1), true},
		{"0xFF", zeroInt128, false},
		{"-1", i64(-1), true},
		{"170141183460469231731687303715884105728", zeroInt128, false},
		{"-170141183460469231731687303715884105729", zeroInt128, false},
	} {
		t.Run(fmt.Sprintf("%d/%s==%d", idx, tc.in, tc.out), func(t *testing.T) {
			
			var result Int128
			n, err := fmt.Sscan(tc.in, &result)
			require.Equal(t,tc.ok, err == nil, "%v", err)
			if err == nil {
				require.Equal(t,1, n)
			} else {
				require.Equal(t,0, n)
			}
			require.Equal(t,tc.out, result)
		})
	}
}

func TestInt128Sign(t *testing.T) {
	for idx, tc := range []struct {
		a    Int128
		sign int
	}{
		{i64(0), 0},
		{i64(1), 1},
		{i64(-1), -1},
		{MinInt128, -1},
		{MaxInt128, 1},
	} {
		t.Run(fmt.Sprintf("%d/%s==%d", idx, tc.a, tc.sign), func(t *testing.T) {
			
			result := tc.a.Sign()
			require.Equal(t,tc.sign, result)
		})
	}
}

func TestInt128Sub(t *testing.T) {
	for idx, tc := range []struct {
		a, b, c Int128
	}{
		{i64(-2), i64(-1), i64(-1)},
		{i64(-2), i64(1), i64(-3)},
		{i64(2), i64(1), i64(1)},
		{i64(2), i64(-1), i64(3)},
		{i64(1), i64(2), i64(-1)},  // crossing zero
		{i64(-1), i64(-2), i64(1)}, // crossing zero

		{MinInt128, i64(1), MaxInt128},  // Overflow wraps
		{MaxInt128, i64(-1), MinInt128}, // Overflow wraps

		{i128s("0x10000000000000000"), i64(1), i128s("0xFFFFFFFFFFFFFFFF")},  // carry down
		{i128s("0xFFFFFFFFFFFFFFFF"), i64(-1), i128s("0x10000000000000000")}, // carry up

		// {i64(maxInt64), i64(1), i128s("18446744073709551616")}, // lo carries to hi
		// {i128s("18446744073709551615"), i128s("18446744073709551615"), i128s("36893488147419103230")},
	} {
		t.Run(fmt.Sprintf("%d/%s-%s=%s", idx, tc.a, tc.b, tc.c), func(t *testing.T) {
			
			require.True(t,tc.c.Equal(tc.a.Sub(tc.b)))
		})
	}
}

func TestInt128Sub64(t *testing.T) {
	for idx, tc := range []struct {
		a Int128
		b int64
		c Int128
	}{
		{i64(-2), -1, i64(-1)},
		{i64(-2), 1, i64(-3)},
		{i64(2), 1, i64(1)},
		{i64(2), -1, i64(3)},
		{i64(1), 2, i64(-1)},  // crossing zero
		{i64(-1), -2, i64(1)}, // crossing zero

		{MinInt128, 1, MaxInt128},  // Overflow wraps
		{MaxInt128, -1, MinInt128}, // Overflow wraps

		{i128s("0x10000000000000000"), 1, i128s("0xFFFFFFFFFFFFFFFF")},  // carry down
		{i128s("0xFFFFFFFFFFFFFFFF"), -1, i128s("0x10000000000000000")}, // carry up
	} {
		t.Run(fmt.Sprintf("%d/%s-%d=%s", idx, tc.a, tc.b, tc.c), func(t *testing.T) {
			
			require.True(t,tc.c.Equal(tc.a.Sub64(tc.b)))
		})
	}
}

var (
	BenchInt128Result            Int128
	BenchInt64Result           int64
	BenchmarkInt128Float64Result float64
)

func BenchmarkInt128Add(b *testing.B) {
	for idx, tc := range []struct {
		a, b Int128
		name string
	}{
		{zeroInt128, zeroInt128, "0+0"},
		{MaxInt128, MaxInt128, "max+max"},
		{i128s("0x7FFFFFFFFFFFFFFF"), i128s("0x7FFFFFFFFFFFFFFF"), "lo-only"},
		{i128s("0xFFFFFFFFFFFFFFFF"), i128s("0x7FFFFFFFFFFFFFFF"), "carry"},
	} {
		b.Run(fmt.Sprintf("%d/%s", idx, tc.name), func(b *testing.B) {
			for i := 0; i < b.N; i++ {
				BenchInt128Result = tc.a.Add(tc.b)
			}
		})
	}
}

func BenchmarkInt128Add64(b *testing.B) {
	for idx, tc := range []struct {
		a    Int128
		b    int64
		name string
	}{
		{zeroInt128, 0, "0+0"},
		{MaxInt128, maxInt64, "max+max"},
		{i64(-1), -1, "-1+-1"},
		{i64(-1), 1, "-1+1"},
		{i64(minInt64), -1, "-min64-1"},
		{i128s("0xFFFFFFFFFFFFFFFF"), 1, "carry"},
	} {
		b.Run(fmt.Sprintf("%d/%s", idx, tc.name), func(b *testing.B) {
			for i := 0; i < b.N; i++ {
				BenchInt128Result = tc.a.Add64(tc.b)
			}
		})
	}
}

func BenchmarkInt128AsFloat(b *testing.B) {
	for idx, tc := range []struct {
		name string
		in   Int128
	}{
		{"zero", Int128{}},
		{"one", i64(1)},
		{"minusone", i64(-1)},
		{"maxInt64", i64(maxInt64)},
		{"gt64bit", i128s("0x1 00000000 00000000")},
		{"minInt64", i64(minInt64)},
		{"minusgt64bit", i128s("-0x1 00000000 00000000")},
	} {
		b.Run(fmt.Sprintf("%d/%s", idx, tc.name), func(b *testing.B) {
			for i := 0; i < b.N; i++ {
				BenchmarkInt128Float64Result = tc.in.AsFloat64()
			}
		})
	}
}

func BenchmarkInt128FromBigInt(b *testing.B) {
	for _, bi := range []*big.Int{
		bigs("0"),
		bigs("0xfedcba98"),
		bigs("0xfedcba9876543210"),
		bigs("0xfedcba9876543210fedcba98"),
		bigs("0xfedcba9876543210fedcba9876543210"),
	} {
		b.Run(fmt.Sprintf("%x", bi), func(b *testing.B) {
			for i := 0; i < b.N; i++ {
				BenchInt128Result, _ = Int128FromBigInt(bi)
			}
		})
	}
}

var (
	I64CastInput int64  = 0x7FFFFFFFFFFFFFFF
	I32CastInput int32  = 0x7FFFFFFF
	U64CastInput uint64 = 0x7FFFFFFFFFFFFFFF
)

func BenchmarkInt128FromCast(b *testing.B) {
	// Establish a baseline for a runtime 64-bit cast:
	b.Run("I64FromU64", func(b *testing.B) {
		for i := 0; i < b.N; i++ {
			BenchInt64Result = int64(U64CastInput)
		}
	})

	b.Run("Int128FromI64", func(b *testing.B) {
		for i := 0; i < b.N; i++ {
			BenchInt128Result = Int128From64(I64CastInput)
		}
	})
	b.Run("Int128FromU64", func(b *testing.B) {
		for i := 0; i < b.N; i++ {
			BenchInt128Result = Int128FromU64(U64CastInput)
		}
	})
	b.Run("Int128FromI32", func(b *testing.B) {
		for i := 0; i < b.N; i++ {
			BenchInt128Result = Int128From32(I32CastInput)
		}
	})
}

func BenchmarkInt128FromFloat(b *testing.B) {
	for _, pow := range []float64{1, 63, 64, 65, 127, 128} {
		b.Run(fmt.Sprintf("pow%d", int(pow)), func(b *testing.B) {
			f := math.Pow(2, pow)
			for i := 0; i < b.N; i++ {
				BenchInt128Result, _ = Int128FromFloat64(f)
			}
		})
	}
}

func BenchmarkInt128IsZero(b *testing.B) {
	for idx, tc := range []struct {
		name string
		v    Int128
	}{
		{"0", zeroInt128},
		{"hizero", i64(1)},
		{"nozero", MaxInt128},
	} {
		b.Run(fmt.Sprintf("%d/%s", idx, tc.name), func(b *testing.B) {
			for i := 0; i < b.N; i++ {
				benchBoolResult = tc.v.IsZero()
			}
		})
	}
}

func BenchmarkInt128LessThan(b *testing.B) {
	for _, iv := range []struct {
		a, b Int128
	}{
		{i64(1), i64(1)},
		{i64(2), i64(1)},
		{i64(1), i64(2)},
		{i64(-1), i64(-1)},
		{i64(-1), i64(-2)},
		{i64(-2), i64(-1)},
		{MaxInt128, MinInt128},
		{MinInt128, MaxInt128},
	} {
		b.Run(fmt.Sprintf("%s<%s", iv.a, iv.b), func(b *testing.B) {
			for i := 0; i < b.N; i++ {
				benchBoolResult = iv.a.LessThan(iv.b)
			}
		})
	}
}

func BenchmarkInt128LessOrEqualTo(b *testing.B) {
	for _, iv := range []struct {
		a, b Int128
	}{
		{i64(1), i64(1)},
		{i64(2), i64(1)},
		{i64(1), i64(2)},
		{i64(-1), i64(-1)},
		{i64(-1), i64(-2)},
		{i64(-2), i64(-1)},
		{MaxInt128, MinInt128},
		{MinInt128, MaxInt128},
	} {
		b.Run(fmt.Sprintf("%s<%s", iv.a, iv.b), func(b *testing.B) {
			for i := 0; i < b.N; i++ {
				benchBoolResult = iv.a.LessOrEqualTo(iv.b)
			}
		})
	}
}

func BenchmarkInt128Mul(b *testing.B) {
	v := Int128From64(maxInt64)
	for i := 0; i < b.N; i++ {
		BenchInt128Result = v.Mul(v)
	}
}

func BenchmarkInt128Mul64(b *testing.B) {
	v := Int128From64(maxInt64)
	lim := int64(b.N)
	for i := int64(0); i < lim; i++ {
		BenchInt128Result = v.Mul64(i)
	}
}

func BenchmarkInt128QuoRem64(b *testing.B) {
	// FIXME: benchmark numbers of various sizes
	v, by := i64(1234), int64(56)
	for i := 0; i < b.N; i++ {
		BenchInt128Result, _ = v.QuoRem64(by)
	}
}

func BenchmarkInt128Sub(b *testing.B) {
	sub := i64(1)
	for _, iv := range []Int128{i64(1), i128s("0x10000000000000000"), MaxInt128} {
		b.Run(fmt.Sprintf("%s", iv), func(b *testing.B) {
			for i := 0; i < b.N; i++ {
				BenchInt128Result = iv.Sub(sub)
			}
		})
	}
}

func BenchmarkInt128MustInt128FromBigEndian(b *testing.B) {
	var bts = make([]byte, 16)
	rand.Read(bts)
	for i := 0; i < b.N; i++ {
		BenchInt128Result = MustUint128FromBigEndian(bts).AsInt128()
	}
}

func accInt64FromBigInt(b *big.Int) int64 {
	if !b.IsInt64() {
		panic(fmt.Errorf("num: inaccurate conversion to I64 in fuzz tester for %s", b))
	}
	return b.Int64()
}

func accInt128FromBigInt(b *big.Int) Int128 {
	i, acc := Int128FromBigInt(b)
	if !acc {
		panic(fmt.Errorf("num: inaccurate conversion to I128 in fuzz tester for %s", b))
	}
	return i
}
