package geometry

type Point64 struct {
	X, Y, Z Int64
}

func (p Point64) IsZero() bool {
	return (p.X == 0) && (p.Y == 0) && (p.Z == 0)
}

func (p *Point64) Dot(b Point64) Int64 {
	return p.X * b.X + p.Y * b.Y + p.Z * b.Z
}