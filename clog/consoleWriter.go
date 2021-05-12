package clog

import (
	"fmt"
	"os"
)

type ConsoleWriter struct {
}

func NewConsoleWriter() *ConsoleWriter {
	return &ConsoleWriter{}
}

func (w *ConsoleWriter) Write(enc *textEncoder) error {
	fmt.Fprint(os.Stdout, string(enc.bytes))

	// 可以自己修改颜色，这里可以进行配置的
	//blue := color.New(color.FgBlue)
	//if _,err :=blue.Fprint(os.Stdout,string(enc.bytes)); err != nil {
	//	panic("blue.Fprint error")
	//}

	return nil
}

func (w *ConsoleWriter) Init() error {
	return nil
}
