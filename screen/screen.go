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
	Screen struct {
		Rows                   [][]uint8
		CurX, CurY, MaxX, MaxY int
		ForceMax               bool
		CharMap                map[uint8][]byte
		Terminal               *term.Terminal
		OnResizeCallback       func(f *Screen)
	}

	Cord struct{ X, Y int }
)

var (
	ErrXOutOfBounds = errors.New("x is out of bounds")
	ErrYOutOfBounds = errors.New("y is out of bounds")
	ErrNoRowsFound  = errors.New("no []uint8s found")

	// t := []Cord{
	// 	{X: 0, Y: 0}, {X: 1, Y: 0}, {X: 2, Y: 0}, {X: 3, Y: 0}, {X: 4, Y: 0},
	// 	{X: 0, Y: 1}, {X: 1, Y: 1}, {X: 2, Y: 1}, {X: 3, Y: 1}, {X: 4, Y: 1},
	// 	{X: 0, Y: 2}, {X: 1, Y: 2}, {X: 2, Y: 2}, {X: 3, Y: 2}, {X: 4, Y: 2},
	// 	{X: 0, Y: 3}, {X: 1, Y: 3}, {X: 2, Y: 3}, {X: 3, Y: 3}, {X: 4, Y: 3},
	// 	{X: 0, Y: 4}, {X: 1, Y: 4}, {X: 2, Y: 4}, {X: 3, Y: 4}, {X: 4, Y: 4},
	// }

	CharMap = map[rune][]Cord{
		'A':  {{1, 0}, {2, 0}, {3, 0}, {0, 1}, {4, 1}, {0, 2}, {1, 2}, {2, 2}, {3, 2}, {4, 2}, {0, 3}, {4, 3}, {0, 4}, {4, 4}},
		'B':  {{0, 0}, {1, 0}, {2, 0}, {3, 0}, {0, 1}, {4, 1}, {0, 2}, {1, 2}, {2, 2}, {3, 2}, {0, 3}, {4, 3}, {0, 4}, {1, 4}, {2, 4}, {3, 4}},
		'C':  {{1, 0}, {2, 0}, {3, 0}, {4, 0}, {0, 1}, {0, 2}, {0, 3}, {1, 4}, {2, 4}, {3, 4}, {4, 4}},
		'D':  {{0, 0}, {1, 0}, {2, 0}, {3, 0}, {0, 1}, {4, 1}, {0, 2}, {4, 2}, {0, 3}, {4, 3}, {0, 4}, {1, 4}, {2, 4}, {3, 4}},
		'E':  {{0, 0}, {1, 0}, {2, 0}, {3, 0}, {4, 0}, {0, 1}, {0, 2}, {1, 2}, {2, 2}, {3, 2}, {0, 3}, {0, 4}, {1, 4}, {2, 4}, {3, 4}, {4, 4}},
		'F':  {{0, 0}, {1, 0}, {2, 0}, {3, 0}, {4, 0}, {0, 1}, {0, 2}, {1, 2}, {2, 2}, {3, 2}, {0, 3}, {0, 4}},
		'G':  {{1, 0}, {2, 0}, {3, 0}, {4, 0}, {0, 1}, {0, 2}, {3, 2}, {4, 2}, {0, 3}, {4, 3}, {1, 4}, {2, 4}, {3, 4}, {4, 4}},
		'H':  {{0, 0}, {4, 0}, {0, 1}, {4, 1}, {0, 2}, {1, 2}, {2, 2}, {3, 2}, {4, 2}, {0, 3}, {4, 3}, {0, 4}, {4, 4}},
		'I':  {{0, 0}, {1, 0}, {2, 0}, {3, 0}, {4, 0}, {2, 1}, {2, 2}, {2, 3}, {0, 4}, {1, 4}, {2, 4}, {3, 4}, {4, 4}},
		'J':  {{1, 0}, {2, 0}, {3, 0}, {4, 0}, {3, 1}, {3, 2}, {0, 3}, {3, 3}, {1, 4}, {2, 4}},
		'K':  {{0, 0}, {4, 0}, {0, 1}, {3, 1}, {0, 2}, {1, 2}, {2, 2}, {0, 3}, {3, 3}, {0, 4}, {4, 4}},
		'L':  {{0, 0}, {0, 1}, {0, 2}, {0, 3}, {0, 4}, {1, 4}, {2, 4}, {3, 4}, {4, 4}},
		'M':  {{0, 0}, {1, 0}, {3, 0}, {4, 0}, {0, 1}, {2, 1}, {4, 1}, {0, 2}, {2, 2}, {4, 2}, {0, 3}, {4, 3}, {0, 4}, {4, 4}},
		'N':  {{0, 0}, {1, 0}, {4, 0}, {0, 1}, {2, 1}, {4, 1}, {0, 2}, {2, 2}, {4, 2}, {0, 3}, {2, 3}, {4, 3}, {0, 4}, {3, 4}, {4, 4}},
		'O':  {{1, 0}, {2, 0}, {3, 0}, {0, 1}, {4, 1}, {0, 2}, {4, 2}, {0, 3}, {4, 3}, {1, 4}, {2, 4}, {3, 4}},
		'P':  {{0, 0}, {1, 0}, {2, 0}, {3, 0}, {0, 1}, {4, 1}, {0, 2}, {4, 2}, {0, 3}, {1, 3}, {2, 3}, {3, 3}, {0, 3}, {0, 4}},
		'Q':  {{1, 0}, {2, 0}, {3, 0}, {0, 2}, {4, 2}, {0, 3}, {3, 3}, {0, 1}, {4, 1}, {1, 4}, {2, 4}, {4, 4}},
		'R':  {{0, 0}, {1, 0}, {2, 0}, {3, 0}, {0, 1}, {4, 1}, {0, 2}, {4, 2}, {0, 3}, {1, 3}, {2, 3}, {3, 3}, {0, 4}, {4, 4}},
		'S':  {{1, 0}, {2, 0}, {3, 0}, {4, 0}, {0, 1}, {0, 2}, {1, 2}, {2, 2}, {3, 2}, {4, 3}, {0, 4}, {1, 4}, {2, 4}, {3, 4}},
		'T':  {{0, 0}, {1, 0}, {2, 0}, {3, 0}, {4, 0}, {2, 1}, {2, 2}, {2, 3}, {2, 4}},
		'U':  {{0, 0}, {4, 0}, {0, 1}, {4, 1}, {0, 2}, {4, 2}, {0, 3}, {4, 3}, {1, 4}, {2, 4}, {3, 4}},
		'V':  {{0, 0}, {4, 0}, {0, 1}, {4, 1}, {0, 2}, {4, 2}, {1, 3}, {3, 3}, {2, 4}},
		'W':  {{0, 0}, {4, 0}, {0, 1}, {4, 1}, {0, 2}, {2, 2}, {4, 2}, {0, 3}, {2, 3}, {4, 3}, {1, 4}, {3, 4}},
		'X':  {{0, 0}, {4, 0}, {1, 1}, {3, 1}, {2, 2}, {1, 3}, {3, 3}, {0, 4}, {4, 4}},
		'Y':  {{0, 0}, {4, 0}, {1, 1}, {3, 1}, {2, 2}, {2, 3}, {2, 4}},
		'Z':  {{0, 0}, {1, 0}, {2, 0}, {3, 0}, {4, 0}, {4, 1}, {1, 2}, {2, 2}, {3, 2}, {0, 3}, {0, 4}, {1, 4}, {2, 4}, {3, 4}, {4, 4}},
		'0':  {{1, 0}, {2, 0}, {3, 0}, {0, 1}, {4, 1}, {0, 2}, {2, 2}, {4, 2}, {0, 3}, {4, 3}, {1, 4}, {2, 4}, {3, 4}},
		'1':  {{2, 0}, {1, 1}, {2, 1}, {0, 2}, {2, 2}, {2, 3}, {0, 4}, {1, 4}, {2, 4}, {3, 4}, {4, 4}},
		'2':  {{0, 0}, {1, 0}, {2, 0}, {3, 0}, {4, 1}, {1, 2}, {2, 2}, {3, 2}, {0, 3}, {0, 4}, {1, 4}, {2, 4}, {3, 4}, {4, 4}},
		'3':  {{0, 0}, {1, 0}, {2, 0}, {3, 0}, {4, 1}, {1, 2}, {2, 2}, {3, 2}, {4, 3}, {0, 4}, {1, 4}, {2, 4}, {3, 4}},
		'4':  {{0, 0}, {3, 0}, {0, 1}, {3, 1}, {0, 2}, {3, 2}, {0, 3}, {1, 3}, {2, 3}, {3, 3}, {4, 3}, {3, 4}},
		'5':  {{0, 0}, {1, 0}, {2, 0}, {3, 0}, {4, 0}, {0, 1}, {0, 2}, {1, 2}, {2, 2}, {3, 2}, {4, 3}, {0, 4}, {1, 4}, {2, 4}, {3, 4}},
		'6':  {{1, 0}, {2, 0}, {3, 0}, {0, 1}, {0, 2}, {1, 2}, {2, 2}, {3, 2}, {0, 3}, {4, 3}, {1, 4}, {2, 4}, {3, 4}},
		'7':  {{0, 0}, {1, 0}, {2, 0}, {3, 0}, {4, 0}, {4, 1}, {3, 2}, {2, 3}, {2, 4}},
		'8':  {{1, 0}, {2, 0}, {3, 0}, {0, 1}, {4, 1}, {1, 2}, {2, 2}, {3, 2}, {0, 3}, {4, 3}, {1, 4}, {2, 4}, {3, 4}},
		'9':  {{1, 0}, {2, 0}, {3, 0}, {0, 1}, {4, 1}, {1, 2}, {2, 2}, {3, 2}, {4, 2}, {4, 3}, {1, 4}, {2, 4}, {3, 4}},
		' ':  {},
		'!':  {{2, 0}, {2, 1}, {2, 2}, {2, 4}},
		'"':  {{0, 0}, {1, 0}, {3, 0}, {4, 0}, {0, 1}, {1, 1}, {3, 1}, {4, 1}, {1, 2}, {4, 2}, {0, 3}, {3, 3}},
		'#':  {{1, 0}, {3, 0}, {0, 1}, {1, 1}, {2, 1}, {3, 1}, {4, 1}, {1, 2}, {3, 2}, {0, 3}, {1, 3}, {2, 3}, {3, 3}, {4, 3}, {1, 4}, {3, 4}},
		'$':  {{1, 0}, {2, 0}, {3, 0}, {4, 0}, {0, 1}, {2, 1}, {0, 2}, {1, 2}, {2, 2}, {3, 2}, {2, 3}, {4, 3}, {0, 4}, {1, 4}, {2, 4}, {3, 4}},
		'%':  {{0, 0}, {1, 0}, {4, 0}, {0, 1}, {1, 1}, {3, 1}, {2, 2}, {1, 3}, {3, 3}, {4, 3}, {0, 4}, {3, 4}, {4, 4}},
		'&':  {{1, 0}, {0, 1}, {2, 1}, {1, 2}, {2, 2}, {4, 2}, {0, 3}, {3, 3}, {1, 4}, {2, 4}, {4, 4}},
		'\'': {{1, 0}, {2, 0}, {1, 1}, {2, 1}, {2, 2}, {1, 3}},
		'(':  {{2, 0}, {1, 1}, {1, 2}, {1, 3}, {2, 4}},
		')':  {{2, 0}, {3, 1}, {3, 2}, {3, 3}, {2, 4}},
		'*':  {{1, 1}, {3, 1}, {2, 2}, {1, 3}, {3, 3}},
		'+':  {{2, 1}, {1, 2}, {2, 2}, {3, 2}, {2, 3}},
		',':  {{2, 3}, {1, 4}, {2, 4}},
		'-':  {{1, 2}, {2, 2}, {3, 2}},
		'.':  {{1, 3}, {2, 3}, {1, 4}, {2, 4}},
		'/':  {{4, 0}, {3, 1}, {2, 2}, {1, 3}, {0, 4}},
		':':  {{1, 0}, {2, 0}, {1, 1}, {2, 1}, {1, 3}, {2, 3}, {1, 4}, {2, 4}},
		';':  {{1, 0}, {2, 0}, {1, 1}, {2, 1}, {2, 3}, {1, 4}, {2, 4}},
		'<':  {{3, 0}, {4, 0}, {1, 1}, {2, 1}, {0, 2}, {1, 3}, {2, 3}, {3, 4}, {4, 4}},
		'=':  {{0, 1}, {1, 1}, {2, 1}, {3, 1}, {4, 1}, {0, 3}, {1, 3}, {2, 3}, {3, 3}, {4, 3}},
		'>':  {{0, 0}, {1, 0}, {2, 1}, {3, 1}, {4, 2}, {2, 3}, {3, 3}, {0, 4}, {1, 4}},
		'?':  {{1, 0}, {2, 0}, {0, 1}, {3, 1}, {2, 2}, {2, 4}},
		'@':  {{3, 0}, {1, 1}, {4, 1}, {0, 2}, {2, 2}, {4, 2}, {0, 3}, {1, 3}, {2, 3}, {4, 3}, {0, 4}, {2, 4}, {3, 4}},
		'[':  {{1, 0}, {2, 0}, {1, 1}, {1, 2}, {1, 3}, {1, 4}, {2, 4}},
		'\\': {{0, 0}, {1, 1}, {2, 2}, {3, 3}, {4, 4}},
		']':  {{2, 0}, {3, 0}, {3, 1}, {3, 2}, {3, 3}, {2, 4}, {3, 4}},
		'^':  {{2, 0}, {1, 1}, {3, 1}, {0, 2}, {4, 2}},
		'_':  {{0, 4}, {1, 4}, {2, 4}, {3, 4}, {4, 4}},
		'`':  {{1, 0}, {2, 0}, {2, 1}},
		'{':  {{2, 0}, {1, 1}, {2, 2}, {1, 3}, {2, 4}},
		'|':  {{2, 0}, {2, 1}, {2, 2}, {2, 3}, {2, 4}},
		'}':  {{2, 0}, {3, 1}, {2, 2}, {3, 3}, {2, 4}},
		'~':  {{1, 1}, {0, 2}, {2, 2}, {4, 2}, {3, 3}},
	}
)

