package utils

type Set map[uint8]bool

func NewSet(elements []uint8) Set {
	set := make(map[uint8]bool)
	for _, element := range elements {
		set[element] = true
	}
	return set
}

// func (s Set) Add(element uint8) {
// 	s[element] = true
// }
//
// func (s Set) Remove(element uint8) {
// 	delete(s, element)
// }

func (s Set) Contains(element uint8) bool {
	return s[element]
}
