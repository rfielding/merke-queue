package merkleq_test

import (
	"github.com/rfielding/merkle-queue/merkleq"
	"testing"
	"crypto/sha256"
	"fmt"
	"encoding/hex"
)

func indexTest(t *testing.T, q *merkleq.Queue, m uint, r uint, x uint) {
	if q.IndexOf(m,r) != x {
		t.Logf("%d,%d -> %d, but got %d", m, r, x, q.IndexOf(m,r))
		t.FailNow()
	}
}

func TestIndexing(t *testing.T) {
	q, err := merkleq.NewQueue(8)
	if err != nil {
		t.FailNow()
	}

	// pin down some random known points
	indexTest(t,q,0,0,2)
	indexTest(t,q,1,0,3)
	indexTest(t,q,0,1,4)
	indexTest(t,q,0,2,8)
	indexTest(t,q,1,1,4)
	indexTest(t,q,255,0,248)
	indexTest(t,q,255,0,248)
	indexTest(t,q,255,0,248)

	// Add some things and verify that hashes change
	fmt.Printf("%s\n", hex.EncodeToString(q.Hashes[0][:]))
 
	q.Append(sha256.Sum256([]byte("this is a test.")))
	fmt.Printf("%s\n", hex.EncodeToString(q.Hashes[0][:]))
 
	q.Append(sha256.Sum256([]byte("another test.")))
	fmt.Printf("%s\n", hex.EncodeToString(q.Hashes[0][:]))
 
	for i := 0; i < 23; i++ {
		q.Append(sha256.Sum256([]byte(fmt.Sprintf("another test %d.",i)))) 
		fmt.Printf("%s\n", hex.EncodeToString(q.Hashes[0][:])) 
	} 
}