func NewScreen(maxX, maxY int, forceMax bool, charMap map[uint8][]byte, terminal *term.Terminal) (*Screen, error) {
	x, y, err := term.GetSize(int(os.Stdin.Fd()))
	if err != nil {
		return &Screen{}, err
	}
	x = min(int(x/2), maxX)
	y = min(y-1, maxY)

	if forceMax {
		x = maxX
		y = maxY
	}

	rows := [][]uint8{}
	for i := 0; i < y; i++ {
		rows = append(rows, make([]uint8, x))
	}

	return &Screen{
		Rows: rows,
		CurX: x - 1, CurY: y - 1, MaxX: maxX, MaxY: maxY,
		ForceMax:         forceMax,
		CharMap:          charMap,
		Terminal:         terminal,
		OnResizeCallback: func(f *Screen) {},
	}, nil
}

func (f *Screen) SetRow(y int, state uint8) error {
	if y > len(f.Rows)-1 || y < 0 {
		return ErrXOutOfBounds
	}

	r := []uint8{}
	for i := 0; i <= f.CurX; i++ {
		r = append(r, state)
	}

	f.Rows = slices.Replace(f.Rows, y, y+1, r)

	return nil
}

func (f *Screen) SetCol(x int, state uint8) error {
	if 0 > len(f.Rows)-1 {
		return ErrNoRowsFound
	}
	if x > len(f.Rows[0])-1 || x < 0 {
		return ErrXOutOfBounds
	}

	for i := range f.Rows {
		f.Rows[i] = slices.Replace(f.Rows[i], x, x+1, state)
	}

	return nil
}

