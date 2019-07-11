package apiserver

import (
	"io"
	"log"

	stan "github.com/nats-io/stan.go"
)

type Apiserver struct {
	natsConn stan.Conn
}

func New(conn stan.Conn) *Apiserver {
	//defer logCloser(conn)
	return &Apiserver{natsConn: conn}
}

func LogCloser(c io.Closer) {
	if err := c.Close(); err != nil {
		log.Printf("close error: %s", err)
	}
}
