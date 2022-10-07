package geometry

import (
	"bytes"
	"fmt"
	"math"
	"math/big"
	"math/rand"
	"strings"
	"testing"
	"time"
)

// masks contains a pre-calculated set of 128-bit masks for use when generating
// random U128s/I128s. It's used to ensure we generate an even distribution of
// bit sizes.
var masks [128]*big.Int

func init() {
	for i := 0; i < 128; i++ {
		bi := new(big.Int)
		for b := 0; b <= i; b++ {
			bi.SetBit(bi, b, 1)
		}
		masks[i] = bi
	}
}

type fuzzOp string
type fuzzType string

// fuzzDefaultIterations should be configured to guarantee all of the argument
// schemes execute at least once for each op in a reasonable time.
// This is the equivalent of passing -num.fuzziter=<...> to 'go test':
const fuzzDefaultIterations = 20000

// These ops are all enabled by default. You can instead pass them explicitly
// on the command line like so: '-num.fuzzop=add -num.fuzzop=sub', or you can
// use the short form '-num.fuzzop=add,sub,mul'.
//
// If you add a new op, search for the string 'NEWOP' in this file for all the
// places you need to update.
const (
	fuzzAbs                fuzzOp = "abs"
	fuzzAdd                fuzzOp = "add"
	fuzzAdd64              fuzzOp = "add64"
	fuzzAnd                fuzzOp = "and"
	fuzzAnd64              fuzzOp = "and64"
	fuzzAndNot             fuzzOp = "andnot"
	fuzzAsFloat64          fuzzOp = "asfloat64"
	fuzzBinBE              fuzzOp = "binbe"
	fuzzBinLE              fuzzOp = "binle"
	fuzzBit                fuzzOp = "bit"
	fuzzBitLen             fuzzOp = "bitlen"
	fuzzCmp                fuzzOp = "cmp"
	fuzzCmp64              fuzzOp = "cmp64"
	fuzzDec                fuzzOp = "dec"
	fuzzEqual              fuzzOp = "equal"
	fuzzEqual64            fuzzOp = "equal64"
	fuzzFromFloat64        fuzzOp = "fromfloat64"
	fuzzGreaterOrEqualTo   fuzzOp = "gte"
	fuzzGreaterOrEqualTo64 fuzzOp = "gte64"
	fuzzGreaterThan        fuzzOp = "gt"
	fuzzGreaterThan64      fuzzOp = "gt64"
	fuzzInc                fuzzOp = "inc"
	fuzzLessOrEqualTo      fuzzOp = "lte"
	fuzzLessOrEqualTo64    fuzzOp = "lte64"
	fuzzLessThan           fuzzOp = "lt"
	fuzzLessThan64         fuzzOp = "lt64"
	fuzzLsh                fuzzOp = "lsh"
	fuzzMul                fuzzOp = "mul"
	fuzzMul64              fuzzOp = "mul64"
	fuzzNeg                fuzzOp = "neg"
	fuzzNot                fuzzOp = "not"
	fuzzOr                 fuzzOp = "or"
	fuzzOr64               fuzzOp = "or64"
	fuzzQuo                fuzzOp = "quo"
	fuzzQuo64              fuzzOp = "quo64"
	fuzzQuoRem             fuzzOp = "quorem"
	fuzzQuoRem64           fuzzOp = "quorem64"
	fuzzRem                fuzzOp = "rem"
	fuzzRem64              fuzzOp = "rem64"
	fuzzRotateLeft         fuzzOp = "rotl"
	fuzzRsh                fuzzOp = "rsh"
	fuzzString             fuzzOp = "string"
	fuzzSetBit             fuzzOp = "setbit"
	fuzzSub                fuzzOp = "sub"
	fuzzSub64              fuzzOp = "sub64"
	fuzzXor                fuzzOp = "xor"
	fuzzXor64              fuzzOp = "xor64"
)

// These types are all enabled by default. You can instead pass them explicitly
// on the command line like so: '-num.fuzztype=u128 -num.fuzztype=i128'
const (
	fuzzTypeUint128 fuzzType = "u128"
	fuzzTypeInt128 fuzzType = "i128"
)

var (
	u128FloatLimit = math.Nextafter(maxRepresentableUint128Float, math.Inf(1))
)

var allFuzzTypes = []fuzzType{fuzzTypeUint128, fuzzTypeInt128}

// allFuzzOps are active by default.
//
// NEWOP: Update this list if a NEW op is added otherwise it won't be
// enabled by default.
//
// Please keep this list alphabetised.
var allFuzzOps = []fuzzOp{
	fuzzAbs,
	fuzzAdd,
	fuzzAdd64,
	fuzzAnd,
	fuzzAnd64,
	fuzzAndNot,
	fuzzAsFloat64,
	fuzzBinBE,
	fuzzBinLE,
	fuzzBit,
	fuzzBitLen,
	fuzzCmp,
	fuzzCmp64,
	fuzzDec,
	fuzzEqual,
	fuzzEqual64,
	fuzzFromFloat64,
	fuzzGreaterOrEqualTo,
	fuzzGreaterOrEqualTo64,
	fuzzGreaterThan,
	fuzzGreaterThan64,
	fuzzInc,
	fuzzLessOrEqualTo,
	fuzzLessOrEqualTo64,
	fuzzLessThan,
	fuzzLessThan64,
	fuzzLsh,
	fuzzMul,
	fuzzMul64,
	fuzzNeg,
	fuzzNot,
	fuzzOr,
	fuzzOr64,
	fuzzQuo,
	fuzzQuo64,
	fuzzQuoRem,
	fuzzQuoRem64,
	fuzzRem,
	fuzzRem64,
	fuzzRotateLeft,
	fuzzRsh,
	fuzzSetBit,
	fuzzString,
	fuzzSub,
	fuzzSub64,
	fuzzXor,
	fuzzXor64,
}

// NEWOP: update this interface if a new op is added.
type fuzzOps interface {
	Name() string // Not an op

	Abs() error
	Add() error
	Add64() error
	And() error
	And64() error
	AndNot() error
	AsFloat64() error
	BinBE() error
	BinLE() error
	Bit() error
	BitLen() error
	Cmp() error
	Cmp64() error
	Dec() error
	Equal() error
	Equal64() error
	FromFloat64() error
	GreaterOrEqualTo() error
	GreaterOrEqualTo64() error
	GreaterThan() error
	GreaterThan64() error
	Inc() error
	LessOrEqualTo() error
	LessOrEqualTo64() error
	LessThan() error
	LessThan64() error
	Lsh() error
	Mul() error
	Mul64() error
	Neg() error
	Not() error
	Or() error
	Or64() error
	Quo() error
	Quo64() error
	QuoRem() error
	QuoRem64() error
	Rem() error
	Rem64() error
	RotateLeft() error
	Rsh() error
	SetBit() error
	String() error
	Sub() error
	Sub64() error
	Xor() error
	Xor64() error
}

func checkEqualInt(u int, b int) error {
	if u != b {
		return fmt.Errorf("128(%v) != big(%v)", u, b)
	}
	return nil
}

func checkEqualBool(u bool, b bool) error {
	if u != b {
		return fmt.Errorf("128(%v) != big(%v)", u, b)
	}
	return nil
}

func checkEqualUint128(n string, u Uint128, b *big.Int) error {
	if u.AsBigInt().Cmp(b) != 0 {
		return fmt.Errorf("%s: u128(%s) != big(%s)", n, u.String(), b.String())
	}
	return nil
}

func checkEqualBytes(n string, b1 []byte, b2 []byte) error {
	if !bytes.Equal(b1, b2) {
		return fmt.Errorf("%s: bytes(%v) != bytes(%v)", n, b1, b2)
	}
	return nil
}

func checkEqualInt128(n string, i Int128, b *big.Int) error {
	if i.AsBigInt().Cmp(b) != 0 {
		return fmt.Errorf("%s: i128(%s) != big(%s)", n, i.String(), b.String())
	}
	return nil
}

func checkEqualString(u fmt.Stringer, b fmt.Stringer) error {
	if u.String() != b.String() {
		return fmt.Errorf("128(%s) != big(%s)", u.String(), b.String())
	}
	return nil
}

func checkFloat(orig *big.Int, result float64, bf *big.Float) error {
	diff := new(big.Float).SetFloat64(result)
	diff.Sub(diff, bf)
	diff.Abs(diff)

	isZero := orig.Cmp(big0) == 0
	if !isZero {
		diff.Quo(diff, bf)
	}

	if (isZero && result != 0) || diff.Abs(diff).Cmp(floatDiffLimit) > 0 {
		return fmt.Errorf("|128(%f) - big(%f)| = %s, > %s", result, bf,
			cleanFloatStr(fmt.Sprintf("%.20f", diff)),
			cleanFloatStr(fmt.Sprintf("%.20f", floatDiffLimit)))
	}
	return nil
}

