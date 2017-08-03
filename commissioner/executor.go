package commissioner

import (
	"io"
	"net/url"
	"strings"
	"time"

	"github.com/appscode/errors"
	"github.com/appscode/log"
	"k8s.io/apimachinery/pkg/apis/meta/internalversion"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	clientset "k8s.io/client-go/kubernetes"
	apiv1 "k8s.io/client-go/pkg/api/v1"
	"k8s.io/client-go/rest"
	"k8s.io/kubernetes/pkg/client/unversioned/remotecommand"
	remotecommandserver "k8s.io/kubernetes/pkg/kubelet/server/remotecommand"
)

type RemoteExecutor interface {
	Execute(*rest.Config, string, *url.URL, []string) (string, error)
}

type RemoteBashExecutor struct{}

func (e *RemoteBashExecutor) Execute(config *rest.Config, method string, url *url.URL, cmds []string) (string, error) {
	exec, err := remotecommand.NewExecutor(config, method, url)
	if err != nil {
		return "", errors.FromErr(err).WithMessage("failed to create executor").Err()
	}
	stdIn := newStringReader(cmds)
	DefaultWriter.Flush()
	err = exec.Stream(remotecommand.StreamOptions{
		SupportedProtocols: remotecommandserver.SupportedStreamingProtocols,
		Stdin:              stdIn,
		Stdout:             DefaultWriter,
		Stderr:             DefaultWriter,
		Tty:                false,
	})
	if err != nil {
		log.Errorln("Error in exec", err)
		return "", errors.FromErr(err).WithMessage("failed to exec").Err()
	}
	return DefaultWriter.Output(), nil
}

type ExecOptions struct {
	Namespace     string
	PodName       string
	ContainerName string
	Command       []string

	Executor RemoteExecutor
	Client   clientset.Interface
	Config   *rest.Config
}

func (p *ExecOptions) Run(retry int) (string, error) {
	err := p.Validate()
	if err != nil {
		return "", errors.FromErr(err).WithMessage("failed to validate").Err()
	}
	var pod *apiv1.Pod
	for i := 0; i < retry; i++ {
		pod, err = p.Client.CoreV1().Pods(p.Namespace).Get(p.PodName, metav1.GetOptions{})
		if err != nil || pod.Status.Phase != apiv1.PodRunning {
			log.Debugln("pod not running waiting, tries", i+1)
			time.Sleep(time.Second * 30)
			continue
		}
		if pod.Status.Phase == apiv1.PodRunning {
			log.Debugln("pod running quiting loop, tries", i+1)
			break
		}
	}
	if pod.Status.Phase != apiv1.PodRunning || err != nil {
		return "", errors.Newf("pod %s is not running and cannot execute commands; current phase is %s", p.PodName, pod.Status.Phase).Err()
	}

	req := p.Client.CoreV1().RESTClient().Post().
		Resource("pods").
		Name(pod.Name).
		Namespace(pod.Namespace).
		SubResource("exec").
		Param("container", p.ContainerName)

	req.VersionedParams(&apiv1.PodExecOptions{
		Container: p.ContainerName,
		Command:   []string{"/bin/bash"},
		Stdin:     true,
		Stdout:    true,
		Stderr:    false,
		TTY:       false,
	}, internalversion.ParameterCodec)

	return p.Executor.Execute(p.Config, "POST", req.URL(), p.Command)
}

func (p *ExecOptions) Validate() error {
	if len(p.PodName) == 0 {
		return errors.New("pod name must be specified").Err()
	}
	if len(p.Command) == 0 {
		return errors.New("you must specify at least one command for the container").Err()
	}
	if p.Executor == nil || p.Client == nil || p.Config == nil {
		return errors.New("client, client config, and executor must be provided").Err()
	}
	return nil
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
	//log.Infoln("$ ", string(b))
	s.data = append(s.data, b...)
	return len(b), nil
}

func newStringReader(ss []string) io.Reader {
	formattedString := strings.Join(ss, "\n")
	reader := strings.NewReader(formattedString)
	return reader
}
