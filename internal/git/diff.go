package git

import (
	"fmt"
	"strings"

	"devflow-backend/internal/models"
)

// editType is the type of a single edit operation from Myers diff.
type editType int

const (
	editEqual  editType = iota
	editInsert
	editDelete
)

type edit struct {
	kind editType
	text string
}

// lcs computes the shortest edit script between two string slices using
func computeEdits(aLines, bLines []string) []edit {
	n, m := len(aLines), len(bLines)
	if n == 0 && m == 0 {
		return nil
	}
	if n == 0 {
		edits := make([]edit, m)
		for i, l := range bLines {
			edits[i] = edit{editInsert, l}
		}
		return edits
	}
	if m == 0 {
		edits := make([]edit, n)
		for i, l := range aLines {
			edits[i] = edit{editDelete, l}
		}
		return edits
	}

	max := n + m
	v := make([]int, 2*max+1)
	trace := [][]int{}

	for d := 0; d <= max; d++ {
		snap := make([]int, len(v))
		copy(snap, v)
		trace = append(trace, snap)

		for k := -d; k <= d; k += 2 {
			var x int
			ki := k + max
			if k == -d || (k != d && v[ki-1] < v[ki+1]) {
				x = v[ki+1]
			} else {
				x = v[ki-1] + 1
			}
			y := x - k
			for x < n && y < m && aLines[x] == bLines[y] {
				x++
				y++
			}
			v[ki] = x
			if x >= n && y >= m {
				return backtrack(trace, aLines, bLines, max)
			}
		}
	}
	return backtrack(trace, aLines, bLines, max)
}

func backtrack(trace [][]int, a, b []string, offset int) []edit {
	x, y := len(a), len(b)
	var edits []edit
	for d := len(trace) - 1; d >= 0; d-- {
		v := trace[d]
		k := x - y
		ki := k + offset
		var prevK int
		if k == -d || (k != d && v[ki-1] < v[ki+1]) {
			prevK = k + 1
		} else {
			prevK = k - 1
		}
		prevX := v[prevK+offset]
		prevY := prevX - prevK
		for x > prevX && y > prevY {
			edits = append([]edit{{editEqual, a[x-1]}}, edits...)
			x--
			y--
		}
		if d > 0 {
			if x == prevX {
				edits = append([]edit{{editInsert, b[y-1]}}, edits...)
				y--
			} else {
				edits = append([]edit{{editDelete, a[x-1]}}, edits...)
				x--
			}
		}
	}
	return edits
}

// intPtr is a helper to take the address of an int.
func intPtr(i int) *int { return &i }

// ComputeDiff produces unified-diff hunks for oldText → newText.
func ComputeDiff(oldText, newText string, contextLines int) []models.DiffHunk {
	aLines := splitLines(oldText)
	bLines := splitLines(newText)

	edits := computeEdits(aLines, bLines)

	// Convert edits into DiffLine slice with line numbers
	type rawLine struct {
		line  models.DiffLine
		isNew bool // true if it's an addition or context (belongs to new side)
		isOld bool // true if it's a deletion or context (belongs to old side)
	}

	var raw []rawLine
	oldNo, newNo := 1, 1
	for _, e := range edits {
		switch e.kind {
		case editEqual:
			o, n := oldNo, newNo
			raw = append(raw, rawLine{
				line:  models.DiffLine{Type: models.DiffContext, Content: " " + e.text, OldNo: intPtr(o), NewNo: intPtr(n)},
				isOld: true, isNew: true,
			})
			oldNo++
			newNo++
		case editDelete:
			o := oldNo
			raw = append(raw, rawLine{
				line:  models.DiffLine{Type: models.DiffDeletion, Content: "-" + e.text, OldNo: intPtr(o)},
				isOld: true,
			})
			oldNo++
		case editInsert:
			n := newNo
			raw = append(raw, rawLine{
				line:  models.DiffLine{Type: models.DiffAddition, Content: "+" + e.text, NewNo: intPtr(n)},
				isNew: true,
			})
			newNo++
		}
	}

	// Group into hunks: find changed lines, expand with contextLines on each side
	type span struct{ start, end int }
	var changeSpans []span
	i := 0
	for i < len(raw) {
		if raw[i].line.Type != models.DiffContext {
			start := i
			for i < len(raw) && raw[i].line.Type != models.DiffContext {
				i++
			}
			changeSpans = append(changeSpans, span{start, i - 1})
		} else {
			i++
		}
	}

	if len(changeSpans) == 0 {
		return nil
	}

	// Merge overlapping/adjacent spans after expanding by contextLines
	type hunkSpan struct{ start, end int }
	var hunkSpans []hunkSpan
	for _, s := range changeSpans {
		lo := max0(s.start-contextLines, 0)
		hi := min0(s.end+contextLines, len(raw)-1)
		if len(hunkSpans) > 0 && lo <= hunkSpans[len(hunkSpans)-1].end+1 {
			hunkSpans[len(hunkSpans)-1].end = hi
		} else {
			hunkSpans = append(hunkSpans, hunkSpan{lo, hi})
		}
	}

	var hunks []models.DiffHunk
	for _, hs := range hunkSpans {
		slice := raw[hs.start : hs.end+1]
		var lines []models.DiffLine
		for _, r := range slice {
			lines = append(lines, r.line)
		}

		// Compute hunk header values
		oldStart, oldCount := 0, 0
		newStart, newCount := 0, 0
		for _, r := range slice {
			if r.isOld {
				if oldStart == 0 && r.line.OldNo != nil {
					oldStart = *r.line.OldNo
				}
				oldCount++
			}
			if r.isNew {
				if newStart == 0 && r.line.NewNo != nil {
					newStart = *r.line.NewNo
				}
				newCount++
			}
		}
		if oldStart == 0 {
			oldStart = 1
		}
		if newStart == 0 {
			newStart = 1
		}

		header := fmt.Sprintf("@@ -%d,%d +%d,%d @@", oldStart, oldCount, newStart, newCount)
		hunks = append(hunks, models.DiffHunk{
			Header:   header,
			OldStart: oldStart,
			OldCount: oldCount,
			NewStart: newStart,
			NewCount: newCount,
			Lines:    lines,
		})
	}
	return hunks
}

func splitLines(text string) []string {
	if text == "" {
		return nil
	}
	lines := strings.Split(text, "\n")
	// Remove trailing empty line caused by trailing newline
	if len(lines) > 0 && lines[len(lines)-1] == "" {
		lines = lines[:len(lines)-1]
	}
	return lines
}

func max0(a, b int) int {
	if a > b {
		return a
	}
	return b
}

func min0(a, b int) int {
	if a < b {
		return a
	}
	return b
}
