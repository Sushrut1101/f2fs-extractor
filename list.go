//
// SPDX-FileCopyrightText: Sushrut1101
//
// SPDX-License-Identifier: GPL-3.0-only
//

package main

import (
	"fmt"
	"path/filepath"
	"sort"
	"strings"
	"time"
)

func (r *F2FSReader) cmdList(path string) error {
	nid, in, err := r.resolvePath(path)
	if err != nil {
		return fmt.Errorf("ls: %w", err)
	}

	if !isDir(in.IMode) {
		mtime := time.Unix(int64(in.IMtime), 0).Format("2006-01-02 15:04")
		fmt.Printf("%s %5d %5d %10d %s %s\n", formatMode(in.IMode), in.IUID, in.IGID, in.ISize, mtime, filepath.Base(path))
		return nil
	}

	entries, err := r.listDir(nid)
	if err != nil {
		return err
	}
	if len(entries) == 0 {
		fmt.Println("(empty directory)")
		return nil
	}

	sort.Slice(entries, func(i, j int) bool {
		ai, aj := entries[i], entries[j]
		if di, dj := ai.FileType == FTDir, aj.FileType == FTDir; di != dj {
			return di
		}
		return ai.Name < aj.Name
	})

	for _, e := range entries {
		child, err := r.getInode(e.Ino)
		if err != nil || child == nil {
			ch, ok := fileTypeChar[e.FileType]
			if !ok {
				ch = '?'
			}
			fmt.Printf("%c?????????     ?     ?          ? ???????????? %s\n", ch, e.Name)
			continue
		}
		mtime := time.Unix(int64(child.IMtime), 0).Format("2006-01-02 15:04")
		suffix := ""
		if e.FileType == FTDir {
			suffix = "/"
		} else if e.FileType == FTSymlink {
			target, _ := r.readFile(e.Ino, 4096)
			suffix = " -> " + strings.TrimRight(string(target), "\x00")
		}
		fmt.Printf("%s %5d %5d %10d %s %s%s\n", formatMode(child.IMode), child.IUID, child.IGID, child.ISize, mtime, e.Name, suffix)
	}
	return nil
}
