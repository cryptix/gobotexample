package ssb

import (
	"context"
	"io"

	"go.cryptoscope.co/luigi"
	"go.cryptoscope.co/muxrpc"
)

const (
	// BlobStoreOpPut is used in put notifications
	BlobStoreOpPut BlobStoreOp = "put"

	// BlobStoreOpRm is used in remove notifications
	BlobStoreOpRm BlobStoreOp = "rm"
)

// BlobStore is the interface of our blob store
//go:generate counterfeiter -o mock/blobstore.go . BlobStore
type BlobStore interface {
	// Get returns a reader of the blob with given ref.
	Get(ref *BlobRef) (io.Reader, error)

	// Put stores the data in the reader in the blob store and returns the address.
	Put(blob io.Reader) (*BlobRef, error)

	// Delete deletes a blob from the blob store.
	Delete(ref *BlobRef) error

	// List returns a source of the refs of all stored blobs.
	List() luigi.Source

	// Size returns the size of the blob with given ref.
	Size(ref *BlobRef) (int64, error)

	// Changes returns a broadcast that emits put and remove notifications.
	Changes() luigi.Broadcast
}

//go:generate counterfeiter -o mock/wantmanager.go . WantManager
type WantManager interface {
	luigi.Broadcast
	Want(ref *BlobRef) error
	Wants(ref *BlobRef) bool
	WantWithDist(ref *BlobRef, dist int64) error
	//Unwant(ref *BlobRef) error
	CreateWants(context.Context, luigi.Sink, muxrpc.Endpoint) luigi.Sink
}

// BlobStoreNotification contains info on a single change of the blob store.
// Op is either "rm" or "put".
type BlobStoreNotification struct {
	Op  BlobStoreOp
	Ref *BlobRef
}

// BlobStoreOp specifies the operation in a blob store notification.
type BlobStoreOp string

// String returns the string representation of the operation.
func (op BlobStoreOp) String() string {
	return string(op)
}
