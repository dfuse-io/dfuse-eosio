// Copied from MIT licensed https://github.com/rendon/testcli
package tests

import (
	"bytes"
	"context"
	"io"
	"os"
	"os/exec"
	"regexp"
	"strings"
	"syscall"
)

// Command is typically constructed through the Command() call and provides state
// to the execution engine.
type Command struct {
	cmd        *exec.Cmd
	env        []string
	exitError  error
	running    bool
	terminated bool
	stdoutBuf  *bytes.Buffer
	stderrBuf  *bytes.Buffer
	stdout     string
	stderr     string
	stdin      io.Reader
}

func NewCommand(ctx context.Context, name string, arg ...string) *Command {
	return &Command{
		cmd: exec.CommandContext(ctx, name, arg...),
	}
}

func (c *Command) validate() {
	if !c.terminated {
		panic("You must call 'Start' then 'Wait' on your cmd instance.")
	}
}

// SetEnv overwrites the environment with the provided one. Otherwise, the
// parent environment will be supplied.
func (c *Command) SetEnv(env []string) {
	c.env = env
}

// SetStdin sets the stdin stream. It makes no attempt to determine if the
// command accepts anything over stdin.
func (c *Command) SetStdin(stdin io.Reader) {
	c.stdin = stdin
}

// Run runs the command.
func (c *Command) Start() {
	if c.stdin != nil {
		c.cmd.Stdin = c.stdin
	}

	if c.env != nil {
		c.cmd.Env = c.env
	} else {
		c.cmd.Env = os.Environ()
	}

	c.stdoutBuf = &bytes.Buffer{}
	c.cmd.Stdout = c.stdoutBuf

	c.stderrBuf = &bytes.Buffer{}
	c.cmd.Stderr = c.stderrBuf

	if err := c.cmd.Start(); err != nil {
		c.exitError = err
		c.terminated = true
		return
	}

	c.running = true
}

func (c *Command) SendInterrupt() {
	if err := c.cmd.Process.Signal(syscall.SIGINT); err != nil {
		c.exitError = err
	}
}

func (c *Command) Wait() {
	if err := c.cmd.Wait(); err != nil {
		c.exitError = err
	}

	c.stdout = string(c.stdoutBuf.Bytes())
	c.stderr = string(c.stderrBuf.Bytes())
	c.terminated = true
}

// Error is the command's error, if any.
func (c *Command) Error() error {
	return c.exitError
}

// Stdout stream for the command
func (c *Command) Stdout() string {
	c.validate()
	return c.stdout
}

// Stderr stream for the command
func (c *Command) Stderr() string {
	c.validate()
	return c.stderr
}

// StdoutContains determines if command's STDOUT contains `str`, this operation
// is case insensitive.
func (c *Command) StdoutContains(str string) bool {
	c.validate()
	str = strings.ToLower(str)
	return strings.Contains(strings.ToLower(c.stdout), str)
}

// StderrContains determines if command's STDERR contains `str`, this operation
// is case insensitive.
func (c *Command) StderrContains(str string) bool {
	c.validate()
	str = strings.ToLower(str)
	return strings.Contains(strings.ToLower(c.stderr), str)
}

// Success is a boolean status which indicates if the program exited non-zero
// or not.
func (c *Command) Success() bool {
	c.validate()
	return c.exitError == nil
}

// Failure is the inverse of Success().
func (c *Command) Failure() bool {
	c.validate()
	return c.exitError != nil
}

// StdoutMatches compares a regex to the stdout produced by the command.
func (c *Command) StdoutMatches(regex string) bool {
	c.validate()
	re := regexp.MustCompile(regex)
	return re.MatchString(c.Stdout())
}

// StderrMatches compares a regex to the stderr produced by the command.
func (c *Command) StderrMatches(regex string) bool {
	c.validate()
	re := regexp.MustCompile(regex)
	return re.MatchString(c.Stderr())
}