func (f *Screen) SetColRow(x, y int, state uint8) error {
	if y > len(f.Rows)-1 || y < 0 {
		return ErrYOutOfBounds
	}
	if x > len(f.Rows[y])-1 || x < 0 {
		return ErrXOutOfBounds
	}

	f.Rows[y] = slices.Replace(f.Rows[y], x, x+1, state)

	return nil
}

func (f *Screen) GetColRow(x, y int) (uint8, error) {
	if y > len(f.Rows)-1 || y < 0 {
		return 0, ErrYOutOfBounds
	}
	if x > len(f.Rows[y])-1 || x < 0 {
		return 0, ErrXOutOfBounds
	}

	return f.Rows[y][x], nil
}

func (f *Screen) Clear() {
	for i := 0; i <= f.CurY; i++ {
		f.Rows[i] = make([]uint8, f.CurX+1)
	}
}

func (f *Screen) Reload() error {
	x, y, err := term.GetSize(int(os.Stdin.Fd()))
	if err != nil {
		return err
	}
	f.CurX = min(int(x/2)-1, f.MaxX)
	f.CurY = min(y-2, f.MaxY)

	if f.ForceMax {
		f.CurX = f.MaxX
		f.CurY = f.MaxY
	}

	f.Rows = [][]uint8{}
	for i := 0; i <= f.CurY; i++ {
		f.Rows = append(f.Rows, make([]uint8, f.CurX+1))
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

			if f.ForceMax {
				f.CurX = f.MaxX
				f.CurY = f.MaxY
			}

			f.Rows = [][]uint8{}
			for i := 0; i <= f.CurY; i++ {
				f.Rows = append(f.Rows, make([]uint8, f.CurX+1))
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

			char, ok = f.CharMap[0]
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

func (f *Screen) RenderString(str string, offsetX, offsetY int, state uint8) {
	for _, r := range str {
		cords, ok := CharMap[unicode.ToUpper(r)]
		if !ok {
			continue
		}

		f.RenderCords(cords, offsetX, offsetY, state)
		offsetX += 6
	}
}

func (f *Screen) RenderStringIf(str string, offsetX, offsetY int, state uint8, set func(uint8) bool) {
	for _, r := range str {
		cords, ok := CharMap[unicode.ToUpper(r)]
		if !ok {
			continue
		}

		f.RenderCordsIf(cords, offsetX, offsetY, state, set)
		offsetX += 6
	}
}

func (f *Screen) RenderCords(cords []Cord, offsetX, offsetY int, state uint8) {
	for _, cord := range cords {
		_ = f.SetColRow(cord.X+offsetX, cord.Y+offsetY, state)
	}
}

func (f *Screen) RenderCordsIf(cords []Cord, offsetX, offsetY int, state uint8, set func(uint8) bool) {
	for _, cord := range cords {
		val, err := f.GetColRow(cord.X+offsetX, cord.Y+offsetY)
		if err != nil {
			continue
		}
		if set(val) {
			_ = f.SetColRow(cord.X+offsetX, cord.Y+offsetY, state)
		}
	}
}
