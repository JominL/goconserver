package plugins

import (
	"fmt"
	"github.com/xcat2/goconserver/common"
	"golang.org/x/crypto/ssh"
	"os"
)

const (
	DRIVER_SSHCMD = "sshcmd"
)

// Experimental plugin, just for test
func init() {
	DRIVER_INIT_MAP[DRIVER_SSHCMD] = NewSSHCMDConsole
	DRIVER_VALIDATE_MAP[DRIVER_SSHCMD] = func(name string, params map[string]string) error {
		if _, ok := params["host"]; !ok {
			return common.NewErr(common.INVALID_PARAMETER, fmt.Sprintf("%s: Please specify the parameter host", name))
		}
		if _, ok := params["user"]; !ok {
			return common.NewErr(common.INVALID_PARAMETER, fmt.Sprintf("%s: Please specify the parameter user", name))
		}
		_, ok1 := params["password"]
		_, ok2 := params["private_key"]
		if !ok1 && !ok2 {
			return common.NewErr(common.INVALID_PARAMETER, fmt.Sprintf("%s: Please specify the parameter private_key or password", name))
		}
		if _, ok := params["cmd"]; !ok {
			return common.NewErr(common.INVALID_PARAMETER, fmt.Sprintf("%s: Please specify the parameter cmd", name))
		}
		return nil
	}
}

type SSHCMDConsole struct {
	*SSHConsole
	string
}

func NewSSHCMDConsole(node string, params map[string]string) (ConsolePlugin, error) {
	sshConsole, err := NewSSHConsole(node, params)
	if err != nil {
		return nil, err
	}
	if _, ok := params["cmd"]; !ok {
		return nil, err
	}
	return &SSHCMDConsole{sshConsole.(*SSHConsole), params["cmd"]}, nil
}

func (s *SSHCMDConsole) startConsole() (*BaseSession, error) {
	tty := common.Tty{}
	ttyWidth, ttyHeight, err := tty.GetSize(os.Stdin)
	if err != nil {
		plog.DebugNode(s.node, "Could not get tty size, use 80,80 as default")
		ttyHeight = 80
		ttyWidth = 80
	}
	modes := ssh.TerminalModes{
		ssh.ECHO:          1,     // Disable echoing
		ssh.TTY_OP_ISPEED: 14400, // input speed = 14.4kbaud
		ssh.TTY_OP_OSPEED: 14400, // output speed = 14.4kbaud
	}
	if err := s.session.RequestPty("xterm-256color", ttyWidth, ttyHeight, modes); err != nil {
		plog.ErrorNode(s.node, err.Error())
		return nil, err
	}
	sshIn, err := s.session.StdinPipe()
	if err != nil {
		plog.ErrorNode(s.node, err.Error())
		return nil, err
	}
	sshOut, err := s.session.StdoutPipe()
	if err != nil {
		plog.ErrorNode(s.node, err.Error())
		return nil, err
	}
	// Start command on remote
	if err := s.session.Start(s.string); err != nil {
		plog.ErrorNode(s.node, err.Error())
		return nil, err
	}
	return &BaseSession{In: sshIn, Out: sshOut, Session: s}, nil
}

// FIXME(chenglch): The original plan is to rewrite the startConsole function only, but seems golang do not support that
// kind of inherit, I have to copy the Start function from ssh plugin.
func (s *SSHCMDConsole) Start() (*BaseSession, error) {
	err := s.connectToHost()
	if err != nil {
		plog.ErrorNode(s.node, err.Error())
		return nil, err
	}
	baseSession, err := s.startConsole()
	if err != nil {
		plog.ErrorNode(s.node, err.Error())
		return nil, err
	}
	return baseSession, nil
}
