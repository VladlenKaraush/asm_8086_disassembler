package model

import (
	"fmt"
	"strconv"
)

type Command struct {
	Cmd    string
	Source string
	Dest   string
	Len    int
}

func (c *Command) Str() string {
	return fmt.Sprintf("%s %s, %s (len:%s)", c.Cmd, c.Dest, c.Source, strconv.Itoa(c.Len))
}
