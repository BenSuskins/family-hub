package chores

// RotationStart returns the index of the candidate that should be tried first.
// When the previously assigned user is still in the candidate list, rotation
// continues from the person after them (robust to membership changes). Otherwise
// it falls back to advancing the stored positional index. An empty pool returns -1.
func RotationStart(candidateIDs []string, lastAssignedUserID *string, lastIndex int) int {
	n := len(candidateIDs)
	if n == 0 {
		return -1
	}
	if lastAssignedUserID != nil {
		for i, userID := range candidateIDs {
			if userID == *lastAssignedUserID {
				return (i + 1) % n
			}
		}
	}
	return ((lastIndex+1)%n + n) % n
}
