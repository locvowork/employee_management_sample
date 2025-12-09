package serviceutils

import (
	"github.com/labstack/echo/v4"
)

type GenericResponse struct {
	Success bool
	Message string
	Data    interface{}
	Error   string
}

func SuccessJSON(data interface{}, msg string) GenericResponse {
	return GenericResponse{
		Success: true,
		Message: msg,
		Data:    data,
	}
}

func ErrorJSON(err error, statusCode int) (GenericResponse, int) {
	return GenericResponse{
		Success: false,
		Error:   err.Error(),
	}, statusCode
}

func ResponseSuccess(c echo.Context, code int, msg string, data interface{}) error {
	return c.JSON(code, GenericResponse{
		Success: true,
		Message: msg,
		Data:    data,
	})
}

func ResponseError(c echo.Context, code int, msg string, err error) error {
	resp := GenericResponse{
		Success: false,
		Message: msg,
	}
	if err != nil {
		resp.Error = err.Error()
	}
	return c.JSON(code, resp)
}