func TestFuzz(t *testing.T) {
	// fuzzOpsActive comes from the -num.fuzzop flag, in TestMain:
	var runFuzzOps = allFuzzOps

	// fuzzTypesActive comes from the -num.fuzzop flag, in TestMain:
	var runFuzzTypes = allFuzzTypes

	var source = newRando(rand.New(rand.NewSource(time.Now().UnixMilli()))) // Classic rando!
	var totalFailures int

	var fuzzTypes []fuzzOps

	for _, fuzzType := range runFuzzTypes {
		switch fuzzType {
		case fuzzTypeUint128:
			fuzzTypes = append(fuzzTypes, &fuzzUint128{source: source})
		case fuzzTypeInt128:
			fuzzTypes = append(fuzzTypes, &fuzzInt128{source: source})
		default:
			panic("unknown fuzz type")
		}
	}

	var failures = make([][]int, len(fuzzTypes))
	var failCount = 0

	for implIdx, fuzzImpl := range fuzzTypes {
		failures[implIdx] = make([]int, len(runFuzzOps))

		for opIdx, op := range runFuzzOps {
			opIterations := source.NextOp(op, fuzzDefaultIterations)

			for i := 0; i < opIterations; i++ {
				source.NextTest()

				var err error

				// NEWOP: add a new branch here in alphabetical order if a new
				// op is added.
				switch op {
				case fuzzAbs:
					err = fuzzImpl.Abs()
				case fuzzAdd:
					err = fuzzImpl.Add()
				case fuzzAdd64:
					err = fuzzImpl.Add64()
				case fuzzAnd:
					err = fuzzImpl.And()
				case fuzzAnd64:
					err = fuzzImpl.And64()
				case fuzzAndNot:
					err = fuzzImpl.AndNot()
				case fuzzAsFloat64:
					err = fuzzImpl.AsFloat64()
				case fuzzBinBE:
					err = fuzzImpl.BinBE()
				case fuzzBinLE:
					err = fuzzImpl.BinLE()
				case fuzzBit:
					err = fuzzImpl.Bit()
				case fuzzBitLen:
					err = fuzzImpl.BitLen()
				case fuzzCmp:
					err = fuzzImpl.Cmp()
				case fuzzCmp64:
					err = fuzzImpl.Cmp64()
				case fuzzDec:
					err = fuzzImpl.Dec()
				case fuzzEqual:
					err = fuzzImpl.Equal()
				case fuzzEqual64:
					err = fuzzImpl.Equal64()
				case fuzzFromFloat64:
					err = fuzzImpl.FromFloat64()
				case fuzzGreaterOrEqualTo:
					err = fuzzImpl.GreaterOrEqualTo()
				case fuzzGreaterOrEqualTo64:
					err = fuzzImpl.GreaterOrEqualTo64()
				case fuzzGreaterThan:
					err = fuzzImpl.GreaterThan()
				case fuzzGreaterThan64:
					err = fuzzImpl.GreaterThan64()
				case fuzzInc:
					err = fuzzImpl.Inc()
				case fuzzLessOrEqualTo:
					err = fuzzImpl.LessOrEqualTo()
				case fuzzLessOrEqualTo64:
					err = fuzzImpl.LessOrEqualTo64()
				case fuzzLessThan:
					err = fuzzImpl.LessThan()
				case fuzzLessThan64:
					err = fuzzImpl.LessThan64()
				case fuzzLsh:
					err = fuzzImpl.Lsh()
				case fuzzMul:
					err = fuzzImpl.Mul()
				case fuzzMul64:
					err = fuzzImpl.Mul64()
				case fuzzNeg:
					err = fuzzImpl.Neg()
				case fuzzNot:
					err = fuzzImpl.Not()
				case fuzzOr:
					err = fuzzImpl.Or()
				case fuzzOr64:
					err = fuzzImpl.Or64()
				case fuzzQuo:
					err = fuzzImpl.Quo()
				case fuzzQuo64:
					err = fuzzImpl.Quo64()
				case fuzzQuoRem:
					err = fuzzImpl.QuoRem()
				case fuzzQuoRem64:
					err = fuzzImpl.QuoRem64()
				case fuzzRem:
					err = fuzzImpl.Rem()
				case fuzzRem64:
					err = fuzzImpl.Rem64()
				case fuzzRotateLeft:
					err = fuzzImpl.RotateLeft()
				case fuzzRsh:
					err = fuzzImpl.Rsh()
				case fuzzSetBit:
					err = fuzzImpl.SetBit()
				case fuzzString:
					err = fuzzImpl.String()
				case fuzzSub:
					err = fuzzImpl.Sub()
				case fuzzSub64:
					err = fuzzImpl.Sub64()
				case fuzzXor:
					err = fuzzImpl.Xor()
				case fuzzXor64:
					err = fuzzImpl.Xor64()
				default:
					panic(fmt.Errorf("unsupported op %q", op))
				}

				if err != nil {
					failures[implIdx][opIdx]++
					failCount++
					t.Logf("impl %s: %s\n%s\n\n", fuzzImpl.Name(), op.Print(source.Operands()...), err)
				}
			}
		}
	}

	if failCount > 0 {
		t.Logf("  ------------- UH OH! ------------")
		t.Logf("")
		t.Logf(`         _.-^^---....,,--          `)
		t.Logf(`      _--                  --_     `)
		t.Logf(`     <                        >)   `)
		t.Logf(`     |                         |   `)
		t.Logf(`      \._                   _./    `)
		t.Logf("         ```--. . , ; .--'''       ")
		t.Logf(`               | |   |             `)
		t.Logf(`            .-=||  | |=-.          `)
		t.Logf("            `-=#$&&@$#=-'          ")
		t.Logf(`               | ;  :|             `)
		t.Logf(`      _____.,-#$&$@$#&#~,._____    `)
		t.Logf("")
	}

	for implIdx, implFailures := range failures {
		fuzzImpl := fuzzTypes[implIdx]
		for opIdx, cnt := range implFailures {
			if cnt > 0 {
				totalFailures += cnt
				t.Logf("impl %s, op %s: %d/%d failed", fuzzImpl.Name(), string(runFuzzOps[opIdx]), cnt, fuzzDefaultIterations)
			}
		}
	}

	if totalFailures > 0 {
		t.Fail()
	}
}

func (op fuzzOp) Print(operands ...*big.Int) string {
	// NEWOP: please add a human-readale format for your op here; this is used
	// for reporting errors and should show the operation, i.e. "2 + 2".
	//
	// It should be safe to assume the appropriate number of operands are set
	// in 'operands'; if not, it's a bug to be fixed elsewhere.
	switch op {
	case fuzzAsFloat64,
		fuzzFromFloat64,
		fuzzBinBE,
		fuzzBinLE,
		fuzzBitLen,
		fuzzString:
		s := strings.TrimRight(op.String(), "()")
		return fmt.Sprintf("%s(%d)", s, operands[0])

	case fuzzSetBit:
		return fmt.Sprintf("%d|(1<<%d)", operands[0], operands[1])

	case fuzzBit:
		return fmt.Sprintf("(%b>>%d)&1", operands[0], operands[1])

	case fuzzInc, fuzzDec:
		return fmt.Sprintf("%d%s", operands[0], op.String())

	case fuzzNeg, fuzzNot:
		return fmt.Sprintf("%s%d", op.String(), operands[0])

	case fuzzAbs:
		return fmt.Sprintf("|%d|", operands[0])

	case fuzzAdd, fuzzAdd64,
		fuzzAnd, fuzzAnd64,
		fuzzAndNot,
		fuzzLessOrEqualTo, fuzzLessOrEqualTo64,
		fuzzLessThan, fuzzLessThan64,
		fuzzLsh,
		fuzzMul, fuzzMul64,
		fuzzOr, fuzzOr64,
		fuzzQuo, fuzzQuo64,
		fuzzQuoRem, fuzzQuoRem64,
		fuzzRem, fuzzRem64,
		fuzzRotateLeft,
		fuzzRsh,
		fuzzXor, fuzzXor64,
		fuzzCmp,
		fuzzEqual,
		fuzzGreaterOrEqualTo, fuzzGreaterOrEqualTo64,
		fuzzGreaterThan, fuzzGreaterThan64,
		fuzzSub:

		// simple binary case:
		return fmt.Sprintf("%d %s %d", operands[0], op.String(), operands[1])

	default:
		return string(op)
	}
}

