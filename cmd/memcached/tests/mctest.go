package main

import "fmt"
import "os"
import "bytes"
import "math/rand"
import "crypto/sha1"
import "time"
import "strconv"

import (
	"github.com/bradfitz/gomemcache/memcache"
)

const letters = "abcdefghijklmnopqrstuvwxyzABCDEFGHIJKLMNOPQRSTUVWXYZ "

var test, fail int

func randbuf(n int) []byte {
	b := make([]byte, n)
	for i := range b {
		b[i] = letters[rand.Intn(len(letters))]
	}
	return b
}

func hash(b []byte) []byte {
	h := sha1.New()
	h.Write(b)
	return h.Sum(nil)
}

func testSet(mc *memcache.Client, k string, v []byte) bool {
	err := mc.Set(&memcache.Item{Key: k, Value: v})
	if err != nil {
		fmt.Println(err)
		return false
	}

	return true
}

func testGet(mc *memcache.Client, k string) (*memcache.Item, bool) {
	it, err := mc.Get(k)
	if err != nil {
		fmt.Println(err)
		return nil, false
	}

	return it, true
}

func testSetnGet(mc *memcache.Client, k string, v []byte, rand bool, n int) bool {

	var b []byte
	var pre, post []byte

	if rand {
		b = randbuf(n)
		pre = hash(b)
	} else {
		pre = hash(v)
		b = v
	}

	ok := testSet(mc, k, b)
	if !ok {
		return false
	}

	it, ok := testGet(mc, k)
	if !ok {
		return false
	}

	post = hash(it.Value)

	if len(it.Value) != len(b) || bytes.Equal(pre, post) == false {
		return false
	}

	//fmt.Printf("Key: %s, Len: %d, Value: %s\n", it.Key, len(it.Value), string(it.Value))
	//fmt.Printf("Hash before: %x, Hash after: %x\n", pre, post)

	return true
}

func countTestFail(res bool, exp bool) {
	if res != exp {
		fail++
	}
	test++
}

func main() {
	port := os.Args[1]
	mc := memcache.New("localhost:" + port)
	rand.Seed(time.Now().UnixNano())

	buf := []byte("Hello, World")
	ok := testSetnGet(mc, "foo", buf, false, 0)
	countTestFail(ok, true)

	buf = []byte("Increment and Decrement. If an item stored is the string representation of a 64bit integer, you may run incr or decr commands to modify that number. You may only incr by positive values, or decr by positive values. They does not accept negative values\n" +
		"If a value does not already exist, incr/decr will fail.")
	ok = testSetnGet(mc, "long", buf, false, 0)
	countTestFail(ok, true)

	buf = []byte("Standard Protocol\n" +
		"The \"standard protocol stuff\" of memcached involves running a command against an \"item\". An item consists of:\n A key (arbitrary string up to 250 bytes in length. No space or newlines for ASCII mode)\n A 32bit \"flag\" value\n An expiration time, in seconds. '0' means never expire. Can be up to 30 days. After 30 days, is treated as a unix timestamp of an exact date.\n A 64bit \"CAS\" value, which is kept unique.\n Arbitrary data\n CAS is optional (can be disabled entirely with -C, and there are more fields that internally make up an item, but these are what your client interacts with.")
	ok = testSetnGet(mc, "longer", buf, false, 0)
	countTestFail(ok, true)

	ok = testSetnGet(mc, "longest", buf, true, 1000)
	countTestFail(ok, true)

	for i := 0; i < 10000; i++ {
		buf = randbuf(10)
		k := "key" + strconv.Itoa(i)
		ok = testSet(mc, k, buf)
		countTestFail(ok, true)
	}

	for i := 0; i < 10000; i++ {
		k := "key" + strconv.Itoa(i)
		_, ok = testGet(mc, k)
		countTestFail(ok, true)
	}

	//Check for non-existent key
	_, ok = testGet(mc, "key01")
	countTestFail(ok, false)

	//Test LRU
	ok = testSet(mc, "key01", []byte("San Francisco"))
	countTestFail(ok, true)

	//key0 should be evicted from cache now
	_, ok = testGet(mc, "key0")
	countTestFail(ok, false)

	_, ok = testGet(mc, "key1")
	countTestFail(ok, true)

	ok = testSet(mc, "key02", []byte("Los Angeles"))
	countTestFail(ok, true)

	//key2 should be evicted now
	_, ok = testGet(mc, "key2")
	countTestFail(ok, false)

	if fail != 0 {
		fmt.Printf("%d of %d tests failed\n", fail, test)
	} else {
		fmt.Printf("All %d tests succeeded\n", test)
	}
}
