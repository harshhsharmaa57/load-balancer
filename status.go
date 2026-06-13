package main

import (
	"net"
	"net/url"
	"time"
)

func isBackendAlive(u *url.URL) bool {
	con, err := net.DialTimeout(
		"tcp",
		u.Host,
		2*time.Second,
	)

	if err != nil {
		return false
	}
	con.Close()
	return true
}
