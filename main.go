package main

import (
	"bytes"
	"fmt"
	"github.com/heyvito/figz/decoder"
	"github.com/heyvito/figz/tikz"
	"github.com/urfave/cli/v2"
	"io"
	"os"
	"strings"
)

func expandTilde(path string) string {
	if !strings.HasPrefix(path, "~") {
		return path
	}
	return strings.Replace(path, "~", os.Getenv("HOME"), 1)
}

func main() {
	app := cli.App{
		Name:        "figz",
		HelpName:    "figz",
		Usage:       "figz [OPTIONS] PATH",
		Version:     tikz.VERSION,
		Description: "Converts figjam files into tikz pictures",
		Flags: []cli.Flag{
			&cli.StringFlag{
				Name:      "output",
				Usage:     "Path to write the generated tikz picture",
				Required:  false,
				Value:     "",
				Aliases:   []string{"o"},
				TakesFile: true,
			},
		},
		Action: run,
		Authors: []*cli.Author{
			{
				Name:  "Vito Sartori",
				Email: "hey@vito.io",
			},
			{
				Name:  "Felipe Mariotti",
				Email: "felipe.mtt95@gmail.com",
			},
		},
		Copyright: "Copyright (c) 2024 Vito Sartori",
	}

	if err := app.Run(os.Args); err != nil {
		panic(err)
	}
}

func run(c *cli.Context) error {
	if c.NArg() == 0 {
		return cli.ShowAppHelp(c)
	}

	input := expandTilde(c.Args().Get(0))
	doc, err := decoder.Decode(input)
	if err != nil {
		_, _ = fmt.Fprintf(os.Stderr, "Failed decoding input file %s: %s\n", input, err)
		os.Exit(1)
	}

	var output io.WriteCloser
	if c.IsSet("output") {
		output, err = os.OpenFile(expandTilde(c.String("output")), os.O_WRONLY|os.O_CREATE|os.O_TRUNC, 0644)
		if err != nil {
			_, _ = fmt.Fprintf(os.Stderr, "Failed opening output file: %v\n", err)
			os.Exit(1)
		}
	} else {
		output = os.Stdout
	}

	str := tikz.NewCompiler(doc.Root.Children[1], &tikz.CompilerOpts{
		FilePath: input,
	})
	_, err = io.Copy(output, bytes.NewBufferString(str))
	if err != nil {
		_, _ = fmt.Fprintf(os.Stderr, "Failed writing output file: %v\n", err)
	}
	_ = output.Close()
	return nil
}
