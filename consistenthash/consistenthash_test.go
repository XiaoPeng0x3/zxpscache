package consistenthash

import (
	"strconv"
	"testing"
	"hash/crc32"
)

func TestHashing(t *testing.T) {
	hash := New(3, func(key []byte) uint32 {
		i, _ := strconv.Atoi(string(key))
		return uint32(i)
	})

	// Given the above hash function, this will give replicas with "hashes":
	// 2, 4, 6, 12, 14, 16, 22, 24, 26
	hash.Add("6", "4", "2")

	testCases := map[string]string{
		"2":  "2",
		"11": "2",
		"23": "4",
		"27": "2",
	}

	for k, v := range testCases {
		if hash.Get(k) != v {
			t.Errorf("Asking for %s, should have yielded %s", k, v)
		}
	}

	// Adds 8, 18, 28
	hash.Add("8")

	// 27 should now map to 8.
	testCases["27"] = "8"

	for k, v := range testCases {
		if hash.Get(k) != v {
			t.Errorf("Asking for %s, should have yielded %s", k, v)
		}
	}

}

func TestRemoveNode(t *testing.T) {
	// 创建哈希环
	hash := New(3, crc32.ChecksumIEEE) // 每个节点 3 个虚拟节点
	nodes := []string{"NodeA", "NodeB", "NodeC"}
	hash.Add(nodes...)

	// 准备一些测试 key
	keys := []string{"apple", "banana", "cherry", "date"}

	// 映射 key 到节点（初始状态）
	keyToNodeBefore := make(map[string]string)
	for _, k := range keys {
		keyToNodeBefore[k] = hash.Get(k)
	}
	t.Logf("Before remove: %+v\n", keyToNodeBefore)

	// 模拟 NodeB 宕机
	hash.Remove("NodeB")

	// 检查每个 key 是否还映射到 NodeB，如果有则说明 Remove 失败
	for _, k := range keys {
		node := hash.Get(k)
		if node == "NodeB" {
			t.Errorf("Key %s still mapped to removed node NodeB", k)
		} else {
			t.Logf("Key %s now maps to %s", k, node)
		}
	}
}