package main

import (
	"fmt"
	"io"
	"os"
	"strconv"
	"strings"
)

const Usage = `Usage:	base64x [-dh] [-b num] [-i in_file] [-o out_file]
  -b, --break   break encoded output up into lines of length num (default: 80)
  -d, --decode  decode input
  -h, --help    display this message
  -i, --input   input file (default: "-" for stdin)
  -o, --output  output file (default: "-" for stdout)`

type Flags struct {
	Break  int
	Decode bool
	Help   bool
	Input  string
	Output string
}

func ParseFlags() (*Flags, error) {
	flags := &Flags{
		Break:  80,
		Input:  "-",
		Output: "-",
	}
	for i := 1; i < len(os.Args); i++ {
		switch os.Args[i] {
		case "-b", "--break":
			if i+1 >= len(os.Args) {
				return nil, fmt.Errorf("base64x: option requires an argument -- %v", os.Args[i])
			}
			num, err := strconv.Atoi(os.Args[i+1])
			if err != nil {
				return nil, fmt.Errorf("base64x: option argument invalid -- %v", os.Args[i])
			}
			flags.Break = num
			i++

		case "-d", "--decode":
			flags.Decode = true

		case "-h", "--help":
			flags.Help = true

		case "-i", "--input":
			if i+1 >= len(os.Args) {
				return nil, fmt.Errorf("base64x: option requires an argument -- %v", os.Args[i])
			}
			flags.Input = strings.TrimSpace(os.Args[i+1])
			i++

		case "-o", "--output":
			if i+1 >= len(os.Args) {
				return nil, fmt.Errorf("base64x: option requires an argument -- %v", os.Args[i])
			}
			flags.Output = strings.TrimSpace(os.Args[i+1])
			i++

		case "-dh", "-hd":
			flags.Decode = true
			flags.Help = true

		default:
			return nil, fmt.Errorf("base64x: unrecognized option: %q", os.Args[i])
		}
	}
	return flags, nil
}

func main() {
	flags, err := ParseFlags()
	if err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}
	if flags.Help {
		fmt.Println(Usage)
		return
	}

	defer func() {
		if err != nil {
			os.Exit(1)
		}
	}()

	var input io.Reader = os.Stdin
	if flags.Input != "-" {
		fp, err := os.Open(flags.Input)
		if err != nil {
			fmt.Fprintln(os.Stderr, err)
			os.Exit(1)
		}
		defer fp.Close()

		input = fp
	}

	var output io.Writer = os.Stdout
	if flags.Output != "-" {
		fp, err := os.OpenFile(flags.Output, os.O_CREATE|os.O_TRUNC|os.O_WRONLY, 0644)
		if err != nil {
			fmt.Fprintln(os.Stderr, err)
			os.Exit(1)
		}
		defer fp.Close()

		output = fp
	}

	if flags.Decode {
		input = NewBase64Decoder(input)
	} else {
		output = NewBase64Encoder(output, flags.Break, '\n')
	}
	_, err = io.Copy(output, input)
	if err != nil {
		fmt.Fprintf(os.Stderr, "\n%v\n", err)
	}
}
