package eqlog

import (
	"context"
	"fmt"
	"time"

	"github.com/xackery/talkeq/model"
)

func (e *Endpoint) runQuery(ctx context.Context, method string, req interface{}) (resp interface{}, err error) {
	respChan := make(chan *model.QueryResponse)
	select {
	case <-e.ctx.Done():
		err = fmt.Errorf("client context completed early during response")
	case <-ctx.Done():
		err = fmt.Errorf("context completed early during request")
	case <-time.After(queryTimeout):
		err = fmt.Errorf("timed out during request")
	case e.queryChan <- &model.QueryRequest{Ctx: ctx, Method: method, Req: req, RespChan: respChan}:
		select {
		case <-e.ctx.Done():
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

func (e *Endpoint) pump() {
	var query *model.QueryRequest
	var err error
	var resp interface{}
	for {
		select {
		case <-e.ctx.Done():
			return
		case query = <-e.queryChan:
			err = nil
			if query.Ctx == nil {
				query.Ctx = context.Background()
			}
			ctx := query.Ctx
			switch query.Method {
			case "ConfigRead":
				resp, err = e.onConfigRead(ctx)
			case "ConfigUpdate":
				req, ok := query.Req.(*model.ConfigEndpoint)
				if !ok {
					err = fmt.Errorf("invalid request type")
				} else {
					err = e.onConfigUpdate(ctx, req)
				}
			case "Connect":
				err = e.onConnect(ctx)
			case "Close":
				err = e.onClose(ctx)
			case "SendMessage":
				req, ok := query.Req.(*model.ChatMessage)
				if !ok {
					err = fmt.Errorf("invalind request type")
				} else {
					resp, err = e.onSendMessage(ctx, req)
				}
			default:
				err = fmt.Errorf("unhandled method on client pump: %s", query.Method)
				return
			}
			if query.RespChan == nil {
				continue
			}
			select {
			case <-e.ctx.Done():
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
