package main

import (
	"net/http"

	"github.com/labstack/echo/v4"
)

func cert(c echo.Context) error {
	wp := new(wpcert)
	c.Bind(&wp)

	err := addCert(*wp)
	if err != nil {
		return echo.NewHTTPError(http.StatusBadRequest, err)
	}
	return c.String(http.StatusOK, "Success")
}
