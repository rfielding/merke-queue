package merkleq_test

import (
	"crypto/sha256"
	"encoding/hex"
	"fmt"
	"github.com/rfielding/merkle-queue/merkleq"
	"testing"
)

func indexTest(t *testing.T, q *merkleq.Queue, m uint, r uint, x uint) {
	if q.IndexOf(m, r) != x {
		t.Logf("%d,%d -> %d, but got %d", m, r, x, q.IndexOf(m, r))
		t.FailNow()
	}
}

func TestIndexing(t *testing.T) {
	q, err := merkleq.NewQueue(10)
	if err != nil {
		t.FailNow()
	}

	// pin down some random known points
	indexTest(t, q, 0, 0, 2)
	indexTest(t, q, 1, 0, 3)
	indexTest(t, q, 0, 1, 4)
	indexTest(t, q, 0, 2, 8)
	indexTest(t, q, 1, 1, 4)
	indexTest(t, q, 97, 0, 193)
	indexTest(t, q, 91, 3, 189)
	indexTest(t, q,100, 3, 206)
	indexTest(t, q,100, 3, q.Left(222,4))
	indexTest(t, q,111, 2, q.Right(221))

	// Add some things and verify that hashes change
	fmt.Printf("%s\n", hex.EncodeToString(q.Hashes[0][:]))

	q.Append(sha256.Sum256([]byte("this is a test.")))
	fmt.Printf("%s\n", hex.EncodeToString(q.Hashes[0][:]))

	q.Append(sha256.Sum256([]byte("another test.")))
	fmt.Printf("%s\n", hex.EncodeToString(q.Hashes[0][:]))

	for i := 0; i < 432; i++ {
		q.Append(sha256.Sum256([]byte(fmt.Sprintf("another test %d.", i))))
	}

}
