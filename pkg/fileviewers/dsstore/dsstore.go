package dsstore

// Record in .DS_Store
type Record struct {
	FileName string // file name
	Extra    uint32 // extra (unknown data)
	Type     string // type
	DataLen  uint32 // explicit data size in bytes
	Data     []byte // raw data
}

// Store of .DS_Store file
type Store struct {
	HeaderExtra []byte   // header extra data (unknown)
	RootExtra   []byte   // root (bookkeeping) extra data (unknown)
	DSDBExtra   []byte   // DSDB extra data (unknown)
	Records     []Record // records
}

const headerMagic1 uint32 = 0x1
const headerMagic2 uint32 = 0x42756431

func blockSize(offset uint32) uint32 {
	return uint32(1) << (offset & uint32(0x1f))
}

func blockOffset(offset uint32) uint32 {
	return offset & ^uint32(0x1f)
}
