package goutils

func AnyFunc[ItemT any](slice []ItemT, match func(ItemT) bool) (ItemT, error) {
	for _, item := range slice {
		if match(item) {
			return item, nil
		}
	}
	var dummy ItemT
	return dummy, ErrAnyCouldNotFind
}

func Any[ItemT comparable](slice []ItemT, value ItemT) (ItemT, error) {
	return AnyFunc(slice, func(it ItemT) bool { return it == value })
}
