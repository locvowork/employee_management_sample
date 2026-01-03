package handler_test

import (
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/labstack/echo/v4"
	"github.com/locvowork/employee_management_sample/apigateway/internal/handler"
	"github.com/stretchr/testify/assert"
)

func TestComparisonEndpoints(t *testing.T) {
	e := echo.New()
	compHandler := handler.NewComparisonHandler()

	t.Run("Idiomatic Wiki Export", func(t *testing.T) {
		req := httptest.NewRequest(http.MethodGet, "/comparison/wiki/idiomatic", nil)
		rec := httptest.NewRecorder()
		c := e.NewContext(req, rec)

		if assert.NoError(t, compHandler.ExportWikiIdiomatic(c)) {
			assert.Equal(t, http.StatusOK, rec.Code)
			assert.Equal(t, "application/vnd.openxmlformats-officedocument.spreadsheetml.sheet", rec.Header().Get(echo.HeaderContentType))
			assert.Contains(t, rec.Header().Get(echo.HeaderContentDisposition), "wiki_names_idiomatic.xlsx")
		}
	})

	t.Run("TPL Wiki Export", func(t *testing.T) {
		req := httptest.NewRequest(http.MethodGet, "/comparison/wiki/tpl", nil)
		rec := httptest.NewRecorder()
		c := e.NewContext(req, rec)

		if assert.NoError(t, compHandler.ExportWikiTPL(c)) {
			assert.Equal(t, http.StatusOK, rec.Code)
			assert.Equal(t, "application/vnd.openxmlformats-officedocument.spreadsheetml.sheet", rec.Header().Get(echo.HeaderContentType))
			assert.Contains(t, rec.Header().Get(echo.HeaderContentDisposition), "wiki_names_tpl.xlsx")
		}
	})
}
