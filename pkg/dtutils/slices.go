package dtutils

func SliceMatchAllFunc[ItemT comparable](s []ItemT, cond func(ItemT) bool) bool {
	for _, item := range s {
		if !cond(item) {
			return false
		}
	}
	return true
}

func SliceMatchAll[ItemT comparable](s []ItemT, value ItemT) bool {
	return SliceMatchAllFunc(s, func(item ItemT) bool { return item == value })
}
