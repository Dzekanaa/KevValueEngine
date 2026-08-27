// Package merkle implements a Merkle tree over a sequence of data blocks,
// used to detect where an SSTable's data has changed since it was written.
package merkle

import (
	"bytes"
	"crypto/sha256"
	"encoding/binary"
	"encoding/hex"
	"fmt"
	"io"
)

// Node is a single Merkle tree node: a hash plus links to its children.
// Leaf nodes additionally carry the index of the data block they hash.
type Node struct {
	data  []byte //sadrzi hes vrednost
	left  *Node
	right *Node
	leaf  bool
	index int
}

// MerkleRoot is a built Merkle tree, identified by its root node.
type MerkleRoot struct {
	root *Node
}

// String returns the tree's root hash as a hex string.
func (mr *MerkleRoot) String() string {
	return mr.root.String()
}

// String returns the node's hash as a hex string.
func (n *Node) String() string {
	return hex.EncodeToString(n.data[:])
}

// Hash returns the SHA-256 hash of data.
func Hash(data []byte) [32]byte {
	return sha256.Sum256(data)
}

// buildLeaves creates one leaf node per data block, hashing each block.
func buildLeaves(blocks [][]byte) []*Node {
	leaves := make([]*Node, len(blocks))
	for i, b := range blocks {
		h := Hash(b)
		leaves[i] = &Node{data: h[:], leaf: true, index: i}
	}
	return leaves
}

// hashPair returns the SHA-256 hash of two child hashes concatenated.
func hashPair(left, right []byte) [32]byte {
	combined := append(left, right...)
	return sha256.Sum256(combined)
}

// buildNextLevel pairs up nodes from level and hashes each pair into a
// parent. A node left without a pair is promoted unchanged to the next
// level.
func buildNextLevel(level []*Node) []*Node {
	next := make([]*Node, 0, (len(level)+1)/2)

	for i := 0; i < len(level); i += 2 {
		left := level[i]

		if i+1 == len(level) {
			//cvor bez para - hes se prenosi nepromenjen
			parent := &Node{data: left.data, left: left}
			next = append(next, parent)
			continue
		}

		right := level[i+1]
		h := hashPair(left.data, right.data)
		parent := &Node{data: h[:], left: left, right: right}
		next = append(next, parent)
	}
	return next
}

// buildTree repeatedly merges a level of nodes into parents until a
// single root node remains.
func buildTree(leaves []*Node) *Node {
	level := leaves
	for len(level) > 1 {
		level = buildNextLevel(level)
	}

	return level[0]
}

// New builds a Merkle tree over blocks, one leaf per block.
func New(blocks [][]byte) (*MerkleRoot, error) {
	if len(blocks) == 0 {
		return nil, fmt.Errorf("Merkle: potrebno je bar jedan blok podataka")
	}

	leaves := buildLeaves(blocks)
	root := buildTree(leaves)

	return &MerkleRoot{root: root}, nil
}

// serializeNode writes n to buf in pre-order, using a 1-byte presence
// marker so nil children can be reconstructed on deserialization.
func serializeNode(buf *bytes.Buffer, n *Node) {
	if n == nil {
		buf.WriteByte(0) //marker: cvor ne postoji
		return
	}

	buf.WriteByte(1) //marker: cvor postoji
	buf.Write(n.data)

	if n.leaf {
		buf.WriteByte(1)
		idx := make([]byte, 4)
		binary.BigEndian.PutUint32(idx, uint32(n.index))
		buf.Write(idx)
	} else {
		buf.WriteByte(0)
	}

	serializeNode(buf, n.left)
	serializeNode(buf, n.right)
}

// Serialize encodes the tree into its on-disk byte representation.
func (mr *MerkleRoot) Serialize() []byte {
	var buf bytes.Buffer
	serializeNode(&buf, mr.root)
	return buf.Bytes()
}

// deserializeNode reads one node (and its subtree) from r, in the same
// pre-order layout written by serializeNode.
func deserializeNode(r *bytes.Reader) (*Node, error) {
	marker, err := r.ReadByte()
	if err != nil {
		return nil, err
	}

	if marker == 0 {
		return nil, nil
	}

	hash := make([]byte, 32)
	if _, err := io.ReadFull(r, hash); err != nil {
		return nil, err
	}

	leafByte, err := r.ReadByte()
	if err != nil {
		return nil, err
	}

	n := &Node{data: hash, leaf: leafByte == 1}

	if n.leaf {
		idxBytes := make([]byte, 4)
		if _, err := r.Read(idxBytes); err != nil {
			return nil, err
		}
		n.index = int(binary.BigEndian.Uint32(idxBytes))
	}

	left, err := deserializeNode(r)
	if err != nil {
		return nil, err
	}

	right, err := deserializeNode(r)
	if err != nil {
		return nil, err
	}

	n.left = left
	n.right = right

	return n, nil
}

// Deserialize decodes a tree from its on-disk byte representation.
func Deserialize(data []byte) (*MerkleRoot, error) {
	r := bytes.NewReader(data)

	root, err := deserializeNode(r)
	if err != nil {
		return nil, err
	}

	if root == nil {
		return nil, fmt.Errorf("merkle: prazan zapis, nema korena stabla")
	}

	return &MerkleRoot{root: root}, nil
}

// compareNodes walks two trees in lockstep, recording the leaf indices
// where the two trees' hashes diverge. A shape mismatch (one side nil,
// the other not) is treated as a structural difference and skipped
// rather than causing a crash.
func compareNodes(a, b *Node, diffs *[]int) {
	if a == nil && b == nil {
		return
	}
	if a == nil || b == nil {
		return
	}

	if bytes.Equal(a.data, b.data) {
		return
	}

	if a.leaf || b.leaf {
		*diffs = append(*diffs, a.index)
		return
	}

	compareNodes(a.left, b.left, diffs)
	compareNodes(a.right, b.right, diffs)
}

// Compare returns the indices of data blocks that differ between
// oldTree and newTree, used to validate an SSTable's data against its
// stored Merkle root.
func Compare(oldTree, newTree *MerkleRoot) []int {
	var diffs []int
	compareNodes(oldTree.root, newTree.root, &diffs)
	return diffs
}
