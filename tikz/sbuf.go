package tikz

import (
	"fmt"
	"strings"
)

type sbuf struct {
	strings.Builder
}

func (s *sbuf) Writef(format string, args ...any) {
	s.WriteString(fmt.Sprintf(format, args...) + "\n")
}
