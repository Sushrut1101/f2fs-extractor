//
// SPDX-FileCopyrightText: Sushrut1101
//
// SPDX-License-Identifier: GPL-3.0-only
//

package main

import (
	"encoding/binary"
	"fmt"
	"io"
	"os"
	"unicode/utf16"
)

const (
	F2FSMagic            uint32 = 0xF2F52010
	F2FSBlkSize                 = 4096
	F2FSSBOffset                = 1024
	F2FSNameLen                 = 255
	F2FSSlotLen                 = 8
	MaxActiveLogs               = 16
	F2FSInlineXAttr             = 0x01
	F2FSInlineData              = 0x02
	F2FSInlineDentry            = 0x04
	F2FSDataExist               = 0x08
	F2FSInlineDots              = 0x10
	F2FSExtraAttr               = 0x20
	FTUnknown                   = 0
	FTRegFile                   = 1
	FTDir                       = 2
	FTChrDev                    = 3
	FTBlkDev                    = 4
	FTFIFO                      = 5
	FTSock                      = 6
	FTSymlink                   = 7
	NullAddr             uint32 = 0x00000000
	NewAddr              uint32 = 0xFFFFFFFF
	CompressAddr         uint32 = 0xFFFFFFFE
	CPLargeNATBitmapFlag        = 0x00000400
	NATEntrySize                = 9
	DirEntrySize                = 11
	NRDentryInBlock             = 214
	DentryBitmapSize            = 27
	NodeFooterSize              = 24
)

type Superblock struct {
	Magic, LogSectorSize, LogSectorsPerBlk, LogBlocksize, LogBlocksPerSeg, SegsPerSec, SecsPerZone                                            uint32
	ChecksumOffset, SectionCount, SegmentCount, SegmentCountCkpt, SegmentCountSIT, SegmentCountNAT                                            uint32
	SegmentCountSSA, SegmentCountMain, Segment0BlkAddr, CPBlkAddr, SITBlkAddr, NATBlkAddr, SSABlkAddr, MainBlkAddr, RootIno, NodeIno, MetaIno uint32
	MajorVer, MinorVer                                                                                                                        uint16
	BlockCount                                                                                                                                uint64
	VolumeName                                                                                                                                string
}

type Inode struct {
	IMode                                             uint16
	IAdvise, IInline                                  uint8
	IUID, IGID, ILinks                                uint32
	ISize, IBlocks, IAtime, ICtime, IMtime            uint64
	ICurrentDepth, IXAttrNID, IFlags, IPino, INameLen uint32
	IName                                             string
	IAddr                                             []uint32
	INID                                              [5]uint32
	FooterNID, FooterIno                              uint32
	addrStart                                         int
	extraISize, inlineXAttrSize                       uint16
	numAddrs, totalAddrSlots                          int
	raw                                               []byte
}

type DirEntry struct {
	Name     string
	Ino      uint32
	FileType int
}

type F2FSReader struct {
	path                                        string
	f                                           *os.File
	sb                                          Superblock
	natBitmap                                   []byte
	blockSize, blocksPerSeg, natEntriesPerBlock int
}

func NewF2FSReader(imagePath string) (*F2FSReader, error) {
	f, err := os.Open(imagePath)
	if err != nil {
		return nil, fmt.Errorf("open: %w", err)
	}
	r := &F2FSReader{path: imagePath, f: f, blockSize: F2FSBlkSize}
	if err := r.parseSuperblock(); err != nil {
		f.Close()
		return nil, err
	}
	return r, nil
}

func (r *F2FSReader) Close() { r.f.Close() }

func (r *F2FSReader) readAt(offset int64, size int) ([]byte, error) {
	buf := make([]byte, size)
	n, err := r.f.ReadAt(buf, offset)
	if err != nil && err != io.EOF {
		return nil, err
	}
	return buf[:n], nil
}

func le16(b []byte, off int) uint16 { return binary.LittleEndian.Uint16(b[off:]) }
func le32(b []byte, off int) uint32 { return binary.LittleEndian.Uint32(b[off:]) }
func le64(b []byte, off int) uint64 { return binary.LittleEndian.Uint64(b[off:]) }

func (r *F2FSReader) parseSuperblock() error {
	data, err := r.readAt(F2FSSBOffset, 512)
	if err != nil {
		return fmt.Errorf("read superblock: %w", err)
	}
	magic := le32(data, 0)
	if magic != F2FSMagic {
		return fmt.Errorf("not an F2FS image")
	}
	sb := &r.sb
	sb.Magic = magic
	sb.MajorVer = le16(data, 4)
	sb.MinorVer = le16(data, 6)
	sb.LogSectorSize = le32(data, 8)
	sb.LogSectorsPerBlk = le32(data, 12)
	sb.LogBlocksize = le32(data, 16)
	sb.LogBlocksPerSeg = le32(data, 20)
	sb.SegsPerSec = le32(data, 24)
	sb.SecsPerZone = le32(data, 28)
	sb.ChecksumOffset = le32(data, 32)
	sb.BlockCount = le64(data, 36)
	sb.SectionCount = le32(data, 44)
	sb.SegmentCount = le32(data, 48)
	sb.SegmentCountCkpt = le32(data, 52)
	sb.SegmentCountSIT = le32(data, 56)
	sb.SegmentCountNAT = le32(data, 60)
	sb.SegmentCountSSA = le32(data, 64)
	sb.SegmentCountMain = le32(data, 68)
	sb.Segment0BlkAddr = le32(data, 72)
	sb.CPBlkAddr = le32(data, 76)
	sb.SITBlkAddr = le32(data, 80)
	sb.NATBlkAddr = le32(data, 84)
	sb.SSABlkAddr = le32(data, 88)
	sb.MainBlkAddr = le32(data, 92)
	sb.RootIno = le32(data, 96)
	sb.NodeIno = le32(data, 100)
	sb.MetaIno = le32(data, 104)

	volData, _ := r.readAt(F2FSSBOffset+0x6C, 512)
	if len(volData) >= 2 {
		u16s := make([]uint16, len(volData)/2)
		for i := range u16s {
			u16s[i] = binary.LittleEndian.Uint16(volData[i*2:])
		}
		end := len(u16s)
		for end > 0 && u16s[end-1] == 0 {
			end--
		}
		sb.VolumeName = string(utf16.Decode(u16s[:end]))
	}
	r.blockSize = 1 << sb.LogBlocksize
	r.blocksPerSeg = 1 << sb.LogBlocksPerSeg
	r.natEntriesPerBlock = r.blockSize / NATEntrySize
	return nil
}
