package main

// t := []Cord{
// 	{X: 0, Y: 0}, {X: 1, Y: 0}, {X: 2, Y: 0}, {X: 3, Y: 0}, {X: 4, Y: 0},
// 	{X: 0, Y: 1}, {X: 1, Y: 1}, {X: 2, Y: 1}, {X: 3, Y: 1}, {X: 4, Y: 1},
// 	{X: 0, Y: 2}, {X: 1, Y: 2}, {X: 2, Y: 2}, {X: 3, Y: 2}, {X: 4, Y: 2},
// 	{X: 0, Y: 3}, {X: 1, Y: 3}, {X: 2, Y: 3}, {X: 3, Y: 3}, {X: 4, Y: 3},
// 	{X: 0, Y: 4}, {X: 1, Y: 4}, {X: 2, Y: 4}, {X: 3, Y: 4}, {X: 4, Y: 4},
// }

func failScreen(f *Frame) {
	g := []Cord{
		{X: 0, Y: 0}, {X: 1, Y: 0}, {X: 2, Y: 0}, {X: 3, Y: 0}, {X: 4, Y: 0},
		{X: 0, Y: 1},
		{X: 0, Y: 2}, {X: 3, Y: 2}, {X: 4, Y: 2},
		{X: 0, Y: 3}, {X: 4, Y: 3},
		{X: 0, Y: 4}, {X: 1, Y: 4}, {X: 2, Y: 4}, {X: 3, Y: 4}, {X: 4, Y: 4},
	}
	for _, cord := range g {
		f.SetColRow(cord.X+2, cord.Y+2, GameObjects.GameOver)
	}

	a := []Cord{
		{X: 2, Y: 0},
		{X: 1, Y: 1}, {X: 3, Y: 1},
		{X: 0, Y: 2}, {X: 4, Y: 2},
		{X: 0, Y: 3}, {X: 1, Y: 3}, {X: 2, Y: 3}, {X: 3, Y: 3}, {X: 4, Y: 3},
		{X: 0, Y: 4}, {X: 4, Y: 4},
	}
	for _, cord := range a {
		f.SetColRow(cord.X+8, cord.Y+2, GameObjects.GameOver)
	}

	m := []Cord{
		{X: 0, Y: 0}, {X: 4, Y: 0},
		{X: 0, Y: 1}, {X: 1, Y: 1}, {X: 3, Y: 1}, {X: 4, Y: 1},
		{X: 0, Y: 2}, {X: 2, Y: 2}, {X: 4, Y: 2},
		{X: 0, Y: 3}, {X: 4, Y: 3},
		{X: 0, Y: 4}, {X: 4, Y: 4},
	}
	for _, cord := range m {
		f.SetColRow(cord.X+14, cord.Y+2, GameObjects.GameOver)
	}

	e := []Cord{
		{X: 0, Y: 0}, {X: 1, Y: 0}, {X: 2, Y: 0}, {X: 3, Y: 0}, {X: 4, Y: 0},
		{X: 0, Y: 1},
		{X: 0, Y: 2}, {X: 1, Y: 2}, {X: 2, Y: 2}, {X: 3, Y: 2}, {X: 4, Y: 2},
		{X: 0, Y: 3},
		{X: 0, Y: 4}, {X: 1, Y: 4}, {X: 2, Y: 4}, {X: 3, Y: 4}, {X: 4, Y: 4},
	}
	for _, cord := range e {
		f.SetColRow(cord.X+20, cord.Y+2, GameObjects.GameOver)
	}

	o := []Cord{
		{X: 1, Y: 0}, {X: 2, Y: 0}, {X: 3, Y: 0},
		{X: 0, Y: 1}, {X: 4, Y: 1},
		{X: 0, Y: 2}, {X: 4, Y: 2},
		{X: 0, Y: 3}, {X: 4, Y: 3},
		{X: 1, Y: 4}, {X: 2, Y: 4}, {X: 3, Y: 4},
	}
	for _, cord := range o {
		f.SetColRow(cord.X+8, cord.Y+8, GameObjects.GameOver)
	}

	v := []Cord{
		{X: 0, Y: 0}, {X: 4, Y: 0},
		{X: 0, Y: 1}, {X: 4, Y: 1},
		{X: 0, Y: 2}, {X: 4, Y: 2},
		{X: 1, Y: 3}, {X: 3, Y: 3},
		{X: 2, Y: 4},
	}
	for _, cord := range v {
		f.SetColRow(cord.X+14, cord.Y+8, GameObjects.GameOver)
	}

	for _, cord := range e {
		f.SetColRow(cord.X+20, cord.Y+8, GameObjects.GameOver)
	}

	r := []Cord{
		{X: 0, Y: 0}, {X: 1, Y: 0}, {X: 2, Y: 0}, {X: 3, Y: 0},
		{X: 0, Y: 1}, {X: 4, Y: 1},
		{X: 0, Y: 2}, {X: 1, Y: 2}, {X: 2, Y: 2}, {X: 3, Y: 2},
		{X: 0, Y: 3}, {X: 3, Y: 3},
		{X: 0, Y: 4}, {X: 4, Y: 4},
	}
	for _, cord := range r {
		f.SetColRow(cord.X+26, cord.Y+8, GameObjects.GameOver)
	}
}

func plusOneScreen(f *Frame) {
	state := GameObjects.PlusOne
	if !plusOneActive {
		state = GameObjects.Empty
	}

	plus := []Cord{
		{X: 2, Y: 1},
		{X: 1, Y: 2}, {X: 2, Y: 2}, {X: 3, Y: 2},
		{X: 2, Y: 3},
	}
	for _, cord := range plus {
		val, err := f.GetColRow(cord.X+2, cord.Y+2)
		if err == nil && (val == GameObjects.Empty || val == GameObjects.PlusOne) {
			f.SetColRow(cord.X+2, cord.Y+2, state)
		}
	}

	one := []Cord{
		{X: 2, Y: 0},
		{X: 1, Y: 1}, {X: 2, Y: 1},
		{X: 0, Y: 2}, {X: 2, Y: 2},
		{X: 2, Y: 3},
		{X: 0, Y: 4}, {X: 1, Y: 4}, {X: 2, Y: 4}, {X: 3, Y: 4}, {X: 4, Y: 4},
	}
	for _, cord := range one {
		val, err := f.GetColRow(cord.X+8, cord.Y+2)
		if err == nil && (val == GameObjects.Empty || val == GameObjects.PlusOne) {
			f.SetColRow(cord.X+8, cord.Y+2, state)
		}
	}
}
