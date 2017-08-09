package cmds

import (
	"os"
	"os/exec"

	appscodeSSH "github.com/appscode/api/ssh/v1beta1"
	"github.com/appscode/appctl/pkg/config"
	"github.com/appscode/appctl/pkg/ssh"
	"github.com/appscode/appctl/pkg/util"
	"github.com/appscode/client/cli"
	term "github.com/appscode/go-term"
	"github.com/mgutz/str"
	"github.com/spf13/cobra"
)

func NewCmdSSH() *cobra.Command {
	var req appscodeSSH.SSHGetRequest

	cmd := &cobra.Command{
		Use:     "ssh",
		Short:   "SSH into a Kubernetes cluster instance",
		Long:    `SSH into a cluster instance.`,
		Example: `appctl cluster ssh -c cluster-name node-name`,
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
				ssh.OpenShell(resp.SshKey.PrivateKey, resp.InstanceAddr, resp.InstancePort, resp.User)
			}
		},
	}
	cmd.Flags().StringVarP(&req.ClusterName, "cluster", "c", "", "Cluster name")

	return cmd
}
