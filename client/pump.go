package client

import (
	"context"
	"fmt"
	"time"

	"github.com/xackery/talkeq/model"
)

func (c *Client) runQuery(ctx context.Context, method string, req interface{}) (resp interface{}, err error) {
	respChan := make(chan *model.QueryResponse)
	select {
	case <-c.ctx.Done():
		err = fmt.Errorf("client context completed early during response")
	case <-ctx.Done():
		err = fmt.Errorf("context completed early during request")
	case <-time.After(queryTimeout):
		err = fmt.Errorf("timed out during request")
	case c.queryChan <- &model.QueryRequest{Ctx: ctx, Method: method, Req: req, RespChan: respChan}:
		select {
		case <-c.ctx.Done():
			err = fmt.Errorf("client context completed early during response")
		case <-ctx.Done():
			err = fmt.Errorf("context completed early during response")
		case <-time.After(queryTimeout):
			err = fmt.Errorf("timed out during response")
		case respMsg := <-respChan:
			resp = respMsg.Resp
			err = respMsg.Error
		}
	}
	return
}

func (c *Client) pump() {
	var query *model.QueryRequest
	var err error
	var resp interface{}
	for {
		select {
		case <-c.ctx.Done():
			return
		case query = <-c.queryChan:
			err = nil
			if query.Ctx == nil {
				query.Ctx = context.Background()
			}
			ctx := query.Ctx
			switch query.Method {
			case "Start":
				err = c.onStart(ctx)
			default:
				err = fmt.Errorf("unhandled method on client pump: %s", query.Method)
			}
			if query.RespChan == nil {
				continue
			}
			select {
			case <-c.ctx.Done():
				return
			case <-ctx.Done():
				continue
			case <-time.After(queryTimeout):
				logger := model.NewLogger(ctx)
				logger.Info().Err(err).Interface("req", query.Req).Interface("resp", resp).Msg("timed out replying to request")
			case query.RespChan <- &model.QueryResponse{Error: err, Resp: resp}:
				continue
			}
		}
	}
}
