package tikz

import "github.com/heyvito/figz/fig"

type DrawingNode struct {
	Node *fig.NodeChange
	Q1   Position
	Q2   Position
	Size Position
}

type Direction int

const (
	TopDirection Direction = iota
	RightDirection
	BottomDirection
	LeftDirection
)

type Side int

func (s Side) Is(o Side) bool { return s&o == o }

const (
	LeftSide Side = 1 << iota
	RightSide
	TopSide
	BottomSide
	HorizontallyEqual
	VerticallyEqual
)

func (d DrawingNode) RelativeSideOf(other DrawingNode) (s Side) {
	if d.Q2.X > other.Q2.X {
		s |= RightSide
	} else if d.Q2.X < other.Q2.X {
		s |= LeftSide
	} else {
		s |= HorizontallyEqual
	}

	if d.Q2.Y > other.Q2.Y {
		s |= BottomSide
	} else if d.Q2.Y < other.Q2.Y {
		s |= TopSide
	} else {
		s |= VerticallyEqual
	}
	return
}

func MakeDrawingNode(v *fig.NodeChange) DrawingNode {
	var (
		m00 = float32(v.Transform.M00)
		m01 = float32(v.Transform.M01)
		m02 = float32(v.Transform.M02) * scale
		m10 = float32(v.Transform.M10)
		m11 = float32(v.Transform.M11)
		m12 = float32(v.Transform.M12) * scale
	)

	var p2 = Position{X: float32(v.Size.X) * scale, Y: float32(v.Size.Y) * scale}
	var q1 = Position{X: m00*0 + m01*0 + m02, Y: m10*0 + m11*0 + m12}
	var q2 = Position{X: m00*p2.X + m01*p2.Y + m02, Y: m10*p2.X + m11*p2.Y + m12}

	return DrawingNode{
		Node: v,
		Q1:   q1,
		Q2:   q2,
		Size: p2,
	}
}
