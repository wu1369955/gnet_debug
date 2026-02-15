package gfd

import (
	"encoding/binary"
	"math"
	"sync/atomic"
)

// Constants for GFD.
const (
	ConnMatrixColumnOffset = 2
	SequenceOffset         = 4
	FdOffset               = 8
	EventLoopIndexMax      = math.MaxUint8 + 1
	ConnMatrixRowMax       = math.MaxUint8 + 1
	ConnMatrixColumnMax    = math.MaxUint16 + 1
)

type monotoneSeq uint32

func (seq *monotoneSeq) Inc() uint32 {
	return atomic.AddUint32((*uint32)(seq), 1)
}

var monoSeq = new(monotoneSeq)

// GFD is a structure to store the fd, eventloop index, connStore indexes.
type GFD [0x10]byte

// Fd returns the underlying fd.
func (gfd GFD) Fd() int {
	return int(binary.BigEndian.Uint64(gfd[FdOffset:]))
}

// EventLoopIndex returns the eventloop index.
func (gfd GFD) EventLoopIndex() int {
	return int(gfd[0])
}

// ConnMatrixRow returns the connMatrix row index.
func (gfd GFD) ConnMatrixRow() int {
	return int(gfd[1])
}

// ConnMatrixColumn returns the connMatrix column index.
func (gfd GFD) ConnMatrixColumn() int {
	return int(binary.BigEndian.Uint16(gfd[ConnMatrixColumnOffset:SequenceOffset]))
}

// Sequence returns the monotonic sequence, only used to prevent fd duplication.
func (gfd GFD) Sequence() uint32 {
	return binary.BigEndian.Uint32(gfd[SequenceOffset:FdOffset])
}

// UpdateIndexes updates the connStore indexes.
func (gfd *GFD) UpdateIndexes(row, column int) {
	(*gfd)[1] = byte(row)
	binary.BigEndian.PutUint16((*gfd)[ConnMatrixColumnOffset:SequenceOffset], uint16(column))
}

// Validate checks if the GFD is valid.
func (gfd GFD) Validate() bool {
	return gfd.Fd() > 2 && gfd.Fd() <= math.MaxInt &&
		gfd.EventLoopIndex() >= 0 && gfd.EventLoopIndex() < EventLoopIndexMax &&
		gfd.ConnMatrixRow() >= 0 && gfd.ConnMatrixRow() < ConnMatrixRowMax &&
		gfd.ConnMatrixColumn() >= 0 && gfd.ConnMatrixColumn() < ConnMatrixColumnMax &&
		gfd.Sequence() > 0
}

// NewGFD creates a new GFD.
func NewGFD(fd, elIndex, row, column int) (gfd GFD) {
	gfd[0] = byte(elIndex)
	gfd[1] = byte(row)
	binary.BigEndian.PutUint16(gfd[ConnMatrixColumnOffset:SequenceOffset], uint16(column))
	binary.BigEndian.PutUint32(gfd[SequenceOffset:FdOffset], monoSeq.Inc())
	binary.BigEndian.PutUint64(gfd[FdOffset:], uint64(fd))
	return
}
