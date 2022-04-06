package git

import "path/filepath"

func objectFanoutPaths(objectID string) (res []string) {
	res = append(res, objectID)
	if len(objectID) <= 2 {
		return
	}

	for _, path := range objectFanoutPaths(objectID[2:]) {
		res = append(res, filepath.Join(objectID[0:2], path))
	}

	return
}
