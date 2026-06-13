// SPDX-License-Identifier: AGPL-3.0-or-later
// Copyright (C) 2026  magiccode (魔法代码)

package handlers

import (
	"strconv"

	"magicmail/models"
	"magicmail/services"

	"github.com/gofiber/fiber/v2"
)

// WebhookHandler Webhook API 处理器
type WebhookHandler struct {
	service *services.WebhookService
}

// NewWebhookHandler 创建 Webhook Handler
func NewWebhookHandler(svc *services.WebhookService) *WebhookHandler {
	return &WebhookHandler{service: svc}
}

// List 获取所有 Webhook
func (h *WebhookHandler) List(c *fiber.Ctx) error {
	hooks, err := h.service.List()
	if err != nil {
		return c.Status(500).JSON(fiber.Map{"error": "获取 Webhook 列表失败"})
	}
	return c.JSON(fiber.Map{"data": hooks})
}

// Get 获取单个 Webhook 详情
func (h *WebhookHandler) Get(c *fiber.Ctx) error {
	id, err := c.ParamsInt("id")
	if err != nil {
		return c.Status(400).JSON(fiber.Map{"error": "无效的 ID"})
	}

	hook, err := h.service.GetByID(uint(id))
	if err != nil {
		return c.Status(404).JSON(fiber.Map{"error": "Webhook 不存在"})
	}

	return c.JSON(hook)
}

// Create 创建 Webhook
func (h *WebhookHandler) Create(c *fiber.Ctx) error {
	var req models.WebhookRequest
	if err := c.BodyParser(&req); err != nil {
		return c.Status(400).JSON(fiber.Map{"error": "请求参数解析失败"})
	}

	if req.Name == "" || req.URL == "" {
		return c.Status(400).JSON(fiber.Map{"error": "名称和 URL 不能为空"})
	}

	hook, err := h.service.Create(req)
	if err != nil {
		return c.Status(500).JSON(fiber.Map{"error": "创建失败", "detail": err.Error()})
	}

	return c.Status(201).JSON(hook)
}

// Update 更新 Webhook
func (h *WebhookHandler) Update(c *fiber.Ctx) error {
	id, err := c.ParamsInt("id")
	if err != nil {
		return c.Status(400).JSON(fiber.Map{"error": "无效的 ID"})
	}

	var req models.WebhookRequest
	if err := c.BodyParser(&req); err != nil {
		return c.Status(400).JSON(fiber.Map{"error": "请求参数解析失败"})
	}

	if req.Name == "" || req.URL == "" {
		return c.Status(400).JSON(fiber.Map{"error": "名称和 URL 不能为空"})
	}

	hook, err := h.service.Update(uint(id), req)
	if err != nil {
		return c.Status(500).JSON(fiber.Map{"error": "更新失败", "detail": err.Error()})
	}

	return c.JSON(hook)
}

// Delete 删除 Webhook
func (h *WebhookHandler) Delete(c *fiber.Ctx) error {
	id, err := c.ParamsInt("id")
	if err != nil {
		return c.Status(400).JSON(fiber.Map{"error": "无效的 ID"})
	}

	if err := h.service.Delete(uint(id)); err != nil {
		return c.Status(500).JSON(fiber.Map{"error": "删除失败", "detail": err.Error()})
	}

	return c.SendStatus(204)
}

// Test 测试 Webhook
func (h *WebhookHandler) Test(c *fiber.Ctx) error {
	id, err := c.ParamsInt("id")
	if err != nil {
		return c.Status(400).JSON(fiber.Map{"error": "无效的 ID"})
	}

	result, err := h.service.Test(uint(id))
	if err != nil {
		return c.Status(404).JSON(fiber.Map{"error": "Webhook 不存在"})
	}

	return c.JSON(result)
}

// GetLogs 获取 Webhook 日志
func (h *WebhookHandler) GetLogs(c *fiber.Ctx) error {
	id, err := c.ParamsInt("id")
	if err != nil {
		return c.Status(400).JSON(fiber.Map{"error": "无效的 ID"})
	}

	limit, _ := strconv.Atoi(c.Query("limit", "20"))
	logs, err := h.service.GetLogs(uint(id), limit)
	if err != nil {
		return c.Status(500).JSON(fiber.Map{"error": "获取日志失败"})
	}

	return c.JSON(fiber.Map{"data": logs})
}
