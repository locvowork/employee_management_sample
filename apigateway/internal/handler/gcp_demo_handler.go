package handler

import (
	"fmt"
	"net/http"
	"strconv"

	"github.com/labstack/echo/v4"
	"github.com/locvowork/employee_management_sample/apigateway/internal/logger"
	"github.com/locvowork/employee_management_sample/apigateway/pkg/googlecloud"
)

type GCPDemoHandler struct {
	client *googlecloud.Client
}

func NewGCPDemoHandler(client *googlecloud.Client) *GCPDemoHandler {
	return &GCPDemoHandler{client: client}
}

// CreateTaskListHandler handles POST /api/v1/gcp/task-lists
func (h *GCPDemoHandler) CreateTaskListHandler(c echo.Context) error {
	ctx := c.Request().Context()
	var list googlecloud.TaskList
	if err := c.Bind(&list); err != nil {
		return echo.NewHTTPError(http.StatusBadRequest, "invalid request body")
	}

	if err := h.client.CreateTaskList(ctx, &list); err != nil {
		logger.ErrorLog(ctx, fmt.Sprintf("failed to create task list: %v", err))
		return echo.NewHTTPError(http.StatusInternalServerError, "failed to create task list")
	}

	return c.JSON(http.StatusCreated, list)
}

// CreateTaskHandler handles POST /api/v1/gcp/task-lists/:id/tasks
func (h *GCPDemoHandler) CreateTaskHandler(c echo.Context) error {
	ctx := c.Request().Context()
	taskListID := c.Param("id")
	var task googlecloud.Task
	if err := c.Bind(&task); err != nil {
		return echo.NewHTTPError(http.StatusBadRequest, "invalid request body")
	}

	if err := h.client.CreateTask(ctx, taskListID, &task); err != nil {
		logger.ErrorLog(ctx, fmt.Sprintf("failed to create task: %v", err))
		return echo.NewHTTPError(http.StatusInternalServerError, "failed to create task")
	}

	return c.JSON(http.StatusCreated, task)
}

// ListTasksHandler handles GET /api/v1/gcp/task-lists/:id/tasks
func (h *GCPDemoHandler) ListTasksHandler(c echo.Context) error {
	ctx := c.Request().Context()
	taskListID := c.Param("id")

	tasks, err := h.client.ListTasksByList(ctx, taskListID)
	if err != nil {
		logger.ErrorLog(ctx, fmt.Sprintf("failed to list tasks: %v", err))
		return echo.NewHTTPError(http.StatusInternalServerError, "failed to list tasks")
	}

	return c.JSON(http.StatusOK, tasks)
}

// ComplexQueryHandler handles GET /api/v1/gcp/tasks/complex
func (h *GCPDemoHandler) ComplexQueryHandler(c echo.Context) error {
	ctx := c.Request().Context()
	minPriorityStr := c.QueryParam("min_priority")
	doneStr := c.QueryParam("done")

	minPriority, _ := strconv.Atoi(minPriorityStr)
	done, _ := strconv.ParseBool(doneStr)

	tasks, err := h.client.ListAllTasksComplex(ctx, minPriority, done)
	if err != nil {
		logger.ErrorLog(ctx, fmt.Sprintf("failed complex query: %v", err))
		return echo.NewHTTPError(http.StatusInternalServerError, "failed complex query")
	}

	return c.JSON(http.StatusOK, tasks)
}
