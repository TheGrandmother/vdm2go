package main

func neg(x bool) (y bool)
func POST_neg(x bool, y bool) bool {
	return x == !y
}
func andb(x, y bool) (z bool)
func POST_andb(x, y bool, z bool) bool {
	return x && y == z || z
}
