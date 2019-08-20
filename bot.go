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
	"go.cryptoscope.co/ssb/message"
	mksbot "go.cryptoscope.co/ssb/sbot"
)

var (
	// helper
	log        logging.Interface
	checkFatal = logging.CheckFatal

	// juicy bits
	appKey  string = "1KHLiKZvAvjbY1ziZEHMXawbCEIM6qwjCDm3VYRan/s="
	hmacSec string

	theBot *mksbot.Sbot
)

func checkAndLog(err error) {
	if err != nil {
		if err := logging.LogPanicWithStack(log, "checkAndLog", err); err != nil {
			panic(err)
		}
	}
}

func Stop() error {
	theBot.Shutdown()
	return theBot.Close()
}

func Start(repoPath string) {
	logging.SetupLogging(os.Stderr)
	log = logging.Logger("sbot")

	ctx := context.Background()

	ak, err := base64.StdEncoding.DecodeString(appKey)
	checkFatal(err)

	listenAddr := ":8008"
	theBot, err = mksbot.New(
		mksbot.WithInfo(log),
		mksbot.WithAppKey(ak),
		mksbot.WithRepoPath(repoPath),
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

	id := theBot.KeyPair.Id
	uf := theBot.UserFeeds
	gb := theBot.GraphBuilder

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
		rv, err := theBot.RootLog.Get(rlSeq.(margaret.BaseSeq))
		if margaret.IsErrNulled(err) {
			continue
		} else {
			checkFatal(err)
		}
		msg := rv.(message.StoredMessage)

		if msg.Sequence.Seq() != userLogSeq.Seq()+1 {
			err = fmt.Errorf("light fsck failed: head of feed mismatch on %s: %d vs %d", authorRef.Ref(), msg.Sequence, userLogSeq.Seq()+1)
			log.Log("warning", err)
			continue
		}

		msgCount += uint(msg.Sequence.Seq())

		f, err := gb.Follows(authorRef)
		checkFatal(err)

		if len(feeds) < 20 {
			h := gb.Hops(authorRef, 2)
			log.Log("info", "currSeq", "feed", authorRef.Ref(), "seq", msg.Sequence.Seq(), "follows", f.Count(), "hops", h.Count())
		}
		followCnt += uint(f.Count())
	}

	log.Log("event", "repo open", "feeds", len(feeds), "msgs", msgCount, "follows", followCnt)

	log.Log("event", "serving", "ID", id.Ref(), "addr", listenAddr)
	go func() {
		for {
			// Note: This is where the serving starts ;)
			err = theBot.Network.Serve(ctx)
			log.Log("event", "sbot node.Serve returned, restarting..", "err", err)
			time.Sleep(1 * time.Second)
			select {
			case <-ctx.Done():
				os.Exit(0)
			default:
			}
		}
	}()
}
