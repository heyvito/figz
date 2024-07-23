package tikz

import (
	"fmt"
	"strings"
)

type Position struct {
	X, Y float32
}

func (p Position) Sum(p2 Position) Position {
	return Position{X: p.X + p2.X, Y: p.Y + p2.Y}
}

func (p Position) MiddleWith(p2 Position) Position {
	pos := p.Sum(p2)
	pos.X /= 2.0
	pos.Y /= 2.0
	return pos
}

type Axis int

const (
	AxisX Axis = iota
	AxisY
)

func (p Position) AxisRelativeTo(p2 Position) Axis {
	if p.X == p2.X {
		return AxisY
	}

	return AxisX
}

func (p Position) String() string {
	return fmt.Sprintf("%f, %f", p.X, p.Y)
}

func (p Position) Diff(of Position) Position {
	return Position{X: p.X - of.X, Y: p.Y - of.Y}
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
	return Position{X: p.X - pos.X, Y: p.Y - pos.Y}
}

type PositionList []Position

func (p PositionList) String() string {
	positions := make([]string, len(p))
	for i, pos := range p {
		positions[i] = "(" + pos.String() + ")"
	}
	return strings.Join(positions, " -- ")
}
