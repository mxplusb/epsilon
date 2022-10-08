package geometry

type DMul[uword,uHword Int32 | Int64 | Int128 | Uint32 | Uint64] struct {}

func (d DMul[uword, uHword]) Mul(a uword, b uHword, aOut *uword, bOut *uword) {

}

func (d DMul[uword, uHword]) high(v Uint64) Uint32 {
	return Uint32(v >> 32)
}

func (d DMul[uword, uHword]) low(v Uint64) Uint32 {
	return Uint32(v)
}

func (d DMul[uword, uHword]) mul(a,b Uint32) Uint64 {
	return Uint64(a) * Uint64(b)
}

func (d DMul[uword, uHword]) shHalf(v *Uint64) {
	*v <<= 32
}

func (DMul[uword, uHword]) i128High(v Int128) Uint64 {
	return v.hi
}