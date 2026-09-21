//
// SPDX-FileCopyrightText: Sushrut1101
//
// SPDX-License-Identifier: GPL-3.0-only
//

package main

import (
	"bytes"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"time"
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

// applyMetadata restores the modification time this inode recorded on the
// real device, and a safe derivative of its permission bits, onto the
// freshly-written file or directory at outPath. Best-effort and
// silent-on-failure: this runs after the content is already written, so a
// metadata problem (e.g. an unrepresentable mode bit) must never turn into a
// lost file.
func applyMetadata(outPath string, in *Inode) {
	os.Chmod(outPath, safeExtractedMode(in.IMode))
	mtime := time.Unix(int64(in.IMtime), 0)
	os.Chtimes(outPath, mtime, mtime)
}

// safeExtractedMode derives a permission mode for extracted output from the
// original inode's mode, not a verbatim copy of it. setuid/setgid/sticky
// bits are dropped: replaying a setuid-root bit from firmware onto a file in
// a shared analysis tree is a real local-privesc surface, not just a
// metadata detail. Read (and, for a directory, search) for everyone is
// always guaranteed on top of whatever the original bits were, so a later
// pipeline stage reading the extracted tree under a different uid than
// whoever ran the extractor can still open every file, the same guarantee
// this project's other extractors already give: ext4-backed firmware goes
// through debugfs's dump/rdump, which writes extracted output at the
// invoking process's default (typically world-readable) umask rather than
// replaying the original device's mode bits at all.
func safeExtractedMode(iMode uint16) os.FileMode {
	perm := os.FileMode(iMode & 0o777)
	perm |= 0o644
	if isDir(iMode) {
		perm |= 0o755
	}
	return perm
}

// Clean F2FS binary padding to prevent Linux syscall crashes
func parseSymlinkTarget(data []byte) string {
	if idx := bytes.IndexByte(data, 0); idx >= 0 {
		data = data[:idx]
	}
	return strings.TrimSpace(string(data))
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

	in, err := r.getInode(nid)
	if err != nil {
		return err
	}
	data, err := r.readFile(nid, -1)
	if err != nil {
		return err
	}
	if err := os.WriteFile(outPath, data, 0644); err != nil {
		return err
	}
	applyMetadata(outPath, in)
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
			// extractDir applies its own directory's metadata itself, once
			// every child underneath it (which would otherwise bump its
			// mtime back to "now") is already written.
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
			childIn, err := r.getInode(e.Ino)
			if err != nil {
				fmt.Fprintf(os.Stderr, "Warning: read inode %s: %v\n", childSrc, err)
				continue
			}
			data, err := r.readFile(e.Ino, -1)
			if err != nil {
				fmt.Fprintf(os.Stderr, "Warning: read %s: %v\n", childSrc, err)
				continue
			}
			if err := os.WriteFile(childDst, data, 0644); err != nil {
				fmt.Fprintf(os.Stderr, "Warning: write %s: %v\n", childDst, err)
			} else {
				applyMetadata(childDst, childIn)
				fmt.Printf("Extracted: %s (%d bytes)\n", childDst, len(data))
			}
		}
	}

	if in, err := r.getInode(nid); err == nil {
		applyMetadata(dest, in)
	}
	return nil
}
