package main

import (
	"log"
	"log/syslog"
)

func main() {
	if err := cmd(); err != nil {
		log.Println(err)
	}
}

func cmd() (err error) {
	log, err := syslog.Dial("tcp", "127.0.0.1:5140", syslog.LOG_DEBUG, "foo.bar")

	if err != nil {
		return
	}

	defer log.Close()

	log.Info("waazzaaaaa")
	return
}
