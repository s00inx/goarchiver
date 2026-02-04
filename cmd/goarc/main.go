package main

import (
	"flag"
	"fmt"
	"os"
	"strings"

	api "github.com/kfcemployee/goarchiver/internal"
)

func main() {
	cmdPack := flag.NewFlagSet("pack", flag.ExitOnError)
	oDir := cmdPack.String("o", "", "указать выходную директорию")

	cmdUnpack := flag.NewFlagSet("unpack", flag.ExitOnError)

	if len(os.Args) < 2 {
		fmt.Fprintln(os.Stdout, "нужна команда : pack или unpack")
		return
	}

	switch os.Args[1] {
	case "pack":
		cmdPack.Parse(os.Args[2:])

		rem := cmdPack.Args()
		if len(rem) == 0 {
			fmt.Fprintln(os.Stderr, "укажите файл")
			return
		}

		inputFile := strings.Trim(rem[0], " ")
		arcDir, err := api.PackFile(inputFile, *oDir)

		if err != nil {
			fmt.Fprintf(os.Stderr, "error packing a file: %s", err)
			return
		}
		fmt.Fprintf(os.Stdout, "сжато успешно. путь к архиву: %s\n", arcDir)
	case "unpack":
		cmdUnpack.Parse(os.Args[2:])
		rem := cmdUnpack.Args()

		if len(rem) == 0 {
			fmt.Fprintln(os.Stderr, "укажите файл")
			return
		}

		output := strings.Trim(rem[0], " ")
		err := api.UnpackFile(output)

		if err != nil {
			fmt.Fprintf(os.Stderr, "ошибка распаковки: %s", err)
			return
		}
		fmt.Fprintf(os.Stdout, "успешно распаковано.")
	default:
		fmt.Fprintf(os.Stderr, "неизвестная команда, список команд: -h/")
		return
	}

}
