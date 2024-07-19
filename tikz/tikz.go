package tikz

import (
	"figz/fig"
	"fmt"
	"github.com/heyvito/carrows"
	"strings"
)

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

const scale = float32(0.018)

func MakeDrawingNode(v *fig.NodeChange) DrawingNode {

	var (
		m00 = float32(v.Transform.M00)
		m01 = float32(v.Transform.M01)
		m02 = float32(v.Transform.M02) * scale
		m10 = float32(v.Transform.M10)
		m11 = float32(v.Transform.M11)
		m12 = float32(v.Transform.M12) * scale
	)

	var p2 = Position{float32(v.Size.X) * scale, float32(v.Size.Y) * scale}
	var q1 = Position{m00*0 + m01*0 + m02, m10*0 + m11*0 + m12}
	var q2 = Position{m00*p2.X + m01*p2.Y + m02, m10*p2.X + m11*p2.Y + m12}

	return DrawingNode{
		Node: v,
		Q1:   q1,
		Q2:   q2,
		Size: p2,
	}
}

func NewCompiler(page *fig.NodeChange) string {
	nodes := make([]DrawingNode, len(page.Children))
	nodeMap := make(map[string]DrawingNode)
	for i, v := range page.Children {
		nodes[i] = MakeDrawingNode(v)
		nodeMap[fmt.Sprintf("%d:%d", v.Guid.SessionId, v.Guid.LocalId)] = nodes[i]
	}
	c := &Compiler{
		page:    page,
		nodes:   nodes,
		nodeMap: nodeMap,
	}
	return c.ConvertPageToTikz()
}

type Compiler struct {
	b       sbuf
	nodes   []DrawingNode
	nodeMap map[string]DrawingNode
	page    *fig.NodeChange
}

func (c *Compiler) FindNode(g *fig.GUID) DrawingNode {
	return c.nodeMap[fmt.Sprintf("%d:%d", g.SessionId, g.LocalId)]
}

type sbuf struct {
	strings.Builder
}

func (s *sbuf) Writef(format string, args ...any) {
	s.WriteString(fmt.Sprintf(format, args...) + "\n")
}

func (c *Compiler) IsArrowDiagonal(start, end Position) bool {
	return !(start.X == end.X || start.Y == end.Y)
}

func (c *Compiler) PositionForNode(node DrawingNode, magnet fig.ConnectorMagnet) (pos Position) {
	toSize := node.Q2.Diff(node.Q1)
	switch magnet {
	case fig.ConnectorMagnetNone, fig.ConnectorMagnetAutoHorizontal,
		fig.ConnectorMagnetAuto, fig.ConnectorMagnetCenter, fig.ConnectorMagnetTop:
		pos = Position{
			X: node.Q2.X - toSize.X/2.0,
			Y: node.Q1.Y - 0.1,
		}
	case fig.ConnectorMagnetLeft:
		pos = Position{
			X: node.Q1.X - 0.1,
			Y: node.Q2.Y - toSize.Y/2.0,
		}
	case fig.ConnectorMagnetBottom:
		pos = Position{
			X: node.Q2.X - toSize.X/2.0,
			Y: node.Q2.Y + 0.1,
		}
	case fig.ConnectorMagnetRight:
		pos = Position{
			X: node.Q2.X + 0.1,
			Y: node.Q2.Y - toSize.Y/2.0,
		}
	}

	return
}

func (c *Compiler) MagnetToCarrow(magnet fig.ConnectorMagnet) carrows.RectSide {
	switch magnet {
	case fig.ConnectorMagnetNone, fig.ConnectorMagnetAutoHorizontal,
		fig.ConnectorMagnetAuto, fig.ConnectorMagnetCenter, fig.ConnectorMagnetTop:
		return carrows.Top
	case fig.ConnectorMagnetLeft:
		return carrows.Left
	case fig.ConnectorMagnetBottom:
		return carrows.Bottom
	case fig.ConnectorMagnetRight:
		return carrows.Right
	default:
		panic("unreachable")
	}
}

var weirdThings = map[string]struct{}{
	"Connector Name":  {},
	"Shape with text": {},
}

func (c *Compiler) CleanupText(t string) string {
	if _, ok := weirdThings[t]; ok {
		return ""
	}
	return t
}

