package controllers

import (
	"fmt"
	"net/http"
	"reflect"
	"strconv"
	"strings"

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

// parseIncludes parses the include query parameter and returns a slice of relationships to preload
func parseIncludes(ctx echo.Context) []string {
	include := ctx.QueryParam("include")
	if include == "" {
		return nil
	}
	return strings.Split(include, ",")
}

// parseExcludes parses the exclude query parameter and returns a slice of fields to exclude
func parseExcludes(ctx echo.Context) []string {
	exclude := ctx.QueryParam("exclude")
	if exclude == "" {
		return nil
	}
	return strings.Split(exclude, ",")
}

// Create handles creation of new entities
func (c *BaseController[T]) Create(ctx echo.Context) error {
	var entity T
	if err := ctx.Bind(&entity); err != nil {
		return echo.NewHTTPError(http.StatusBadRequest, "invalid request body "+err.Error())
	}

	if err := ctx.Validate(&entity); err != nil {
		return err
	}

	includes := parseIncludes(ctx)
	if err := c.service.Create(ctx.Request().Context(), &entity, includes...); err != nil {
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
	includes := parseIncludes(ctx)
	entity, err := c.service.Get(ctx.Request().Context(), id, includes...)
	if err != nil {
		return echo.NewHTTPError(http.StatusNotFound, "entity not found")
	}

	return ctx.JSON(http.StatusOK, entity)
}

func (c *BaseController[T]) applyFilters(ctx echo.Context, filters map[string]interface{}) map[string]interface{} {
	// add a teamID filter
	teamID := ctx.Get("teamID")
	if teamID != nil {
		var entity T
		entityType := reflect.TypeOf(entity)
		if _, found := entityType.FieldByName("TeamID"); found {
			filters["team_id"] = teamID
		}
	}
	if userID := ctx.Get("userID"); userID != nil {
		// Check if entity supports user_id field using reflection
		var entity T
		entityType := reflect.TypeOf(entity)
		if _, found := entityType.FieldByName("UserID"); found {
			filters["user_id"] = userID
		}
	}

	return filters
}

// List handles retrieval of multiple entities with pagination and filtering
func (c *BaseController[T]) List(ctx echo.Context) error {
	// Parse pagination parameters
	page, _ := strconv.Atoi(ctx.QueryParam("page"))
	limit, _ := strconv.Atoi(ctx.QueryParam("limit"))
	exclude := parseExcludes(ctx)
	if page < 1 {
		page = 1
	}
	if limit < 1 {
		limit = 10
	}

	// Parse filters from query parameters
	filters := make(map[string]interface{})
	for key, values := range ctx.QueryParams() {
		if key != "page" && key != "limit" && key != "include" && key != "exclude" && len(values) > 0 {
			filters[key] = values[0]
		}
	}

	filters = c.applyFilters(ctx, filters)

	includes := parseIncludes(ctx)

	excludeFields := make(map[string]bool)

	for _, field := range exclude {
		excludeFields[field] = true
	}

	entities, total, err := c.service.List(ctx.Request().Context(), page, limit, filters, excludeFields, includes...)

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

	// 🔒 Check if request body contains forbidden fields
	requestMap := make(map[string]interface{})
	if err := ctx.Bind(&requestMap); err != nil {
		return echo.NewHTTPError(http.StatusBadRequest, err.Error())
	}

	// 🚫 Check for forbidden fields in request body
	forbiddenFields := []string{"id", "ID", "team_id", "teamId", "TeamID"}
	for _, field := range forbiddenFields {
		if _, exists := requestMap[field]; exists {
			return echo.NewHTTPError(http.StatusBadRequest,
				fmt.Sprintf("Field '%s' cannot be modified", field))
		}
	}

	var entity T
	if err := ctx.Bind(&entity); err != nil {
		return echo.NewHTTPError(http.StatusBadRequest, err.Error())
	}

	if err := ctx.Validate(&entity); err != nil {
		return echo.NewHTTPError(http.StatusBadRequest, err.Error())
	}

	includes := parseIncludes(ctx)
	if err := c.service.Update(ctx.Request().Context(), id, &entity, includes...); err != nil {
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
			// validate the request body
			g.POST(path, c.Create)
		case "GET":
			g.GET(path+"/:id", c.Get)
			g.GET(path, c.List)
		case "PUT":
			// validate the request body
			g.PUT(path+"/:id", c.Update)
		case "DELETE":
			g.DELETE(path+"/:id", c.Delete)
		}
	}
}
