package tabwriter

import (
	"bytes"
	"fmt"
	"io"
	"strings"
	"text/tabwriter"
)

const (
	tabwriterMinWidth = 6
	tabwriterWidth    = 4
	tabwriterPadding  = 3
	tabwriterPadChar  = ' '
)

type Writer struct {
	delegate *tabwriter.Writer

	header []string
	out    io.Writer
	buf    bytes.Buffer
}

func toStringList(args ...interface{}) []string {
	strLst := make([]string, len(args))
	for i, arg := range args {
		strLst[i] = fmt.Sprint(arg)
	}
	return strLst
}

func (w *Writer) Render() error {
	// print header
	_, err := fmt.Fprintln(w.delegate, strings.Join(w.header, "\t"))
	if err != nil {
		return err
	}

	// print content
	_, err = w.buf.WriteTo(w.delegate)
	if err != nil {
		return err
	}
	return w.delegate.Flush()
}

func (w *Writer) Append(args ...interface{}) {
	_, _ = fmt.Fprintln(&w.buf, strings.Join(toStringList(args...), "\t"))
}

func (w *Writer) AppendAndFlush(args ...interface{}) error {
	_, _ = fmt.Fprintln(w.delegate, strings.Join(toStringList(args...), "\t"))
	return w.delegate.Flush()
}

func (w *Writer) SetHeader(header []string) {
	upperHeader := make([]string, len(header))
	for i, col := range header {
		upperHeader[i] = strings.ToUpper(col)
	}
	w.header = upperHeader
}

func (w *Writer) Write(buf []byte) (n int, err error) {
	return w.buf.Write(buf)
}

func (w *Writer) Reset() {
	w.delegate.Init(w.out, tabwriterMinWidth, tabwriterWidth, tabwriterPadding, tabwriterPadChar, 0)
}

func New(out io.Writer) *Writer {
	return &Writer{
		out: out,
		delegate: tabwriter.NewWriter(out,
			tabwriterMinWidth,
			tabwriterWidth,
			tabwriterPadding,
			tabwriterPadChar,
			0),
	}
}
