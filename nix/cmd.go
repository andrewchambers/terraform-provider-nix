package nix

import (
	"bufio"
	"bytes"
	"fmt"
	"io"
	"log"
	"os/exec"
	"strconv"
)

func runCommandWithLogging(c *exec.Cmd, stdout io.Writer) error {
	log.Printf("running %v in env %v", c.Args, c.Env)

	er, ew := io.Pipe()
	or, ow := io.Pipe()
	c.Stdout = ow
	c.Stderr = ew
	c.Stdin = nil

	capture := func(r io.Reader, label string) {
		brdr := bufio.NewReader(r)

		for {
			s, err := brdr.ReadString('\n')
			if len(s) != 0 {
				log.Printf("[INFO] %s: %s", label, s)
			}
			if err != nil {
				break
			}
		}

	}

	stderrSaver := &prefixSuffixSaver{N: 32 << 10}
	terr := io.TeeReader(er, stderrSaver)
	tout := io.TeeReader(or, stdout)

	ioDone := make(chan struct{})

	go func() { capture(terr, "stderr"); ioDone <- struct{}{} }()
	go func() { capture(tout, "stdout"); ioDone <- struct{}{} }()

	err := c.Run()

	_ = or.Close()
	_ = er.Close()

	<-ioDone
	<-ioDone

	if err != nil {
		if ee, ok := err.(*exec.ExitError); ok {
			err = fmt.Errorf("%s: %v", ee.String(), string(stderrSaver.Bytes()))
		}
	}
	return err
}

// prefixSuffixSaver code originally from go stdlib.

// prefixSuffixSaver is an io.Writer which retains the first N bytes
// and the last N bytes written to it. The Bytes() methods reconstructs
// it with a pretty error message.
type prefixSuffixSaver struct {
	N         int // max size of prefix or suffix
	prefix    []byte
	suffix    []byte // ring buffer once len(suffix) == N
	suffixOff int    // offset to write into suffix
	skipped   int64
}

func (w *prefixSuffixSaver) Write(p []byte) (n int, err error) {
	lenp := len(p)
	p = w.fill(&w.prefix, p)

	// Only keep the last w.N bytes of suffix data.
	if overage := len(p) - w.N; overage > 0 {
		p = p[overage:]
		w.skipped += int64(overage)
	}
	p = w.fill(&w.suffix, p)

	// w.suffix is full now if p is non-empty. Overwrite it in a circle.
	for len(p) > 0 { // 0, 1, or 2 iterations.
		n := copy(w.suffix[w.suffixOff:], p)
		p = p[n:]
		w.skipped += int64(n)
		w.suffixOff += n
		if w.suffixOff == w.N {
			w.suffixOff = 0
		}
	}
	return lenp, nil
}

// fill appends up to len(p) bytes of p to *dst, such that *dst does not
// grow larger than w.N. It returns the un-appended suffix of p.
func (w *prefixSuffixSaver) fill(dst *[]byte, p []byte) (pRemain []byte) {
	if remain := w.N - len(*dst); remain > 0 {
		add := minInt(len(p), remain)
		*dst = append(*dst, p[:add]...)
		p = p[add:]
	}
	return p
}

func (w *prefixSuffixSaver) Bytes() []byte {
	if w.suffix == nil {
		return w.prefix
	}
	if w.skipped == 0 {
		return append(w.prefix, w.suffix...)
	}
	var buf bytes.Buffer
	buf.Grow(len(w.prefix) + len(w.suffix) + 50)
	buf.Write(w.prefix)
	buf.WriteString("\n... omitting ")
	buf.WriteString(strconv.FormatInt(w.skipped, 10))
	buf.WriteString(" bytes ...\n")
	buf.Write(w.suffix[w.suffixOff:])
	buf.Write(w.suffix[:w.suffixOff])
	return buf.Bytes()
}

func minInt(a, b int) int {
	if a < b {
		return a
	}
	return b
}
