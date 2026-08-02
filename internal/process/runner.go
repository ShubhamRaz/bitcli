// Package process owns safe external process execution for backend adapters.
package process

import (
	"bufio"
	"bytes"
	"context"
	"errors"
	"io"
	"os/exec"
	"sync"
)

// Event is a chunk of stdout or stderr emitted by an external process.
type Event struct {
	Stream string
	Text   string
}

// Result describes a completed process.
type Result struct {
	ExitCode int
}

// Runner starts commands through exec.CommandContext with argument slices.
type Runner struct{}

// NewRunner creates a process runner.
func NewRunner() *Runner {
	return &Runner{}
}

// RunStream starts a process and streams stdout and stderr as text events.
func (r *Runner) RunStream(ctx context.Context, dir, name string, args ...string) (<-chan Event, <-chan error) {
	events := make(chan Event, 32)
	errs := make(chan error, 1)

	go func() {
		defer close(events)
		defer close(errs)

		cmd := exec.CommandContext(ctx, name, args...)
		if dir != "" {
			cmd.Dir = dir
		}
		cmd.Stdin = bytes.NewReader(nil)
		stdout, err := cmd.StdoutPipe()
		if err != nil {
			errs <- err
			return
		}
		stderr, err := cmd.StderrPipe()
		if err != nil {
			errs <- err
			return
		}
		if err := cmd.Start(); err != nil {
			errs <- err
			return
		}

		var wg sync.WaitGroup
		wg.Add(2)
		go scanStream(ctx, &wg, stdout, "stdout", events)
		go scanStream(ctx, &wg, stderr, "stderr", events)
		wg.Wait()

		if err := cmd.Wait(); err != nil {
			errs <- err
			return
		}
		errs <- nil
	}()

	return events, errs
}

// RunWait runs a command and waits for it to finish without streaming output.
func (r *Runner) RunWait(ctx context.Context, dir, name string, args ...string) error {
	cmd := exec.CommandContext(ctx, name, args...)
	if dir != "" {
		cmd.Dir = dir
	}
	return cmd.Run()
}

func scanStream(ctx context.Context, wg *sync.WaitGroup, reader io.Reader, stream string, events chan<- Event) {
	defer wg.Done()
	scanner := bufio.NewScanner(reader)
	scanner.Buffer(make([]byte, 4096), 1024*1024)
	for scanner.Scan() {
		select {
		case <-ctx.Done():
			return
		case events <- Event{Stream: stream, Text: scanner.Text() + "\n"}:
		}
	}
	if err := scanner.Err(); err != nil && !errors.Is(err, context.Canceled) {
		select {
		case events <- Event{Stream: stream, Text: err.Error()}:
		case <-ctx.Done():
		}
	}
}

