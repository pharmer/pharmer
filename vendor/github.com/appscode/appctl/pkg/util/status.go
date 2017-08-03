package util

import (
	"os"

	term "github.com/appscode/go-term"
	tracer "github.com/appscode/go-tracer"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
)

func PrintStatus(err error) {
	if s, ok := status.FromError(err); ok {
		tracer.SetStatus(s.Code().String())
		if s.Code() == codes.OK {
			return
		}
		term.Errorln(s.Code().String(), s.Message())
		for _, d := range s.Proto().Details {
			term.Errorln(d.TypeUrl, d.String())
		}
		os.Exit(1)
	}
	term.ExitOnError(err)
}
