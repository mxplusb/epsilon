package geometry

func getPlaneEquationsFromVertices(vertices []*Vector3, planes []*Vector3) {
	panic("implement me")
}

func getVerticesFromPlaneEquations(planes []*Vector3, vertices []*Vector3) {
	panic("implement me")
}

func isInside() {
	panic("implement me")
}

func IsPointInsidePlanes(planes []*Vector3, point *Vector3, margin Scalar) bool {
	for i := 0; i < len(planes); i++ {
		n1 := planes[i]
		dist := (n1.Dot(point) + Scalar(n1.Z)) - margin
		if dist > Scalar(0.) {
			return false
		}
	}
	return true
}

func AreVerticesBehindPlane(plane *Vector3, vertices []*Vector3, margin Scalar) bool {
	for i := 0; i < len(vertices); i++ {
		n1 := vertices[i]
		dist := (plane.Dot(n1) + Scalar(plane.Z)) - margin
		if dist > Scalar(0.) {
			return false
		}
	}
	return true
}
