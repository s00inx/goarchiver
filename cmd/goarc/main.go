package main

import (
	"flag"
	"os"
	"strings"

	api "github.com/kfcemployee/goarchiver/internal"
)

func main() {
	if len(os.Args) < 2 {
		msg := []byte("unknown command: use goarc -h for help.")
		os.Stdout.Write(msg)
		return
	}

	cmdPack := flag.NewFlagSet("p", flag.ExitOnError)
	oDir := cmdPack.String("o", "", "enter output dir")
	cmdUnpack := flag.NewFlagSet("unpack", flag.ContinueOnError)

	help := flag.Bool("h", false, "show help message")
	flag.Parse()

	if *help {
		showHelp()
		return
	}

	switch os.Args[1] {
	case "p":
		cmdPack.Parse(os.Args[2:])

		rem := cmdPack.Args()
		if len(rem) == 0 {
			msg := []byte("enter a file.")
			os.Stdout.Write(msg)
			return
		}

		inputFile := strings.Trim(rem[0], " ")
		err := api.PackFile(inputFile, *oDir)

		if err != nil {
			msg := []byte(err.Error())
			os.Stderr.Write(msg)
			return
		}
	case "u":
		cmdUnpack.Parse(os.Args[2:])
		rem := cmdUnpack.Args()

		if len(rem) == 0 {
			msg := []byte("enter filename")
			os.Stdout.Write(msg)
			return
		}

		output := strings.Trim(rem[0], " ")
		err := api.UnpackFile(output)

		if err != nil {
			msg := []byte(err.Error())
			os.Stderr.Write(msg)
			return
		}
	default:
		msg := []byte("unknown command: use goarc -h for help.")
		os.Stdout.Write(msg)
		return
	}
}

func showHelp() {
	os.Stdout.Write([]byte("usage of goarchiver:\n"))
	os.Stdout.Write([]byte("  p [options] <file>\tcompress file\n"))
	os.Stdout.Write([]byte("  u <file>.arc      \tunpack file\n\n"))
	os.Stdout.Write([]byte("options for pack:\n"))
	os.Stdout.Write([]byte("  -o string\tenter output dir\n\n"))
	os.Stdout.Write([]byte("usage example:\tgoarc p -o ./out my.txt\n"))
}
