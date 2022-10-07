package geometry

type Vector3 struct {
	X float64
	Y float64
	Z float64
	W float64
}

func (v3 Vector3) Dot(v *Vector3) Scalar {
	return Scalar(v3.X*v.X +
		v3.Y*v.Y +
		v3.Z*v.Z)
}
