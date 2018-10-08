package model

import "context"

// QueryRequest is used in pumps to query a method
type QueryRequest struct {
	Ctx      context.Context
	Method   string
	Req      interface{}
	RespChan chan *QueryResponse
}

// QueryResponse is used in pumps to query a method
type QueryResponse struct {
	Resp  interface{}
	Error error
}
