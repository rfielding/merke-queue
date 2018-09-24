This is a simple MerkleQueue library that will

- Start off with all hashes in the tree as zeroes, the "genesis block".
- As items are appended, they are hashed with the existing entries, including running into old entries that we are about to overwrite.
- The root entry should update when hashes are added.


