package main

import "fmt"
import "os"
import "sync"
import "strconv"

import (
	"github.com/bradfitz/gomemcache/memcache"
)

var test, fail int
var mutex = &sync.Mutex{}
var wg, rwg sync.WaitGroup

func testSet(mc *memcache.Client, k string, v []byte) bool {
	defer wg.Done()
	err := mc.Set(&memcache.Item{Key: k, Value: v})
	if err != nil {
		fmt.Println(err)
		countTestFail(false, true)
		return false
	}
	countTestFail(true, true)

	return true
}

func testGet(mc *memcache.Client, k string) (*memcache.Item, bool) {
	defer rwg.Done()
	it, err := mc.Get(k)
	if err != nil {
		fmt.Println(err)
		countTestFail(false, true)
		return nil, false
	}

	countTestFail(true, true)
	return it, true
}

func countTestFail(res bool, exp bool) {
	mutex.Lock()
	if res != exp {
		fail++
	}
	test++
	mutex.Unlock()
}

func main() {
	if len(os.Args) != 2 {
		fmt.Println("Usage: go run conmctest.go <port>")
	}
	port := os.Args[1]
	mc := memcache.New("localhost:" + port)

	wg.Add(100)
	for i := 0; i < 100; i++ {
		go testSet(mc, "key99", []byte("Sunnyvale"+strconv.Itoa(i)))
	}

	wg.Wait()

	rwg.Add(100)
	for i := 0; i < 100; i++ {
		go testGet(mc, "key99")
	}

	rwg.Wait()
	rwg.Add(1)
	it, ok := testGet(mc, "key99")
	rwg.Wait()

	if ok {
		fmt.Println(string(it.Value))
	}

	if fail != 0 {
		fmt.Printf("%d of %d tests failed\n", fail, test)
	} else {
		fmt.Printf("All %d tests succeeded\n", test)
	}
}
