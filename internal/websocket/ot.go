package websocket

import "devflow-backend/internal/models"

// Transform adjusts op2 so it can be applied after op1 has already been applied
func Transform(op1, op2 models.OTOperation) models.OTOperation {
	result := op2 
	if op1.Insert != "" && op2.Insert != "" {
		// Both are inserts
		if op1.Index <= op2.Index {
			result.Index += len([]rune(op1.Insert))
		}
		return result
	}
	
	if op1.Insert != "" && op2.Delete > 0 {
		// op1 inserts, op2 deletes
		insertLen := len([]rune(op1.Insert))
		if op1.Index <= op2.Index {
			result.Index += insertLen
		}
		return result
	}

	if op1.Delete > 0 && op2.Insert != "" {
		// op1 deletes, op2 inserts
		if op1.Index < op2.Index {
			del := op1.Delete
			if op2.Index-op1.Index < del {
				del = op2.Index - op1.Index
			}
			result.Index -= del
			if result.Index < op1.Index {
				result.Index = op1.Index
			}
		}
		return result
	}

	if op1.Delete > 0 && op2.Delete > 0 {
		// Both are deletes
		op1End := op1.Index + op1.Delete
		op2End := op2.Index + op2.Delete

		if op1End <= op2.Index {
			// op1 is entirely before op2
			result.Index -= op1.Delete
		} else if op1.Index >= op2End {
			// op1 is entirely after op2 — no adjustment needed
		} else { 
			// Overlapping deletes — shrink op2's delete range
			overlapStart := max(op1.Index,  op2.Index)
			overlapEnd := min(op1End, op2End)
			overlap := overlapEnd - overlapStart
			result.Delete -= overlap
			if op1.Index <= op2.Index {
				result.Index = op1.Index
			}
		}
		return result
	}
	return result
}

// ApplyOp applies an OTOperation to a rune slice and returns the new text
func ApplyOp(doc []rune, op models.OTOperation) []rune {
	n := len(doc)

	if op.Insert != "" {
		idx := clamp(op.Index, 0, n)
		ins := []rune(op.Insert)
		result := make([]rune, 0, n+len(ins))
		result = append(result, doc[:idx]...)
		result = append(result, ins...)
		result = append(result, doc[idx:]...)
		return result
	}

	if op.Delete > 0 {
		idx := clamp(op.Index, 0, n)
		end := clamp(op.Index+op.Delete, 0, n)
		result := make([]rune, 0, n-(end-idx))
		result = append(result, doc[:idx]...)
		result = append(result, doc[end:]...)
		return result
	}
	return doc
}

func clamp(v, lo, hi int) int {
	if v < lo {
		return lo
	}
	if v > hi {
		return hi
	}
	return v
}

func max(a, b int) int {
	if a > b {
		return a
	}
	return b
}

func min(a, b int) int {
	if a < b {
		return a
	}
	return b
}
