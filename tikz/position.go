package tikz

import "fmt"

type Position struct {
	X, Y float32
}

func (p Position) Sum(p2 Position) Position {
	return Position{p.X + p2.X, p.Y + p2.Y}
}

func (p Position) MiddleWith(p2 Position) Position {
	pos := p.Sum(p2)
	pos.X /= 2.0
	pos.Y /= 2.0
	return pos
}

func (p Position) String() string {
	return fmt.Sprintf("%f, %f", p.X, p.Y)
}

func (p Position) Diff(of Position) Position {
	return Position{p.X - of.X, p.Y - of.Y}
}

func (p Position) DirectionTo(pos Position) Direction {
	switch {
	case pos.X > p.X:
		return RightDirection
	case pos.X < p.X:
		return LeftDirection
	case pos.Y > p.Y:
		return BottomDirection
	case pos.Y < p.Y:
		return TopDirection
	case pos.Y == p.Y:
		return TopDirection
	case pos.X == p.X:
		return LeftDirection // ?
	}

	panic("unreachable")
}

func (p Position) Sub(pos Position) Position {
	return Position{p.X - pos.X, p.Y - pos.Y}
}