func (op fuzzOp) String() string {
	// NEWOP: please add a short string representation of this op, as if
	// the operands were in a sum (if that's possible)
	switch op {
	case fuzzAbs:
		return "|x|"
	case fuzzAdd, fuzzAdd64:
		return "+"
	case fuzzAnd, fuzzAnd64:
		return "&"
	case fuzzAndNot:
		return "&^"
	case fuzzAsFloat64:
		return "float64()"
	case fuzzBit:
		return "bit()"
	case fuzzBitLen:
		return "bitlen()"
	case fuzzCmp, fuzzCmp64:
		return "<=>"
	case fuzzDec:
		return "--"
	case fuzzEqual, fuzzEqual64:
		return "=="
	case fuzzFromFloat64:
		return "fromfloat64()"
	case fuzzGreaterThan, fuzzGreaterThan64:
		return ">"
	case fuzzGreaterOrEqualTo, fuzzGreaterOrEqualTo64:
		return ">="
	case fuzzInc:
		return "++"
	case fuzzLessThan, fuzzLessThan64:
		return "<"
	case fuzzLessOrEqualTo, fuzzLessOrEqualTo64:
		return "<="
	case fuzzLsh:
		return "<<"
	case fuzzMul, fuzzMul64:
		return "*"
	case fuzzNeg:
		return "-"
	case fuzzNot:
		return "^"
	case fuzzOr:
		return "|"
	case fuzzQuo, fuzzQuo64:
		return "/"
	case fuzzQuoRem, fuzzQuoRem64:
		return "/%"
	case fuzzRem, fuzzRem64:
		return "%"
	case fuzzRotateLeft:
		return "rotl()"
	case fuzzRsh:
		return ">>"
	case fuzzSetBit:
		return "setbit()"
	case fuzzString:
		return "string()"
	case fuzzSub, fuzzSub64:
		return "-"
	case fuzzXor, fuzzXor64:
		return "^"
	default:
		return string(op)
	}
}

type fuzzUint128 struct {
	source *rando
}

func (f fuzzUint128) Name() string { return "u128" }

func (f fuzzUint128) Abs() error {
	return nil // Always succeeds!
}

func (f fuzzUint128) Inc() error {
	b1 := f.source.BigUint128()
	u1 := accUint128FromBigInt(b1)
	rb := new(big.Int).Add(b1, big1)
	ru := u1.Inc()
	if rb.Cmp(wrapBigUint128) >= 0 {
		rb = new(big.Int).Sub(rb, wrapBigUint128) // simulate overflow
	}
	return checkEqualUint128("inc", ru, rb)
}

func (f fuzzUint128) Dec() error {
	b1 := f.source.BigUint128()
	u1 := accUint128FromBigInt(b1)
	rb := new(big.Int).Sub(b1, big1)
	if rb.Cmp(big0) < 0 {
		rb = new(big.Int).Add(wrapBigUint128, rb) // simulate underflow
	}
	ru := u1.Dec()
	return checkEqualUint128("dec", ru, rb)
}

func (f fuzzUint128) Add() error {
	b1, b2 := f.source.BigUint128x2()
	u1, u2 := accUint128FromBigInt(b1), accUint128FromBigInt(b2)
	rb := new(big.Int).Add(b1, b2)
	rb = simulateBigUint128Overflow(rb)
	ru := u1.Add(u2)
	return checkEqualUint128("add", ru, rb)
}

func (f fuzzUint128) Add64() error {
	b1, b2 := f.source.BigUint128And64()
	u1, u2 := accUint128FromBigInt(b1), accU64FromBigInt(b2)
	rb := new(big.Int).Add(b1, b2)
	rb = simulateBigUint128Overflow(rb)
	ru := u1.Add64(u2)
	return checkEqualUint128("add64", ru, rb)
}

func (f fuzzUint128) Sub() error {
	b1, b2 := f.source.BigUint128x2()
	u1, u2 := accUint128FromBigInt(b1), accUint128FromBigInt(b2)
	rb := new(big.Int).Sub(b1, b2)
	if rb.Cmp(big0) < 0 {
		rb = new(big.Int).Add(wrapBigUint128, rb) // simulate underflow
	}
	ru := u1.Sub(u2)
	return checkEqualUint128("sub", ru, rb)
}

func (f fuzzUint128) Sub64() error {
	b1, b2 := f.source.BigUint128And64()
	u1, u2 := accUint128FromBigInt(b1), accU64FromBigInt(b2)
	rb := new(big.Int).Sub(b1, b2)
	if rb.Cmp(big0) < 0 {
		rb = new(big.Int).Add(wrapBigUint128, rb) // simulate underflow
	}
	ru := u1.Sub64(u2)
	return checkEqualUint128("sub64", ru, rb)
}

func (f fuzzUint128) Mul() error {
	b1, b2 := f.source.BigUint128x2()
	u1, u2 := accUint128FromBigInt(b1), accUint128FromBigInt(b2)
	rb := new(big.Int).Mul(b1, b2)
	rb = simulateBigUint128Overflow(rb)
	ru := u1.Mul(u2)
	return checkEqualUint128("mul", ru, rb)
}

func (f fuzzUint128) Mul64() error {
	b1, b2 := f.source.BigUint128And64()
	u1, u2 := accUint128FromBigInt(b1), accU64FromBigInt(b2)
	rb := new(big.Int).Mul(b1, b2)
	rb = simulateBigUint128Overflow(rb)
	ru := u1.Mul64(u2)
	return checkEqualUint128("mul64", ru, rb)
}

func (f fuzzUint128) Quo() error {
	b1, b2 := f.source.BigUint128x2()
	u1, u2 := accUint128FromBigInt(b1), accUint128FromBigInt(b2)
	if b2.Cmp(big0) == 0 {
		return nil // Just skip this iteration, we know what happens!
	}
	rb := new(big.Int).Quo(b1, b2)
	ru := u1.Quo(u2)
	return checkEqualUint128("quo", ru, rb)
}

func (f fuzzUint128) Quo64() error {
	b1, b2 := f.source.BigUint128And64()
	u1, u2 := accUint128FromBigInt(b1), accU64FromBigInt(b2)
	if b2.Cmp(big0) == 0 {
		return nil // Just skip this iteration, we know what happens!
	}
	rb := new(big.Int).Quo(b1, b2)
	ru := u1.Quo64(u2)
	return checkEqualUint128("quo64", ru, rb)
}

func (f fuzzUint128) Rem() error {
	b1, b2 := f.source.BigUint128x2()
	u1, u2 := accUint128FromBigInt(b1), accUint128FromBigInt(b2)
	if b2.Cmp(big0) == 0 {
		return nil // Just skip this iteration, we know what happens!
	}
	rb := new(big.Int).Rem(b1, b2)
	ru := u1.Rem(u2)
	return checkEqualUint128("rem", ru, rb)
}

func (f fuzzUint128) Rem64() error {
	b1, b2 := f.source.BigUint128And64()
	u1, u2 := accUint128FromBigInt(b1), accU64FromBigInt(b2)
	if b2.Cmp(big0) == 0 {
		return nil // Just skip this iteration, we know what happens!
	}
	rb := new(big.Int).Rem(b1, b2)
	ru := u1.Rem64(u2)
	return checkEqualUint128("rem64", ru, rb)
}

func (f fuzzUint128) QuoRem() error {
	b1, b2 := f.source.BigUint128x2()
	u1, u2 := accUint128FromBigInt(b1), accUint128FromBigInt(b2)
	if b2.Cmp(big0) == 0 {
		return nil // Just skip this iteration, we know what happens!
	}

	rbq := new(big.Int).Quo(b1, b2)
	rbr := new(big.Int).Rem(b1, b2)
	ruq, rur := u1.QuoRem(u2)
	if err := checkEqualUint128("quo", ruq, rbq); err != nil {
		return err
	}
	if err := checkEqualUint128("rem", rur, rbr); err != nil {
		return err
	}
	return nil
}

func (f fuzzUint128) QuoRem64() error {
	b1, b2 := f.source.BigUint128And64()
	u1, u2 := accUint128FromBigInt(b1), accU64FromBigInt(b2)
	if b2.Cmp(big0) == 0 {
		return nil // Just skip this iteration, we know what happens!
	}

	rbq := new(big.Int).Quo(b1, b2)
	rbr := new(big.Int).Rem(b1, b2)
	ruq, rur := u1.QuoRem64(u2)
	if err := checkEqualUint128("quo64", ruq, rbq); err != nil {
		return err
	}
	if err := checkEqualUint128("rem64", rur, rbr); err != nil {
		return err
	}
	return nil
}

