package io_test

import (
	"context"
	"errors"
	"fmt"
	"testing"

	"github.com/redjack/marionette"
	"github.com/redjack/marionette/mock"
	"github.com/redjack/marionette/plugins/io"
)

func TestPuts(t *testing.T) {
	t.Run("OK", func(t *testing.T) {
		conn := mock.DefaultConn()
		conn.WriteFn = func(p []byte) (int, error) {
			if string(p) != "foo" {
				t.Fatalf("unexpected write: %q", p)
			}
			copy(p, []byte("foo"))
			return 3, nil
		}
		fsm := mock.NewFSM(&conn, marionette.NewStreamSet())
		fsm.PartyFn = func() string { return marionette.PartyClient }

		if err := io.Puts(context.Background(), &fsm, "foo"); err != nil {
			t.Fatal(err)
		}
	})

	// Ensure writes are continually attempted if there is a timeout error.
	t.Run("Timeout", func(t *testing.T) {
		var i int
		conn := mock.DefaultConn()
		conn.WriteFn = func(p []byte) (int, error) {
			defer func() { i++ }()
			switch i {
			case 0:
				return 1, &TimeoutError{}
			case 1:
				return 0, &TimeoutError{}
			case 2:
				return 2, nil
			default:
				return 0, fmt.Errorf("too many writes: %d", i)
			}
		}
		fsm := mock.NewFSM(&conn, marionette.NewStreamSet())
		fsm.PartyFn = func() string { return marionette.PartyClient }

		if err := io.Puts(context.Background(), &fsm, "foo"); err != nil {
			t.Fatal(err)
		}
	})

	t.Run("ErrNotEnoughArguments", func(t *testing.T) {
		conn := mock.DefaultConn()
		fsm := mock.NewFSM(&conn, marionette.NewStreamSet())
		fsm.PartyFn = func() string { return marionette.PartyClient }
		if err := io.Puts(context.Background(), &fsm); err == nil || err.Error() != `not enough arguments` {
			t.Fatalf("unexpected error: %q", err)
		}
	})

	t.Run("ErrInvalidArgument", func(t *testing.T) {
		conn := mock.DefaultConn()
		fsm := mock.NewFSM(&conn, marionette.NewStreamSet())
		fsm.PartyFn = func() string { return marionette.PartyClient }
		if err := io.Puts(context.Background(), &fsm, 123); err == nil || err.Error() != `invalid argument type` {
			t.Fatalf("unexpected error: %q", err)
		}
	})

	// Ensure write errors are passed through.
	t.Run("ErrWrite", func(t *testing.T) {
		errMarker := errors.New("marker")
		conn := mock.DefaultConn()
		conn.WriteFn = func(p []byte) (int, error) {
			return 0, errMarker
		}
		fsm := mock.NewFSM(&conn, marionette.NewStreamSet())
		fsm.PartyFn = func() string { return marionette.PartyClient }

		if err := io.Puts(context.Background(), &fsm, "foo"); err != errMarker {
			t.Fatalf("unexpected error: %q", err)
		}
	})
}

type TimeoutError struct{}

func (e TimeoutError) Error() string { return "timeout" }
func (e TimeoutError) Timeout() bool { return true }
