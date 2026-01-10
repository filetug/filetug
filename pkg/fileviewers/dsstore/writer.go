package dsstore

import (
	"bytes"
	"encoding/binary"
	"errors"
	"io"
	"os"
	"sort"

	"golang.org/x/text/encoding/unicode"
	"golang.org/x/text/transform"
)

type freeBlock struct {
	offset uint32
	size   uint32
}

func (s *Store) writeFreeMapCreate() []freeBlock {
	freeBlocks := make([]freeBlock, 0)
	for i := 5; i < 31; i++ {
		pow := uint32(1 << i)
		freeBlocks = append(freeBlocks, freeBlock{pow, pow})
	}
	return freeBlocks
}

func (s *Store) writeFreeMapSort(freeBlocks []freeBlock) {
	sort.SliceStable(freeBlocks, func(i, j int) bool {
		if freeBlocks[i].size < freeBlocks[j].size {
			return true
		}
		if freeBlocks[i].size < freeBlocks[j].size {
			return freeBlocks[i].offset < freeBlocks[j].offset
		}
		return false
	})
}

func (s *Store) writeFreeMapAlloc(freeBlocks []freeBlock, size uint32, capacity uint32) (uint32, []freeBlock) {
	// check capacity
	if capacity < size {
		capacity = size
	}
	// calculate size needed size that powered by 2
	var powCapacity uint32 = 0
	var powSize uint32 = 0
	var powIndex uint32 = 32
	for i := uint32(5); i < 32; i++ {
		powCapacity = uint32(1) << i
		if powCapacity >= size && powIndex >= 32 {
			powIndex = i
			powSize = powCapacity
		}
		if powCapacity >= capacity {
			if powIndex < 32 {
				break
			}
		}
	}
	// sort map
	s.writeFreeMapSort(freeBlocks)
	// find block in partly used blocks
	for i, block := range freeBlocks {
		if block.size != block.offset && block.size == powCapacity {
			freeBlocks = append(freeBlocks[:i], freeBlocks[i+1:]...)
			if block.size > powSize {
				freeBlocks = append(freeBlocks, freeBlock{block.offset + powSize, block.size - powSize})
			}
			return block.offset | powIndex, freeBlocks
		}
	}
	// find block in fully cleared blocks
	for i, block := range freeBlocks {
		if block.size >= powCapacity {
			freeBlocks = append(freeBlocks[:i], freeBlocks[i+1:]...)
			if block.size > powSize {
				freeBlocks = append(freeBlocks, freeBlock{block.offset + powSize, block.size - powSize})
			}
			return block.offset | powIndex, freeBlocks
		}
	}
	return 0, freeBlocks
}

func (s *Store) writeAlignBlock(b *bytes.Buffer, minSize uint32) error {
	var lenBytes = uint32(b.Len())
	for i := 0; i < 32; i++ {
		var aligned uint32 = 1 << i
		if lenBytes <= aligned && minSize <= aligned {
			// add dummy bytes
			if _, err := b.Write(make([]byte, aligned-lenBytes)); err != nil {
				return err
			}
			break
		}
	}
	return nil
}

func (s *Store) writeBlockData(b *bytes.Buffer, records []Record) error {
	// nextBlock is always 0. Storing all records in the one block
	if err := binary.Write(b, binary.BigEndian, uint32(0)); err != nil {
		return err
	}
	// count of records
	if err := binary.Write(b, binary.BigEndian, uint32(len(records))); err != nil {
		return err
	}
	// records
	for _, r := range records {
		// r.FileName
		n, _, err := transform.Bytes(unicode.UTF16(unicode.BigEndian, unicode.IgnoreBOM).NewEncoder(), []byte(r.FileName))
		if err != nil {
			return err
		}
		if err := binary.Write(b, binary.BigEndian, uint32(len(n)/2)); err != nil {
			return err
		}
		if _, err := b.Write(n); err != nil {
			return err
		}
		// unknown extra 4 bytes
		if err := binary.Write(b, binary.BigEndian, uint32(r.Extra)); err != nil {
			return err
		}
		// r.Type (4-bytes string)
		t := make([]byte, 4)
		copy(t, []byte(r.Type))
		if _, err := b.Write(t); err != nil {
			return err
		}
		// r.DataLen for blob, ustr etc
		if r.DataLen > 0 {
			if err := binary.Write(b, binary.BigEndian, uint32(r.DataLen)); err != nil {
				return err
			}
		}
		// r.Data
		if _, err := b.Write(r.Data); err != nil {
			return err
		}
	}
	return nil
}