func (f fuzzUint128) Cmp() error {
	b1, b2 := f.source.BigUint128x2()
	u1, u2 := accUint128FromBigInt(b1), accUint128FromBigInt(b2)
	return checkEqualInt(u1.Cmp(u2), b1.Cmp(b2))
}

func (f fuzzUint128) Cmp64() error {
	b1, b2 := f.source.BigUint128And64()
	u1, u2 := accUint128FromBigInt(b1), accU64FromBigInt(b2)
	return checkEqualInt(u1.Cmp64(u2), b1.Cmp(b2))
}

func (f fuzzUint128) Equal() error {
	b1, b2 := f.source.BigUint128x2()
	u1, u2 := accUint128FromBigInt(b1), accUint128FromBigInt(b2)
	return checkEqualBool(u1.Equal(u2), b1.Cmp(b2) == 0)
}

func (f fuzzUint128) Equal64() error {
	b1, b2 := f.source.BigUint128And64()
	u1, u2 := accUint128FromBigInt(b1), accU64FromBigInt(b2)
	return checkEqualBool(u1.Equal64(u2), b1.Cmp(b2) == 0)
}

func (f fuzzUint128) GreaterThan() error {
	b1, b2 := f.source.BigUint128x2()
	u1, u2 := accUint128FromBigInt(b1), accUint128FromBigInt(b2)
	return checkEqualBool(u1.GreaterThan(u2), b1.Cmp(b2) > 0)
}

func (f fuzzUint128) GreaterThan64() error {
	b1, b2 := f.source.BigUint128And64()
	u1, u2 := accUint128FromBigInt(b1), accU64FromBigInt(b2)
	return checkEqualBool(u1.GreaterThan64(u2), b1.Cmp(b2) > 0)
}

func (f fuzzUint128) GreaterOrEqualTo() error {
	b1, b2 := f.source.BigUint128x2()
	u1, u2 := accUint128FromBigInt(b1), accUint128FromBigInt(b2)
	return checkEqualBool(u1.GreaterOrEqualTo(u2), b1.Cmp(b2) >= 0)
}

func (f fuzzUint128) GreaterOrEqualTo64() error {
	b1, b2 := f.source.BigUint128And64()
	u1, u2 := accUint128FromBigInt(b1), accU64FromBigInt(b2)
	return checkEqualBool(u1.GreaterOrEqualTo64(u2), b1.Cmp(b2) >= 0)
}

func (f fuzzUint128) LessThan() error {
	b1, b2 := f.source.BigUint128x2()
	u1, u2 := accUint128FromBigInt(b1), accUint128FromBigInt(b2)
	return checkEqualBool(u1.LessThan(u2), b1.Cmp(b2) < 0)
}

func (f fuzzUint128) LessThan64() error {
	b1, b2 := f.source.BigUint128And64()
	u1, u2 := accUint128FromBigInt(b1), accU64FromBigInt(b2)
	return checkEqualBool(u1.LessThan64(u2), b1.Cmp(b2) < 0)
}

func (f fuzzUint128) LessOrEqualTo() error {
	b1, b2 := f.source.BigUint128x2()
	u1, u2 := accUint128FromBigInt(b1), accUint128FromBigInt(b2)
	return checkEqualBool(u1.LessOrEqualTo(u2), b1.Cmp(b2) <= 0)
}

func (f fuzzUint128) LessOrEqualTo64() error {
	b1, b2 := f.source.BigUint128And64()
	u1, u2 := accUint128FromBigInt(b1), accU64FromBigInt(b2)
	return checkEqualBool(u1.LessOrEqualTo64(u2), b1.Cmp(b2) <= 0)
}

func (f fuzzUint128) And() error {
	b1, b2 := f.source.BigUint128x2()
	u1, u2 := accUint128FromBigInt(b1), accUint128FromBigInt(b2)
	rb := new(big.Int).And(b1, b2)
	ru := u1.And(u2)
	return checkEqualUint128("and", ru, rb)
}

func (f fuzzUint128) And64() error {
	b1, b2 := f.source.BigUint128And64()
	u1, u2 := accUint128FromBigInt(b1), accU64FromBigInt(b2)
	rb := new(big.Int).And(b1, b2)
	ru := u1.And64(u2)
	return checkEqualUint128("and64", ru, rb)
}

func (f fuzzUint128) AndNot() error {
	b1, b2 := f.source.BigUint128x2()
	u1, u2 := accUint128FromBigInt(b1), accUint128FromBigInt(b2)
	rb := new(big.Int).AndNot(b1, b2)
	ru := u1.AndNot(u2)
	return checkEqualUint128("andnot", ru, rb)
}

func (f fuzzUint128) Or() error {
	b1, b2 := f.source.BigUint128x2()
	u1, u2 := accUint128FromBigInt(b1), accUint128FromBigInt(b2)
	rb := new(big.Int).Or(b1, b2)
	ru := u1.Or(u2)
	return checkEqualUint128("or", ru, rb)
}

func (f fuzzUint128) Or64() error {
	b1, b2 := f.source.BigUint128And64()
	u1, u2 := accUint128FromBigInt(b1), accU64FromBigInt(b2)
	rb := new(big.Int).Or(b1, b2)
	ru := u1.Or64(u2)
	return checkEqualUint128("or", ru, rb)
}

func (f fuzzUint128) Xor() error {
	b1, b2 := f.source.BigUint128x2()
	u1, u2 := accUint128FromBigInt(b1), accUint128FromBigInt(b2)
	rb := new(big.Int).Xor(b1, b2)
	ru := u1.Xor(u2)
	return checkEqualUint128("xor", ru, rb)
}

func (f fuzzUint128) Xor64() error {
	b1, b2 := f.source.BigUint128And64()
	u1, u2 := accUint128FromBigInt(b1), accU64FromBigInt(b2)
	rb := new(big.Int).Xor(b1, b2)
	ru := u1.Xor64(u2)
	return checkEqualUint128("xor", ru, rb)
}

func (f fuzzUint128) Lsh() error {
	b1, by := f.source.BigUint128AndBitSize()
	u1 := accUint128FromBigInt(b1)
	rb := new(big.Int).Lsh(b1, by)
	rb.And(rb, maxBigUint128)
	ru := u1.Lsh(by)
	return checkEqualUint128("lsh", ru, rb)
}

func (f fuzzUint128) Rsh() error {
	b1, by := f.source.BigUint128AndBitSize()
	u1 := accUint128FromBigInt(b1)
	rb := new(big.Int).Rsh(b1, by)
	ru := u1.Rsh(by)
	return checkEqualUint128("rsh", ru, rb)
}

func (f fuzzUint128) RotateLeft() error {
	b1, by := f.source.BigUint128AndBitSize()
	u1 := accUint128FromBigInt(b1)
	rb1 := new(big.Int).Set(b1)
	rb1.Lsh(rb1, by)
	rb1.And(rb1, maxBigUint128)
	rb2 := new(big.Int).Set(b1)
	rb2.Rsh(rb2, 128-by)
	rb1.Or(rb1, rb2)

	// FIXME: this does not check RotateLeft with a negative input:
	ru := u1.RotateLeft(int(by))
	return checkEqualUint128("rotl", ru, rb1)
}

func (f fuzzUint128) Neg() error {
	return nil // nothing to do here
}

func (f fuzzUint128) BinBE() error {
	b1 := f.source.BigUint128()
	u1 := accUint128FromBigInt(b1)

	b1bts := make([]byte, 16)
	b1.FillBytes(b1bts)

	u1bts := make([]byte, 16)
	u1.PutBigEndian(u1bts)

	if err := checkEqualBytes("binbe", b1bts, u1bts); err != nil {
		return err
	}

	u2 := MustUint128FromBigEndian(u1bts)
	if !u1.Equal(u2) {
		return fmt.Errorf("binbe: u128(%s) != u128(%s)", u1.String(), u2.String())
	}
	return nil
}

func (f fuzzUint128) BinLE() error {
	b1 := f.source.BigUint128()
	u1 := accUint128FromBigInt(b1)

	b1bts := make([]byte, 16)
	b1.FillBytes(b1bts)

	// big.Int writes big endian; reverse the slice:
	for i, j := 0, len(b1bts)-1; i < j; i, j = i+1, j-1 {
		b1bts[i], b1bts[j] = b1bts[j], b1bts[i]
	}

	u1bts := make([]byte, 16)
	u1.PutLittleEndian(u1bts)

	if err := checkEqualBytes("binle", b1bts, u1bts); err != nil {
		return err
	}

	u2 := MustUint128FromLittleEndian(u1bts)
	if !u1.Equal(u2) {
		return fmt.Errorf("binbe: u128(%s) != u128(%s)", u1.String(), u2.String())
	}
	return nil
}

