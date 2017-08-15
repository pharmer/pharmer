package cmds

import (
	"fmt"
	"os"
	"os/exec"

	appscodeSSH "github.com/appscode/api/ssh/v1beta1"
	"github.com/appscode/appctl/pkg/config"
	"github.com/appscode/appctl/pkg/util"
	"github.com/appscode/client/cli"
	term "github.com/appscode/go-term"
	"github.com/mgutz/str"
	"github.com/spf13/cobra"
	"golang.org/x/crypto/ssh"
	"golang.org/x/crypto/ssh/terminal"
)

func NewCmdSSH() *cobra.Command {
	var req appscodeSSH.SSHGetRequest

	cmd := &cobra.Command{
		Use:               "ssh",
		Short:             "SSH into a Kubernetes cluster instance",
		Long:              `SSH into a cluster instance.`,
		Example:           `appctl cluster ssh -c cluster-name node-name`,
		DisableAutoGenTag: true,
		Run: func(cmd *cobra.Command, args []string) {
			if len(args) > 0 {
				req.InstanceName = args[0]
			} else {
				term.Fatalln("Missing instance name")
			}

			c := config.ClientOrDie()
			req.Namespace = cli.GetAuthOrDie().TeamId
			resp, err := c.SSH().Get(c.Context(), &req)
			util.PrintStatus(err)

			// Closing the connection so no idle connection stays alive.
			c.Close()

			if resp.Command != "" {
				term.Infoln("Running", resp.Command)
				arg := str.ToArgv(resp.Command)
				name, arg := arg[0], arg[1:]
				cmd := exec.Command(name, arg...)
				cmd.Stdin = os.Stdin
				cmd.Stdout = os.Stdout
				cmd.Stderr = os.Stderr

				err = cmd.Start()
				if err != nil {
					term.Fatalln("Failed to execute commnand", err)
				}
				err = cmd.Wait()
				if err != nil {
					term.Fatalln("Error waiting for command", err)
				}
			} else {
				openShell(resp.SshKey.PrivateKey, resp.InstanceAddr, resp.InstancePort, resp.User)
			}
		},
	}
	cmd.Flags().StringVarP(&req.ClusterName, "cluster", "c", "", "Cluster name")

	return cmd
}

// http://stackoverflow.com/questions/26315572/ssh-executing-nsenter-as-remote-command-with-interactive-shell-in-golang-to-debu
func openShell(privateKey []byte, addr string, port int32, user string) {
	keySigner, err := ssh.ParsePrivateKey(privateKey)

	term.ExitOnError(err)
	// Create client config
	config := &ssh.ClientConfig{
		User: user,
		Auth: []ssh.AuthMethod{
			ssh.PublicKeys(keySigner),
		},
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

	session.Shell()

	session.Wait()
	terminal.Restore(fd, oldState)
}