func (s *Store) writeBlockDSDB(b *bytes.Buffer, index uint32) error {
	// write data block index
	err := binary.Write(b, binary.BigEndian, index)
	if err != nil {
		return err
	}
	// levels. always is 0
	if err = binary.Write(b, binary.BigEndian, uint32(0)); err != nil {
		return err
	}
	// records
	if err = binary.Write(b, binary.BigEndian, uint32(len(s.Records))); err != nil {
		return err
	}
	// nodes. always is 1 (storing all data in one data block)
	if err = binary.Write(b, binary.BigEndian, uint32(1)); err != nil {
		return err
	}
	// dummy 0x1000 value
	if err = binary.Write(b, binary.BigEndian, uint32(0x1000)); err != nil {
		return err
	}
	// other unknown data
	if _, err := b.Write(s.DSDBExtra); err != nil {
		return err
	}
	return nil
}

func (s *Store) writeOffsets(b *bytes.Buffer, offsetRoot, offsetDBDS, offsetData uint32) error {
	// count of offset. always 3 blocks only
	if err := binary.Write(b, binary.BigEndian, uint32(3)); err != nil {
		return err
	}
	// dummy 4 bytes
	if err := binary.Write(b, binary.BigEndian, uint32(0)); err != nil {
		return err
	}
	// offsets
	if err := binary.Write(b, binary.BigEndian, offsetRoot); err != nil {
		return err
	}
	if err := binary.Write(b, binary.BigEndian, offsetDBDS); err != nil {
		return err
	}
	if err := binary.Write(b, binary.BigEndian, offsetData); err != nil {
		return err
	}
	// dummy 253 zero offsets
	for i := 0; i < 253; i++ {
		if err := binary.Write(b, binary.BigEndian, uint32(0)); err != nil {
			return err
		}
	}
	return nil
}

func (s *Store) writeTopicDBDS(b *bytes.Buffer, index uint32) error {
	// always one topic (DBDS)
	if err := binary.Write(b, binary.BigEndian, uint32(1)); err != nil {
		return err
	}
	// topic name len. Always is 4
	if err := b.WriteByte(4); err != nil {
		return err
	}
	// topic name
	if _, err := b.Write([]byte("DSDB")); err != nil {
		return err
	}
	// topic offset
	if err := binary.Write(b, binary.BigEndian, uint32(index)); err != nil {
		return err
	}
	return nil
}

func (s *Store) writeFreeBlocks(b *bytes.Buffer, freeBlocks []freeBlock) error {
	// sort blocks by size
	s.writeFreeMapSort(freeBlocks)
	// it is magic
	for i := 0; i < 32; i++ {
		// current block size (iterating block sizes 1, 2, 4, 8, 16, ..., 1024, 2048, 4096, ...)
		var blockSize = uint32(1) << i
		var blockCount uint32 = 0
		for j := 0; j < len(freeBlocks); j++ {
			power := (freeBlocks[j].size & (freeBlocks[j].size - 1)) == 0
			// we need take blocks with size == blockSize
			// but also we take all blocks bigger than blockSize that are not round by 2 power
			//
			// For example we have the following blocks allocation (globally 2 blocks - 2048-4096, 4096-8092):
			// |             |####1024####                 |
			// 2048          4096                          8092
			//
			// Used space is 4096-5120. Free space are 2048-4095 and 5121-8091 (sub part of block 4096-8092).
			// Free blocks for blockSize == 2048 are 2048-4096 and 5120-8092 (because it can't be assinged to blocks with size >= 4096)
			// Free blocks for blockSize == 4096 are none
			// here we just calculate count of free blocks with needed size
			if freeBlocks[j].size == blockSize || (freeBlocks[j].size > blockSize && !power) {
				blockCount++
				continue
			}
			if freeBlocks[j].size >= blockSize {
				break
			}
		}
		if err := binary.Write(b, binary.BigEndian, uint32(blockCount)); err != nil {
			return err
		}
		if blockCount > 0 {
			// writing blocks offsets
			for j := 0; j < len(freeBlocks); j++ {
				power := (freeBlocks[j].size & (freeBlocks[j].size - 1)) == 0
				if freeBlocks[j].size == blockSize || (freeBlocks[j].size > blockSize && !power) {
					if err := binary.Write(b, binary.BigEndian, uint32(freeBlocks[j].offset)); err != nil {
						return err
					}
					continue
				}
				if freeBlocks[j].size >= blockSize {
					break
				}
			}
		}
	}
	return nil
}