func (f fuzzUint128) AsFloat64() error {
	b1 := f.source.BigUint128()
	u1 := accUint128FromBigInt(b1)
	bf := new(big.Float).SetInt(b1)
	ruf := u1.AsFloat64()
	return checkFloat(b1, ruf, bf)
}

func (f fuzzUint128) FromFloat64() error {
	b1 := f.source.BigUint128()
	u1 := accUint128FromBigInt(b1)
	bf1 := new(big.Float).SetInt(b1)
	f64, _ := bf1.Float64()

	if f64 == u128FloatLimit {
		// This is a bit of a cheat to allow rando to use MaxUint128, which is
		// technically unrepresentable as a float64 due to precision errors.
		// The float64 after converting MaxUint128 will be the next representable
		// float _after_ the maximum one representable within a 128-bit integer.
		f64 = maxRepresentableUint128Float
	}

	r1, inRange := Uint128FromFloat64(f64)
	if !inRange {
		panic(fmt.Errorf("float out of u128 range; expected <= %s, found %f", u1, f64)) // FIXME: error
	}

	diff := DifferenceUint128(u1, r1)

	isZero := b1.Cmp(big0) == 0
	if isZero {
		return checkEqualUint128("fromfloat64", r1, b1)
	} else {
		diffFloat := new(big.Float).Quo(diff.AsBigFloat(), bf1)
		if diffFloat.Cmp(floatDiffLimit) > 0 {
			return fmt.Errorf("|128(%s) - big(%s)| = %s, > %s", r1, b1,
				cleanFloatStr(fmt.Sprintf("%s", diff)),
				cleanFloatStr(fmt.Sprintf("%.20f", floatDiffLimit)))
		}
	}
	return nil
}

func (f fuzzUint128) String() error {
	b1 := f.source.BigUint128()
	u1 := accUint128FromBigInt(b1)
	return checkEqualString(u1, b1)
}

func (f fuzzUint128) SetBit() error {
	b1, bt, bv := f.source.BigUint128AndBitSizeAndBitValue()
	u1 := accUint128FromBigInt(b1)

	bvi := uint(0)
	if bv {
		bvi = 1
	}

	rb := new(big.Int).SetBit(b1, int(bt), bvi)
	ru := u1.SetBit(int(bt), bvi)
	return checkEqualUint128("setbit", ru, rb)
}

func (f fuzzUint128) Bit() error {
	b1, bt := f.source.BigUint128AndBitSize()
	u1 := accUint128FromBigInt(b1)
	return checkEqualInt(int(b1.Bit(int(bt))), int(u1.Bit(int(bt))))
}

func (f fuzzUint128) Not() error {
	b1 := f.source.BigUint128()
	u1 := accUint128FromBigInt(b1)

	ru := u1.Not()
	if ru.Equal(u1) {
		return fmt.Errorf("input unchanged by Not: %v", u1)
	}
	rd := ru.Not()
	if !rd.Equal(u1) {
		return fmt.Errorf("double-not does not equal input. expected %d, found %d", u1, rd)
	}

	return nil
}

func (f fuzzUint128) BitLen() error {
	b1 := f.source.BigUint128()
	u1 := accUint128FromBigInt(b1)

	rb := b1.BitLen()
	ru := u1.BitLen()

	return checkEqualInt(rb, ru)
}

// NEWOP: func (f fuzzUint128) ...() error {}

type fuzzInt128 struct {
	source *rando
}

func (f fuzzInt128) Name() string { return "i128" }

func (f fuzzInt128) Abs() error {
	b1 := f.source.BigInt128()
	i1 := accInt128FromBigInt(b1)

	rb := new(big.Int).Abs(b1)
	ib := simulateBigInt128Overflow(rb)
	if err := checkEqualInt128("abs128", i1.Abs(), ib); err != nil {
		return fmt.Errorf("Abs() failed: %v", err)
	}
	if err := checkEqualUint128("absu128", i1.AbsUint128(), rb); err != nil {
		return fmt.Errorf("AbsUint128() failed: %v", err)
	}

	return nil
}

func (f fuzzInt128) Inc() error {
	b1 := f.source.BigInt128()
	u1 := accInt128FromBigInt(b1)
	rb := new(big.Int).Add(b1, big1)
	ru := u1.Inc()
	rb = simulateBigInt128Overflow(rb)
	return checkEqualInt128("inc", ru, rb)
}

func (f fuzzInt128) Dec() error {
	b1 := f.source.BigInt128()
	u1 := accInt128FromBigInt(b1)
	rb := new(big.Int).Sub(b1, big1)
	rb = simulateBigInt128Overflow(rb)
	ru := u1.Dec()
	return checkEqualInt128("dec", ru, rb)
}

func (f fuzzInt128) Add() error {
	b1, b2 := f.source.BigInt128x2()
	u1, u2 := accInt128FromBigInt(b1), accInt128FromBigInt(b2)
	rb := new(big.Int).Add(b1, b2)
	rb = simulateBigInt128Overflow(rb)
	ru := u1.Add(u2)
	return checkEqualInt128("add", ru, rb)
}

func (f fuzzInt128) Add64() error {
	b1, b2 := f.source.BigInt128And64()
	i1, i2 := accInt128FromBigInt(b1), accI64FromBigInt(b2)
	rb := new(big.Int).Add(b1, b2)
	rb = simulateBigInt128Overflow(rb)
	ri := i1.Add64(i2)
	return checkEqualInt128("add64", ri, rb)
}

func (f fuzzInt128) Sub() error {
	b1, b2 := f.source.BigInt128x2()
	i1, i2 := accInt128FromBigInt(b1), accInt128FromBigInt(b2)
	rb := new(big.Int).Sub(b1, b2)
	rb = simulateBigInt128Overflow(rb)
	ri := i1.Sub(i2)
	return checkEqualInt128("sub", ri, rb)
}

func (f fuzzInt128) Sub64() error {
	b1, b2 := f.source.BigInt128And64()
	i1, i2 := accInt128FromBigInt(b1), accI64FromBigInt(b2)
	rb := new(big.Int).Sub(b1, b2)
	rb = simulateBigInt128Overflow(rb)
	ri := i1.Sub64(i2)
	return checkEqualInt128("sub64", ri, rb)
}

func (f fuzzInt128) Mul() error {
	b1, b2 := f.source.BigInt128x2()
	u1, u2 := accInt128FromBigInt(b1), accInt128FromBigInt(b2)
	rb := new(big.Int).Mul(b1, b2)
	rb = simulateBigInt128Overflow(rb)
	ru := u1.Mul(u2)
	return checkEqualInt128("mul", ru, rb)
}

func (f fuzzInt128) Mul64() error {
	b1, b2 := f.source.BigInt128And64()
	i1, i2 := accInt128FromBigInt(b1), accI64FromBigInt(b2)
	rb := new(big.Int).Mul(b1, b2)
	rb = simulateBigInt128Overflow(rb)
	ri := i1.Mul64(i2)
	return checkEqualInt128("mul64", ri, rb)
}

func (f fuzzInt128) Quo() error {
	b1, b2 := f.source.BigInt128x2()
	u1, u2 := accInt128FromBigInt(b1), accInt128FromBigInt(b2)
	if b2.Cmp(big0) == 0 {
		return nil // Just skip this iteration, we know what happens!
	}
	if u1 == MinInt128 && u2 == minusOne {
		return nil // Skip overflow corner case, it's handled in the unit tests and not meaningful here in the fuzzer.
	}
	rb := new(big.Int).Quo(b1, b2)
	ru := u1.Quo(u2)
	return checkEqualInt128("quo", ru, rb)
}

func (f fuzzInt128) Quo64() error {
	b1, b2 := f.source.BigInt128And64()
	i1, i2 := accInt128FromBigInt(b1), accI64FromBigInt(b2)
	if b2.Cmp(big0) == 0 {
		return nil // Just skip this iteration, we know what happens!
	}
	if i1 == MinInt128 && i2 == -1 {
		// Skip overflow corner case, it's (not yet, FIXME) handled in the
		// unit tests and not meaningful here in the fuzzer.
		return nil
	}
	rb := new(big.Int).Quo(b1, b2)
	ri := i1.Quo64(i2)
	return checkEqualInt128("quo64", ri, rb)
}

