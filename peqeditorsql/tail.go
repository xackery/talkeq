package peqeditorsql

import (
	"bytes"
	"context"
	"fmt"
	"os"
	"text/template"
	"time"

	"github.com/hpcloud/tail"
	"github.com/xackery/talkeq/tlog"
)

// tail wraps the tail tool for each file being watched
type tailWatch struct {
	rootCtx context.Context
	ctx     context.Context
	cancel  context.CancelFunc
	req     *tailReq
	tailer  *tail.Tail
}

type tailReq struct {
	id          int
	filePattern string
	basePath    string
	cfg         tail.Config
	isNextMonth bool
}

func newTailWatch(rootCtx context.Context, req *tailReq, msgChan chan string) (*tailWatch, error) {

	e := &tailWatch{
		rootCtx: rootCtx,
		req:     req,
	}
	e.ctx, e.cancel = context.WithCancel(context.Background())

	err := e.restart(msgChan)
	if err != nil {
		return nil, fmt.Errorf("restart: %w", err)
	}

	return e, nil
}

func (e *tailWatch) restart(msgChan chan string) error {
	var err error
	e.cancel()
	time.Sleep(1 * time.Second)
	e.ctx, e.cancel = context.WithCancel(context.Background())
	buf := new(bytes.Buffer)
	tmpl := template.New("filePattern")
	tmpl.Parse(e.req.filePattern)

	month := time.Now().Format("01")
	if e.req.isNextMonth {
		month = time.Now().AddDate(0, 1, 0).Format("01")
	}

	tmpl.Execute(buf, struct {
		Year  int
		Month string
	}{
		time.Now().Year(),
		month,
	})
	finalPath := fmt.Sprintf("%s/%s", e.req.basePath, buf.String())

	fi, err := os.Stat(finalPath)
	if err == nil {
		e.req.cfg.Location = &tail.SeekInfo{Offset: fi.Size()}
	}

	e.tailer, err = tail.TailFile(finalPath, e.req.cfg)
	if err != nil {
		return fmt.Errorf("tail: %w", err)
	}
	go e.loop(msgChan)
	return nil
}

func (e *tailWatch) loop(msgChan chan string) {
	defer func() {
		tlog.Debugf("[peqeditorsql] tail%d loop exiting for %s", e.req.id, e.tailer.Filename)
		e.tailer.Cleanup()
	}()

	select {
	case <-e.rootCtx.Done():
		return
	case <-e.ctx.Done():
		return
	case line := <-e.tailer.Lines:
		if line.Err != nil {
			tlog.Warnf("[peqeditorsql] tail%d error: %s", e.req.id, line.Err)
			return
		}
		msgChan <- line.Text
	}
}