func (s *Store) writeBlockRoot(b *bytes.Buffer, offsetRoot, offsetDBDS, offsetData uint32, freeBlocks []freeBlock) error {
	// offsets
	if err := s.writeOffsets(b, offsetRoot, offsetDBDS, offsetData); err != nil {
		return err
	}
	// topic. index is always 1
	if err := s.writeTopicDBDS(b, 1); err != nil {
		return err
	}
	// free blocks
	if err := s.writeFreeBlocks(b, freeBlocks); err != nil {
		return err
	}
	// write extra (unknown data)
	if _, err := b.Write(s.RootExtra); err != nil {
		return err
	}
	return nil
}

func (s *Store) writeHeader(b *bytes.Buffer, offsetRoot, size uint32) error {
	// magic1
	if err := binary.Write(b, binary.BigEndian, headerMagic1); err != nil {
		return err
	}
	// magic2
	if err := binary.Write(b, binary.BigEndian, headerMagic2); err != nil {
		return err
	}
	// offset of root block
	if err := binary.Write(b, binary.BigEndian, offsetRoot); err != nil {
		return err
	}
	// size of root block
	if err := binary.Write(b, binary.BigEndian, size); err != nil {
		return err
	}
	// offset of root block
	if err := binary.Write(b, binary.BigEndian, offsetRoot); err != nil {
		return err
	}
	// write extra
	headerExtra := make([]byte, 16)
	copy(headerExtra, s.HeaderExtra)
	if _, err := b.Write(headerExtra); err != nil {
		return err
	}
	// check size
	if b.Len() != 36 {
		return errors.New("invalid header size")
	}
	return nil
}

// WriteStore writes .DS_Store to io.Writer
func (s *Store) Write(w io.Writer) error {
	// prepare data block
	blockData := new(bytes.Buffer)
	if err := s.writeBlockData(blockData, s.Records); err != nil {
		return err
	}
	// prepare DSDB block (always 2 index)
	blockDSDB := new(bytes.Buffer)
	if err := s.writeBlockDSDB(blockDSDB, 2); err != nil {
		return err
	}
	// prepare Root block
	blockList := s.writeFreeMapCreate()
	blockRoot := new(bytes.Buffer)
	if err := s.writeBlockRoot(blockRoot, 0, 0, 0, blockList); err != nil {
		return err
	}
	// align blocks
	if err := s.writeAlignBlock(blockData, 32); err != nil {
		return err
	}
	if err := s.writeAlignBlock(blockDSDB, 32); err != nil {
		return err
	}
	if err := s.writeAlignBlock(blockRoot, 32); err != nil {
		return err
	}
	// create blocks map
	blockDataOffset, blockList := s.writeFreeMapAlloc(blockList, uint32(blockData.Len()), 0)
	blockDSDBOffset, blockList := s.writeFreeMapAlloc(blockList, uint32(blockDSDB.Len()), 0)
	blockRootOffset, blockList := s.writeFreeMapAlloc(blockList, uint32(blockRoot.Len()), 0)
	// calc real offset
	blockDataOffsetReal := blockOffset(blockDataOffset)
	blockDSDBOffsetReal := blockOffset(blockDSDBOffset)
	blockRootOffsetReal := blockOffset(blockRootOffset)
	// calc end of blocks
	blockDataEnd := blockDataOffsetReal + blockSize(blockDataOffset)
	blockDSDBEnd := blockDSDBOffsetReal + blockSize(blockDSDBOffset)
	blockRootEnd := blockRootOffsetReal + blockSize(blockRootOffset)
	// re-create root block with correct offsets
	blockRoot.Reset()
	if err := s.writeBlockRoot(blockRoot, blockRootOffset, blockDSDBOffset, blockDataOffset, blockList); err != nil {
		return err
	}
	// write header
	blockHeader := new(bytes.Buffer)
	if err := s.writeHeader(blockHeader, blockRootOffsetReal, uint32(blockRoot.Len())); err != nil {
		return err
	}
	// calculate file size
	var size uint32 = 32
	if blockRootEnd > size {
		size = blockRootEnd
	}
	if blockDSDBEnd > size {
		size = blockDSDBEnd
	}
	if blockDataEnd > size {
		size = blockDataEnd
	}
	// create full file
	fileData := make([]byte, size+4)
	copy(fileData[0:], blockHeader.Bytes())
	copy(fileData[4+blockRootOffsetReal:], blockRoot.Bytes())
	copy(fileData[4+blockDSDBOffsetReal:], blockDSDB.Bytes())
	copy(fileData[4+blockDataOffsetReal:], blockData.Bytes())
	// write it
	_, err := w.Write(fileData)
	return err
}

// WriteFile writes .DS_Store to the file
func (s *Store) WriteFile(filename string, perm os.FileMode) error {
	buffer := new(bytes.Buffer)
	if err := s.Write(buffer); err != nil {
		return err
	}
	return os.WriteFile(filename, buffer.Bytes(), perm)
}
