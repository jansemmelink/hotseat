package api

import (
	"context"
	"net/http"
)

type Handler func(httpRes http.ResponseWriter, httpReq *http.Request)

type ContextHandler func(ctx context.Context, httpRes http.ResponseWriter, httpReq *http.Request) (httpStatus int, res interface{})
