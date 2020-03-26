//server functionality for memcached. Handles network exchanges with client & parses
//commands received from the client before handing off to cache.
package main

import (
	"bufio"
	"errors"
	"fmt"
	"io"
	"net"
	"os"
	"strconv"
	"strings"
	"sync"
	"utils"
)

//States for managing set command
type State int

const (
	Waiting   State = 0
	WaitValue State = 1
	MaxState  State = 2
)

//Maximum size of value - 64MB
const maxValueBytes = 64 * 1048576

//Maximum length of key
const maxKeyLen = 250

//Context for managing set command
type setCtx struct {
	size    int
	flags   int16
	exptime int64
	noreply bool
	key     string
}

// structure to maintain information about a connection
type conn struct {
	c     net.Conn
	state State
	ctx   setCtx
}

//pointer to map of connections
type connections struct {
	m    map[net.Conn]*conn
	lock *sync.RWMutex
}

//table of connections
var conns connections

//Initialize connection table & cache
func memcachedInit() {
	conns.m = make(map[net.Conn]*conn)
	conns.lock = new(sync.RWMutex)
	CacheInit(10000)
}

//Updates connection state
func updateState(c net.Conn, s State) {

	conns.lock.Lock()
	conns.m[c].state = s
	conns.lock.Unlock()
}

// Handles set command that has the following format:
// set <key> <flags> <exptime> <bytes> [noreply]\r\n
func handleSet(c net.Conn, arr []string) error {

	if len(arr) < 5 {
		err := errors.New("Invalid number of arguments")
		c.Write([]byte("ERROR\r\n"))
		return err
	}

	key := arr[1]
	if len(key) > maxKeyLen {
		err := errors.New("Invalid number of arguments")
		c.Write([]byte("CLIENT_ERROR bad command line format\r\n"))
		return err
	}

	//skip over exptime & flags
	nbytes := arr[4]
	if strings.Trim(nbytes, " ") == "" {
		err := errors.New("Invalid size")
		c.Write([]byte("ERROR\r\n"))
		return err
	}

	b, err := strconv.Atoi(nbytes)
	if err != nil || b > maxValueBytes {
		err := errors.New("Invalid value size")
		c.Write([]byte("CLIENT_ERROR Invalid size\r\n"))
		return err
	}

	conns.lock.Lock()
	conns.m[c].state = WaitValue
	conns.m[c].ctx.size = b
	conns.m[c].ctx.key = key
	conns.lock.Unlock()

	return nil

}

// Handle set command to update value for specified key
func handleSetValue(c net.Conn, val string) error {
	conns.lock.RLock()
	size := conns.m[c].ctx.size
	key := conns.m[c].ctx.key
	conns.lock.RUnlock()

	if len(val) != size {
		c.Write([]byte("CLIENT_ERROR bad data chunk\r\n"))
		err := errors.New("Bad data chunk")
		return err
	}
	err := Upsert(key, val)
	if err != nil {
		c.Write([]byte("ERROR\r\n"))
		return err
	}

	c.Write([]byte("STORED\r\n"))
	return nil
}

//Cleans up a connection upon disconnection
func cleanup(c net.Conn) {
	conns.lock.Lock()
	delete(conns.m, c)
	conns.lock.Unlock()
	c.Close()
}

//Handles get|gets commands with the following format
// get|gets <key1> [<key2> <key3>... <keyN>]
// Retrieves values from the cache and responds to client
func handleGet(c net.Conn, arr []string) error {

	w := bufio.NewWriter(c)
	for i := 1; i < len(arr); i++ {
		key := strings.Trim(arr[i], "\r\n")
		if key == "" {
			continue
		}
		val, err := Get(key)
		if err != nil {
			fmt.Println("Error handling key: " + key + err.Error())
			continue
		}

		w.Write([]byte("VALUE " + key + " 0 " + strconv.Itoa(len(val)) + "\r\n"))
		w.Write([]byte(val + "\r\n"))
		w.Flush()
	}
	w.Write([]byte("END" + "\r\n"))
	w.Flush()

	return nil
}

//Parses incoming requests. Supports only "set" & "get | gets"
func parseRootCmd(c net.Conn, message string) {
	arr := strings.SplitN(string(message), " ", -1)
	switch arr[0] {
	case "set":
		err := handleSet(c, arr)
		if err != nil {
			return
		}
	case "get", "gets":
		err := handleGet(c, arr)
		if err != nil {
			return
		}
	default:
		fmt.Println("Received unknown keyword " + arr[0])
		c.Write([]byte("ERROR\r\n"))
	}
}

//Handles errors received while reading data from client
// If EOF is received, the connection is cleaned up
// Else an error is logged
func handleConnErr(c net.Conn, err error) {

	if err == io.EOF || err == io.ErrUnexpectedEOF {
		cleanup(c)
	} else if err != nil {
		fmt.Println(err)
	}

	return
}

// routine to handle an incoming connection
// behavior forks based on the connection state
func handleConnection(co net.Conn) {

	b := bufio.NewReader(co)

	conns.lock.Lock()
	c, ok := conns.m[co]

	if !ok {
		c = new(conn)
		if c == nil {
			//handle error
			return
		}
		c.state = Waiting
		conns.m[co] = c
	}

	conns.lock.Unlock()

	for {

		switch c.state {
		case Waiting:
			message, err := b.ReadBytes('\n')
			if err != nil {
				handleConnErr(co, err)
				return
			}
			buf := strings.Trim(string(message), "\r\n")
			parseRootCmd(co, buf)

		case WaitValue:
			message := make([]byte, c.ctx.size+2)
			n, err := io.ReadFull(b, message)
			if err != nil || n < len(message) {
				handleConnErr(co, err)
				return
			}
			buf := strings.Trim(string(message), "\r\n")
			err = handleSetValue(co, buf)
			updateState(co, Waiting)
		}
	}

	return
}

func main() {
	args := os.Args
	if len(args) > 2 {
		checkExit(errors.New("Usage: ./memcached <port>"))
		return
	}

	var addr string

	if len(args) == 1 {
		//If no port specified, use default port
		addr = ":11211"
	} else {
		addr = ":" + args[1]
	}

	//start listening
	l, err := net.Listen("tcp", addr)
	checkExit(err)

	defer l.Close()

	memcachedInit()

	for {
		c, err := l.Accept()
		utils.CheckExit(err)
		//goroutine to handle a connection
		//1:1 mapping between connection & a goroutine
		go handleConnection(c)
	}

	return
}
