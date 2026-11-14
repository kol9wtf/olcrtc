// Package main provides the kc CLI entry point.
package main

import (
	"flag"
	"fmt"
	"io"
	"os"

	"codeberg.org/rape4me/kc/ivf"
	"codeberg.org/rape4me/kc/vp8"
)

func main() {
	encCmd := flag.NewFlagSet("enc", flag.ExitOnError)
	encOut := encCmd.String("o", "out.ivf", "output IVF file")
	encQP := encCmd.Int("qp", 4, "quantizer 0-127")
	encFPS := encCmd.Int("fps", 120, "frames per second")
	encW := encCmd.Int("w", 1024, "frame width")
	encH := encCmd.Int("h", 1024, "frame height")

	decCmd := flag.NewFlagSet("dec", flag.ExitOnError)
	decIn := decCmd.String("i", "out.ivf", "input IVF file")
	decOut := decCmd.String("o", "-", "output raw Y frames (- = stdout)")

	if len(os.Args) < 2 {
		usage()
		os.Exit(1)
	}

	switch os.Args[1] {
	case "enc":
		encCmd.Parse(os.Args[2:]) //nolint:errcheck // ExitOnError handles it.
		if err := runEncode(*encOut, *encQP, *encFPS, *encW, *encH); err != nil {
			fmt.Fprintf(os.Stderr, "kc: %v\n", err)
			os.Exit(1)
		}
	case "dec":
		decCmd.Parse(os.Args[2:]) //nolint:errcheck // ExitOnError handles it.
		if err := runDecode(*decIn, *decOut); err != nil {
			fmt.Fprintf(os.Stderr, "kc: %v\n", err)
			os.Exit(1)
		}
	default:
		usage()
		os.Exit(1)
	}
}

func usage() {
	fmt.Fprintf(os.Stderr, "usage: kc <enc|dec> [flags]\n")
	fmt.Fprintf(os.Stderr, "  enc - encode raw grayscale frames (stdin) to VP8/IVF\n")
	fmt.Fprintf(os.Stderr, "  dec - decode VP8/IVF to raw grayscale frames\n")
}

// runEncode reads raw grayscale frames from stdin and writes VP8/IVF.
func runEncode(outPath string, qp, fps, w, h int) error {
	f, err := os.Create(outPath) //nolint:gosec // user CLI arg.
	if err != nil {
		return fmt.Errorf("create %s: %w", outPath, err)
	}
	defer f.Close() //nolint:errcheck // best-effort.

	iw, err := ivf.NewWriter(f, w, h, fps)
	if err != nil {
		return fmt.Errorf("ivf writer: %w", err)
	}

	enc := vp8.NewEncoder(w, h, qp)
	frameSize := w * h
	buf := make([]byte, frameSize)
	var pts uint64

	for {
		_, err := io.ReadFull(os.Stdin, buf)
		if err != nil {
			break
		}
		frame, err := enc.Encode(buf)
		if err != nil {
			return fmt.Errorf("encode: %w", err)
		}
		if err := iw.WriteFrame(frame, pts); err != nil {
			return fmt.Errorf("write frame: %w", err)
		}
		pts++
	}

	if err := iw.Close(); err != nil {
		return fmt.Errorf("ivf close: %w", err)
	}
	fmt.Fprintf(os.Stderr, "encoded %d frames -> %s\n", pts, outPath)
	return nil
}

// runDecode reads VP8/IVF and writes raw grayscale frames.
func runDecode(inPath, outPath string) error {
	f, err := os.Open(inPath) //nolint:gosec // user CLI arg.
	if err != nil {
		return fmt.Errorf("open %s: %w", inPath, err)
	}
	defer f.Close() //nolint:errcheck // best-effort.

	ir, err := ivf.NewReader(f)
	if err != nil {
		return fmt.Errorf("ivf reader: %w", err)
	}

	var out io.Writer = os.Stdout
	if outPath != "-" {
		of, err := os.Create(outPath) //nolint:gosec // user CLI arg.
		if err != nil {
			return fmt.Errorf("create %s: %w", outPath, err)
		}
		defer of.Close() //nolint:errcheck // best-effort.
		out = of
	}

	dec := vp8.NewDecoder()
	var count int
	for {
		data, _, err := ir.ReadFrame()
		if err != nil {
			break
		}
		frame, err := dec.Decode(data)
		if err != nil {
			return fmt.Errorf("decode frame %d: %w", count, err)
		}
		if _, err := out.Write(frame.Grayscale()); err != nil {
			return fmt.Errorf("write frame %d: %w", count, err)
		}
		count++
	}
	fmt.Fprintf(os.Stderr, "decoded %d frames\n", count)
	return nil
}
