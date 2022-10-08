package geometry

type Face struct {
	Next                     *Face
	NearbyVertex             *Vertex
	NextWithSameNearbyVertex *Face
	Origin                   Point32
	Dir0                     Point32
	Dir1                     Point32
}

func NewFace() *Face {
	return &Face{
		Next:                     nil,
		NearbyVertex:             nil,
		NextWithSameNearbyVertex: nil,
	}
}

func (f *Face) init(a, b, c *Vertex) {
	f.NearbyVertex = a
	f.Origin = a.Point
	f.Dir0 = b.Subtract(*a)
	f.Dir1 = c.Subtract(*a)

	if a.LastNearbyFace != nil {
		a.LastNearbyFace.NextWithSameNearbyVertex = f
	} else {
		a.FirstNearbyFace = f
	}
	a.LastNearbyFace = f
}

func (f *Face) GetNormal() Point64 {
	return f.Dir0.Cross32(f.Dir1)
}
