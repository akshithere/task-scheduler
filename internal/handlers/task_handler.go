package handlers

import (
	"strconv"

	"github.com/akshithere/task-scheduler/internal/models"
	"github.com/akshithere/task-scheduler/internal/services"
	"github.com/gofiber/fiber/v2"
	"github.com/google/uuid"
)

type TaskHandler struct {
	service *services.TaskService
}

func NewTaskHandler(service *services.TaskService) *TaskHandler {
	return &TaskHandler{
		service: service,
	}
}

func (h *TaskHandler) CreateTask(c *fiber.Ctx) error {
	var task models.Task
	if err := c.BodyParser(&task); err != nil {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{
			"error": "Invalid request body",
		})
	}

	if err := h.service.CreateTask(&task); err != nil {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{
			"error": err.Error(),
		})
	}

	return c.Status(fiber.StatusCreated).JSON(task)
}

func (h *TaskHandler) GetTask(c *fiber.Ctx) error {
	id, err := uuid.Parse(c.Params("id"))
	if err != nil {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{
			"error": "Invalid task ID",
		})
	}

	task, err := h.service.GetTask(id)
	if err != nil {
		return c.Status(fiber.StatusNotFound).JSON(fiber.Map{
			"error": err.Error(),
		})
	}

	return c.JSON(task)
}

func (h *TaskHandler) ListTasks(c *fiber.Ctx) error {
	status := c.Query("status")
	var statusPtr *string
	if status != "" {
		statusPtr = &status
	}

	limit, _ := strconv.Atoi(c.Query("limit", "20"))
	page, _ := strconv.Atoi(c.Query("page", "1"))

	if limit <= 0 || limit > 100 {
		limit = 20
	}
	if page <= 0 {
		page = 1
	}

	offset := (page - 1) * limit

	tasks, total, err := h.service.ListTasks(statusPtr, limit, offset)
	if err != nil {
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{
			"error": err.Error(),
		})
	}

	return c.JSON(fiber.Map{
		"data":       tasks,
		"total":      total,
		"page":       page,
		"limit":      limit,
		"total_pages": (total + int64(limit) - 1) / int64(limit),
	})
}

func (h *TaskHandler) UpdateTask(c *fiber.Ctx) error {
	id, err := uuid.Parse(c.Params("id"))
	if err != nil {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{
			"error": "Invalid task ID",
		})
	}

	var updates models.Task
	if err := c.BodyParser(&updates); err != nil {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{
			"error": "Invalid request body",
		})
	}

	task, err := h.service.UpdateTask(id, &updates)
	if err != nil {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{
			"error": err.Error(),
		})
	}

	return c.JSON(task)
}

func (h *TaskHandler) DeleteTask(c *fiber.Ctx) error {
	id, err := uuid.Parse(c.Params("id"))
	if err != nil {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{
			"error": "Invalid task ID",
		})
	}

	if err := h.service.CancelTask(id); err != nil {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{
			"error": err.Error(),
		})
	}

	return c.JSON(fiber.Map{
		"message": "Task cancelled successfully",
	})
}

func (h *TaskHandler) GetTaskResults(c *fiber.Ctx) error {
	id, err := uuid.Parse(c.Params("id"))
	if err != nil {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{
			"error": "Invalid task ID",
		})
	}

	limit, _ := strconv.Atoi(c.Query("limit", "20"))
	page, _ := strconv.Atoi(c.Query("page", "1"))

	if limit <= 0 || limit > 100 {
		limit = 20
	}
	if page <= 0 {
		page = 1
	}

	offset := (page - 1) * limit

	results, total, err := h.service.GetTaskResults(id, limit, offset)
	if err != nil {
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{
			"error": err.Error(),
		})
	}

	return c.JSON(fiber.Map{
		"data":       results,
		"total":      total,
		"page":       page,
		"limit":      limit,
		"total_pages": (total + int64(limit) - 1) / int64(limit),
	})
}

func (h *TaskHandler) ListAllResults(c *fiber.Ctx) error {
	var taskIDPtr *uuid.UUID
	if taskIDStr := c.Query("task_id"); taskIDStr != "" {
		taskID, err := uuid.Parse(taskIDStr)
		if err == nil {
			taskIDPtr = &taskID
		}
	}

	var successPtr *bool
	if successStr := c.Query("success"); successStr != "" {
		success := successStr == "true"
		successPtr = &success
	}

	limit, _ := strconv.Atoi(c.Query("limit", "20"))
	page, _ := strconv.Atoi(c.Query("page", "1"))

	if limit <= 0 || limit > 100 {
		limit = 20
	}
	if page <= 0 {
		page = 1
	}

	offset := (page - 1) * limit

	results, total, err := h.service.ListAllResults(taskIDPtr, successPtr, limit, offset)
	if err != nil {
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{
			"error": err.Error(),
		})
	}

	return c.JSON(fiber.Map{
		"data":       results,
		"total":      total,
		"page":       page,
		"limit":      limit,
		"total_pages": (total + int64(limit) - 1) / int64(limit),
	})
}
