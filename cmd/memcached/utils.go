//common utility functions for memcached
package main

import (
	"fmt"
	"os"
)

// Checks err and exits program execution upon non nil error
func checkExit(err error) {
	if err != nil {
		fmt.Println(err)
		os.Exit(1)
	}
	return
}
