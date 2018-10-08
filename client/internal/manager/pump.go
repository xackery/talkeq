package manager

import (
	"context"
	"fmt"
	"time"

	"github.com/xackery/talkeq/model"
)

func (m *Manager) runQuery(ctx context.Context, method string, req interface{}) (resp interface{}, err error) {
	respChan := make(chan *model.QueryResponse)
	select {
	case <-m.ctx.Done():
		err = fmt.Errorf("manager context completed early during response")
	case <-ctx.Done():
		err = fmt.Errorf("context completed early during request")
	case <-time.After(queryTimeout):
		err = fmt.Errorf("timed out during request")
	case m.queryChan <- &model.QueryRequest{Ctx: ctx, Method: method, Req: req, RespChan: respChan}:
		select {
		case <-m.ctx.Done():
			err = fmt.Errorf("manager context completed early during response")
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

func (m *Manager) runClientQuery(ctx context.Context, method string, req interface{}) (resp interface{}, err error) {
	respChan := make(chan *model.QueryResponse)
	select {
	case <-m.ctx.Done():
		err = fmt.Errorf("manager context completed early during response")
	case <-ctx.Done():
		err = fmt.Errorf("context completed early during request")
	case <-time.After(queryTimeout):
		err = fmt.Errorf("timed out during request")
	case m.clientQueryChan <- &model.QueryRequest{Ctx: ctx, Method: method, Req: req, RespChan: respChan}:
		select {
		case <-m.ctx.Done():
			err = fmt.Errorf("manager context completed early during response")
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

func (m *Manager) pump() {
	var query *model.QueryRequest
	var err error
	var resp interface{}
	for {
		select {
		case <-m.ctx.Done():
			return
		case query = <-m.queryChan:
			err = nil
			if query.Ctx == nil {
				query.Ctx = context.Background()
			}
			ctx := query.Ctx
			switch query.Method {
			case "ChatMessage":
				req, ok := query.Req.(*model.ChatMessage)
				if !ok {
					err = fmt.Errorf("invalid request type")
				} else {
					err = m.onChatMessage(ctx, req)
				}
			default:
				err = fmt.Errorf("unhandled method on manager pump: %s", query.Method)
				return
			}
			if query.RespChan == nil {
				continue
			}
			select {
			case <-m.ctx.Done():
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
