package geometry

type Vertex struct {
	Next, Previous *Vertex
	Edge *Edge
	FirstNearbyFace, LastNearbyFace *Face
	Point128 PointRational128
	Point Point32
	copy int
}

func (vx Vertex) Subtract(vy Vertex) Point32 {
	return vx.Point.Subtract(vx.Point)
}

//func (vx Vertex) Dot(p Point64) Rational128 {
//	if vx.Point.index >= 0 {
//		r := Rational128{}
//		r.FromInt64(vx.Point.Dot64(p))
//		return r
//	} else {
//		x := vx.Point128.X.Mul(p.X)
//		return NewRational128(x + vx.Point128.Y * p.Y + vx.Point128.Z * p.Z)
//	}
//}