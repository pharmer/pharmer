package cloud

import (
	"fmt"
	"io"
	"os"
	"os/exec"
	"strings"

	"github.com/appscode/go/log"
	"golang.org/x/crypto/ssh"
)

func ExecuteTCPCommand(command, addr string, config *ssh.ClientConfig) (string, error) {
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
	if config.User != "root" {
		command = fmt.Sprintf("sudo %s", command)
	}
	session.Run(command)
	output := DefaultWriter.Output()
	session.Close()
	return output, nil
}

func ExecuteSSHCommand(name string, arg []string, stdIn io.Reader) (string, error) {
	cmd := exec.Command(name, arg...)
	cmd.Stdin = stdIn
	cmd.Stdout = DefaultWriter
	cmd.Stderr = DefaultWriter
	err := cmd.Run()
	output := DefaultWriter.Output()
	return output, err
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

func NewStringReader(ss []string) io.Reader {
	formattedString := strings.Join(ss, "\n")
	reader := strings.NewReader(formattedString)
	return reader
}
