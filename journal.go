//
// SPDX-FileCopyrightText: Sushrut1101
//
// SPDX-License-Identifier: GPL-3.0-only
//

package main


// F2FS batches recent NAT (node address table) updates into a journal
// embedded in the checkpoint, and only periodically flushes them back into
// the main on-disk NAT table blocks the rest of this reader walks. Right
// after a burst of writes the true location of a just-created inode can live
// only in this journal; the raw table still has a stale or empty entry for
// it. Confirmed against the real kernel implementation
// (fs/f2fs/segment.c:read_compacted_summaries, fs/f2fs/checkpoint.c:
// f2fs_get_valid_checkpoint) and f2fs-tools' f2fs_fs.h struct layout, not
// guessed: without this, resolving those nids from the raw table alone
// returns the wrong block, which readNode now reports as an error rather
// than returning as if it were correct.
const (
	// struct f2fs_checkpoint field byte offsets (f2fs_fs.h). Fixed part is
	// 192 bytes; MAX_ACTIVE_NODE_LOGS=8 and MAX_ACTIVE_DATA_LOGS=8 fields
	// sit between free_segment_count and ckpt_flags.
	cpVerOffset            = 0  // __le64 checkpoint_ver
	cpFlagsOffset          = 132 // __le32 ckpt_flags
	cpPackTotalBlksOffset  = 136 // __le32 cp_pack_total_block_count
	cpPackStartSumOffset   = 140 // __le32 cp_pack_start_sum
	cpCompactSumFlag       = 0x00000004

	// struct f2fs_journal (f2fs_fs.h): __le16 n_nats, followed by a union
	// spare area also used for the SIT journal / extra info. In compact
	// summary mode the whole 507-byte journal sits at byte 0 of the block
	// at start_sum_block, ahead of any per-log summary entries (confirmed
	// in read_compacted_summaries: "Step 1: restore nat cache" copies from
	// kaddr, offset 0, before the summary-entries step that follows it).
	natJournalNNatsOffset  = 0
	natJournalEntriesStart = 2
	natJournalEntrySize    = 13 // __le32 nid + f2fs_nat_entry{u8 version; le32 ino; le32 block_addr}
	// NAT_JOURNAL_ENTRIES = (SUM_JOURNAL_SIZE(507) - 2) / 13
	natJournalMaxEntries = 38
)

// verAfter mirrors the kernel's ver_after() macro: a is more recent than b,
// wraparound-safe via signed subtraction.
func verAfter(a, b uint64) bool {
	return int64(a-b) > 0
}

// checkpointPack is one of the two redundant, self-mirrored checkpoint
// copies F2FS keeps for crash consistency.
type checkpointPack struct {
	startAddr     uint32
	version       uint64
	flags         uint32
	packStartSum  uint32
	valid         bool
}

// readCheckpointPack reads and internally validates one checkpoint pack: its
// header block must agree with its own footer copy (the last block of the
// pack, a repeated header) on the version number, the same consistency check
// f2fs_get_valid_checkpoint performs before trusting a pack.
func (r *F2FSReader) readCheckpointPack(startAddr uint32) checkpointPack {
	header, err := r.readBlock(startAddr)
	if err != nil {
		return checkpointPack{}
	}
	version := le64(header, cpVerOffset)
	totalBlocks := le32(header, cpPackTotalBlksOffset)
	// Sanity bound: a pack can't be larger than one segment nor smaller
	// than the two mirrored header/footer blocks it must contain.
	if totalBlocks <= 2 || totalBlocks > uint32(r.blocksPerSeg) {
		return checkpointPack{}
	}
	footerAddr := startAddr + totalBlocks - 1
	footer, err := r.readBlock(footerAddr)
	if err != nil {
		return checkpointPack{}
	}
	if le64(footer, cpVerOffset) != version {
		return checkpointPack{}
	}
	return checkpointPack{
		startAddr:    startAddr,
		version:      version,
		flags:        le32(header, cpFlagsOffset),
		packStartSum: le32(header, cpPackStartSumOffset),
		valid:        true,
	}
}

// loadNATJournal locates the current checkpoint pack and, when it uses the
// compact summary format, parses the NAT journal entries out of it. Returns
// nil (not an error) when no journal is available or it uses the
// non-compact per-log summary format this reader does not parse; readNode's
// existing raw-table path remains the fallback for those nids either way.
func (r *F2FSReader) loadNATJournal() map[uint32]uint32 {
	pack1 := r.readCheckpointPack(r.sb.CPBlkAddr)
	pack2 := r.readCheckpointPack(r.sb.CPBlkAddr + uint32(r.blocksPerSeg))

	var current checkpointPack
	switch {
	case pack1.valid && pack2.valid:
		if verAfter(pack2.version, pack1.version) {
			current = pack2
		} else {
			current = pack1
		}
	case pack1.valid:
		current = pack1
	case pack2.valid:
		current = pack2
	default:
		return nil
	}

	if current.flags&cpCompactSumFlag == 0 {
		// Non-compact (per-log) summary format: the NAT journal is split
		// across up to 3 separate node-log summary blocks instead of one
		// merged block, a different layout this reader does not parse.
		return nil
	}

	sumBlkAddr := current.startAddr + current.packStartSum
	block, err := r.readBlock(sumBlkAddr)
	if err != nil {
		return nil
	}

	nNats := int(le16(block, natJournalNNatsOffset))
	if nNats > natJournalMaxEntries {
		nNats = natJournalMaxEntries
	}
	journal := make(map[uint32]uint32, nNats)
	for i := 0; i < nNats; i++ {
		off := natJournalEntriesStart + i*natJournalEntrySize
		nid := le32(block, off)
		// entry layout: nid(4) + ne{version(1) + ino(4) + block_addr(4)}.
		// block_addr sits at off+9 (4 for nid, 1 for version, 4 for ino),
		// not off+5 -- that offset is only right for the RAW nat table's
		// f2fs_nat_entry, which has no leading nid field of its own.
		blkAddr := le32(block, off+9)
		if blkAddr != 0 && blkAddr != NewAddr && blkAddr < uint32(r.sb.BlockCount) {
			journal[nid] = blkAddr
		}
	}
	return journal
}
