//
// SPDX-FileCopyrightText: Sushrut1101
//
// SPDX-License-Identifier: GPL-3.0-only
//

package main

import (
	"fmt"
	"os"
)

const globalUsage = `F2FS Raw Image Extractor - browse and extract Android F2FS images

Usage:
	f2fs-extractor <command> <image.img> [options] [args...]

Commands:
	help        Display this message

Run 'f2fs-extractor <command> -h' for help on a specific command.
`

func main() {
	if len(os.Args) < 2 {
		fmt.Print(globalUsage)
		os.Exit(1)
	}

	command := os.Args[1]

	// Catch global help requests
	if command == "-h" || command == "--help" {
		fmt.Print(globalUsage)
		os.Exit(0)
	}

	if len(os.Args) < 3 {
		fmt.Fprintln(os.Stderr, "Error: Missing image path.")
		fmt.Print(globalUsage)
		os.Exit(1)
	}

	imagePath := os.Args[2]

	switch command {
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
}
