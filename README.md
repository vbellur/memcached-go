### About

This is a simple memcached server written in Go. Supports only set & get
operations as of now.

## Installing

### Using *go get*

    $ go get github.com/vbellur/memcached-go/cmd/memcached

After this operation, source code will be located in:

    $GOPATH/src/github.com/vbellur/memcached-go/cmd/memcached

From this directory, the following command can be issued to build the binary:

	$go build .


## Details

memcached-go provides a LRU cache that can be accessed using `get` and `set` commands over the [memcached ascii protocol](https://github.com/memcached/memcached/blob/master/doc/protocol.txt). The LRU is maintained using a combination of:

1. Doubly Linked List (DLL) containing nodes that have key, value and other metadata.
2. Map of keys to pointers for nodes in the DLL.

A set operation results in both the DLL and map being updated.  An existing key's value gets updated during a set or a new key is added to the cache. If the cache is full, the node at the tail of the DLL is evicted to make space for the new key, value tuple.

A get operation causes the node pertaining to the key to be promoted to the head of the list.

By default, the maximum number of keys in the cache is 10000. This can be altered by changing the value passed to `CacheInit()` in `memcached.go`.

When a connection is established from the client, a goroutine is used to manage
the connection and commands that are issued over the connection by the client.

## Running memcached

The `memcached` binary generated after `go build` can be started using:

`$memcached`

This will launch a foreground process that will run on port 11211 by default. In case, you need to run the server on a different port, please specify the port as an argument to memcached, like:

`$memcached 9001`

Once the server is up and running, you can use your favorite memcached client tool/interface to work with the server.

## Running tests

There are two categories of tests provided with the source:

1. Unit tests - These can be run using `go test -v` in the memcached directory.
2. Functional tests - There are functional tests in the tests/ directory. Currently, there are two test units:
		a> mctest.go - Runs a bunch of set and get tests & also verifies data integrity by hash comparison.
		b>conmctest.go - Runs concurrent sets and gets using goroutines.

Both these test units utilize [gomemcache](https://github.com/bradfitz/gomemcache) client library for running tests and gomemcache will need to be installed on your system before these tests are run. Both test units talk to a single instance of the memcached server by default.

## Known Issues

1. Authentication is not supported with `set` command as of yet.
2. flags, exptime and noreply are ignored by the server.
3.  [Segmented LRU](https://memcached.org/blog/modern-lru/) is not yet supported.
4. memcached is reachable only over TCP. There is no support for UDP.
5. LRU eviction happens only when the cache capacity is full. Other schemes like eviction based on size, etc. are not yet available.
6. verbose mode for logging is not yet available.

## Reporting Bugs

Please use github issues for reporting bugs & issues observed.
