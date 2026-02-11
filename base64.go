package main

import (
	"bytes"
	"encoding/base64"
	"errors"
	"fmt"
	"io"
	"slices"
	"unicode/utf8"
)

type base64Encoder struct {
	enc *base64.Encoding
	w   io.Writer
	brk int
	sep []byte
}

func NewBase64Encoder(w io.Writer, breakNum int, sep ...byte) io.Writer {
	be := &base64Encoder{enc: base64.StdEncoding, w: w, brk: breakNum, sep: sep}
	if breakNum > 0 && len(sep) == 0 {
		be.sep = []byte{'\n'}
	}
	return be
}

func (be *base64Encoder) Write(p []byte) (int, error) {
	d := make([]byte, be.enc.EncodedLen(len(p)))
	be.enc.Encode(d, p)

	if be.brk > 0 {
		var lines [][]byte
		for line := range slices.Chunk(d, be.brk) {
			lines = append(lines, line)
		}
		lines = append(lines, nil)
		d = bytes.Join(lines, be.sep)
	}

	n, e := be.w.Write(d)
	if e == nil && n < len(d) {
		e = io.ErrShortWrite
	}
	return len(p), e
}

type base64Decoder struct {
	r io.Reader
	d *base64.Encoding
	b []byte
	p []byte
	e error
}

func NewBase64Decoder(r io.Reader) io.Reader {
	return &base64Decoder{
		r: &newlineFilteringReader{&compatibleReader{r}},
		d: base64.StdEncoding,
	}
}

func (bd *base64Decoder) Read(b []byte) (int, error) {
	if len(bd.b) > 0 {
		return bd.readBuffer(b)
	}
	if bd.e != nil {
		return 0, bd.e
	}

	bd.fill(b)

	if len(bd.b) > 0 {
		return bd.readBuffer(b)
	}
	return 0, bd.e
}

func (bd *base64Decoder) readBuffer(b []byte) (int, error) {
	rn := copy(b, bd.b)
	if rn == len(bd.b) {
		bd.b = bd.b[:0]
		return rn, bd.e
	}
	bd.b = bd.b[:copy(bd.b, bd.b[rn:])]
	return rn, nil
}

func (bd *base64Decoder) fill(b []byte) {
	rn, re := bd.r.Read(b)
	if rn > 0 {
		bd.p = append(bd.p, b[:rn]...)
	}
	if re != nil {
		bd.e = re
	}

	n := len(bd.p) / 4 * 4
	if n > 0 {
		bd.decode(n)
	}
	if bd.e != nil && len(bd.p) > 0 {
		if bd.e == io.EOF {
			bd.e = io.ErrUnexpectedEOF
		}
		bd.leftover()
	}
}

func (bd *base64Decoder) decode(n int) {
	d := make([]byte, bd.d.DecodedLen(n))
	for _, p := range split(bd.p[:n]) {
		dn, de := bd.d.Decode(d, p)
		if dn > 0 {
			bd.b = append(bd.b, d[:dn]...)
		}
		if de != nil {
			bd.e = de
			var i base64.CorruptInputError
			if errors.As(de, &i) {
				bd.e = illegalError(p[i:])
				bd.p = p[i/4*4 : i : i]
				bd.leftover()
			}
			bd.p = nil
			return
		}
	}
	bd.p = bd.p[:copy(bd.p, bd.p[n:])]
}

func (bd *base64Decoder) leftover() {
	n := len(bd.p)
	if n == 0 {
		return
	}

	bd.p = append(bd.p, "000"...)
	var d [3]byte
	_, de := bd.d.Decode(d[:], bd.p[:4])
	if de != nil {
		var i base64.CorruptInputError
		if errors.As(de, &i) {
			bd.e = illegalError(bd.p[i:])
			n = int(i)
		}
	}
	bd.p = bd.p[:0]

	switch n {
	case 2:
		if de == nil {
			bd.b = append(bd.b, d[0])
		} else {
			bd.p = bd.p[:2]
			bd.leftover()
		}
	case 3:
		bd.b = append(bd.b, d[0], d[1])
	}
}

func illegalError(p []byte) error {
	if p[0] <= utf8.RuneSelf {
		return fmt.Errorf("illegal base64 byte: %q", p[:1])
	}
	_, n := utf8.DecodeRune(p)
	return fmt.Errorf("illegal base64 byte: %q", p[:n])
}

func split(s []byte) [][]byte {
	var a [][]byte
	start := 0
	for i := 4; i <= len(s); i += 4 {
		if s[i-1] == '=' {
			a = append(a, s[start:i])
			start = i
		}
	}
	if start < len(s) {
		a = append(a, s[start:])
	}
	return a
}

type newlineFilteringReader struct {
	wrapped io.Reader
}

func (r *newlineFilteringReader) Read(p []byte) (int, error) {
	n, err := r.wrapped.Read(p)
	for n > 0 {
		offset := 0
		for i, b := range p[:n] {
			if b != '\r' && b != '\n' {
				if i != offset {
					p[offset] = b
				}
				offset++
			}
		}
		if offset > 0 {
			return offset, err
		}
		// Previous buffer entirely whitespace, read again
		n, err = r.wrapped.Read(p)
	}
	return n, err
}

type compatibleReader struct {
	r io.Reader
}

func (cr *compatibleReader) Read(p []byte) (int, error) {
	n, err := cr.r.Read(p)
	if n > 0 {
		for i := range p {
			switch p[i] {
			case '-':
				p[i] = '+'
			case '_':
				p[i] = '/'
			}
		}
	}
	return n, err
}