func (c *Compiler) ConvertPageToTikz() string {
	b := sbuf{}
	b.Writef("\\begin{tikzpicture}[yscale=-1]")
	for _, v := range c.page.Children {
		w := MakeDrawingNode(v)
		switch v.Type {
		case fig.NodeTypeText:
			c.drawText(&b, v)
		case fig.NodeTypeShapeWithText:
			c.drawShapeWithText(&b, w)

		case fig.NodeTypeConnector:
			nodeFrom := c.FindNode(v.ConnectorStart.EndpointNodeId)
			magnetFrom := v.ConnectorStart.Magnet
			nodeTo := c.FindNode(v.ConnectorEnd.EndpointNodeId)
			magnetTo := v.ConnectorEnd.Magnet
			var (
				positionStart = c.PositionForNode(nodeFrom, magnetFrom)
				positionEnd   = c.PositionForNode(nodeTo, magnetTo)
			)

			//b.Writef(`\filldraw[color=red] (%s) circle (3pt);`, positionStart)
			//b.Writef(`\filldraw[color=red] (%s) circle (3pt);`, positionEnd)

			var points []string

			//connectorText := c.CleanupText(v.Name)Vai me
			if v.ConnectorControlPoints != nil {
				//for _, con := range v.ConnectorControlPoints {
				//	pos := Position{
				//		X: float32(con.Position.X) * scale,
				//		Y: float32(con.Position.Y) * scale,
				//	}
				//	b.Writef(`\filldraw[color=blue] (%s) circle (3pt);`, pos)
				//}
				curPos := positionStart
				points = append(points, fmt.Sprintf("(%s)", positionStart))

				for _, con := range v.ConnectorControlPoints {
					var newPos Position
					if con.Axis.X == 1 {
						newPos = Position{
							X: curPos.X,
							Y: float32(con.Position.Y) * scale,
						}
					} else {
						newPos = Position{
							X: float32(con.Position.X) * scale,
							Y: curPos.Y,
						}
					}
					points = append(points, fmt.Sprintf(`(%s)`, newPos))
					curPos = newPos
				}
				rawPos := v.ConnectorControlPoints[len(v.ConnectorControlPoints)-1].Position
				lastPos := Position{float32(rawPos.X), float32(rawPos.Y)}
				lastPos.X *= scale
				lastPos.Y *= scale
				points = append(points, fmt.Sprintf(`(%s)`, lastPos))
				allPoints := strings.Join(points, " -- ")
				b.Writef(`\draw [thick, rounded corners] %s;`, allPoints)
				straight := c.isStraight(curPos, lastPos, magnetTo)

				if straight {
					b.Writef(`\draw[-To, thick] (%s) -- (%s);`, lastPos, positionEnd)
				} else {
					midPath := Position{
						X: positionEnd.X,
						Y: lastPos.Y,
					}
					finalPosition := positionEnd
					finalPosition.Y += 0.3
					b.Writef(`\draw [thick, rounded corners] (%s) -- (%s) -- (%s); %% midpoint`, lastPos, midPath, finalPosition)
					b.Writef(`\draw[-To, thick] (%s) -- (%s);`, finalPosition, positionEnd)
				}

			} else if c.IsArrowDiagonal(positionStart, positionEnd) {
				arr := carrows.GetArrow(float64(positionStart.X), float64(positionStart.Y), float64(positionEnd.X), float64(positionEnd.Y), &carrows.Opts{
					ControlPointStretch: 1.8,
					AllowedStartSides:   []carrows.RectSide{c.MagnetToCarrow(magnetFrom)},
					AllowedEndSides:     []carrows.RectSide{c.MagnetToCarrow(magnetTo)},
				})
				b.Writef(`\draw[-To, thick] (%f, %f) .. controls (%f,%f) and (%f,%f) .. (%f, %f);`,
					arr.Sx, arr.Sy, arr.C1x, arr.C1y, arr.C2x, arr.C2y, arr.Ex, arr.Ey)

			} else {
				b.Writef(`\draw[-To, thick] (%s) -- (%s);`, positionStart, positionEnd)
			}
		}
	}
	b.Writef(`\end{tikzpicture}` + "\n")

	return b.String()
}

