package merkleq

import (
	"bytes"
	"crypto/sha256"
	"encoding/binary"
	"log"
	"os"
	"sync"
)

const HashSize = 32

type State struct {
	IndexBits uint32
	Epoch     uint32
	Head      uint64
}

type Queue struct {
	Fname string
	State State
	File  *os.File
	Mutex sync.RWMutex
}

func LogInfo(msg string, args ...interface{}) {
	log.Printf("INFO:"+msg, args...)
}

func LogDebug(msg string, args ...interface{}) {
	if os.Getenv("MERKLE_DEBUG") == "true" {
		log.Printf("DEBUG:"+msg, args...)
	}
}

func seek(f *os.File, n int64) error {
	LogDebug("Seek %d * 32", n/32)
	_, err := f.Seek(n, 0)
	return err
}

func NewQueue(fname string, bits uint32) (*Queue, error) {
	LogDebug("looking for the file")
	var f *os.File
	wasCreated := false
	_, err := os.Stat(fname)
	if os.IsNotExist(err) {
		LogDebug("create the file empty")
		f, err = os.Create(fname)
		if err != nil {
			return nil, err
		}
		// Write the last byte of the file as a zero
		err = seek(f, HashSize*(1<<(bits))-1)
		if err != nil {
			return nil, err
		}
		_, err = f.Write([]byte{0})
		if err != nil {
			return nil, err
		}
		wasCreated = true
	} else {
		if err != nil {
			return nil, err
		} else {
			LogDebug("open existing file")
			f, err = os.OpenFile(fname, os.O_RDWR, 0700)
			if err != nil {
				return nil, err
			}
		}
	}

	q := &Queue{
		File:  f,
		Fname: fname,
		State: State{
			IndexBits: bits,
			Head:      0,
			Epoch:     0,
		},
	}
	if wasCreated {
		err = q.WriteState()
		if err != nil {
			return nil, err
		}
	} else {
		err = q.ReadState()
		if err != nil {
			return nil, err
		}
	}
	return q, nil
}

func (q *Queue) GetHash(p uint64) ([HashSize]byte, error) {
	LogDebug("getHash %d", p)
	buf := *new([32]byte)
	err := seek(q.File, int64(HashSize*p))
	if err != nil {
		return buf, err
	}
	_, err = q.File.Read(buf[:])
	return buf, err
}

func (q *Queue) SetHash(p uint64, buf [HashSize]byte) error {
	LogDebug("SetHash %d", p)
	err := seek(q.File, int64(HashSize*p))
	if err != nil {
		return err
	}
	_, err = q.File.Write(buf[:])
	return err
}

func (q *Queue) WriteState() error {
	LogDebug("WriteState IndexBits: %d Epoch: %d Head: %d", q.State.IndexBits, q.State.Epoch, q.State.Head)
	buf := make([]byte, HashSize)
	binary.BigEndian.PutUint32(buf[4:], q.State.IndexBits)
	binary.BigEndian.PutUint32(buf[4+4:], q.State.Epoch)
	binary.BigEndian.PutUint64(buf[4+4+8:], q.State.Head)
	err := seek(q.File, 1*HashSize)
	if err != nil {
		return err
	}
	_, err = q.File.Write(buf)
	if err != nil {
		return err
	}
	return nil
}

func (q *Queue) ReadState() error {
	LogDebug("ReadState")
	buf := make([]byte, HashSize)
	err := seek(q.File, 1*HashSize)
	if err != nil {
		return err
	}
	_, err = q.File.Read(buf)
	if err != nil {
		return err
	}
	q.State.IndexBits = binary.BigEndian.Uint32(buf[4:])
	q.State.Epoch = binary.BigEndian.Uint32(buf[4+4:])
	q.State.Head = binary.BigEndian.Uint64(buf[4+4+8:])
	LogDebug("Read IndexBits: %d Epoch: %d Head: %d", q.State.IndexBits, q.State.Epoch, q.State.Head)
	return nil
}

func (q *Queue) Close() error {
	LogDebug("Close")
	return q.File.Close()
}

func (q *Queue) Delete() error {
	LogDebug("Delete")
	_, err := os.Stat(q.Fname)
	if os.IsExist(err) {
		return os.Remove(q.Fname)
	}
	return err
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
func (q *Queue) IndexOf(m uint64, r uint32) uint64 {
	LogDebug("IndexOf %d", m)
	return q.Down(q.indexOfRoot(m), m, r)
}

func (q *Queue) indexOfRoot(m uint64) uint64 {
	p := uint64(0)
	for k := uint32(0); k < q.State.IndexBits; k++ {
		pow2 := uint64(1 << k)
		bit := (m & pow2) >> k
		if bit != 0 {
			p += (2*pow2 - 1)
		}
	}
	return (p + 2) % (1 << q.State.IndexBits)
}

func (q *Queue) Down(p uint64, m uint64, r uint32) uint64 {
	for ri := uint32(1); ri <= r; ri++ {
		LogDebug("Down %d %d %d", p, m, r)
		bit := (m & (1 << (ri - 1))) >> (ri - 1)
		if bit == 0 {
			p += (1 << ri)
		} else {
			p += 1
		}
	}
	return p % (1 << q.State.IndexBits)
}

func (q *Queue) Left(p uint64, r uint32) uint64 {
	mod := uint64(1 << q.State.IndexBits)
	return (p - (1 << r) + uint64(2*mod)) % uint64(mod)
}

func (q *Queue) Right(p uint64) uint64 {
	mod := uint64(1 << q.State.IndexBits)
	return (p - 1 + uint64(2*mod)) % uint64(mod)
}

var allZeroes = *new([32]byte)

// Append writes an entry to the log,
// - we hash up the tree
// - move the head forward
func (q *Queue) Append(h [32]byte) error {
	LogDebug("Append")
	mod := uint64(1 << q.State.IndexBits)
	q.Mutex.Lock()
	defer q.Mutex.Unlock()
	// Write this hash to the head
	m := uint64(q.State.Head % mod)
	p := q.IndexOf(m, 0)
	err := q.SetHash(p, h)
	if err != nil {
		return err
	}
	// Fix up parent hashes
	for r := uint32(1); r < q.State.IndexBits; r++ {
		p = q.IndexOf(m, r)
		lp := q.Left(p, r)
		rp := q.Right(p)
		zeroes := allZeroes[:]
		hl, err := q.GetHash(lp)
		if err != nil {
			return err
		}
		leftzeroes := bytes.Compare(zeroes, hl[:]) == 0
		hr, err := q.GetHash(rp)
		if err != nil {
			return err
		}
		rightzeroes := bytes.Compare(zeroes, hr[:]) == 0
		if (leftzeroes && rightzeroes) == false {
			if leftzeroes {
				q.SetHash(p, hr)
				if err != nil {
					return err
				}
			} else {
				if rightzeroes {
					err = q.SetHash(p, hl)
					if err != nil {
						return err
					}
				} else {
					err = q.SetHash(p, sha256.Sum256(
						append(hl[:], hr[:]...),
					))
					if err != nil {
						return err
					}
				}
			}
		}
	}
	q.State.Head++
	if q.State.Head == 0 {
		q.State.Epoch++
	}
	q.State.Head = q.State.Head % uint64(q.State.IndexBits)
	return q.WriteState()
}
