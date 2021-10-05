package calc

/*
 * the consistenthasher
 * @author liuxiaofeng
 * @mail neo532@126.com
 */

import (
	"hash/crc32"
	"sort"
	"strconv"
	"sync"
)

// ConsistentHash holds the information about consistent hashing.
type ConsistentHash struct {
	hash2node        map[uint32]string
	hashList         slots
	NumOfVirtualNode int
	lock             *sync.Mutex
}

type slots []uint32

// Len is a method for sort.
func (s slots) Len() int {
	return len(s)
}

// Less is a method for sort.
func (s slots) Less(i, j int) bool {
	return s[i] < s[j]
}

// Swap is a method for sort.
func (s slots) Swap(i, j int) {
	s[i], s[j] = s[j], s[i]
}

// NewCHash returns a instance of ConsistentHash.
func NewCHash() *ConsistentHash {
	return &ConsistentHash{
		hash2node:        make(map[uint32]string),
		NumOfVirtualNode: 2,
		lock:             &sync.Mutex{},
	}
}

// hashNode returns  a unit32 after hashing by inputing parameter.
func (c *ConsistentHash) hashNode(node string, element int) uint32 {
	return c.hash(strconv.Itoa(element) + node)
}

// hash returns a unit32 after hashing by inputing string.
func (c *ConsistentHash) hash(key string) uint32 {
	return crc32.ChecksumIEEE([]byte(key))
}

// Add adds the nodes of ConsistentHash by inputing nodes.
func (c *ConsistentHash) Add(nodeList ...string) {
	c.lock.Lock()
	defer c.lock.Unlock()

	for _, n := range nodeList {
		for i := 0; i < c.NumOfVirtualNode; i++ {
			hash := c.hashNode(n, i)
			c.hash2node[hash] = n
			c.hashList = append(c.hashList, hash)
		}
	}

	sort.Sort(c.hashList)
}

// Del deletes the nodes of ConsistentHash by inputing nodes.
func (c *ConsistentHash) Del(nodeList ...string) {
	c.lock.Lock()
	defer c.lock.Unlock()

	for _, n := range nodeList {
		for i := 0; i < c.NumOfVirtualNode; i++ {
			delete(c.hash2node, c.hashNode(n, i))
		}
	}

	var HL slots
	for n, _ := range c.hash2node {
		HL = append(HL, n)
	}
	sort.Sort(HL)
	c.hashList = HL
}

// Get returns a node of ConsistentHash by key.
func (c *ConsistentHash) Get(key string) string {
	hash := c.hash(key)
	lenHL := len(c.hashList)
	index := sort.Search(lenHL, func(i int) bool {
		return c.hashList[i] >= hash
	})
	if index >= lenHL {
		index = 0
	}

	if node, ok := c.hash2node[c.hashList[index]]; ok {
		return node
	}
	return ""
}