func (c *Compiler) drawShapeWithText(b *sbuf, w DrawingNode) {
	text := c.CleanupText(w.Node.Name)

	switch w.Node.ShapeWithTextType {
	case fig.ShapeWithTextTypeSquare, fig.ShapeWithTextTypePredefinedProcess:
		b.Writef(`\draw (%s) rectangle node{%s} (%s);`, w.Q1, text, w.Q2)

	case fig.ShapeWithTextTypeEllipse:
	case fig.ShapeWithTextTypeDiamond:
		pos := w.Q1.MiddleWith(w.Q2)
		b.Writef(`\draw[rotate around={45:(%s)}, scale around={0.75:(%s)}] (%s) rectangle node{%s} (%s);`, pos, pos, w.Q1, text, w.Q2)
	case fig.ShapeWithTextTypeTriangleUp:
	case fig.ShapeWithTextTypeTriangleDown:
	case fig.ShapeWithTextTypeRoundedRectangle:
	case fig.ShapeWithTextTypeParallelogramRight:
	case fig.ShapeWithTextTypeParallelogramLeft:
	case fig.ShapeWithTextTypeEngDatabase:
	case fig.ShapeWithTextTypeEngQueue:
	case fig.ShapeWithTextTypeEngFile:
	case fig.ShapeWithTextTypeEngFolder:
	case fig.ShapeWithTextTypeTrapezoid:
	case fig.ShapeWithTextTypeShield:
	case fig.ShapeWithTextTypeDocumentSingle:
	case fig.ShapeWithTextTypeDocumentMultiple:
	case fig.ShapeWithTextTypeManualInput:
	case fig.ShapeWithTextTypeHexagon:
	case fig.ShapeWithTextTypeChevron:
	case fig.ShapeWithTextTypePentagon:
	case fig.ShapeWithTextTypeOctagon:
	case fig.ShapeWithTextTypeStar:
	case fig.ShapeWithTextTypePlus:
	case fig.ShapeWithTextTypeArrowLeft:
	case fig.ShapeWithTextTypeArrowRight:
	case fig.ShapeWithTextTypeSummingJunction:
	case fig.ShapeWithTextTypeOr:
	case fig.ShapeWithTextTypeSpeechBubble:
	case fig.ShapeWithTextTypeInternalStorage:
	}
}

func (c *Compiler) isStraight(start, lastPos Position, magnet fig.ConnectorMagnet) (straight bool) {
	mag := directionFromMagnet(magnet)
	src := lastPos.DirectionTo(start)
	switch {
	case mag == BottomDirection && src == TopDirection,
		mag == TopDirection && src == BottomDirection,
		mag == LeftDirection && src == RightDirection,
		mag == RightDirection && src == LeftDirection:
		// Already aligned, just go straight
		straight = true
	case mag == BottomDirection && src == LeftDirection,
		mag == TopDirection && src == LeftDirection:
	case mag == BottomDirection && src == RightDirection,
		mag == TopDirection && src == RightDirection:
	case mag == LeftDirection && src == TopDirection,
		mag == RightDirection && src == TopDirection:
	case mag == LeftDirection && src == BottomDirection,
		mag == RightDirection && src == BottomDirection:
	case mag == BottomDirection && src == BottomDirection:
		// TODO: This is a problem
	case mag == TopDirection && src == TopDirection:
		// TODO: This is a problem
	case mag == LeftDirection && src == LeftDirection:
		// TODO: This is a problem
	case mag == RightDirection && src == RightDirection:
		// TODO: This is a problem
	default:
		panic("unreachable")
	}
	return
}

func (c *Compiler) drawText(s *sbuf, v *fig.NodeChange) {

}

func directionFromMagnet(mag fig.ConnectorMagnet) Direction {
	switch mag {
	case fig.ConnectorMagnetNone, fig.ConnectorMagnetAutoHorizontal,
		fig.ConnectorMagnetAuto, fig.ConnectorMagnetCenter, fig.ConnectorMagnetTop:
		return TopDirection
	case fig.ConnectorMagnetLeft:
		return LeftDirection
	case fig.ConnectorMagnetBottom:
		return BottomDirection
	case fig.ConnectorMagnetRight:
		return RightDirection
	default:
		panic("unreachable")
	}
}
