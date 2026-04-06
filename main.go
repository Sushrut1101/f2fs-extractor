//
// SPDX-FileCopyrightText: Sushrut1101
//
// SPDX-License-Identifier: GPL-3.0-only
//

package main

import (
	"flag"
	"fmt"
	"os"
)

const globalUsage = `F2FS Raw Image Extractor - browse and extract Android F2FS images

Usage:
	f2fs-extractor <command> <image.img> [options] [args...]

Commands:
	help        Display this message
	info        Show filesystem information
	list        List directory contents
	extract     Extract a file or directory (defaults to root '/')

Run 'f2fs-extractor <command> -h' for help on a specific command.
`

func main() {
	if len(os.Args) < 2 {
		fmt.Print(globalUsage)
		os.Exit(1)
	}

	command := os.Args[1]

	// Catch global help requests
	if command == "-h" || command == "--help" || command == "help" {
		fmt.Print(globalUsage)
		os.Exit(0)
	}

	if len(os.Args) < 3 {
		fmt.Fprintln(os.Stderr, "Error: Missing image path.")
		fmt.Print(globalUsage)
		os.Exit(1)
	}

	imagePath := os.Args[2]
	cmdArgs := os.Args[3:]

	if imagePath == "-h" || imagePath == "--help" {
		cmdArgs = []string{"-h"}
		imagePath = ""
	}

	listCmd := flag.NewFlagSet("list", flag.ExitOnError)

	extractCmd := flag.NewFlagSet("extract", flag.ExitOnError)
	extFile := extractCmd.String("file", "/", "Path inside the F2FS image to extract (defaults to root)")

	switch command {
	case "info":
	case "list":
		listCmd.Parse(cmdArgs)
		cmdArgs = listCmd.Args()
	case "extract":
		extractCmd.Parse(cmdArgs)
		cmdArgs = extractCmd.Args()
	default:
		fmt.Printf("Unknown command: %s\n\n", command)
		fmt.Print(globalUsage)
		os.Exit(1)
	}

	if imagePath == "" {
		os.Exit(0)
	}

	if _, err := os.Stat(imagePath); err != nil {
		fmt.Fprintf(os.Stderr, "Error: Image '%s' not found\n", imagePath)
		os.Exit(1)
	}

	reader, err := NewF2FSReader(imagePath)
	if err != nil {
		fmt.Fprintf(os.Stderr, "Error opening image: %v\n", err)
		os.Exit(1)
	}
	defer reader.Close()

	var cmdErr error

	switch command {
	case "info":
		reader.cmdInfo()
	case "list":
		path := "/"
		if len(cmdArgs) > 0 {
			path = cmdArgs[0]
		}
		cmdErr = reader.cmdList(path)
	case "extract":
		dest := "."
		if len(cmdArgs) > 0 {
			dest = cmdArgs[0]
		}
		cmdErr = reader.cmdExtract(*extFile, dest)
	}

	if cmdErr != nil {
		fmt.Fprintf(os.Stderr, "Error: %v\n", cmdErr)
		os.Exit(1)
	}
}