func (f fuzzInt128) Rem() error {
	b1, b2 := f.source.BigInt128x2()
	u1, u2 := accInt128FromBigInt(b1), accInt128FromBigInt(b2)
	if b2.Cmp(big0) == 0 {
		return nil // Just skip this iteration, we know what happens!
	}
	if u1 == MinInt128 && u2 == minusOne {
		return nil // Skip overflow corner case, it's handled in the unit tests and not meaningful here in the fuzzer.
	}
	rb := new(big.Int).Rem(b1, b2)
	ru := u1.Rem(u2)
	return checkEqualInt128("rem", ru, rb)
}

func (f fuzzInt128) Rem64() error {
	b1, b2 := f.source.BigInt128And64()
	i1, i2 := accInt128FromBigInt(b1), accI64FromBigInt(b2)
	if b2.Cmp(big0) == 0 {
		return nil // Just skip this iteration, we know what happens!
	}
	if i1 == MinInt128 && i2 == -1 {
		// Skip overflow corner case, it's (not yet, FIXME) handled in the
		// unit tests and not meaningful here in the fuzzer.
		return nil
	}
	rb := new(big.Int).Rem(b1, b2)
	ri := i1.Rem64(i2)
	return checkEqualInt128("rem64", ri, rb)
}

func (f fuzzInt128) QuoRem() error {
	b1, b2 := f.source.BigInt128x2()
	u1, u2 := accInt128FromBigInt(b1), accInt128FromBigInt(b2)
	if b2.Cmp(big0) == 0 {
		return nil // Just skip this iteration, we know what happens!
	}
	if u1 == MinInt128 && u2 == minusOne {
		return nil // Skip overflow corner case, it's handled in the unit tests and not meaningful here in the fuzzer.
	}

	rbq := new(big.Int).Quo(b1, b2)
	rbr := new(big.Int).Rem(b1, b2)
	ruq, rur := u1.QuoRem(u2)
	if err := checkEqualInt128("quo", ruq, rbq); err != nil {
		return err
	}
	if err := checkEqualInt128("rem", rur, rbr); err != nil {
		return err
	}
	return nil
}

func (f fuzzInt128) QuoRem64() error {
	b1, b2 := f.source.BigInt128And64()
	i1, i2 := accInt128FromBigInt(b1), accI64FromBigInt(b2)
	if b2.Cmp(big0) == 0 {
		return nil // Just skip this iteration, we know what happens!
	}
	if i1 == MinInt128 && i2 == -1 {
		// Skip overflow corner case, it's (not yet, FIXME) handled in the
		// unit tests and not meaningful here in the fuzzer.
		return nil
	}

	rbq := new(big.Int).Quo(b1, b2)
	rbr := new(big.Int).Rem(b1, b2)
	riq, rir := i1.QuoRem64(i2)
	if err := checkEqualInt128("quo64", riq, rbq); err != nil {
		return err
	}
	if err := checkEqualInt128("rem64", rir, rbr); err != nil {
		return err
	}
	return nil
}

func (f fuzzInt128) Cmp() error {
	b1, b2 := f.source.BigInt128x2()
	i1, i2 := accInt128FromBigInt(b1), accInt128FromBigInt(b2)
	return checkEqualInt(i1.Cmp(i2), b1.Cmp(b2))
}

func (f fuzzInt128) Cmp64() error {
	b1, b2 := f.source.BigInt128And64()
	i1, i2 := accInt128FromBigInt(b1), accI64FromBigInt(b2)
	return checkEqualInt(i1.Cmp64(i2), b1.Cmp(b2))
}

func (f fuzzInt128) Equal() error {
	b1, b2 := f.source.BigInt128x2()
	i1, i2 := accInt128FromBigInt(b1), accInt128FromBigInt(b2)
	return checkEqualBool(i1.Equal(i2), b1.Cmp(b2) == 0)
}

func (f fuzzInt128) Equal64() error {
	b1, b2 := f.source.BigInt128And64()
	i1, i2 := accInt128FromBigInt(b1), accI64FromBigInt(b2)
	return checkEqualBool(i1.Equal64(i2), b1.Cmp(b2) == 0)
}

func (f fuzzInt128) GreaterThan() error {
	b1, b2 := f.source.BigInt128x2()
	i1, i2 := accInt128FromBigInt(b1), accInt128FromBigInt(b2)
	return checkEqualBool(i1.GreaterThan(i2), b1.Cmp(b2) > 0)
}

func (f fuzzInt128) GreaterThan64() error {
	b1, b2 := f.source.BigInt128And64()
	i1, i2 := accInt128FromBigInt(b1), accI64FromBigInt(b2)
	return checkEqualBool(i1.GreaterThan64(i2), b1.Cmp(b2) > 0)
}

func (f fuzzInt128) GreaterOrEqualTo() error {
	b1, b2 := f.source.BigInt128x2()
	i1, i2 := accInt128FromBigInt(b1), accInt128FromBigInt(b2)
	return checkEqualBool(i1.GreaterOrEqualTo(i2), b1.Cmp(b2) >= 0)
}

func (f fuzzInt128) GreaterOrEqualTo64() error {
	b1, b2 := f.source.BigInt128And64()
	i1, i2 := accInt128FromBigInt(b1), accI64FromBigInt(b2)
	return checkEqualBool(i1.GreaterOrEqualTo64(i2), b1.Cmp(b2) >= 0)
}

func (f fuzzInt128) LessThan() error {
	b1, b2 := f.source.BigInt128x2()
	i1, i2 := accInt128FromBigInt(b1), accInt128FromBigInt(b2)
	return checkEqualBool(i1.LessThan(i2), b1.Cmp(b2) < 0)
}

func (f fuzzInt128) LessThan64() error {
	b1, b2 := f.source.BigInt128And64()
	i1, i2 := accInt128FromBigInt(b1), accI64FromBigInt(b2)
	return checkEqualBool(i1.LessThan64(i2), b1.Cmp(b2) < 0)
}

func (f fuzzInt128) LessOrEqualTo() error {
	b1, b2 := f.source.BigInt128x2()
	i1, i2 := accInt128FromBigInt(b1), accInt128FromBigInt(b2)
	return checkEqualBool(i1.LessOrEqualTo(i2), b1.Cmp(b2) <= 0)
}

func (f fuzzInt128) LessOrEqualTo64() error {
	b1, b2 := f.source.BigInt128And64()
	i1, i2 := accInt128FromBigInt(b1), accI64FromBigInt(b2)
	return checkEqualBool(i1.LessOrEqualTo64(i2), b1.Cmp(b2) <= 0)
}

func (f fuzzInt128) AsFloat64() error {
	b1 := f.source.BigInt128()
	i1 := accInt128FromBigInt(b1)
	bf := new(big.Float).SetInt(b1)
	rif := i1.AsFloat64()
	return checkFloat(b1, rif, bf)
}

func (f fuzzInt128) FromFloat64() error {
	b1 := f.source.BigInt128()
	i1 := accInt128FromBigInt(b1)
	bf1 := new(big.Float).SetInt(b1)
	f1, _ := bf1.Float64()
	r1, inRange := Int128FromFloat64(f1)
	if !inRange {
		panic("float out of range") // FIXME: error
	}

	diff := DifferenceInt128(i1, r1)

	isZero := b1.Cmp(big0) == 0
	if isZero {
		return checkEqualInt128("fromfloat64", r1, b1)
	} else {
		diffFloat := new(big.Float).Quo(diff.AsBigFloat(), bf1)
		if diffFloat.Cmp(floatDiffLimit) > 0 {
			return fmt.Errorf("|128(%s) - big(%s)| = %s, > %s", r1, b1,
				cleanFloatStr(fmt.Sprintf("%s", diff)),
				cleanFloatStr(fmt.Sprintf("%.20f", floatDiffLimit)))
		}
	}
	return nil
}

// Bitwise operations on Int128 are not supported:
func (f fuzzInt128) And() error        { return nil }
func (f fuzzInt128) And64() error      { return nil }
func (f fuzzInt128) AndNot() error     { return nil }
func (f fuzzInt128) Or() error         { return nil }
func (f fuzzInt128) Or64() error       { return nil }
func (f fuzzInt128) Xor() error        { return nil }
func (f fuzzInt128) Xor64() error      { return nil }
func (f fuzzInt128) Lsh() error        { return nil }
func (f fuzzInt128) Rsh() error        { return nil }
func (f fuzzInt128) SetBit() error     { return nil }
func (f fuzzInt128) Bit() error        { return nil }
func (f fuzzInt128) BitLen() error     { return nil }
func (f fuzzInt128) Not() error        { return nil }
func (f fuzzInt128) RotateLeft() error { return nil }

