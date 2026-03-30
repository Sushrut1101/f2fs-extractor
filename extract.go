//
// SPDX-FileCopyrightText: Sushrut1101
//
// SPDX-License-Identifier: GPL-3.0-only
//

package main

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"
)

func (r *F2FSReader) cmdExtract(srcPath, dest string) error {
	nid, in, err := r.resolvePath(srcPath)
	if err != nil {
		return err
	}

	if isDir(in.IMode) {
		return r.extractDir(nid, srcPath, dest)
	}
	return r.extractFile(nid, srcPath, dest)
}

func (r *F2FSReader) extractFile(nid uint32, srcPath, dest string) error {
	outPath := dest
	// If the destination is an existing directory, put the file inside it
	if info, err := os.Stat(outPath); err == nil && info.IsDir() {
		outPath = filepath.Join(outPath, filepath.Base(srcPath))
	} else if outPath == "" {
		outPath = filepath.Base(srcPath)
	}

	data, err := r.readFile(nid, -1)
	if err != nil {
		return err
	}
	if err := os.WriteFile(outPath, data, 0644); err != nil {
		return err
	}
	fmt.Printf("Extracted file: %s -> %s (%d bytes)\n", srcPath, outPath, len(data))
	return nil
}

func (r *F2FSReader) extractDir(nid uint32, srcPath, dest string) error {
	if dest == "" {
		dest = "extracted_root"
	}
	if err := os.MkdirAll(dest, 0755); err != nil {
		return err
	}

	entries, err := r.listDir(nid)
	if err != nil {
		return err
	}
	for _, e := range entries {
		if e.Name == "." || e.Name == ".." {
			continue
		}
		childSrc := strings.TrimRight(srcPath, "/") + "/" + e.Name
		childDst := filepath.Join(dest, e.Name)

		switch e.FileType {
		case FTDir:
			if err := r.extractDir(e.Ino, childSrc, childDst); err != nil {
				fmt.Fprintf(os.Stderr, "Warning: extractdir %s: %v\n", childSrc, err)
			}
		case FTSymlink:
			target, _ := r.readFile(e.Ino, 4096)
			targetStr := strings.TrimRight(string(target), "\x00")
			_ = os.Remove(childDst)
			if err := os.Symlink(targetStr, childDst); err != nil {
				fmt.Fprintf(os.Stderr, "Warning: symlink %s -> %s: %v\n", childDst, targetStr, err)
			} else {
				fmt.Printf("Symlink: %s -> %s\n", childSrc, targetStr)
			}
		default:
			data, err := r.readFile(e.Ino, -1)
			if err != nil {
				fmt.Fprintf(os.Stderr, "Warning: read %s: %v\n", childSrc, err)
				continue
			}
			if err := os.WriteFile(childDst, data, 0644); err != nil {
				fmt.Fprintf(os.Stderr, "Warning: write %s: %v\n", childDst, err)
			} else {
				fmt.Printf("Extracted: %s (%d bytes)\n", childSrc, len(data))
			}
		}
	}
	return nil
}
