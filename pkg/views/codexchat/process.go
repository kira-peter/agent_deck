package codexchat

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"os/exec"
	"strings"
	"time"
)

type rpcMessage struct {
	Method string          `json:"method"`
	ID     json.RawMessage `json:"id"`
	Params json.RawMessage `json:"params"`
	Result json.RawMessage `json:"result"`
	Error  *struct {
		Message string `json:"message"`
	} `json:"error"`
}

type processEvent struct {
	message rpcMessage
	err     error
}

type appServer struct {
	cmd    *exec.Cmd
	stdin  io.WriteCloser
	writer *json.Encoder
	stop   context.CancelFunc
	done   chan struct{}
}

func startServer(cwd, home string, deliver func(processEvent)) (*appServer, error) {
	path, err := exec.LookPath("codex")
	if err != nil {
		return nil, err
	}
	cmd := serverCommand(path)
	cmd.Dir = cwd
	cmd.Env = append(cmd.Environ(), "CODEX_HOME="+home)
	var stderr bytes.Buffer
	cmd.Stderr = &stderr
	stdin, err := cmd.StdinPipe()
	if err != nil {
		return nil, err
	}
	stdout, err := cmd.StdoutPipe()
	if err != nil {
		stdin.Close()
		return nil, err
	}
	if err := cmd.Start(); err != nil {
		stdin.Close()
		stdout.Close()
		return nil, err
	}
	ctx, cancel := context.WithCancel(context.Background())
	s := &appServer{cmd: cmd, stdin: stdin, writer: json.NewEncoder(stdin), stop: cancel, done: make(chan struct{})}
	go func() {
		decoder := json.NewDecoder(stdout)
		var readErr error
		for ctx.Err() == nil {
			var msg rpcMessage
			readErr = decoder.Decode(&msg)
			if readErr != nil || ctx.Err() != nil {
				break
			}
			deliver(processEvent{message: msg})
		}
		if readErr != nil && !errors.Is(readErr, io.EOF) {
			_ = cmd.Process.Kill()
		}
		waitErr := cmd.Wait()
		close(s.done)
		if ctx.Err() == nil {
			err := errors.New("Codex app-server exited")
			switch {
			case readErr != nil && !errors.Is(readErr, io.EOF):
				err = fmt.Errorf("decode Codex app-server response: %w", readErr)
			case strings.TrimSpace(stderr.String()) != "":
				err = errors.New(strings.TrimSpace(stderr.String()))
			case waitErr != nil:
				err = waitErr
			}
			deliver(processEvent{err: err})
		}
	}()
	return s, nil
}

func (s *appServer) close() {
	s.stop()
	_ = s.stdin.Close()
	select {
	case <-s.done:
	case <-time.After(750 * time.Millisecond):
		_ = s.cmd.Process.Kill()
	}
}
