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
	b       *sbuf
	nodes   []DrawingNode
	nodeMap map[string]DrawingNode
	page    *fig.NodeChange
	opts    *CompilerOpts
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
	c.b.Writef(`\end{tikzpicture}` + "\n")

	return c.b.String()
}

func (c *Compiler) AddElement(el any) {}

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
		c.AddElement(FillDraw{
			Attributes: AttributeList{ColorAttribute("red")},
			Position:   positionStart,
			Shape:      "circle",
			Size:       "3pt",
		})
		c.AddElement(FillDraw{
			Attributes: AttributeList{ColorAttribute("red")},
			Position:   positionEnd,
			Shape:      "circle",
			Size:       "3pt",
		})
	}

	var points []string

	if v.ConnectorControlPoints != nil {
		if c.opts.DebugControlPoints {
			for _, con := range v.ConnectorControlPoints {
				pos := Position{
					X: float32(con.Position.X) * scale,
					Y: float32(con.Position.Y) * scale,
				}
				c.AddElement(FillDraw{
					Attributes: AttributeList{ColorAttribute("red")},
					Position:   pos,
					Shape:      "blue",
					Size:       "3pt",
				})
			}
		}
		curPos := positionStart
		points = append(points, fmt.Sprintf("(%s)", positionStart))
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
			points = append(points, fmt.Sprintf(`(%s)`, newPos))
			curPos = newPos
		}
		rawPos := v.ConnectorControlPoints[len(v.ConnectorControlPoints)-1].Position
		lastPos := Position{float32(rawPos.X), float32(rawPos.Y)}
		lastPos.X *= scale
		lastPos.Y *= scale
		points = append(points, fmt.Sprintf(`(%s)`, lastPos))
		allPoints := strings.Join(points, " -- ")
		c.b.Writef(`\draw [thick, rounded corners=10] %s;`, allPoints)
		straight := c.isStraight(curPos, lastPos, magnetTo)

		completePath = append(completePath, lastPos)

		if straight {
			completePath = append(completePath, positionEnd)
			c.b.Writef(`\draw[-To, thick] (%s) -- (%s);`, lastPos, positionEnd)
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
			c.b.Writef(`\draw [thick, rounded corners=10] (%s) -- (%s) -- (%s); %% midpoint`, lastPos, midPath, finalPosition)
			c.b.Writef(`\draw[-To, thick] (%s) -- (%s);`, finalPosition, positionEnd)
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

		c.b.Writef(`\draw[-To, thick, rounded corners=10] (%s) -- (%s) -- (%s) -- (%s);`, positionStart, cp1, cp2, positionEnd)
	} else {
		c.b.Writef(`\draw[-To, thick] (%s) -- (%s);`, positionStart, positionEnd)

		if len(connectorText) != 0 {
			midPoint := v.ConnectorTextMidpoint
			c.drawArrowTextStraight(positionStart, positionEnd, connectorText, midPoint)
		}
	}
}

func (c *Compiler) drawShapeWithText(w DrawingNode) {
	text := c.CleanupText(w.Node.Name)

	switch w.Node.ShapeWithTextType {
	case fig.ShapeWithTextTypeSquare, fig.ShapeWithTextTypePredefinedProcess:
		c.b.Writef(`\draw (%s) rectangle node{%s} (%s);`, w.Q1, text, w.Q2)

	case fig.ShapeWithTextTypeEllipse:
	case fig.ShapeWithTextTypeDiamond:
		pos := w.Q1.MiddleWith(w.Q2)
		c.b.Writef(`\draw[rotate around={45:(%s)}, scale around={0.75:(%s)}] (%s) rectangle node{%s} (%s);`, pos, pos, w.Q1, text, w.Q2)
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
	c.b.Writef(`\node at (%s) {%s};`, point, strings.ReplaceAll(v.Node.Name, "_", "\\_"))
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
	c.b.Writef(`\node[draw=none,fill=white] at (%s) {%s};`, pos, text)
}

func (c *Compiler) drawArrowTextComplex(path []Position, text string, midpoint *fig.ConnectorTextMidpoint) {
	if len(path)%2 != 0 {
		panic("odd number of path segments?")
	}

	if midpoint == nil {
		midpoint = &fig.ConnectorTextMidpoint{
			Section:       fig.ConnectorTextSectionMiddleToStart,
			Offset:        0,
			OffAxisOffset: 0,
		}
	}

	totalLen := float32(0.0)
	for i := 1; i < len(path); i += 1 {
		last := path[i-1]
		current := path[i]
		if last.Y == current.Y {
			totalLen += float32(math.Abs(float64(current.X - last.X)))
		} else {
			totalLen += float32(math.Abs(float64(current.Y - last.Y)))
		}
	}

	midPoint := totalLen / 2.0
	var positionInLen float32
	if midpoint.Section == fig.ConnectorTextSectionMiddleToEnd { // 1 == end, 0 = middle
		positionInLen = totalLen - midPoint*float32(midpoint.Offset)
	} else { // 1 = start, 0 = middle
		positionInLen = midPoint * float32(midpoint.Offset)
	}

	var totalRan float32
	var finalPaths = append([]Position{}, path...)
	if midpoint.Section == fig.ConnectorTextSectionMiddleToEnd {
		slices.Reverse(finalPaths)
	}
	for i := 1; i < len(finalPaths); i += 1 {
		var last, current Position
		if midpoint.Section == fig.ConnectorTextSectionMiddleToEnd {
			last = finalPaths[i]
			current = finalPaths[i-1]
		} else {
			last = finalPaths[i-1]
			current = finalPaths[i]
		}

		var lineLen float32
		if last.Y == current.Y {
			lineLen = float32(math.Abs(float64(current.X - last.X)))
		} else {
			lineLen = float32(math.Abs(float64(current.Y - last.Y)))
		}

		if positionInLen > totalRan && positionInLen < totalRan+lineLen {
			var x, y float32
			if midpoint.Section == fig.ConnectorTextSectionMiddleToEnd { // 1 == end, 0 = middle
				if last.Y == current.Y {
					y = current.Y - positionInLen - totalRan
					x = current.X
				} else {
					x = current.X - positionInLen - totalRan
					y = current.Y
				}
			} else {
				if last.Y == current.Y {
					y = current.Y - positionInLen - totalRan
					x = current.X
				} else {
					x = current.X + positionInLen - totalRan
					y = current.Y
				}
			}
			c.b.Writef(`\node[draw=none,fill=white] at (%s) {%s};`, Position{x, y}, text)
			return
		}

		totalRan += lineLen
	}
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
