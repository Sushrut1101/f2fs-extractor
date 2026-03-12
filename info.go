//
// SPDX-FileCopyrightText: Sushrut1101
//
// SPDX-License-Identifier: GPL-3.0-only
//

package main

import (
	"fmt"
	"strings"
)

func (r *F2FSReader) cmdInfo() {
	sb := r.sb
	totalMB := float64(sb.BlockCount) * float64(r.blockSize) / (1024 * 1024)
	fmt.Printf("F2FS Filesystem Info\n")
	fmt.Printf("%s\n", strings.Repeat("=", 50))
	fmt.Printf("  Version:          %d.%d\n", sb.MajorVer, sb.MinorVer)
	fmt.Printf("  Volume:           %s\n", sb.VolumeName)
	fmt.Printf("  Block size:       %d\n", r.blockSize)
	fmt.Printf("  Blocks/segment:   %d\n", r.blocksPerSeg)
	fmt.Printf("  Total blocks:     %d\n", sb.BlockCount)
	fmt.Printf("  Total size:       %.1f MB\n", totalMB)
	fmt.Printf("  Segments:         %d\n", sb.SegmentCount)
	fmt.Printf("  Main segments:    %d\n", sb.SegmentCountMain)
	fmt.Printf("  Root inode:       %d\n", sb.RootIno)
	fmt.Printf("  NAT segments:     %d\n", sb.SegmentCountNAT)
	fmt.Printf("  SIT segments:     %d\n", sb.SegmentCountSIT)
	fmt.Printf("  CP blkaddr:       %d\n", sb.CPBlkAddr)
	fmt.Printf("  NAT blkaddr:      %d\n", sb.NATBlkAddr)
	fmt.Printf("  Main blkaddr:     %d\n", sb.MainBlkAddr)
}
