package cmds

import (
	"context"
	"fmt"
	"os"

	"github.com/appscode/go/log"
	"github.com/appscode/go/term"
	"github.com/pharmer/pharmer/cloud"
	"github.com/pharmer/pharmer/cloud/cmds/options"
	"github.com/pharmer/pharmer/config"
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
			cfgFile, _ := config.GetConfigFile(cmd.Flags())
			cfg, err := config.LoadConfig(cfgFile)
			if err != nil {
				log.Fatalln(err)
			}
			ctx := cloud.NewContext(context.Background(), cfg, config.GetEnv(cmd.Flags()))

			cluster, err := cloud.Store(ctx).Owner(opts.Owner).Clusters().Get(opts.ClusterName)
			if err != nil {
				term.Fatalln(err)
			}
			sshConfig, err := cloud.GetSSHConfig(ctx, opts.Owner, opts.NodeName, cluster)
			if err != nil {
				log.Fatalln(err)
			}
			OpenShell(sshConfig.PrivateKey, sshConfig.HostIP, sshConfig.HostPort, sshConfig.User)
		},
	}
	opts.AddFlags(cmd.Flags())

	return cmd
}

// http://stackoverflow.com/questions/26315572/ssh-executing-nsenter-as-remote-command-with-interactive-shell-in-golang-to-debu
func OpenShell(privateKey []byte, addr string, port int32, user string) {
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

	session.Shell()

	session.Wait()
	terminal.Restore(fd, oldState)
}