func (f fuzzInt128) Neg() error {
	b1 := f.source.BigInt128()
	u1 := accInt128FromBigInt(b1)

	// overflow is possible if you negate minBig128
	rb := simulateBigInt128Overflow(new(big.Int).Neg(b1))

	ru := u1.Neg()
	return checkEqualInt128("neg", ru, rb)
}

func (f fuzzInt128) BinBE() error {
	// Nothing to do
	return nil
}

func (f fuzzInt128) BinLE() error {
	// Nothing to do
	return nil
}

func (f fuzzInt128) String() error {
	b1 := f.source.BigInt128()
	i1 := accInt128FromBigInt(b1)
	return checkEqualString(i1, b1)
}

// NEWOP: func (f fuzzInt128) ...() error {}

type bigGenKind int

const (
	bigGenZero  bigGenKind = 0
	bigGenBits  bigGenKind = 1
	bigGenSame  bigGenKind = 2
	bigGenFixed bigGenKind = 3
)

type bigUint128Gen struct {
	kind  bigGenKind
	bits  int
	fixed *big.Int
}

func (gen bigUint128Gen) Value(r *rando) (v *big.Int) {
	switch gen.kind {
	case bigGenZero:
		v = new(big.Int)

	case bigGenBits:
		v = new(big.Int)
		if gen.bits <= 0 || gen.bits > 128 {
			panic("misconfigured bits")
		} else if gen.bits <= 64 {
			v = v.Rand(r.rng, maxBigUint64)
		} else {
			v = v.Rand(r.rng, maxBigUint128)
		}
		idx := gen.bits - 1
		v.And(v, masks[idx])
		v.SetBit(v, idx, 1)

	case bigGenSame:
		oper := r.Operands()
		v = oper[len(oper)-1]

	case bigGenFixed:
		v = new(big.Int)
		v.Set(gen.fixed)

	default:
		panic("unknown gen kind")
	}

	r.operands = append(r.operands, v)

	return v
}

type bigInt128Gen struct {
	kind  bigGenKind
	bits  int
	neg   bool
	fixed *big.Int
}

func (gen bigInt128Gen) Value(r *rando) (v *big.Int) {
	switch gen.kind {
	case bigGenZero:
		v = new(big.Int)

	case bigGenBits:
		v = new(big.Int)
		if gen.bits <= 0 || gen.bits > 127 { // 128th bit is set aside for the sign
			panic("misconfigured bits")
		} else if gen.bits <= 64 {
			v = v.Rand(r.rng, maxBigUint64)
		} else {
			v = v.Rand(r.rng, maxBigUint128)
		}
		idx := gen.bits - 1
		v.And(v, masks[idx])
		v.SetBit(v, idx, 1)
		if gen.neg {
			v.Neg(v)
		}

	case bigGenSame:
		oper := r.Operands()
		v = oper[len(oper)-1]

	case bigGenFixed:
		v = new(big.Int)
		v.Set(gen.fixed)

	default:
		panic("unknown gen kind")
	}

	r.operands = append(r.operands, v)

	return v
}

type bigUint128AndBitSizeGen struct {
	u128  bigUint128Gen
	shift uint // 0 to 128
}

func (gen bigUint128AndBitSizeGen) Values(r *rando) (v *big.Int, shift uint) {
	val := gen.u128.Value(r)
	r.operands = append(r.operands, val)
	return val, gen.shift
}

type bigUint128AndBitSizeAndBitValueGen struct {
	u128  bigUint128Gen
	shift uint // 0 to 127
	value bool // 0 or 1
}

func (gen bigUint128AndBitSizeAndBitValueGen) Values(r *rando) (v *big.Int, shift uint, value bool) {
	return gen.u128.Value(r), gen.shift, gen.value
}

// rando provides schemes for argument generation with heuristics that try to
// ensure coverage of the differences that matter.
//
// classic rando!
type rando struct {
	operands []*big.Int
	rng      *rand.Rand

	bigUint128Schemes []bigUint128Gen
	bigUint128Cur     int

	bigInt128Schemes []bigInt128Gen
	bigInt128Cur     int

	bigUint128x2Schemes [][2]bigUint128Gen
	bigUint128x2Cur     int

	bigInt128x2Schemes [][2]bigInt128Gen
	bigInt128x2Cur     int

	bigUint128And64Schemes [][2]bigUint128Gen
	bigUint128And64Cur     int

	bigInt128And64Schemes [][2]bigInt128Gen
	bigInt128And64Cur     int

	bigUint128AndBitSizeSchemes []bigUint128AndBitSizeGen
	bigUint128AndBitSizeCur     int

	bigUint128AndBitSizeAndBitValueSchemes []bigUint128AndBitSizeAndBitValueGen
	bigUint128AndBitSizeAndBitValueCur     int

	// This test has run; subsequent rando requests should fail until NewTest
	// is called again:
	testHasRun bool
}

func newRando(rng *rand.Rand) *rando {
	// Number of times to repeat the "both arguments identical" test for schemes
	// that have two of the same kind of argument.
	//
	// We need this because the chance of even two random 128-bit operands being
	// the same is unfathomable.
	samesies := 5

	r := &rando{ // classic rando!
		rng: rng,
	}

	{ // build bigUint128Schemes
		r.bigUint128Schemes = []bigUint128Gen{
			bigUint128Gen{kind: bigGenZero},
			bigUint128Gen{kind: bigGenFixed, fixed: maxBigUint64},
			bigUint128Gen{kind: bigGenFixed, fixed: maxBigUint128},
		}
		for i := 1; i <= 128; i++ {
			r.bigUint128Schemes = append(r.bigUint128Schemes, bigUint128Gen{kind: bigGenBits, bits: i})
		}
	}

	{ // build bigUint128AndBitSizeSchemes
		for _, u := range r.bigUint128Schemes {
			for shift := uint(0); shift < 128; shift++ {
				r.bigUint128AndBitSizeSchemes = append(
					r.bigUint128AndBitSizeSchemes, bigUint128AndBitSizeGen{u128: u, shift: shift})
			}
		}
	}

	{ // build bigUint128AndBitSizeAndBitValueSchemes
		for _, u := range r.bigUint128Schemes {
			for shift := uint(0); shift < 128; shift++ {
				for value := 0; value < 2; value++ {
					r.bigUint128AndBitSizeAndBitValueSchemes = append(
						r.bigUint128AndBitSizeAndBitValueSchemes, bigUint128AndBitSizeAndBitValueGen{u128: u, shift: shift, value: value == 1})
				}
			}
		}
	}

	{ // build bigUint128x2Schemes
		for _, u1 := range r.bigUint128Schemes {
			for _, u2 := range r.bigUint128Schemes {
				r.bigUint128x2Schemes = append(r.bigUint128x2Schemes, [2]bigUint128Gen{u1, u2})
			}
			for i := 0; i < samesies; i++ {
				r.bigUint128x2Schemes = append(r.bigUint128x2Schemes, [2]bigUint128Gen{u1, bigUint128Gen{kind: bigGenSame}})
			}
		}
	}

	{ // build bigUint128And64Schemes
		bigU64Schemes := []bigUint128Gen{
			bigUint128Gen{kind: bigGenZero},
			bigUint128Gen{kind: bigGenFixed, fixed: maxBigUint64},
		}
		for i := 1; i <= 64; i++ {
			bigU64Schemes = append(bigU64Schemes, bigUint128Gen{kind: bigGenBits, bits: i})
		}
		for _, u1 := range r.bigUint128Schemes {
			for _, u2 := range bigU64Schemes {
				r.bigUint128And64Schemes = append(r.bigUint128And64Schemes, [2]bigUint128Gen{u1, u2})
			}

			// FIXME: Samesies doesn't work here due to bit size mismatches:
			// for i := 0; i < samesies; i++ {
			//     r.bigUint128And64Schemes = append(r.bigUint128And64Schemes, [2]bigUint128Gen{u1, bigUint128Gen{kind: bigGenSame}})
			// }
		}
	}

	{ // build bigInt128Schemes
		r.bigInt128Schemes = []bigInt128Gen{
			bigInt128Gen{kind: bigGenZero},
			bigInt128Gen{kind: bigGenFixed, fixed: maxBigInt64},
			bigInt128Gen{kind: bigGenFixed, fixed: minBigInt64},
		}
		for i := 1; i <= 127; i++ {
			for n := 0; n < 2; n++ {
				r.bigInt128Schemes = append(r.bigInt128Schemes, bigInt128Gen{kind: bigGenBits, bits: i, neg: n == 0})
			}
		}
	}

	{ // build bigInt128x2Schemes
		for _, u1 := range r.bigInt128Schemes {
			for _, u2 := range r.bigInt128Schemes {
				r.bigInt128x2Schemes = append(r.bigInt128x2Schemes, [2]bigInt128Gen{u1, u2})
			}
			for i := 0; i < samesies; i++ {
				r.bigInt128x2Schemes = append(r.bigInt128x2Schemes, [2]bigInt128Gen{u1, bigInt128Gen{kind: bigGenSame}})
			}
		}
	}

	{ // build bigInt128And64Schemes
		bigI64Schemes := []bigInt128Gen{
			bigInt128Gen{kind: bigGenZero},
			bigInt128Gen{kind: bigGenFixed, fixed: maxBigInt64},
			bigInt128Gen{kind: bigGenFixed, fixed: minBigInt64},
		}
		for i := 1; i <= 63; i++ {
			for n := 0; n < 2; n++ {
				bigI64Schemes = append(bigI64Schemes, bigInt128Gen{kind: bigGenBits, bits: i, neg: n == 0})
			}
		}
		for _, u1 := range r.bigInt128Schemes {
			for _, u2 := range bigI64Schemes {
				r.bigInt128And64Schemes = append(r.bigInt128And64Schemes, [2]bigInt128Gen{u1, u2})
			}

			// FIXME: Samesies doesn't work here due to bit size mismatches:
			// for i := 0; i < samesies; i++ {
			//     r.bigInt128And64Schemes = append(r.bigInt128And64Schemes, [2]bigInt128Gen{u1, bigInt128Gen{kind: bigGenSame}})
			// }
		}
	}

	return r
}

