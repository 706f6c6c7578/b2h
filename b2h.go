package main

import (
	"bufio"
	"encoding/hex"
	"flag"
	"fmt"
	"io"
	"os"
	"time"
)

const bufferSize = 32 * 1024 // 32KB

func main() {
	decode := flag.Bool("d", false, "Decode mode")
	wrap := flag.Int("w", 0, "Wrap width (0 for no wrapping)")

	flag.Usage = func() {
		fmt.Fprintf(os.Stderr, "Usage: %s [OPTIONS] < infile > outfile\n\n", os.Args[0])
		fmt.Fprintf(os.Stderr, "Options:\n")
		flag.PrintDefaults()
	}

	flag.Parse()

	if *wrap < 0 {
		fmt.Fprintf(os.Stderr, "Error: Wrap width must be non-negative\n")
		flag.Usage()
		os.Exit(1)
	}

	reader := bufio.NewReaderSize(os.Stdin, bufferSize)
	writer := bufio.NewWriterSize(os.Stdout, bufferSize)
	defer writer.Flush()

	startTime := time.Now()

	var err error
	if *decode {
		err = decodeStream(reader, writer)
	} else {
		err = encodeStream(reader, writer, *wrap)
	}

	duration := time.Since(startTime)

	if err != nil {
		fmt.Fprintf(os.Stderr, "Error: %v\n", err)
		os.Exit(1)
	}

	fmt.Fprintf(os.Stderr, "\nProcess completed in %v\n\n", duration)
}

func decodeStream(r io.Reader, w io.Writer) error {
	decoder := hex.NewDecoder(&whitespaceFilter{r: r})
	_, err := io.Copy(w, decoder)
	if err != nil {
		return fmt.Errorf("decode error: %w", err)
	}
	return nil
}

func encodeStream(r io.Reader, w io.Writer, wrapWidth int) error {
	lw := &lineWrapper{w: w, width: wrapWidth}
	encoder := hex.NewEncoder(lw)
	_, err := io.Copy(encoder, r)
	if err != nil {
		return fmt.Errorf("encode error: %w", err)
	}
	if err := lw.Flush(); err != nil {
		return fmt.Errorf("flush error: %w", err)
	}
	return nil
}

type whitespaceFilter struct {
	r io.Reader
}

func (wf *whitespaceFilter) Read(p []byte) (n int, err error) {
	for {
		n, err = wf.r.Read(p)
		if err != nil {
			return
		}
		j := 0
		for i := 0; i < n; i++ {
			if p[i] != '\r' && p[i] != '\n' && p[i] != ' ' && p[i] != '\t' {
				p[j] = p[i]
				j++
			}
		}
		if j > 0 {
			return j, nil
		}
	}
}

type lineWrapper struct {
	w     io.Writer
	width int
	count int
}

func (lw *lineWrapper) Write(p []byte) (n int, err error) {
	for _, b := range p {
		if lw.width > 0 && lw.count >= lw.width {
			if err := lw.writeCRLF(); err != nil {
				return n, err
			}
		}
		if _, err := lw.w.Write([]byte{b}); err != nil {
			return n, fmt.Errorf("write error: %w", err)
		}
		n++
		lw.count++
	}
	return n, nil
}

func (lw *lineWrapper) writeCRLF() error {
	_, err := lw.w.Write([]byte("\r\n"))
	if err != nil {
		return fmt.Errorf("write CRLF error: %w", err)
	}
	lw.count = 0
	return nil
}

func (lw *lineWrapper) Flush() error {
	if lw.count > 0 {
		return lw.writeCRLF()
	}
	return nil
}
