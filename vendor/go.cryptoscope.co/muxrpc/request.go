package muxrpc // import "go.cryptoscope.co/muxrpc"

import (
	"context"
	"os"
	"strings"

	"github.com/pkg/errors"

	"go.cryptoscope.co/luigi"
	"go.cryptoscope.co/muxrpc/codec"
)

type Method []string

func (m Method) String() string {
	return strings.Join(m, ".")
}

// Request assembles the state of an RPC call
type Request struct {
	// Stream allows sending and receiving packets
	Stream Stream `json:"-"`

	// Method is the name of the called function
	Method Method `json:"name"`
	// Args contains the call arguments
	Args []interface{} `json:"args"`
	// Type is the type of the call, i.e. async, sink, source or duplex
	Type CallType `json:"type"`

	// in is the sink that incoming packets are passed to
	in luigi.Sink

	// same as packet.Req - the numerical identifier for the stream
	id int32

	// tipe is a value that has the type of data we expect to receive.
	// This is needed for unmarshaling JSON.
	tipe interface{}
}

// Return is a helper that returns on an async call
func (req *Request) Return(ctx context.Context, v interface{}) error {
	if req.Type != "async" && req.Type != "sync" {
		return errors.Errorf("cannot return value on %q stream", req.Type)
	}

	err := req.Stream.Pour(ctx, v)
	if err != nil {
		return errors.Wrap(err, "error pouring return value")
	}

	return nil
}

func (req *Request) CloseWithError(cerr error) error {
	var inErr error
	if cerr == nil || luigi.IsEOS(errors.Cause(cerr)) {
		inErr = req.in.Close()
	} else {
		inErr = req.in.(luigi.ErrorCloser).CloseWithError(cerr)
	}
	if inErr != nil {
		return errors.Wrap(inErr, "failed to close request input")
	}

	// we really need to make sure we close the streams
	// "you can't" doesn't work here
	s := req.Stream.(*stream)
	err := s.doCloseWithError(cerr)
	if errors.Cause(err) == os.ErrClosed || isAlreadyClosed(err) {
		return nil
	}
	return errors.Wrap(err, "muxrpc: failed to close request stream")
}

func (req *Request) Close() error {
	return req.CloseWithError(luigi.EOS{})
}

// CallType is the type of a call
type CallType string

// Flags returns the packet flags of the respective call type
func (t CallType) Flags() codec.Flag {
	switch t {
	case "source", "sink", "duplex":
		return codec.FlagStream
	default:
		return 0
	}
}
