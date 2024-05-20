package main

import (
	"errors"
	"fmt"
	"slices"
	"strings"

	"golang.org/x/term"
)

type (
	errFrames struct{ XOutOfBounds, YOutOfBounds, NoRowsFound error }

	Row []int8

	Frame struct {
		Rows                   []Row
		CurX, CurY, MaxX, MaxY int
		CharMap                map[int8]string
	}

	colors struct{ Reset, Gray, Red, Green, Yellow, Blue, Purple, Cyan, White string }
)

var (
	ErrFrames = errFrames{
		XOutOfBounds: errors.New("y is out of bounds"),
		YOutOfBounds: errors.New("y is out of bounds"),
		NoRowsFound:  errors.New("no rows found"),
	}

	Colors = colors{
		Reset:  "\033[0m",
		Gray:   "\033[30m",
		Red:    "\033[31m",
		Green:  "\033[32m",
		Yellow: "\033[33m",
		Blue:   "\033[34m",
		Purple: "\033[35m",
		Cyan:   "\033[36m",
		White:  "\033[97m",
	}
)

func NewFrame(maxX int, maxY int, charMap map[int8]string) (*Frame, error) {
	x, y, err := term.GetSize(0)
	if err != nil {
		return &Frame{}, err
	}
	x = min(int(x/2), maxX)
	y = min(y-1, maxY)

	rows := []Row{}
	for i := 0; i < y; i++ {
		rows = append(rows, make(Row, x))
	}

	return &Frame{Rows: rows, CurX: x - 1, CurY: y - 1, MaxX: maxX, MaxY: maxY, CharMap: charMap}, nil
}

func (f *Frame) SetRow(y int, state int8) error {
	if y > len(f.Rows)-1 || y < 0 {
		return ErrFrames.XOutOfBounds
	}

	row := Row{}
	for i := 0; i <= f.CurX; i++ {
		row = append(row, state)
	}

	f.Rows = slices.Replace(f.Rows, y, y+1, row)

	return nil
}

func (f *Frame) SetCol(x int, state int8) error {
	if 0 > len(f.Rows)-1 {
		return ErrFrames.NoRowsFound
	}
	if x > len(f.Rows[0])-1 || x < 0 {
		return ErrFrames.XOutOfBounds
	}

	for i, _ := range f.Rows {
		f.Rows[i] = slices.Replace(f.Rows[i], x, x+1, state)
	}

	return nil
}

func (f *Frame) SetColRow(x int, y int, state int8) error {
	if y > len(f.Rows)-1 || y < 0 {
		return ErrFrames.YOutOfBounds
	}
	if x > len(f.Rows[y])-1 || x < 0 {
		return ErrFrames.XOutOfBounds
	}

	f.Rows[y] = slices.Replace(f.Rows[y], x, x+1, state)

	return nil
}

func (f *Frame) GetColRow(x int, y int) (int8, error) {
	if y > len(f.Rows)-1 || y < 0 {
		return -1, ErrFrames.YOutOfBounds
	}
	if x > len(f.Rows[y])-1 || x < 0 {
		return -1, ErrFrames.XOutOfBounds
	}

	return f.Rows[y][x], nil
}

func (f *Frame) Clear() {
	for i := 0; i <= f.CurY; i++ {
		f.Rows[i] = make(Row, f.CurX+1)
	}
}

func (f *Frame) Reload() error {
	x, y, err := term.GetSize(0)
	if err != nil {
		return err
	}
	f.CurX = min(int(x/2)-1, f.MaxX)
	f.CurY = min(y-2, f.MaxY)

	f.Rows = []Row{}
	for i := 0; i <= f.CurY; i++ {
		f.Rows = append(f.Rows, make(Row, f.CurX+1))
	}

	return nil
}

func (f *Frame) Draw() error {
	fmt.Print("\033[2J")
	fmt.Print("\033[0;0H")

	lines := []string{}
	for _, row := range f.Rows {
		line := ""
		for _, col := range row {
			char, ok := f.CharMap[col]
			if ok {
				line += char
				continue
			}

			char, ok = f.CharMap[-1]
			if ok {
				line += char
				continue
			}

			if col != 0 {
				line += "█"
				continue
			}

			line += " "
		}

		lines = append(lines, line)
	}

	fmt.Print(strings.Join(lines, "\r\n"))

	return nil
}
