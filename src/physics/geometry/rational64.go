package geometry

func Rational64FromInt64s(numerator Int64, denominator Int64) Rational64 {
	r := Rational64{}

	if numerator > 0 {
		r.sign = 1
		r.numerator = Uint64(numerator)
	} else if numerator < 0 {
		r.sign = -1
		r.numerator = Uint64(-numerator)
	} else {
		r.sign = 0
		numerator = 0
	}

	if denominator > 0 {
		r.denominator = Uint64(denominator)
	} else if denominator < 0 {
		r.sign = -r.sign
		r.denominator = Uint64(-denominator)
	} else {
		r.denominator = 0
	}

	return r
}

type Rational64 struct {
	numerator   Uint64
	denominator Uint64
	sign        int
}

func (r Rational64) IsNegativeInfinity() bool {
	return (r.sign < 0) && (r.denominator == 0)
}

func (r Rational64) IsNaN() bool {
	return (r.sign == 0) && (r.denominator == 0)
}

func (r Rational64) ToScalar() Scalar {
	if r.denominator == 0 {
		return Scalar(float64(r.sign) * Infinity)
	} else {
		return Scalar(r.sign) * Scalar(r.numerator / r.denominator)
	}
}
