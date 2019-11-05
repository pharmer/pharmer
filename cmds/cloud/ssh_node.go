/*
Copyright The Pharmer Authors.

Licensed under the Apache License, Version 2.0 (the "License");
you may not use this file except in compliance with the License.
You may obtain a copy of the License at

    http://www.apache.org/licenses/LICENSE-2.0

Unless required by applicable law or agreed to in writing, software
distributed under the License is distributed on an "AS IS" BASIS,
WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
See the License for the specific language governing permissions and
limitations under the License.
*/
package cloud

import (
	"fmt"
	"os"

	"pharmer.dev/pharmer/cloud"
	"pharmer.dev/pharmer/cmds/cloud/options"
	"pharmer.dev/pharmer/store"

	"github.com/appscode/go/term"
	"github.com/spf13/cobra"
	"golang.org/x/crypto/ssh"
	"golang.org/x/crypto/ssh/terminal"
)

func NewCmdSSH() *cobra.Command {
	opts := options.NewNodeSSHConfig()
	cmd := &cobra.Command{
		Use:               "node",
		Short:             "SSH into a Kubernetes cluster instance",
		Long:              `SSH into a cluster instance.`,
		Example:           `pharmer ssh node -k cluster-name node-name`,
		DisableAutoGenTag: true,
		Run: func(cmd *cobra.Command, args []string) {
			if err := opts.ValidateFlags(cmd, args); err != nil {
				term.Fatalln(err)
			}

			storeProvider, err := store.GetStoreProvider(cmd)
			term.ExitOnError(err)

			sshConfig, err := cloud.GetSSHConfig(storeProvider, opts.ClusterName, opts.NodeName)
			term.ExitOnError(err)

			err = OpenShell(sshConfig.PrivateKey, sshConfig.HostIP, sshConfig.HostPort, sshConfig.User)
			if err != nil {
				term.Fatalln(err)
			}
		},
	}
	opts.AddFlags(cmd.Flags())

	return cmd
}

// http://stackoverflow.com/questions/26315572/ssh-executing-nsenter-as-remote-command-with-interactive-shell-in-golang-to-debu
func OpenShell(privateKey []byte, addr string, port int32, user string) error {
	keySigner, err := ssh.ParsePrivateKey(privateKey)
	term.ExitOnError(err)

	// Create client config
	config := &ssh.ClientConfig{
		User: user,
		Auth: []ssh.AuthMethod{
			ssh.PublicKeys(keySigner),
		},
		HostKeyCallback: ssh.InsecureIgnoreHostKey(),
	}

	// Connect to ssh server
	conn, err := ssh.Dial("tcp", fmt.Sprintf("%v:%v", addr, port), config)
	term.ExitOnError(err)

	defer conn.Close()

	// Create a session
	session, err := conn.NewSession()
	term.ExitOnError(err)

	// The following two lines makes the terminal work properly because of
	// side-effects I don't understand.
	fd := int(os.Stdin.Fd())
	oldState, err := terminal.MakeRaw(fd)
	term.ExitOnError(err)

	session.Stdout = os.Stdout
	session.Stderr = os.Stderr
	session.Stdin = os.Stdin

	termWidth, termHeight, err := terminal.GetSize(fd)
	term.ExitOnError(err)

	// Set up terminal modes
	modes := ssh.TerminalModes{
		ssh.ECHO:          1,     // enable echoing
		ssh.TTY_OP_ISPEED: 14400, // input speed = 14.4kbaud
		ssh.TTY_OP_OSPEED: 14400, // output speed = 14.4kbaud
	}

	// Request pseudo terminal
	if err := session.RequestPty("xterm-256color", termHeight, termWidth, modes); err != nil {
		term.ExitOnError(err)
	}

	err = session.Shell()
	if err != nil {
		return err
	}

	err = session.Wait()
	if err != nil {
		return err
	}
	err = terminal.Restore(fd, oldState)
	if err != nil {
		return err
	}
	return nil
}
