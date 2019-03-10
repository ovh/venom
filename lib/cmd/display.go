package cmd

import (
	"context"
	"fmt"
	"sync"
	"time"

	"github.com/sethgrid/curse"
)

type Container struct {
	screenLines  int
	screenWidth  int
	startingLine int
	sync.Mutex
	content        []*Display
	lastLineNumber int
}

type Display string

func (c *Container) Display(ctx context.Context) {
	if c == nil {
		return
	}
	width, lines, _ := curse.GetScreenDimensions()
	_, line, _ := curse.GetCursorPosition()

	c.screenLines = lines
	c.screenWidth = width
	c.startingLine = line

	if c == nil {
		return
	}
	refresh := time.NewTicker(100 * time.Millisecond)
	for {
		select {
		case <-ctx.Done():
			return
		case <-refresh.C:
			c.update()
		}
	}
}

func (c *Container) Append(d *Display) {
	if c == nil {
		return
	}
	c.content = append(c.content, d)
}

func (c *Container) update() {
	if c == nil {
		return
	}

	for c.lastLineNumber < len(c.content)-1 {
		fmt.Println()
		c.lastLineNumber++
	}

	cu, _ := curse.New()
	for i := range c.content {
		line := c.content[len(c.content)-(i+1)]
		cu.Move(1, c.startingLine-i)
		cu.EraseCurrentLine()
		fmt.Printf("\r%v", line)
	}
	cu.Move(cu.StartingPosition.X, cu.StartingPosition.Y)
}

// Printf update the displayed message
func (d *Display) Printf(format string, args ...interface{}) {
	*d = Display(fmt.Sprintf(format, args...))
}

func (d *Display) String() string {
	return string(*d)
}
