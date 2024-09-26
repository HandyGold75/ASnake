package screen

import (
	"bytes"
	"errors"
	"os"
	"slices"
	"unicode"

	"golang.org/x/term"
)

type (
	row []int8

	screenHelper struct {
		f *Screen
	}

	Screen struct {
		Rows                   []row
		CurX, CurY, MaxX, MaxY int
		CharMap                map[int8][]byte
		Terminal               *term.Terminal
		OnResizeCallback       func(f *Screen)
		H                      screenHelper
	}

	Cord       struct{ X, Y int }
	colors     struct{ Reset, Gray, Red, Green, Yellow, Blue, Purple, Cyan, White string }
	errScreens struct{ XOutOfBounds, YOutOfBounds, NoRowsFound, InvalidSTInOut error }
)

var (
	ErrScreens = errScreens{
		XOutOfBounds: errors.New("x is out of bounds"),
		YOutOfBounds: errors.New("y is out of bounds"),
		NoRowsFound:  errors.New("no rows found"),
	}
)

func NewScreen(maxX, maxY int, charMap map[int8][]byte, terminal *term.Terminal) (*Screen, error) {
	x, y, err := term.GetSize(int(os.Stdin.Fd()))
	if err != nil {
		return &Screen{}, err
	}
	x = min(int(x/2), maxX)
	y = min(y-1, maxY)

	rows := []row{}
	for i := 0; i < y; i++ {
		rows = append(rows, make(row, x))
	}

	f := &Screen{
		Rows: rows,
		CurX: x - 1, CurY: y - 1, MaxX: maxX, MaxY: maxY,
		CharMap:  charMap,
		Terminal: terminal,
	}
	f.H.f = f

	return f, nil
}

func (f *Screen) SetRow(y int, state int8) error {
	if y > len(f.Rows)-1 || y < 0 {
		return ErrScreens.XOutOfBounds
	}

	r := row{}
	for i := 0; i <= f.CurX; i++ {
		r = append(r, state)
	}

	f.Rows = slices.Replace(f.Rows, y, y+1, r)

	return nil
}

func (f *Screen) SetCol(x int, state int8) error {
	if 0 > len(f.Rows)-1 {
		return ErrScreens.NoRowsFound
	}
	if x > len(f.Rows[0])-1 || x < 0 {
		return ErrScreens.XOutOfBounds
	}

	for i := range f.Rows {
		f.Rows[i] = slices.Replace(f.Rows[i], x, x+1, state)
	}

	return nil
}

func (f *Screen) SetColRow(x, y int, state int8) error {
	if y > len(f.Rows)-1 || y < 0 {
		return ErrScreens.YOutOfBounds
	}
	if x > len(f.Rows[y])-1 || x < 0 {
		return ErrScreens.XOutOfBounds
	}

	f.Rows[y] = slices.Replace(f.Rows[y], x, x+1, state)

	return nil
}

func (f *Screen) GetColRow(x, y int) (int8, error) {
	if y > len(f.Rows)-1 || y < 0 {
		return -1, ErrScreens.YOutOfBounds
	}
	if x > len(f.Rows[y])-1 || x < 0 {
		return -1, ErrScreens.XOutOfBounds
	}

	return f.Rows[y][x], nil
}

func (f *Screen) Clear() {
	for i := 0; i <= f.CurY; i++ {
		f.Rows[i] = make(row, f.CurX+1)
	}
}

func (f *Screen) Reload() error {
	x, y, err := term.GetSize(int(os.Stdin.Fd()))
	if err != nil {
		return err
	}
	f.CurX = min(int(x/2)-1, f.MaxX)
	f.CurY = min(y-2, f.MaxY)

	f.Rows = []row{}
	for i := 0; i <= f.CurY; i++ {
		f.Rows = append(f.Rows, make(row, f.CurX+1))
	}

	return nil
}

func (f *Screen) Draw() error {
	if f.OnResizeCallback != nil {
		x, y, err := term.GetSize(int(os.Stdin.Fd()))
		if err != nil {
			return err
		}

		if f.CurX != min(int(x/2)-1, f.MaxX) || f.CurY != min(y-2, f.MaxY) {
			f.CurX = min(int(x/2)-1, f.MaxX)
			f.CurY = min(y-2, f.MaxY)

			f.Rows = []row{}
			for i := 0; i <= f.CurY; i++ {
				f.Rows = append(f.Rows, make(row, f.CurX+1))
			}

			if _, err := f.Terminal.Write([]byte("\033[2J")); err != nil {
				return err
			}
			f.OnResizeCallback(f)
		}
	}

	lines := [][]byte{}
	for _, r := range f.Rows {
		line := []byte{}
		for _, col := range r {
			char, ok := f.CharMap[col]
			if ok {
				line = append(line, char...)
				continue
			}

			char, ok = f.CharMap[-1]
			if ok {
				line = append(line, char...)
				continue
			}

			if col != 0 {
				line = append(line, []byte("██")...)
				continue
			}

			line = append(line, []byte("  ")...)
		}
		lines = append(lines, line)
	}
	lines = append(lines, []byte{})

	if _, err := f.Terminal.Write(append([]byte("\033[0;0H"), bytes.Join(lines, []byte("\r\n"))...)); err != nil {
		return err
	}

	return nil
}

func (fh *screenHelper) RenderString(str string, offsetX, offsetY int, state int8) {
	for _, r := range str {
		cords, ok := charMap[unicode.ToUpper(r)]
		if !ok {
			continue
		}

		fh.RenderCords(cords, offsetX, offsetY, state)
		offsetX += 6
	}
}

func (fh *screenHelper) RenderStringIf(str string, offsetX, offsetY int, state int8, set func(int8) bool) {
	for _, r := range str {
		cords, ok := charMap[unicode.ToUpper(r)]
		if !ok {
			continue
		}

		fh.RenderCordsIf(cords, offsetX, offsetY, state, set)
		offsetX += 6
	}
}

func (fh *screenHelper) RenderCords(cords []Cord, offsetX, offsetY int, state int8) {
	for _, cord := range cords {
		fh.f.SetColRow(cord.X+offsetX, cord.Y+offsetY, state)
	}
}

func (fh *screenHelper) RenderCordsIf(cords []Cord, offsetX, offsetY int, state int8, set func(int8) bool) {
	for _, cord := range cords {
		val, err := fh.f.GetColRow(cord.X+offsetX, cord.Y+offsetY)
		if err != nil {
			continue
		}
		if set(val) {
			fh.f.SetColRow(cord.X+offsetX, cord.Y+offsetY, state)
		}
	}
}
