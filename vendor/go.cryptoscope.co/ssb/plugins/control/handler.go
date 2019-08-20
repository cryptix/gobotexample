package control

import (
	"context"
	"net"
	"os"

	"github.com/cryptix/go/logging"
	"github.com/pkg/errors"
	"go.cryptoscope.co/muxrpc"
	"go.cryptoscope.co/netwrap"
	"go.cryptoscope.co/secretstream"

	"go.cryptoscope.co/ssb"
	"go.cryptoscope.co/ssb/internal/multiserver"
)

type handler struct {
	node ssb.Network
	info logging.Interface
}

func New(i logging.Interface, n ssb.Network) muxrpc.Handler {
	return &handler{
		info: i,
		node: n,
	}
}

func (h *handler) check(err error) {
	if err != nil && errors.Cause(err) != os.ErrClosed {
		h.info.Log("error", err)
	}
}

func (h *handler) HandleConnect(ctx context.Context, e muxrpc.Endpoint) {}

func (h *handler) HandleCall(ctx context.Context, req *muxrpc.Request, edp muxrpc.Endpoint) {
	if req.Type == "" {
		req.Type = "async"
	}

	var closed bool
	checkAndClose := func(err error) {
		h.check(err)
		if err != nil {
			closed = true
			closeErr := req.Stream.CloseWithError(err)
			h.check(errors.Wrapf(closeErr, "error closeing request. %s", req.Method))
		}
	}

	defer func() {
		if !closed {
			h.check(errors.Wrapf(req.Stream.Close(), "gossip: error closing call: %s", req.Method))
		}
	}()

	switch req.Method.String() {

	case "ctrl.connect":
		if len(req.Args) != 1 {
			// TODO: use secretstream
			h.info.Log("error", "usage", "args", req.Args, "method", req.Method)
			checkAndClose(errors.New("usage: ctrl.connect host:port:key"))
			return
		}
		destString, ok := req.Args[0].(string)
		if !ok {
			err := errors.Errorf("ctrl.connect call: expected argument to be string, got %T", req.Args[0])
			checkAndClose(err)
			return
		}
		if err := h.connect(ctx, destString); err != nil {
			checkAndClose(errors.Wrap(err, "ctrl.connect failed."))
			return
		}
		closed = true
		h.check(req.Return(ctx, "connected"))

	default:
		checkAndClose(errors.Errorf("unknown command: %s", req.Method))
	}
}

func (h *handler) connect(ctx context.Context, dest string) error {
	msaddr, err := multiserver.ParseNetAddress([]byte(dest))
	if err != nil {
		return errors.Wrapf(err, "gossip.connect call: failed to parse input: %s", dest)
	}

	addr := &net.TCPAddr{
		IP:   msaddr.Host,
		Port: msaddr.Port,
	}

	wrappedAddr := netwrap.WrapAddr(addr, secretstream.Addr{PubKey: msaddr.Ref.ID})
	h.info.Log("event", "doing gossip.connect", "remote", wrappedAddr.String())
	// TODO: add context to tracker to cancel connections
	err = h.node.Connect(context.Background(), wrappedAddr)
	return errors.Wrapf(err, "gossip.connect call: error connecting to %q", addr)
}
