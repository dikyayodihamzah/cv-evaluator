package exception

import (
	"errors"
	"fmt"
	"strings"

	"github.com/dikyayodihamzah/cv-evaluator/pkg/env"
	"github.com/dikyayodihamzah/cv-evaluator/pkg/lib"
	"github.com/dikyayodihamzah/cv-evaluator/pkg/log"
	"github.com/go-playground/validator/v10"
	"github.com/gofiber/fiber/v2"
	"github.com/iancoleman/strcase"
)

var ShowErrorDescription bool = strings.EqualFold(env.GetString("ENVIRONMENT"), "LOCAL") || strings.EqualFold(env.GetString("ENVIRONMENT"), "DEVELOPMENT")

func Handler(c *fiber.Ctx, err error) error {
	log.Error("%s", err.Error())
	code := fiber.StatusInternalServerError

	var e *fiber.Error
	if errors.As(err, &e) {
		code = e.Code
	}

	if e, ok := err.(*lib.Response); ok {
		if env.GetString("ENVIRONMENT") == "DEVELOPMENT" || env.GetString("ENVIRONMENT") == "LOCAL" {
			e.Log = lib.AddLog()
		}

		return c.Status(e.Code).JSON(e)
	}

	if errs, ok := err.(validator.ValidationErrors); ok {
		if len(errs) > 0 {
			err := errs[0]
			responses := lib.Response{
				Code:    fiber.StatusBadRequest,
				Message: err.Error(),
				Status:  false,
				Data:    nil,
			}
			if m, ok := lib.ValidateMessages[err.Tag()]; ok {
				responses.Message = fmt.Sprintf(m, strcase.ToSnake(err.Field()), fmt.Sprint(err.Param()))
			}
			return c.Status(responses.Code).JSON(responses)
		}
		code = fiber.StatusBadRequest
	}
	return c.Status(code).JSON(err)
}

func ErrorBadRequest(message ...string) error {
	if len(message) == 0 {
		message = append(message, "Bad Request")
	}
	var errorDesc *string = nil
	if len(message) > 1 && ShowErrorDescription {
		errorDesc = lib.Pointer(message[1])
	}
	return &lib.Response{
		Code:             fiber.StatusBadRequest,
		Status:           false,
		Message:          message[0],
		ErrorDescription: errorDesc,
	}
}

func ErrorUnauthorized(message ...string) error {
	if len(message) == 0 {
		message = append(message, "Unauthorized")
	}
	var errorDesc *string = nil
	if len(message) > 1 && ShowErrorDescription {
		errorDesc = lib.Pointer(message[1])
	}
	return &lib.Response{
		Code:             fiber.StatusUnauthorized,
		Status:           false,
		Message:          message[0],
		ErrorDescription: errorDesc,
	}
}

func ErrorForbidden(message ...string) error {
	if len(message) == 0 {
		message = append(message, "Forbidden")
	}
	var errorDesc *string = nil
	if len(message) > 1 && ShowErrorDescription {
		errorDesc = lib.Pointer(message[1])
	}
	return &lib.Response{
		Code:             fiber.StatusForbidden,
		Status:           false,
		Message:          message[0],
		ErrorDescription: errorDesc,
	}
}

func ErrorNotFound(message ...string) error {
	if len(message) == 0 {
		message = append(message, "Not Found")
	}
	var errorDesc *string = nil
	if len(message) > 1 && ShowErrorDescription {
		errorDesc = lib.Pointer(message[1])
	}
	return &lib.Response{
		Code:             fiber.StatusNotFound,
		Status:           false,
		Message:          message[0],
		ErrorDescription: errorDesc,
	}
}

func ErrorConflict(message ...string) error {
	if len(message) == 0 {
		message = append(message, "Conflict")
	}
	var errorDesc *string = nil
	if len(message) > 1 && ShowErrorDescription {
		errorDesc = lib.Pointer(message[1])
	}
	return &lib.Response{
		Code:             fiber.StatusConflict,
		Status:           false,
		Message:          message[0],
		ErrorDescription: errorDesc,
	}
}

func ErrorInternal(message ...string) error {
	if len(message) == 0 {
		message = append(message, "Internal Server Error")
	}
	var errorDesc *string = nil
	if len(message) > 1 && ShowErrorDescription {
		errorDesc = lib.Pointer(message[1])
	}
	return &lib.Response{
		Code:             fiber.StatusInternalServerError,
		Status:           false,
		Message:          message[0],
		ErrorDescription: errorDesc,
	}
}

func ErrorSessionExpired(message ...string) error {
	if len(message) == 0 {
		message = append(message, "Session Expired")
	}
	var errorDesc *string = nil
	if len(message) > 1 && ShowErrorDescription {
		errorDesc = lib.Pointer(message[1])
	}
	return &lib.Response{
		Code:             fiber.StatusGone,
		Status:           false,
		Message:          message[0],
		ErrorDescription: errorDesc,
	}
}
