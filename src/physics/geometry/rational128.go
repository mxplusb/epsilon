package geometry

func NewRational128(numerator Int128, denominator Int128) Rational128 {
	r := Rational128{}
	sign := numerator.Sign()
	if sign >= 0 {
		r.numerator = numerator
	} else {
		// swap the sign
		r.numerator = numerator.Mul64(-1)
	}

	dsign := denominator.Sign()
	if dsign >= 0 {
		r.denominator = denominator
	} else {
		r.sign = -sign
		// swap the sign
		r.denominator = denominator.Mul64(-1)
	}
	r.isInt64 = false
	return r
}

func Rational128FromInt64(v Int64) (r Rational128) {
	r = Rational128{}
	if v > 0 {
		r.sign = 1
		r.numerator = Int128From64(int64(v))
	} else if v < 0 {
		r.sign = -1
		r.numerator = Int128From64(int64(-v))
	} else {
		r.sign = 0
		r.numerator = Int128FromInt(0)
	}
	r.denominator = Int128FromInt(1)
	r.isInt64 = true

	return r
}

type Rational128 struct {
	numerator   Int128
	denominator Int128
	sign int
	isInt64 bool
}

func (r *Rational128) ToScalar() Scalar {
	if r.denominator.Sign() == 0 {
		return Scalar(float64(r.sign) * Infinity)
	} else {
		return Scalar(r.sign) * r.numerator.ToScalar() / r.denominator.ToScalar()
	}
}
