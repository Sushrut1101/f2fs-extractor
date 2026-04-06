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
	if isSymlink(in.IMode) {
		return r.extractSymlink(nid, srcPath, dest)
	}
	return r.extractFile(nid, srcPath, dest)
}

// Clean F2FS binary padding to prevent Linux syscall crashes
func parseSymlinkTarget(data []byte) string {
	// Aggressively trim null bytes and whitespace from BOTH ends of the string
	target := strings.Trim(string(data), "\x00\n\r\t ")

	// If there is still a rogue null byte in the middle of the string, it will
	// crash the Linux kernel syscall with 'invalid argument' (EINVAL). Cut it off.
	if idx := strings.IndexByte(target, 0); idx >= 0 {
		target = target[:idx]
	}

	return target
}

func (r *F2FSReader) extractSymlink(nid uint32, srcPath, dest string) error {
	outPath := dest
	if info, err := os.Stat(outPath); err == nil && info.IsDir() {
		outPath = filepath.Join(outPath, filepath.Base(srcPath))
	} else if outPath == "" || outPath == "." {
		outPath = filepath.Base(srcPath)
	}

	if err := os.MkdirAll(filepath.Dir(outPath), 0755); err != nil {
		return err
	}

	in, err := r.getInode(nid)
	if err != nil {
		return err
	}

	targetData, err := r.readFile(nid, int64(in.ISize))
	if err != nil {
		return err
	}

	targetStr := parseSymlinkTarget(targetData)
	if targetStr == "" {
		return fmt.Errorf("symlink target is empty after parsing")
	}

	_ = os.RemoveAll(outPath)
	if err := os.Symlink(targetStr, outPath); err != nil {
		return fmt.Errorf("failed to create symlink %s -> %s: %w", outPath, targetStr, err)
	}

	fmt.Printf("Extracted symlink: %s -> %s\n", outPath, targetStr)
	return nil
}

func (r *F2FSReader) extractFile(nid uint32, srcPath, dest string) error {
	outPath := dest
	if info, err := os.Stat(outPath); err == nil && info.IsDir() {
		outPath = filepath.Join(outPath, filepath.Base(srcPath))
	} else if outPath == "" || outPath == "." {
		outPath = filepath.Base(srcPath)
	}

	if err := os.MkdirAll(filepath.Dir(outPath), 0755); err != nil {
		return err
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
	if dest == "" || dest == "." {
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
			childIn, err := r.getInode(e.Ino)
			if err != nil {
				fmt.Fprintf(os.Stderr, "Warning: read symlink inode %s: %v\n", childSrc, err)
				continue
			}

			targetData, _ := r.readFile(e.Ino, int64(childIn.ISize))
			targetStr := parseSymlinkTarget(targetData)

			if targetStr == "" {
				fmt.Fprintf(os.Stderr, "Warning: empty target for symlink %s\n", childDst)
				continue
			}

			// Ensure parent directory exists before creating symlink
			os.MkdirAll(filepath.Dir(childDst), 0755)

			_ = os.RemoveAll(childDst)
			if err := os.Symlink(targetStr, childDst); err != nil {
				fmt.Fprintf(os.Stderr, "Warning: symlink %s -> %s: %v\n", childDst, targetStr, err)
			} else {
				fmt.Printf("Symlink: %s -> %s\n", childDst, targetStr)
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
				fmt.Printf("Extracted: %s (%d bytes)\n", childDst, len(data))
			}
		}
	}
	return nil
}
