package blobs

import (
	"context"
	"fmt"
	"io"

	"github.com/cryptix/go/logging"
	"github.com/pkg/errors"

	"go.cryptoscope.co/muxrpc"

	"go.cryptoscope.co/ssb"
)

type getHandler struct {
	bs  ssb.BlobStore
	log logging.Interface
}

func (getHandler) HandleConnect(context.Context, muxrpc.Endpoint) {}

func (h getHandler) HandleCall(ctx context.Context, req *muxrpc.Request, edp muxrpc.Endpoint) {
	h.log.Log("event", "onCall", "handler", "get", "args", fmt.Sprintf("%v", req.Args), "method", req.Method)
	defer h.log.Log("event", "onCall", "handler", "get-return", "method", req.Method)
	// TODO: push manifest check into muxrpc
	if req.Type == "" {
		req.Type = "source"
	}

	if len(req.Args) != 1 {
		return
	}

	var refStr string
	switch arg := req.Args[0].(type) {
	case string:
		refStr = arg
	case map[string]interface{}:
		refStr, _ = arg["key"].(string)
	}

	ref, err := ssb.ParseBlobRef(refStr)
	checkAndLog(h.log, errors.Wrap(err, "error parsing blob reference"))
	if err != nil {
		return
	}

	r, err := h.bs.Get(ref)
	if err != nil {
		err = req.Stream.CloseWithError(errors.New("do not have blob"))
		checkAndLog(h.log, errors.Wrap(err, "error closing stream with error"))
		return
	}

	w := muxrpc.NewSinkWriter(req.Stream)
	_, err = io.Copy(w, r)
	checkAndLog(h.log, errors.Wrap(err, "error sending blob"))

	err = w.Close()
	checkAndLog(h.log, errors.Wrap(err, "error closing blob output"))
}
