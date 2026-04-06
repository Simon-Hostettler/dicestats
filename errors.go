package dicestats

import "fmt"

type ParseError struct {
	Pos     int
	Message string
}

func (e *ParseError) Error() string {
	return fmt.Sprintf("parse error at %d: %s", e.Pos, e.Message)
}
