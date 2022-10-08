package geometry

type Point32 struct {
	X, Y, Z Int32
	index int
}

func NewPoint32(x Int32, y Int32, z Int32,) Point32 {
	return Point32{X: x, Y: y, Z: z, index: -1}
}

func (p Point32) IsZero() bool {
	return (p.X == 0) && (p.Y == 0) && (p.Z == 0)
}

func (p *Point32) Cross32(b Point32) Point64 {
	return Point64{
		X: Int64(p.Y*b.Z - p.Z*b.Y), // y * b.z - z * b.y
		Y: Int64(p.Z*b.X - p.X*b.Z), // z * b.x - x * b.z
		Z: Int64(p.X*b.Y - p.Y*b.X), // x * b.y - y * b.x
	}
}

func (p *Point32) Cross64(b Point64) Point64 {
	return Point64{
		X: Int64(p.Y)*b.Z - Int64(p.Z)*b.Y, // y * b.z - z * b.y
		Y: Int64(p.Z)*b.X - Int64(p.X)*b.Z, // z * b.x - x * b.z
		Z: Int64(p.X)*b.Y - Int64(p.Y)*b.X, // x * b.y - y * b.x
	}
}

func (p *Point32) Dot32(b Point32) Int64 {
	return Int64(p.X*b.X + p.Y*b.Y + p.Z*b.Z)
}

func (p *Point32) Dot64(b Point64) Int64 {
	return Int64(p.X)*b.X + Int64(p.Y)*b.Y + Int64(p.Z)*b.Z
}

func (p *Point32) Equals(b Point32) bool {
	return (p.X == b.X) && (p.Y == b.Y) && (p.Z == b.Z)
}

func (p *Point32) NotEquals(b Point32) bool {
	return (p.X != b.X) || (p.Y != b.Y) || (p.Z != b.Z)
}

func (p *Point32) Add(b Point32) Point32 {
	return Point32{
		X: p.X + b.X,
		Y: p.Y + b.Y,
		Z: p.Z + b.Z,
	}
}

func (p *Point32) Subtract(b Point32) Point32 {
	return Point32{
		X: p.X - b.X,
		Y: p.Y - b.Y,
		Z: p.Z - b.Z,
	}
}
