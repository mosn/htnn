package api

import "net/http"

type ResultAction interface {
	OK()
}
type isResultAction struct {
}

var (
	Continue ResultAction = nil
)

func (i *isResultAction) OK() {}

type LocalResponse struct {
	isResultAction

	Code   int
	Msg    string
	Header http.Header
}
