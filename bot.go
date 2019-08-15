package gobotexample

import (
	"context"
	"encoding/base64"
	"fmt"
	"os"
	"time"

	"github.com/cryptix/go/logging"
	"go.cryptoscope.co/margaret"

	"go.cryptoscope.co/ssb"
	mksbot "go.cryptoscope.co/ssb/sbot"
)

var (
	// helper
	log        logging.Interface
	checkFatal = logging.CheckFatal

	// juicy bits
	appKey  string = "1KHLiKZvAvjbY1ziZEHMXawbCEIM6qwjCDm3VYRan/s="
	hmacSec string
)

func checkAndLog(err error) {
	if err != nil {
		if err := logging.LogPanicWithStack(log, "checkAndLog", err); err != nil {
			panic(err)
		}
	}
}

func Start() {
	logging.SetupLogging(os.Stderr)
	log = logging.Logger("sbot")

	ctx := context.Background()

	ak, err := base64.StdEncoding.DecodeString(appKey)
	checkFatal(err)

	listenAddr := "192.168.0.101:8008"
	sbot, err := mksbot.New(
		mksbot.WithInfo(log),
		mksbot.WithAppKey(ak),
		mksbot.WithRepoPath("/storage/my/ssb/folder"),
		mksbot.WithListenAddr(listenAddr),
		mksbot.EnableAdvertismentBroadcasts(true),
		mksbot.EnableAdvertismentDialing(true),
	)
	checkFatal(err)

	/* shutdown handling with ctrl+c, could be a function
	c := make(chan os.Signal)
	signal.Notify(c, os.Interrupt, syscall.SIGTERM)
	go func() {
		sig := <-c
		log.Log("event", "killed", "msg", "received signal, shutting down", "signal", sig.String())
		cancel()
		sbot.Shutdown()
		time.Sleep(2 * time.Second)

		err := sbot.Close()
		checkAndLog(err)

		time.Sleep(2 * time.Second)
		os.Exit(0)
	}()
	logging.SetCloseChan(c)
	*/

	id := sbot.KeyPair.Id
	uf, ok := sbot.GetMultiLog("userFeeds")
	if !ok {
		checkAndLog(fmt.Errorf("missing userFeeds index"))
		return
	}
	gb := sbot.GraphBuilder

	feeds, err := uf.List()
	checkFatal(err)

	var followCnt, msgCount uint
	for _, author := range feeds {
		authorRef, err := ssb.ParseFeedRef(string(author))
		checkFatal(err)

		subLog, err := uf.Get(author)
		checkFatal(err)

		userLogV, err := subLog.Seq().Value()
		checkFatal(err)
		userLogSeq := userLogV.(margaret.Seq)
		rlSeq, err := subLog.Get(userLogSeq)
		if margaret.IsErrNulled(err) {
			continue
		} else {
			checkFatal(err)
		}
		rv, err := sbot.RootLog.Get(rlSeq.(margaret.BaseSeq))
		if margaret.IsErrNulled(err) {
			continue
		} else {
			checkFatal(err)
		}
		msg := rv.(ssb.Message)

		if msg.Seq() != userLogSeq.Seq()+1 {
			err = fmt.Errorf("light fsck failed: head of feed mismatch on %s: %d vs %d", authorRef.Ref(), msg.Seq(), userLogSeq.Seq()+1)
			log.Log("warning", err)
			continue
		}

		msgCount += uint(msg.Seq())

		f, err := gb.Follows(authorRef)
		checkFatal(err)

		if len(feeds) < 20 {
			h := gb.Hops(authorRef, 2)
			log.Log("info", "currSeq", "feed", authorRef.Ref(), "seq", msg.Seq(), "follows", f.Count(), "hops", h.Count())
		}
		followCnt += uint(f.Count())
	}

	log.Log("event", "repo open", "feeds", len(feeds), "msgs", msgCount, "follows", followCnt)

	log.Log("event", "serving", "ID", id.Ref(), "addr", listenAddr)
	for {
		// Note: This is where the serving starts ;)
		err = sbot.Network.Serve(ctx)
		log.Log("event", "sbot node.Serve returned", "err", err)
		time.Sleep(1 * time.Second)
		select {
		case <-ctx.Done():
			os.Exit(0)
		default:
		}
	}
}
