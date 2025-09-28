package model

import "github.com/dikyayodihamzah/cv-evaluator/pkg/model/cvweb"

type BaseResponse struct {
	ID      string            `json:"id"`
	Status  string            `json:"status"`
	Message string            `json:"message,omitempty"`
	Result  *cvweb.CVResponse `json:"result,omitempty"`
	Error   *string           `json:"error,omitempty"`
}
