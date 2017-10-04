package cloud

import (
	"io"
	"os"
	"strings"

	"github.com/appscode/log"
	"golang.org/x/crypto/ssh"
)

func ExecuteCommand(command, addr string, config *ssh.ClientConfig) (string, error) {
	conn, err := ssh.Dial("tcp", addr, config)
	if err != nil {
		return "", err
	}
	defer conn.Close()
	session, err := conn.NewSession()
	if err != nil {
		return "", err
	}
	session.Stdout = DefaultWriter
	session.Stderr = DefaultWriter
	session.Stdin = os.Stdin
	session.Run(command)
	output := DefaultWriter.Output()
	session.Close()
	return output, nil
}

var DefaultWriter = &StringWriter{
	data: make([]byte, 0),
}

type StringWriter struct {
	data []byte
}

func (s *StringWriter) Flush() {
	s.data = make([]byte, 0)
}

func (s *StringWriter) Output() string {
	return string(s.data)
}

func (s *StringWriter) Write(b []byte) (int, error) {
	log.Infoln("$ ", string(b))
	s.data = append(s.data, b...)
	return len(b), nil
}

func newStringReader(ss []string) io.Reader {
	formattedString := strings.Join(ss, "\n")
	reader := strings.NewReader(formattedString)
	return reader
}
