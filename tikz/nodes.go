package tikz

import (
	"fmt"
	"strings"
)

type Attribute interface {
	HasPosition() bool
	GetPosition() Position
	SetPosition(p Position)
	String() string
}
type AttributeList []Attribute

func (a AttributeList) String() string {
	allAttributes := make([]string, len(a))
	for i, v := range a {
		allAttributes[i] = v.String()
	}
	return strings.Join(allAttributes, ", ")
}

type ToAttribute struct{}

func (t *ToAttribute) HasPosition() bool { return false }

func (t *ToAttribute) GetPosition() Position {
	panic("ToAttribute has no position")
}

func (t *ToAttribute) SetPosition(p Position) {
	panic("ToAttribute has no position")
}

func (t *ToAttribute) String() string { return "-To" }

type ThickAttribute struct{}

func (t *ThickAttribute) HasPosition() bool { return false }

func (t *ThickAttribute) GetPosition() Position {
	panic("ThickAttribute has no position")
}

func (t *ThickAttribute) SetPosition(p Position) {
	panic("ThickAttribute has no position")
}

func (t *ThickAttribute) String() string { return "thick" }

type FillAttribute struct{ Value string }

func (f *FillAttribute) HasPosition() bool { return false }

func (f *FillAttribute) GetPosition() Position {
	panic("FillAttribute has no position")
}

func (f *FillAttribute) SetPosition(p Position) {
	panic("FillAttribute has no position")
}

func (f *FillAttribute) String() string {
	return fmt.Sprintf("fill=%s", f.Value)
}

type RoundedCornersAttribute struct{ Value int }

func (r *RoundedCornersAttribute) HasPosition() bool { return false }

func (r *RoundedCornersAttribute) GetPosition() Position {
	panic("RoundedCornersAttribute has no position")
}

func (r *RoundedCornersAttribute) SetPosition(p Position) {
	panic("RoundedCornersAttribute has no position")
}

func (r *RoundedCornersAttribute) String() string {
	return fmt.Sprintf("rounded corners=%d", r.Value)
}

type RotateAroundAttribute struct {
	Degrees  int
	Position Position
}

func (r *RotateAroundAttribute) HasPosition() bool { return true }

func (r *RotateAroundAttribute) GetPosition() Position { return r.Position }

func (r *RotateAroundAttribute) SetPosition(p Position) { r.Position = p }

func (r *RotateAroundAttribute) String() string {
	return fmt.Sprintf("rotate around={%d:(%s)}", r.Degrees, r.Position)
}

type ScaleAroundAttribute struct {
	Scale    float32
	Position Position
}

func (s *ScaleAroundAttribute) HasPosition() bool { return true }

func (s *ScaleAroundAttribute) GetPosition() Position { return s.Position }

func (s *ScaleAroundAttribute) SetPosition(p Position) { s.Position = p }

func (s *ScaleAroundAttribute) String() string {
	return fmt.Sprintf("scale around={%f:(%s)}", s.Scale, s.Position)
}

type ColorAttribute string

func (c ColorAttribute) HasPosition() bool { return false }

func (c ColorAttribute) GetPosition() Position {
	panic("ColorAttribute has no position")
}

func (c ColorAttribute) SetPosition(p Position) {
	panic("ColorAttribute has no position")
}

func (c ColorAttribute) String() string { return "color=" + string(c) }

type Draw struct {
	Attributes AttributeList
	Points     PositionList
	Text       *string
	Kind       *string
}

func (d Draw) String() string {
	data := []string{`\draw`}
	if len(d.Attributes) > 0 {
		data = append(data, fmt.Sprintf("[%s]", d.Attributes.String()))
	}
	if d.Kind != nil {
		data = append(data, fmt.Sprintf(" %s", *d.Kind))
	}
	if d.Text != nil {
		data = append(data, fmt.Sprintf(" node{%s}", *d.Text))
	}
	if len(d.Points) > 0 {
		data = append(data, fmt.Sprintf(" %s", d.Points.String()))
	}
	data = append(data, ";")
	return strings.Join(data, "")
}

type Node struct {
	Attributes AttributeList
	Position   Position
	Text       *string
}

func (n Node) String() string {
	data := []string{`\node`}
	if len(n.Attributes) > 0 {
		data = append(data, fmt.Sprintf("[%s]", n.Attributes.String()))
	}
	data = append(data, fmt.Sprintf(" at (%s)", n.Position))
	if n.Text != nil {
		data = append(data, fmt.Sprintf(" {%s}", *n.Text))
	}
	data = append(data, ";")
	return strings.Join(data, "")
}

type FillDraw struct {
	Attributes AttributeList
	Position   Position
	Shape      string
	Size       string
}

func (f FillDraw) String() string {
	data := []string{`\filldraw`}
	if len(f.Attributes) > 0 {
		data = append(data, fmt.Sprintf("[%s]", f.Attributes.String()))
	}

	data = append(data, fmt.Sprintf(" (%s)", f.Position))
	data = append(data, fmt.Sprintf(" %s", f.Shape))
	data = append(data, fmt.Sprintf("(%s)", f.Size))
	data = append(data, ";")
	return strings.Join(data, "")
}
