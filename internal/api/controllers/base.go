package controllers

import (
	"net/http"
	"strconv"

	"kori/internal/services"

	"github.com/labstack/echo/v4"
)

// BaseController provides generic CRUD operations for any model
type BaseController[T any] struct {
	service services.BaseService[T]
}

// NewBaseController creates a new base controller
func NewBaseController[T any](service services.BaseService[T]) *BaseController[T] {
	return &BaseController[T]{
		service: service,
	}
}

// Create handles creation of new entities
func (c *BaseController[T]) Create(ctx echo.Context) error {
	var entity T
	if err := ctx.Bind(&entity); err != nil {
		return echo.NewHTTPError(http.StatusBadRequest, "invalid request body")
	}

	if err := ctx.Validate(&entity); err != nil {
		return err
	}

	if err := c.service.Create(ctx.Request().Context(), &entity); err != nil {
		return echo.NewHTTPError(http.StatusInternalServerError, err.Error())
	}

	return ctx.JSON(http.StatusCreated, entity)
}

// Get handles retrieval of a single entity
func (c *BaseController[T]) Get(ctx echo.Context) error {
	id := ctx.Param("id")
	if id == "" {
		return echo.NewHTTPError(http.StatusBadRequest, "missing id parameter")
	}

	entity, err := c.service.Get(ctx.Request().Context(), id)
	if err != nil {
		return echo.NewHTTPError(http.StatusNotFound, "entity not found")
	}

	return ctx.JSON(http.StatusOK, entity)
}

// List handles retrieval of multiple entities with pagination and filtering
func (c *BaseController[T]) List(ctx echo.Context) error {
	// Parse pagination parameters
	page, _ := strconv.Atoi(ctx.QueryParam("page"))
	limit, _ := strconv.Atoi(ctx.QueryParam("limit"))
	if page < 1 {
		page = 1
	}
	if limit < 1 {
		limit = 10
	}

	// Parse filters from query parameters
	filters := make(map[string]interface{})
	for key, values := range ctx.QueryParams() {
		if key != "page" && key != "limit" && len(values) > 0 {
			filters[key] = values[0]
		}
	}

	entities, total, err := c.service.List(ctx.Request().Context(), page, limit, filters)
	if err != nil {
		return echo.NewHTTPError(http.StatusInternalServerError, err.Error())
	}

	return ctx.JSON(http.StatusOK, map[string]interface{}{
		"data":  entities,
		"total": total,
		"page":  page,
		"limit": limit,
	})
}

// Update handles updating an existing entity
func (c *BaseController[T]) Update(ctx echo.Context) error {
	id := ctx.Param("id")
	if id == "" {
		return echo.NewHTTPError(http.StatusBadRequest, "missing id parameter")
	}

	var entity T
	if err := ctx.Bind(&entity); err != nil {
		return echo.NewHTTPError(http.StatusBadRequest, "invalid request body")
	}

	if err := ctx.Validate(&entity); err != nil {
		return err
	}

	if err := c.service.Update(ctx.Request().Context(), id, &entity); err != nil {
		return echo.NewHTTPError(http.StatusInternalServerError, err.Error())
	}

	return ctx.JSON(http.StatusOK, entity)
}

// Delete handles deletion of an entity
func (c *BaseController[T]) Delete(ctx echo.Context) error {
	id := ctx.Param("id")
	if id == "" {
		return echo.NewHTTPError(http.StatusBadRequest, "missing id parameter")
	}

	if err := c.service.Delete(ctx.Request().Context(), id); err != nil {
		return echo.NewHTTPError(http.StatusInternalServerError, err.Error())
	}

	return ctx.NoContent(http.StatusNoContent)
}

// RegisterRoutes registers CRUD routes for the controller
func (c *BaseController[T]) RegisterRoutes(g *echo.Group, path string, methods ...string) {
	if len(methods) == 0 {
		methods = []string{"POST", "GET", "PUT", "DELETE"}
	}

	for _, method := range methods {
		switch method {
		case "POST":
			g.POST(path, c.Create)
		case "GET":
			g.GET(path+"/:id", c.Get)
			g.GET(path, c.List)
		case "PUT":
			g.PUT(path+"/:id", c.Update)
		case "DELETE":
			g.DELETE(path+"/:id", c.Delete)
		}
	}
}
