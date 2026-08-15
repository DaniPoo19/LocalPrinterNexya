package escpos

import (
	"bytes"
	"fmt"
	"strings"
)

type Builder struct {
	buf       bytes.Buffer
	lineWidth int // 32 for 58mm, 48 for 80mm
}

func NewBuilder(paperWidth string) *Builder {
	width := 48
	if paperWidth == "58mm" {
		width = 32
	}
	b := &Builder{lineWidth: width}
	b.Init()
	return b
}

func (b *Builder) Init() *Builder {
	b.buf.Write(CmdInit)
	return b
}

func (b *Builder) SetAlign(align string) *Builder {
	switch strings.ToLower(align) {
	case "center":
		b.buf.Write(CmdAlignCenter)
	case "right":
		b.buf.Write(CmdAlignRight)
	default:
		b.buf.Write(CmdAlignLeft)
	}
	return b
}

func (b *Builder) SetBold(enable bool) *Builder {
	if enable {
		b.buf.Write(CmdBoldOn)
	} else {
		b.buf.Write(CmdBoldOff)
	}
	return b
}

func (b *Builder) SetDoubleSize(enable bool) *Builder {
	if enable {
		b.buf.Write(CmdSizeDoubleAll)
	} else {
		b.buf.Write(CmdSizeNormal)
	}
	return b
}

func (b *Builder) SetDoubleHeight(enable bool) *Builder {
	if enable {
		b.buf.Write(CmdSizeDoubleH)
	} else {
		b.buf.Write(CmdSizeNormal)
	}
	return b
}

func (b *Builder) SetSmallFont(enable bool) *Builder {
	if enable {
		b.buf.Write(CmdFontB)
	} else {
		b.buf.Write(CmdFontA)
	}
	return b
}

func (b *Builder) PrintLine(text string) *Builder {
	b.buf.WriteString(text)
	b.buf.Write(CmdLineFeed)
	return b
}

func (b *Builder) FeedLines(n int) *Builder {
	for i := 0; i < n; i++ {
		b.buf.Write(CmdLineFeed)
	}
	return b
}

func (b *Builder) PrintDivider(char string) *Builder {
	if char == "" {
		char = "-"
	}
	repeated := strings.Repeat(char, b.lineWidth)
	b.buf.WriteString(repeated)
	b.buf.Write(CmdLineFeed)
	return b
}

// PrintRow2Cols imprime 2 columnas justificadas a los extremos (ej: "Total" a la izquierda, "$25.000" a la derecha)
func (b *Builder) PrintRow2Cols(left, right string) *Builder {
	totalWidth := b.lineWidth
	spaceCount := totalWidth - len(left) - len(right)
	if spaceCount < 1 {
		spaceCount = 1
	}
	line := left + strings.Repeat(" ", spaceCount) + right
	b.buf.WriteString(line)
	b.buf.Write(CmdLineFeed)
	return b
}

// PrintItemRow imprime producto con cantidad, nombre y precio
func (b *Builder) PrintItemRow(qty int, name string, priceStr string) *Builder {
	qtyStr := fmt.Sprintf("%dx ", qty)
	maxNameWidth := b.lineWidth - len(qtyStr) - len(priceStr) - 1
	if maxNameWidth < 5 {
		maxNameWidth = 5
	}

	displayName := name
	if len(displayName) > maxNameWidth {
		displayName = displayName[:maxNameWidth]
	}

	left := qtyStr + displayName
	b.PrintRow2Cols(left, priceStr)
	return b
}

func (b *Builder) OpenDrawer() *Builder {
	b.buf.Write(CmdOpenDrawerPin2)
	b.buf.Write(CmdOpenDrawerPin5)
	return b
}

func (b *Builder) Beep() *Builder {
	b.buf.Write(CmdBeep)
	return b
}

func (b *Builder) Cut(partial bool) *Builder {
	b.FeedLines(3)
	if partial {
		b.buf.Write(CmdCutPartial)
	} else {
		b.buf.Write(CmdCutFull)
	}
	return b
}

func (b *Builder) Bytes() []byte {
	return b.buf.Bytes()
}
