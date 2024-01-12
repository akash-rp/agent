package main

import (
	"github.com/labstack/echo/v4"
	"net/http"
)

func AbortWithErrorMessage(c echo.Context, message string) error {
	errorMessage := new(Error)
	errorMessage.Message = message
	return c.JSON(http.StatusBadRequest, errorMessage)
}
