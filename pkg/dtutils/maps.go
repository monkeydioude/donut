package dtutils

func MapMatchAllFunc[KeyT comparable, ValueT any](m map[KeyT]ValueT, cond func(KeyT, ValueT) bool) bool {
	for k, v := range m {
		if !cond(k, v) {
			return false
		}
	}
	return true
}

func MapMatchAll[KeyT, ValueT comparable](m map[KeyT]ValueT, value ValueT) bool {
	return MapMatchAllFunc(m, func(_ KeyT, v ValueT) bool { return v == value })
}

func MapMatchKeysAllFunc[KeyT comparable, ValueT any](m map[KeyT]ValueT, cond func(KeyT, ValueT) bool, keys ...KeyT) bool {
	for _, key := range keys {
		v, ok := m[key]
		if !ok {
			continue
		}
		if !cond(key, v) {
			return false
		}
	}
	return true
}

func MapMatchKeysAll[KeyT comparable, ValueT comparable](m map[KeyT]ValueT, value ValueT, keys ...KeyT) bool {
	return MapMatchKeysAllFunc(m, func(_ KeyT, v ValueT) bool { return v == value }, keys...)
}

func MapKeysByValue[KeyT comparable, ValueT comparable](m map[KeyT]ValueT, value ValueT) []KeyT {
	keys := []KeyT{}
	for k, v := range m {
		if v == value {
			keys = append(keys, k)
		}
	}
	return keys
}
