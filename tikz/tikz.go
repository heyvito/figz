package tikz

import (
	"figz/fig"
	"fmt"
	"math"
	"slices"
	"strings"
)

const scale = float32(0.018)

type CompilerOpts struct {
	DebugMagnets       bool
	DebugControlPoints bool
}

func NewCompiler(page *fig.NodeChange, opts *CompilerOpts) string {
	if opts == nil {
		opts = &CompilerOpts{}
	}
	nodes := make([]DrawingNode, len(page.Children))
	nodeMap := make(map[string]DrawingNode)
	for i, v := range page.Children {
		nodes[i] = MakeDrawingNode(v)
		nodeMap[fmt.Sprintf("%d:%d", v.Guid.SessionId, v.Guid.LocalId)] = nodes[i]
	}
	c := &Compiler{
		b:       &sbuf{},
		opts:    opts,
		page:    page,
		nodes:   nodes,
		nodeMap: nodeMap,
	}
	return c.ConvertPageToTikz()
}

type Compiler struct {
	b        *sbuf
	nodes    []DrawingNode
	nodeMap  map[string]DrawingNode
	page     *fig.NodeChange
	opts     *CompilerOpts
	elements []fmt.Stringer
}

func (c *Compiler) FindNode(g *fig.GUID) DrawingNode {
	return c.nodeMap[fmt.Sprintf("%d:%d", g.SessionId, g.LocalId)]
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

func (c *Compiler) MagnetToDirection(magnet fig.ConnectorMagnet) Direction {
	switch magnet {
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

var weirdThings = map[string]struct{}{
	"Connector Name":  {},
	"Shape with text": {},
	"Connector line":  {},
}

func (c *Compiler) CleanupText(t string) string {
	if _, ok := weirdThings[t]; ok {
		return ""
	}
	return t
}

func (c *Compiler) ConvertPageToTikz() string {
	c.b.Writef("\\begin{tikzpicture}[yscale=-1]")
	for _, v := range c.page.Children {
		w := MakeDrawingNode(v)
		switch v.Type {
		case fig.NodeTypeText:
			c.drawText(w)
		case fig.NodeTypeShapeWithText:
			c.drawShapeWithText(w)
		case fig.NodeTypeConnector:
			c.drawArrow(w)
		}
	}

	minX := c.findMinX()

	for _, v := range c.elements {
		v.(XAdjuster).AdjustX(minX)
		c.b.Writef(v.String())
	}
	c.b.Writef(`\end{tikzpicture}` + "\n")

	return c.b.String()
}

func (c *Compiler) AddElement(el fmt.Stringer) {
	c.elements = append(c.elements, el)
}

func (c *Compiler) drawArrow(w DrawingNode) {
	var (
		v             = w.Node
		nodeFrom      = c.FindNode(v.ConnectorStart.EndpointNodeId)
		magnetFrom    = v.ConnectorStart.Magnet
		nodeTo        = c.FindNode(v.ConnectorEnd.EndpointNodeId)
		magnetTo      = v.ConnectorEnd.Magnet
		connectorText = c.CleanupText(v.Name)
		positionStart = c.PositionForNode(nodeFrom, magnetFrom)
		positionEnd   = c.PositionForNode(nodeTo, magnetTo)
	)

	if c.opts.DebugMagnets {
		c.AddElement(&FillDraw{
			Attributes: AttributeList{ColorAttribute("red")},
			Position:   positionStart,
			Shape:      "circle",
			Size:       "3pt",
		})
		c.AddElement(&FillDraw{
			Attributes: AttributeList{ColorAttribute("red")},
			Position:   positionEnd,
			Shape:      "circle",
			Size:       "3pt",
		})
	}

	var points []Position

	if v.ConnectorControlPoints != nil {
		if c.opts.DebugControlPoints {
			for _, con := range v.ConnectorControlPoints {
				pos := Position{
					X: float32(con.Position.X) * scale,
					Y: float32(con.Position.Y) * scale,
				}
				c.AddElement(&FillDraw{
					Attributes: AttributeList{ColorAttribute("red")},
					Position:   pos,
					Shape:      "blue",
					Size:       "3pt",
				})
			}
		}
		curPos := positionStart
		points = append(points, positionStart)
		completePath := []Position{positionStart}

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
			completePath = append(completePath, newPos)
			points = append(points, newPos)
			curPos = newPos
		}
		rawPos := v.ConnectorControlPoints[len(v.ConnectorControlPoints)-1].Position
		lastPos := Position{X: float32(rawPos.X), Y: float32(rawPos.Y)}
		lastPos.X *= scale
		lastPos.Y *= scale
		points = append(points, lastPos)
		c.AddElement(&Draw{
			Attributes: AttributeList{&ThickAttribute{}, &RoundedCornersAttribute{10}},
			Points:     points,
			Text:       nil,
			Kind:       nil,
		})
		straight := c.isStraight(curPos, lastPos, magnetTo)

		completePath = append(completePath, lastPos)

		if straight {
			completePath = append(completePath, positionEnd)
			c.AddElement(&Draw{
				Attributes: AttributeList{&ToAttribute{}, &ThickAttribute{}},
				Points:     []Position{lastPos, positionEnd},
				Text:       nil,
				Kind:       nil,
			})
		} else {
			midPath := Position{
				X: positionEnd.X,
				Y: lastPos.Y,
			}
			finalPosition := positionEnd
			if lastPos.Y <= finalPosition.Y {
				finalPosition.Y -= 0.3
			} else {
				finalPosition.Y += 0.3
			}
			completePath = append(completePath, midPath, positionEnd)
			c.AddElement(&Draw{
				Attributes: AttributeList{&ThickAttribute{}, &RoundedCornersAttribute{10}},
				Points:     []Position{lastPos, midPath, finalPosition},
				Text:       nil,
				Kind:       nil,
			})
			c.AddElement(&Draw{
				Attributes: AttributeList{&ToAttribute{}, &ThickAttribute{}},
				Points:     []Position{finalPosition, positionEnd},
				Text:       nil,
				Kind:       nil,
			})
		}

		if connectorText != "" {
			c.drawArrowTextComplex(completePath, connectorText, v.ConnectorTextMidpoint)
		}

	} else if c.IsArrowDiagonal(positionStart, positionEnd) {
		midPoint := float32(math.Abs(float64(positionStart.X-positionEnd.X))) / 2.0

		var x1, x2 float32
		switch positionStart.DirectionTo(positionEnd) {
		case RightDirection:
			x1 = positionStart.X + midPoint
			x2 = positionEnd.X - midPoint
		case LeftDirection:
			x1 = positionStart.X - midPoint
			x2 = positionEnd.X + midPoint
		case TopDirection:
		case BottomDirection:
		}
		cp1 := Position{
			Y: positionStart.Y,
			X: x1,
		}

		cp2 := Position{
			Y: positionEnd.Y,
			X: x2,
		}

		c.AddElement(&Draw{
			Attributes: AttributeList{&ToAttribute{}, &ThickAttribute{}, &RoundedCornersAttribute{10}},
			Points:     []Position{positionStart, cp1, cp2, positionEnd},
		})
	} else {
		c.AddElement(&Draw{
			Attributes: AttributeList{&ToAttribute{}, &ThickAttribute{}},
			Points:     []Position{positionStart, positionEnd},
		})

		if len(connectorText) != 0 {
			midPoint := v.ConnectorTextMidpoint
			c.drawArrowTextStraight(positionStart, positionEnd, connectorText, midPoint)
		}
	}
}

func (c *Compiler) findMinX() float32 {
	minX := float32(math.MaxFloat32)
	for _, v := range c.elements {
		switch t := v.(type) {
		case *Draw:
			hasMinAttr := t.Attributes != nil
			minAttrX := float32(math.MaxFloat32)
			if hasMinAttr {
				minAttrX = t.Attributes.MinX()
			}
			theMin := float32(math.MaxFloat32)
			for _, p := range t.Points {
				if p.X < theMin {
					theMin = p.X
				}
			}
			theMin = min(minAttrX, theMin)
			if theMin < minX {
				minX = theMin
			}
		case *Shape:
			hasMinAttr := t.Attributes != nil
			minAttrX := float32(math.MaxFloat32)
			if hasMinAttr {
				minAttrX = t.Attributes.MinX()
			}
			theMin := min(minAttrX, t.P1.X)
			if theMin < minX {
				minX = theMin
			}
		case *Node:
			hasMinAttr := t.Attributes != nil
			minAttrX := float32(math.MaxFloat32)
			if hasMinAttr {
				minAttrX = t.Attributes.MinX()
			}
			theMin := min(minAttrX, t.Position.X)
			if theMin < minX {
				minX = theMin
			}
		case *FillDraw:
			hasMinAttr := t.Attributes != nil
			minAttrX := float32(math.MaxFloat32)
			if hasMinAttr {
				minAttrX = t.Attributes.MinX()
			}
			theMin := min(minAttrX, t.Position.X)
			if theMin < minX {
				minX = theMin
			}
		}
	}
	return minX
}

func (c *Compiler) drawShapeWithText(w DrawingNode) {
	text := c.CleanupText(w.Node.Name)

	switch w.Node.ShapeWithTextType {
	case fig.ShapeWithTextTypeSquare, fig.ShapeWithTextTypePredefinedProcess:
		c.AddElement(&Shape{
			P1:   w.Q1,
			P2:   w.Q2,
			Text: &text,
			Kind: "rectangle",
		})

	case fig.ShapeWithTextTypeEllipse:
	case fig.ShapeWithTextTypeDiamond:
		pos := w.Q1.MiddleWith(w.Q2)
		c.AddElement(&Shape{
			Attributes: AttributeList{&RotateAroundAttribute{45, pos}, &ScaleAroundAttribute{0.75, pos}},
			P1:         w.Q1,
			P2:         w.Q2,
			Text:       &text,
			Kind:       "rectangle",
		})
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

func (c *Compiler) drawText(v DrawingNode) {
	point := v.Q1
	point.X += (v.Q2.Sub(v.Q1)).X/2.0 - 0.55
	point.Y += (v.Q2.Sub(v.Q1)).Y / 2.0
	text := strings.ReplaceAll(v.Node.Name, "_", "\\_")
	c.AddElement(&Node{
		Position: point,
		Text:     &text,
	})
}

func (c *Compiler) drawArrowTextStraight(startPos Position, endPos Position, text string, midPoint *fig.ConnectorTextMidpoint) {
	dir := startPos.DirectionTo(endPos)
	pos := Position{}
	var start, end float32

	isVertical := dir == TopDirection || dir == BottomDirection

	if isVertical {
		start, end = startPos.Y, endPos.Y
		pos.X = startPos.X
	} else {
		start, end = startPos.X, endPos.X
		pos.Y = startPos.Y
	}

	length := float32(math.Abs(float64(end-start) / 2.0))
	fPos := end - length

	if midPoint.Section == fig.ConnectorTextSectionMiddleToEnd {
		fPos += length * float32(midPoint.Offset)
	} else {
		fPos -= length * float32(midPoint.Offset)
	}

	if isVertical {
		pos.Y = fPos
	} else {
		pos.X = fPos
	}

	c.AddElement(&Node{
		Attributes: AttributeList{DrawAttribute("none"), &FillAttribute{"white"}},
		Position:   pos,
		Text:       &text,
	})
}

func (c *Compiler) drawArrowTextComplex(path []Position, text string, connectorMid *fig.ConnectorTextMidpoint) {
	totalLength := float32(0)
	for i := 1; i < len(path); i++ {
		last := path[i-1]
		curr := path[i]
		var l float32
		if last.Y == curr.Y {
			l = float32(math.Abs(float64(curr.X - last.X)))
		} else {
			l = float32(math.Abs(float64(curr.Y - last.Y)))
		}
		totalLength += l
	}

	middlePoint := totalLength / 2.0
	offset := middlePoint * float32(math.Abs(connectorMid.Offset-1.0))
	if connectorMid.Section == fig.ConnectorTextSectionMiddleToEnd {
		slices.Reverse(path)
	}

	textPos := Position{}
	isVertical := false
	for i := 1; i < len(path); i++ {
		prev := path[i-1]
		current := path[i]
		var distance float32
		if prev.X == current.X {
			distance = float32(math.Abs(float64(current.Y - prev.Y)))
			textPos.X = current.X
			isVertical = true

			if current.Y > prev.Y {
				textPos.Y = prev.Y + offset
			} else {
				textPos.Y = prev.Y - offset
			}

		} else {
			distance = float32(math.Abs(float64(current.X - prev.X)))
			textPos.Y = current.Y
			isVertical = false

			if current.X > prev.X {
				textPos.X = prev.X + offset
			} else {
				textPos.X = prev.X - offset
			}
		}

		if distance > offset {
			break
		}

		offset = offset - distance
	}

	attrs := AttributeList{DrawAttribute("none"), &FillAttribute{"white"}}
	if isVertical {
		attrs = append(attrs, AnchorAttribute("south"))
	}

	c.AddElement(&Node{
		Attributes: attrs,
		Position:   textPos,
		Text:       &text,
	})

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