func (r *rando) Operands() []*big.Int { return r.operands }

func (r *rando) NextOp(op fuzzOp, configuredIterations int) (opIterations int) {
	r.bigUint128x2Cur = 0
	r.bigUint128Cur = 0
	r.bigInt128x2Cur = 0
	r.bigInt128Cur = 0
	r.bigUint128AndBitSizeCur = 0
	r.bigUint128AndBitSizeAndBitValueCur = 0
	return configuredIterations
}

func (r *rando) NextTest() {
	r.testHasRun = false
	for i := range r.operands {
		r.operands[i] = nil
	}
	r.operands = r.operands[:0]
}

func (r *rando) ensureOnePerTest() {
	if r.testHasRun {
		panic("may only call source once per test")
	}
	r.testHasRun = true
}

func (r *rando) BigUint128x2() (b1, b2 *big.Int) {
	r.ensureOnePerTest()

	schemes := r.bigUint128x2Schemes[r.bigUint128x2Cur]
	r.bigUint128x2Cur++
	if r.bigUint128x2Cur >= len(r.bigUint128x2Schemes) {
		r.bigUint128x2Cur = 0
	}
	return schemes[0].Value(r), schemes[1].Value(r)
}

func (r *rando) BigInt128x2() (b1, b2 *big.Int) {
	r.ensureOnePerTest()

	schemes := r.bigInt128x2Schemes[r.bigInt128x2Cur]
	r.bigInt128x2Cur++
	if r.bigInt128x2Cur >= len(r.bigInt128x2Schemes) {
		r.bigInt128x2Cur = 0
	}
	return schemes[0].Value(r), schemes[1].Value(r)
}

func (r *rando) BigUint128And64() (b1, b2 *big.Int) {
	r.ensureOnePerTest()

	schemes := r.bigUint128And64Schemes[r.bigUint128And64Cur]
	r.bigUint128And64Cur++
	if r.bigUint128And64Cur >= len(r.bigUint128And64Schemes) {
		r.bigUint128And64Cur = 0
	}
	return schemes[0].Value(r), schemes[1].Value(r)
}

func (r *rando) BigInt128And64() (b1, b2 *big.Int) {
	r.ensureOnePerTest()

	schemes := r.bigInt128And64Schemes[r.bigInt128And64Cur]
	r.bigInt128And64Cur++
	if r.bigInt128And64Cur >= len(r.bigInt128And64Schemes) {
		r.bigInt128And64Cur = 0
	}
	return schemes[0].Value(r), schemes[1].Value(r)
}

func (r *rando) BigUint128AndBitSize() (*big.Int, uint) {
	r.ensureOnePerTest()

	scheme := r.bigUint128AndBitSizeSchemes[r.bigUint128AndBitSizeCur]
	r.bigUint128AndBitSizeCur++
	if r.bigUint128AndBitSizeCur >= len(r.bigUint128AndBitSizeSchemes) {
		r.bigUint128AndBitSizeCur = 0
	}
	return scheme.Values(r)
}

func (r *rando) BigUint128AndBitSizeAndBitValue() (*big.Int, uint, bool) {
	r.ensureOnePerTest()

	scheme := r.bigUint128AndBitSizeAndBitValueSchemes[r.bigUint128AndBitSizeAndBitValueCur]
	r.bigUint128AndBitSizeAndBitValueCur++
	if r.bigUint128AndBitSizeAndBitValueCur >= len(r.bigUint128AndBitSizeAndBitValueSchemes) {
		r.bigUint128AndBitSizeAndBitValueCur = 0
	}
	return scheme.Values(r)
}

func (r *rando) BigInt128() *big.Int {
	r.ensureOnePerTest()
	scheme := r.bigInt128Schemes[r.bigInt128Cur]
	r.bigInt128Cur++
	if r.bigInt128Cur >= len(r.bigInt128Schemes) {
		r.bigInt128Cur = 0
	}
	return scheme.Value(r)
}

func (r *rando) BigUint128() *big.Int {
	r.ensureOnePerTest()
	scheme := r.bigUint128Schemes[r.bigUint128Cur]
	r.bigUint128Cur++
	if r.bigUint128Cur >= len(r.bigUint128Schemes) {
		r.bigUint128Cur = 0
	}
	return scheme.Value(r)
}


func accUint128FromBigInt(b *big.Int) Uint128 {
	u, acc := Uint128FromBigInt(b)
	if !acc {
		panic(fmt.Errorf("num: inaccurate conversion to Uint128 in fuzz tester for %s", b))
	}
	return u
}

func accU64FromBigInt(b *big.Int) uint64 {
	if !b.IsUint64() {
		panic(fmt.Errorf("num: inaccurate conversion to U64 in fuzz tester for %s", b))
	}
	return b.Uint64()
}

func accI64FromBigInt(b *big.Int) int64 {
	if !b.IsInt64() {
		panic(fmt.Errorf("num: inaccurate conversion to I64 in fuzz tester for %s", b))
	}
	return b.Int64()
}

type StringList []string

func (s StringList) Strings() []string { return s }

func (s *StringList) String() string {
	if s == nil {
		return ""
	}
	return strings.Join(*s, ",")
}

func (s *StringList) Set(v string) error {
	vs := strings.Split(v, ",")
	for _, vi := range vs {
		vi = strings.TrimSpace(vi)
		if vi != "" {
			*s = append(*s, vi)
		}
	}
	return nil
}

func randomBigUint128(rng *rand.Rand) *big.Int {
	if rng == nil {
		rng = rand.New(rand.NewSource(time.Now().UnixMilli()))
	}

	var v = new(big.Int)
	bits := rng.Intn(129) - 1 // 128 bits, +1 for "0 bits"
	if bits < 0 {
		return v // "-1 bits" == "0"
	} else if bits <= 64 {
		v = v.Rand(rng, maxBigUint64)
	} else {
		v = v.Rand(rng, maxBigUint128)
	}
	v.And(v, masks[bits])
	v.SetBit(v, bits, 1)
	return v
}

func simulateBigUint128Overflow(rb *big.Int) *big.Int {
	for rb.Cmp(wrapBigUint128) >= 0 {
		rb = new(big.Int).And(rb, maxBigUint128)
	}
	return rb
}

func simulateBigInt128Overflow(rb *big.Int) *big.Int {
	if rb.Cmp(maxBigInt128) > 0 {
		// simulate overflow
		gap := new(big.Int)
		gap.Sub(rb, minBigInt128)
		r := new(big.Int).Rem(gap, wrapBigUint128)
		rb = new(big.Int).Add(r, minBigInt128)

	} else if rb.Cmp(minBigInt128) < 0 {
		// simulate underflow
		gap := new(big.Int).Set(rb)
		gap.Sub(maxBigInt128, gap)
		r := new(big.Int).Rem(gap, wrapBigUint128)
		rb = new(big.Int).Sub(maxBigInt128, r)
	}

	return rb
}