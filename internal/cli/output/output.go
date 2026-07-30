package output

import (
    "fmt"
    "strings"
)

const (
    reset  = "\033[0m"
    bold   = "\033[1m"
    green  = "\033[32m"
    red    = "\033[31m"
    yellow = "\033[33m"
    cyan   = "\033[36m"
    gray   = "\033[90m"
)

func Success(msg string) { fmt.Println(green + "✓ " + msg + reset) }
func Error(msg string)   { fmt.Println(red + "✗ " + msg + reset) }
func Info(msg string)    { fmt.Println(cyan + "  " + msg + reset) }
func Warn(msg string)    { fmt.Println(yellow + "⚠ " + msg + reset) }
func Bold(msg string)    { fmt.Println(bold + msg + reset) }
func Dim(msg string)     { fmt.Println(gray + msg + reset) }

func Table(headers []string, rows [][]string) {
	widths := make([]int, len(headers))
	for i, h := range headers {
		widths[i] = len(h)
	}
	for _, row := range rows {
		for i, cell := range row {
			if  i < len(widths) && len(cell) > widths[i] {
				widths[i] = len(cell)
			}
		}
	}

	sep := func ()  {
		fmt.Print("+")
		for _, w := range widths {
			fmt.Print(strings.Repeat("-", w+2) + "+")
		}
		fmt.Println()
	}
	row := func (cells []string)  {
		fmt.Print("|")
		for i, cell := range cells {
			if i >= len(widths) {
				break
		}
		fmt.Printf(" %-*s |", widths[i], cell)
	}
	fmt.Println()
}

	sep()
	row(headers)
	sep()
	for _, r := range rows {
		row(r)
	}
	sep()
}