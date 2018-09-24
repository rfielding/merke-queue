package merkleq

import (
	"crypto/sha256"
	"sync"
)

type Queue struct {
	Mutex     sync.RWMutex
	IndexBits uint
	Head      uint64
	Epoch     uint64
	Hashes    [][32]byte
}

func NewQueue(bits uint) (*Queue, error) {
	return &Queue{
		IndexBits: bits,
		Hashes:    make([][32]byte, (1 << bits)),
	}, nil
}

// IndexOf gets the physical index of ring 0 (outer ring)
//
// - an epoch is a cyle around the ring
// - logical indexes are called "n", and can be from many epochs ago,
//   or for a future epoch
// - m is "n" mod 2^bits
// - physical indexes are called "p", and are mod 2*2^bits
// - index of a bit is called "k", and is from 0..bits
// - this gives the index p, regardless of the epoch of n
// - r is the ring
//
// The logical to physical mapping happens to be the same as
// a depth-first-search post-order traversal.  This physical
// ordering is why we can let the writes wrap around and overwrite.
// Such entries are guaranteed to be the oldest entries.
//
// We put a bound on how far back into the past a query can go,
// which lets us garbage collect.
//
//
// We seek to the n-th slot, and go down r rings
//
func (q *Queue) IndexOf(m uint, r uint) uint {
	return q.Down( q.IndexOfRoot(m), m, r)
}

func (q *Queue) IndexOfRoot(m uint) uint {
	p := uint(0)
	for k := uint(0); k < q.IndexBits; k++ {
		pow2 := uint(1 << k)
		bit := (m & pow2) >> k
		if bit != 0 {
			p += (2*pow2 - 1)
		}
	}
	return (p + 2) % (1 << q.IndexBits)
}

func (q *Queue) Down(p uint, m uint, r uint) uint {
	for ri := uint(1); ri <= r; ri++ {
		bit := (m & (1 << (ri - 1))) >> (ri - 1)
		if bit == 0 {
			p += (1 << ri)
		} else {
			p += 1
		}
	}
	return p % (1 << q.IndexBits)
}

// Append writes an entry to the log,
// - we hash up the tree
// - move the head forward
func (q *Queue) Append(h [32]byte) {
	mod := uint64(1 << q.IndexBits)
	q.Mutex.Lock()
	// Write this hash to the head
	m := uint(q.Head % mod)
	p := q.IndexOf(m, 0)
	q.Hashes[p] = h
	// Fix up parent hashes
	for r := uint(1); r < q.IndexBits; r++ {
		p = q.IndexOf(m, r)
		rightLeaf := p + uint(2*mod - uint64(r))
		//XXX BUG -- some issue with physical vs logical indices
		leftLeaf := p + uint(2*mod - uint64(2*(1<<r) - 2))
		rp := q.Down(rightLeaf, m, r-1)
		lp := q.Down(leftLeaf, m, r-1)
		q.Hashes[p] = sha256.Sum256(append(q.Hashes[lp][:], q.Hashes[rp][:]...))
	}
	q.Head++
	if q.Head == 0 {
		q.Epoch++
	}
	q.Head = q.Head % uint64(q.IndexBits)
	q.Mutex.Unlock()
}
