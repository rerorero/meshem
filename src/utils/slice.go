package utils

func ContainsString(slice []string, value string) (index int, ok bool) {
	for i, v := range slice {
		if value == v {
			return i, true
		}
	}
	return -1, false
}

// RemoveFromStringSlice removes a string entity from the string slice.
func RemoveFromStringSlice(slice []string, value string) (removed []string) {
	for _, v := range slice {
		if v != value {
			removed = append(removed, v)
		}
	}
	return removed
}

// FilterNotContainsString removes all items in 'filterElements' from the 'slice'.
func FilterNotContainsString(slice []string, filterElements []string) (filtered []string) {
	elements := map[string]struct{}{}
	var i int
	for i = 0; i < len(filterElements); i++ {
		elements[filterElements[i]] = struct{}{}
	}
	for _, s := range slice {
		_, ok := elements[s]
		if !ok {
			filtered = append(filtered, s)
		}
	}
	return filtered
}

// IntersectStringSlice returns intersections.
func IntersectStringSlice(sliceA []string, sliceB []string) (intersect []string) {
	elements := map[string]struct{}{}
	var i int
	for i = 0; i < len(sliceB); i++ {
		elements[sliceB[i]] = struct{}{}
	}
	for _, s := range sliceA {
		_, ok := elements[s]
		if ok {
			intersect = append(intersect, s)
		}
	}
	return intersect
}
