package geometry

type PointRational128 struct {
	X,Y,Z,Denominator Int128
}

func NewPointRational128(x Int128, y Int128, z Int128, denominator Int128) PointRational128 {
	return PointRational128{X: x, Y: y, Z: z, Denominator: denominator}
}

func (r PointRational128) XScalar() Scalar {
	return r.X.ToScalar() / r.Denominator.ToScalar()
}

func (r PointRational128) YScalar() Scalar {
	return r.Y.ToScalar() / r.Denominator.ToScalar()
}

func (r PointRational128) ZScalar() Scalar {
	return r.Z.ToScalar() / r.Denominator.ToScalar()
}