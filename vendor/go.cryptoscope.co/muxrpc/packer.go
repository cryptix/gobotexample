package muxrpc // import "go.cryptoscope.co/muxrpc"

import (
	"context"
	stderr "errors"
	"io"
	"os"
	"sync"

	"go.cryptoscope.co/luigi"
	"go.cryptoscope.co/muxrpc/codec"

	"github.com/pkg/errors"
)

// Packer is a duplex stream that sends and receives *codec.Packet values.
// Usually wraps a network connection or stdio.
type Packer interface {
	luigi.Source
	luigi.Sink
}

// NewPacker takes an io.ReadWriteCloser and returns a Packer.
func NewPacker(rwc io.ReadWriteCloser) Packer {
	return &packer{
		r: codec.NewReader(rwc),
		w: codec.NewWriter(rwc),
		c: rwc,

		closing:  make(chan struct{}),
		closeLis: make([]chan struct{}, 0),
	}
}

// packer wraps an io.ReadWriteCloser and implements Packer.
type packer struct {
	rl sync.Mutex
	wl sync.Mutex

	r *codec.Reader
	w *codec.Writer
	c io.Closer

	cl        sync.Mutex
	closeErr  error
	closeOnce sync.Once
	closing   chan struct{}
	closeLis  []chan struct{}
}

type CloseNotifier interface {
	// Closed returns a channel that is closed once the packer has to stop operating
	// this allows other parts of the stack to see when the the packer stopped working
	Closed() <-chan struct{}
}

func (pkr *packer) Closed() <-chan struct{} {
	pkr.cl.Lock()
	defer pkr.cl.Unlock()
	ch := make(chan struct{})
	pkr.closeLis = append(pkr.closeLis, ch)
	return ch
}

// Next returns the next packet from the underlying stream.
func (pkr *packer) Next(_ context.Context) (interface{}, error) {
	pkr.rl.Lock()
	defer pkr.rl.Unlock()

	pkt, err := pkr.r.ReadPacket()
	select {
	case <-pkr.closing:
		if err != nil {
			return nil, luigi.EOS{}
		}
	default:
	}

	if err != nil {
		if cerr := pkr.Close(); cerr != nil {
			return nil, errors.Wrapf(cerr, "error closing connection on read error %v", err)
		}

		if errors.Cause(err) == io.EOF {
			return nil, luigi.EOS{}
		}

		return nil, errors.Wrap(err, "error reading packet")
	}

	pkt.Req = -pkt.Req

	return pkt, nil
}

// Pour sends a packet to the underlying stream.
func (pkr *packer) Pour(_ context.Context, v interface{}) error {
	select {
	case <-pkr.closing:
		return errSinkClosed
	default:
	}

	pkt, ok := v.(*codec.Packet)
	if !ok {
		return errors.Errorf("packer sink expected type *codec.Packet, got %T", v)
	}

	pkr.wl.Lock()
	defer pkr.wl.Unlock()
	err := pkr.w.WritePacket(pkt)
	if err != nil {

		if cerr := pkr.Close(); cerr != nil {
			return errors.Wrapf(cerr, "error closing connection on write err:%v", err)
		}

	}

	return errors.Wrap(err, "muxrpc: error writing packet")
}

var errSinkClosed = stderr.New("muxrpc: pour to closed sink")

// IsSinkClosed should be moved to luigi to gether with the error
func IsSinkClosed(err error) bool {
	if err == nil {
		return false
	}
	causeErr := errors.Cause(err)
	if causeErr == errSinkClosed {
		return true
	}
	return false
}

func isAlreadyClosed(err error) bool {
	if err == nil {
		return false
	}

	causeErr := errors.Cause(err)
	if causeErr == os.ErrClosed {
		return true
	}

	if sysErr, ok := (causeErr).(*os.PathError); ok {
		if sysErr.Err == os.ErrClosed {
			// fmt.Printf("debug: found syscall err: %T) %s\n", causeErr, causeErr)
			return true
		}
	}
	return false
}

// Close closes the packer.
func (pkr *packer) Close() error {
	pkr.cl.Lock()
	defer pkr.cl.Unlock()
	select {
	case <-pkr.closing:
		return errors.Wrap(pkr.closeErr, "packer: already closed")
	default:
	}

	var err error

	pkr.closeOnce.Do(func() {
		for _, ch := range pkr.closeLis {
			close(ch)
		}
		pkr.closeLis = nil
		err = pkr.c.Close()
		close(pkr.closing)
	})
	err = errors.Wrap(err, "error closing underlying closer")
	pkr.closeErr = err
	return err
}
