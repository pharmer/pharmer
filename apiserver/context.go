package apiserver

import (
	"context"
	"io"
	"log"

	stan "github.com/nats-io/stan.go"
)

type Apiserver struct {
	ctx      context.Context
	natsConn stan.Conn
}

func New(ctx context.Context, conn stan.Conn) *Apiserver {
	//defer logCloser(conn)
	return &Apiserver{ctx: ctx, natsConn: conn}
}

func LogCloser(c io.Closer) {
	if err := c.Close(); err != nil {
		log.Printf("close error: %s", err)
	}
}
