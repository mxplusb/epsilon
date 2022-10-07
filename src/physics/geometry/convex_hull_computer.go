package geometry

type ConvexHullComputer struct {
	Vertices []Vector3
	Edges    []Edge
	Faces    []int
}

func (c *ConvexHullComputer) Compute(coords float64, stride int, count int, shrink Scalar, shrinkClamp Scalar) Scalar {
	if count <= 0 {
		c.Vertices = nil
		c.Edges = nil
		c.Faces = nil
		return 0
	}

	return 0
}
