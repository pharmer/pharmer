package inspector

import (
	"io"
	"net/url"
	"strings"
	"time"
	"github.com/appscode/go/log"
	. "github.com/pharmer/pharmer/cloud"
	"github.com/pkg/errors"
	core "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/client-go/kubernetes"
	"k8s.io/client-go/rest"
	"k8s.io/client-go/tools/remotecommand"
)

type RemoteExecutor interface {
	Execute(*rest.Config, string, *url.URL, []string) (string, error)
}

type RemoteBashExecutor struct{}

func (e *RemoteBashExecutor) Execute(config *rest.Config, method string, url *url.URL, cmds []string) (string, error) {
	exec, err := remotecommand.NewSPDYExecutor(config, method, url)
	if err != nil {
		return "", errors.Wrap(err, "failed to create executor")
	}
	stdIn := newStringReader(cmds)
	DefaultWriter.Flush()
	err = exec.Stream(remotecommand.StreamOptions{
		Stdin:  stdIn,
		Stdout: DefaultWriter,
		Stderr: DefaultWriter,
		Tty:    false,
	})
	if err != nil {
		log.Errorln("Error in exec", err)
		return "", errors.Wrap(err, "failed to exec")
	}
	return DefaultWriter.Output(), nil
}

type ExecOptions struct {
	Namespace     string
	PodName       string
	ContainerName string
	Command       []string

	Executor RemoteExecutor
	Client   kubernetes.Interface
	Config   *rest.Config
}

func (p *ExecOptions) Run(retry int) (string, error) {
	err := p.Validate()
	if err != nil {
		return "", errors.Wrap(err, "failed to validate")
	}
	var pod *core.Pod
	for i := 0; i < retry; i++ {
		pod, err = p.Client.CoreV1().Pods(p.Namespace).Get(p.PodName, metav1.GetOptions{})
		if err != nil || pod.Status.Phase != core.PodRunning {
			log.Debugln("pod not running waiting, tries", i+1)
			time.Sleep(time.Second * 30)
			continue
		}
		if pod.Status.Phase == core.PodRunning {
			log.Debugln("pod running quiting loop, tries", i+1)
			break
		}
	}
	if pod.Status.Phase != core.PodRunning || err != nil {
		return "", errors.Errorf("pod %s is not running and cannot execute commands; current phase is %s", p.PodName, pod.Status.Phase)
	}

	req := p.Client.CoreV1().RESTClient().Post().
		Resource("pods").
		Name(pod.Name).
		Namespace(pod.Namespace).
		SubResource("exec").
		Param("container", p.ContainerName).
		Param("command", "/bin/sh").
		Param("stdin", "true").
		Param("stdout", "false").
		Param("stderr", "false").
		Param("tty", "false")

	return p.Executor.Execute(p.Config, "POST", req.URL(), p.Command)
}

func (p *ExecOptions) Validate() error {
	if len(p.PodName) == 0 {
		return errors.New("pod name must be specified")
	}
	if len(p.Command) == 0 {
		return errors.New("you must specify at least one command for the container")
	}
	if p.Executor == nil || p.Client == nil || p.Config == nil {
		return errors.New("client, client config, and executor must be provided")
	}
	return nil
}

func newStringReader(ss []string) io.Reader {
	formattedString := strings.Join(ss, "\n")
	reader := strings.NewReader(formattedString)
	return reader
}
