package merkleq_test

import (
	"crypto/sha256"
	"encoding/hex"
	"fmt"
	"github.com/rfielding/merkle-queue/merkleq"
	"testing"
)

func indexTest(t *testing.T, q *merkleq.Queue, m uint64, r uint32, x uint64) {
	if q.IndexOf(m, r) != x {
		t.Logf("%d,%d -> %d, but got %d", m, r, x, q.IndexOf(m, r))
		t.FailNow()
	}
}

func dumpAppend(t *testing.T, data string, q *merkleq.Queue) {
	err := q.Append(sha256.Sum256([]byte(data)))
	if err != nil {
		t.Logf("%v", err)
		t.FailNow()
	}
	h, err := q.GetHash(0)
	if err != nil {
		t.Logf("%v", err)
		t.FailNow()
	}
	fmt.Printf("%s\n", hex.EncodeToString(h[:]))
}

func TestIndexing(t *testing.T) {
	f := "merkle.q"
	q, err := merkleq.NewQueue(f, 8)
	if err != nil {
		t.Logf("%v", err)
		t.FailNow()
	}
	defer func() {
		q.Close()
		q.Delete()
	}()

	// pin down some random known points
	indexTest(t, q, 0, 0, 2)
	indexTest(t, q, 1, 0, 3)
	indexTest(t, q, 0, 1, 4)
	indexTest(t, q, 0, 2, 8)
	indexTest(t, q, 1, 1, 4)
	indexTest(t, q, 97, 0, 193)
	indexTest(t, q, 91, 3, 189)
	indexTest(t, q, 100, 3, 206)
	indexTest(t, q, 100, 3, q.Left(222, 4))
	indexTest(t, q, 111, 2, q.Right(221))

	dumpAppend(t, "this is a test", q)
	dumpAppend(t, "another test", q)
	for i := 0; i < 500; i++ {
		dumpAppend(t, fmt.Sprintf("another test %d", i), q)
	}

}
