package geometry

type Vertex struct {
	Next            *Vertex
	Previous        *Vertex
	Edge            *Edge
	FirstNearbyFace *Face
	LastNearbyFace  *Face
	Point128        PointRational128
	Point           Point32
	copy            int
}

func (vx Vertex) Subtract(vy Vertex) Point32 {
	return vx.Point.Subtract(vy.Point)
}

func (vx Vertex) Dot(p Point64) Rational128 {
	if vx.Point.index >= 0 {
		return Rational128FromInt64(vx.Point.Dot64(p))
	} else {
		// long form of x*x + y*y + z*z
		x := vx.Point128.X.Mul64(p.X)
		y := vx.Point128.Y.Mul64(p.Y)
		z := vx.Point128.Z.Mul64(p.Z)
		a := x.Add(y).Add(z)
		return NewRational128(a, vx.Point128.Denominator)
	}
}

func (vx Vertex) XScalar() Scalar {
	if vx.Point.index >= 0 {
		return Scalar(vx.Point.X)
	} else {
		return vx.Point128.XScalar()
	}
}

func (vx Vertex) YScalar() Scalar {
	if vx.Point.index >= 0 {
		return Scalar(vx.Point.Y)
	} else {
		return vx.Point128.YScalar()
	}
}

func (vx Vertex) ZScalar() Scalar {
	if vx.Point.index >= 0 {
		return Scalar(vx.Point.Z)
	} else {
		return vx.Point128.ZScalar()
	}
}
