package editor

import (
	"bytes"
	"fmt"
	"io"
	"io/ioutil"
	"math/rand"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"github.com/golang/glog"
	"github.com/pharmer/pharmer/utils"
	"github.com/pkg/errors"
	"k8s.io/apimachinery/pkg/util/yaml"
	"k8s.io/kubernetes/pkg/kubectl/util/term"
)

const defaultEditor = "nano"

type Editor struct {
	Args  []string
	Shell bool
}

func NewDefaultEditor() Editor {
	var editorName string
	editorName = os.Getenv("EDITOR")
	if len(editorName) != 0 {
		editorName = os.ExpandEnv(editorName)
		return Editor{
			Args:  []string{editorName},
			Shell: false,
		}
	}

	editorName = os.Getenv("KUBEDB_EDITOR")
	if len(editorName) != 0 {
		editorName = os.ExpandEnv(editorName)
		return Editor{
			Args:  []string{editorName},
			Shell: false,
		}
	}

	return Editor{
		Args:  []string{defaultEditor},
		Shell: false,
	}
}

func (e Editor) LaunchTempFile(prefix, suffix string, r io.Reader) ([]byte, string, error) {
	f, err := tempFile(prefix, suffix)
	if err != nil {
		return nil, "", err
	}
	defer f.Close()
	path := f.Name()
	if _, err := io.Copy(f, r); err != nil {
		os.Remove(path)
		return nil, path, err
	}
	// This file descriptor needs to close so the next process (Launch) can claim it.
	f.Close()
	if err := e.Launch(path); err != nil {
		return nil, path, err
	}
	bytes, err := ioutil.ReadFile(path)
	return bytes, path, err
}

func tempFile(prefix, suffix string) (f *os.File, err error) {
	dir := os.TempDir()

	for i := 0; i < 10000; i++ {
		name := filepath.Join(dir, prefix+randSeq(5)+suffix)
		f, err = os.OpenFile(name, os.O_RDWR|os.O_CREATE|os.O_EXCL, 0600)
		if os.IsExist(err) {
			continue
		}
		break
	}
	return
}

// Launch opens the described or returns an error. The TTY will be protected, and
// SIGQUIT, SIGTERM, and SIGINT will all be trapped.
func (e Editor) Launch(path string) error {
	if len(e.Args) == 0 {
		return errors.Errorf("no editor defined, can't open %s", path)
	}
	abs, err := filepath.Abs(path)
	if err != nil {
		return err
	}
	args := e.args(abs)
	cmd := exec.Command(args[0], args[1:]...)
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr
	cmd.Stdin = os.Stdin
	glog.V(5).Infof("Opening file with editor %v", args)
	if err := (term.TTY{In: os.Stdin, TryDev: true}).Safe(cmd.Run); err != nil {
		if err, ok := err.(*exec.Error); ok {
			if err.Err == exec.ErrNotFound {
				return errors.Errorf("unable to launch the editor %q", strings.Join(e.Args, " "))
			}
		}
		return errors.Errorf("there was a problem with the editor %q", strings.Join(e.Args, " "))
	}
	return nil
}

func (e Editor) args(path string) []string {
	args := make([]string, len(e.Args))
	copy(args, e.Args)
	if e.Shell {
		last := args[len(args)-1]
		args[len(args)-1] = fmt.Sprintf("%s %q", last, path)
	} else {
		args = append(args, path)
	}
	return args
}

var letters = []rune("abcdefghijklmnopqrstuvwxyz0123456789")

func randSeq(n int) string {
	b := make([]rune, n)
	for i := range b {
		b[i] = letters[rand.Intn(len(letters))]
	}
	return string(b)
}

func PreconditionFailedError() error {
	return errors.New(`At least one of the following was changed:
	apiVersion
	kind
	name
	namespace
	status`)
}

type EditReason struct {
	Head  string
	Other []string
}

type EditHeader struct {
	Reasons []EditReason
}

type EditResults struct {
	Header EditHeader
	File   string
}

// writeTo outputs the current header information into a stream
func (h *EditHeader) WriteTo(w io.Writer) (int64, error) {
	fmt.Fprint(w, `# Please edit the object below. Lines beginning with a '#' will be ignored,
# and an empty file will abort the edit. If an error occurs while saving this file will be
# reopened with the relevant failures.
#
`)
	for _, r := range h.Reasons {
		if len(r.Other) > 0 {
			fmt.Fprintf(w, "# %s:\n", r.Head)
		} else {
			fmt.Fprintf(w, "# %s\n", r.Head)
		}
		for _, o := range r.Other {
			fmt.Fprintf(w, "# * %s\n", o)
		}
		fmt.Fprintln(w, "#")
	}
	return 0, nil
}

func PreservedFile(err error, path string, out io.Writer) error {
	if len(path) > 0 {
		if _, err := os.Stat(path); !os.IsNotExist(err) {
			fmt.Fprintf(out, "A copy of your changes has been stored to %q\n", path)
		}
	}
	return err
}

func StripComments(file []byte) []byte {
	stripped := file
	stripped, err := yaml.ToJSON(stripped)
	if err != nil {
		stripped = ManualStrip(file)
	}
	return stripped
}

func ManualStrip(file []byte) []byte {
	stripped := []byte{}
	lines := bytes.Split(file, []byte("\n"))
	for i, line := range lines {
		if bytes.HasPrefix(bytes.TrimSpace(line), []byte("#")) {
			continue
		}
		stripped = append(stripped, line...)
		if i < len(lines)-1 {
			stripped = append(stripped, '\n')
		}
	}
	return stripped
}

func ConditionalPreconditionFailedError(kind string) error {
	str := utils.PreconditionSpecField[kind]
	strList := strings.Join(str, "\n\t")
	return errors.Errorf(`At least one of the following was changed:
	%v`, strList)
}
