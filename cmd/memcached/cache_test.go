//Unit tests for cache.go
package main

import "testing"
import "strconv"

func TestCacheInit(t *testing.T) {
	//Test Case 1
	err := CacheInit(10)
	if err != nil {
		t.Error(err)
	}

	//Test Case 2
	err = CacheInit(0)
	if err == nil {
		t.Error(err)
	}

	//Test Case 3
	err = CacheInit(-1)
	if err == nil {
		t.Error(err)
	}

}

func insertElements(t *testing.T, n int) {

	for i := 1; i <= n; i++ {
		k := "key" + strconv.Itoa(i)
		err := Upsert(k, strconv.Itoa(i*10))
		if err != nil {
			t.Error(err)
		}
	}
}

func TestUpsert(t *testing.T) {
	err := CacheInit(5)
	if err != nil {
		t.Error(err)
	}

	err = Upsert("foo", "10")
	if err != nil {
		t.Error(err)
	}

	insertElements(t, 4)

	//Cache is at capacity now, test if another insert works fine
	err = Upsert("key5", "60")
	if err != nil {
		t.Error(err)
	}

}

func TestGet(t *testing.T) {
	err := CacheInit(5)
	if err != nil {
		t.Error(err)
	}

	_, err = Get("foo")
	if err == nil {
		t.Error(err)
	}

	err = Upsert("foo", "10")
	if err != nil {
		t.Error(err)
	}

	_, err = Get("foo")
	if err != nil {
		t.Error(err)
	}

	insertElements(t, 4)

	//Cache is at capacity now, test another insert works fine
	err = Upsert("key5", "50")
	if err != nil {
		t.Error(err)
	}

	// key "foo" should not be in cache anymore
	_, err = Get("foo")
	if err == nil {
		t.Error(err)
	}

	//Verify the inserted elements exist in cache
	for i := 1; i <= 5; i++ {
		k := "key" + strconv.Itoa(i)
		v, err := Get(k)
		if err != nil {
			t.Error(err)
		}
		n, err := strconv.Atoi(v)
		if err != nil {
			t.Error(err)
		}

		if n != (10 * i) {
			t.Error(err)
		}
	}

}
