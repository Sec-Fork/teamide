package ssh

import (
	"github.com/team-ide/go-tool/util"
	"go.uber.org/zap"
	"golang.org/x/crypto/ssh"
	"io"
	"sync"
	"teamide/pkg/terminal"
)

func NewTerminalService(config *Config) (res *terminalService) {
	res = &terminalService{
		config: config,
	}
	return
}

type terminalService struct {
	config       *Config
	sshClient    *ssh.Client
	sshSession   *ssh.Session
	stdout       io.Reader
	stdin        io.Writer
	onClose      func()
	readeLock    sync.Mutex
	readeErrLock sync.Mutex
	writeLock    sync.Mutex
}

func (this_ *terminalService) IsWindows() (isWindows bool, err error) {
	isWindows = false
	return
}

func (this_ *terminalService) Stop() {
	if this_.sshSession != nil {
		_ = this_.sshSession.Close()
	}
	if this_.sshClient != nil {
		_ = this_.sshClient.Close()
	}
	if this_.stdout != nil {
		if readerCloser, ok := this_.stdout.(io.ReadCloser); ok {
			_ = readerCloser.Close()
		}
	}
	if this_.stdin != nil {
		if writeCloser, ok := this_.stdin.(io.WriteCloser); ok {
			_ = writeCloser.Close()
		}
	}
}

func (this_ *terminalService) ChangeSize(size *terminal.Size) (err error) {

	if this_.sshSession == nil {
		return
	}
	if size.Cols > 0 && size.Rows > 0 {
		err = this_.sshSession.WindowChange(size.Rows, size.Cols)
		if err != nil {
			util.Logger.Error("SSH Session Window Change error", zap.Error(err))
			return
		}
	}
	return
}

func (this_ *terminalService) Start(size *terminal.Size) (err error) {

	this_.sshClient, err = NewClient(*this_.config)
	if err != nil {
		util.Logger.Error("SSH NewClient error", zap.Error(err))
		return
	}
	util.Logger.Info("SSH NewClient success", zap.Any("address", this_.config.Address))
	go func() {
		err = this_.sshClient.Wait()
		this_.Stop()
		util.Logger.Info("SSH Client end", zap.Any("address", this_.config.Address))
	}()

	this_.sshSession, err = this_.sshClient.NewSession()
	if err != nil {
		util.Logger.Error("SSH NewSession Error", zap.Error(err))
		return
	}
	util.Logger.Info("SSH NewSession success", zap.Any("address", this_.config.Address))

	err = NewSSHShell(size, this_.sshSession)
	if err != nil {
		util.Logger.Error("Create SSH Shell Error", zap.Error(err))
		return
	}
	util.Logger.Info("SSH Format Shell Session success", zap.Any("address", this_.config.Address))
	this_.stdout, err = this_.sshSession.StdoutPipe()
	if err != nil {
		util.Logger.Error("ssh session StdoutPipe error", zap.Error(err))
		return
	}
	this_.stdin, err = this_.sshSession.StdinPipe()
	if err != nil {
		util.Logger.Error("ssh session StdinPipe error", zap.Error(err))
		return
	}

	return
}

func (this_ *terminalService) Write(buf []byte) (n int, err error) {
	this_.writeLock.Lock()
	defer this_.writeLock.Unlock()

	n = len(buf)
	err = util.Write(this_.stdin, buf, nil)
	return
}

func (this_ *terminalService) Read(buf []byte) (n int, err error) {
	this_.readeLock.Lock()
	defer this_.readeLock.Unlock()

	n, err = this_.stdout.Read(buf)
	return
}
