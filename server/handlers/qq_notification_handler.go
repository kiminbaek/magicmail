// SPDX-License-Identifier: AGPL-3.0-or-later
// Copyright (C) 2026  magiccode (魔法代码) — original MagicMail project
// Copyright (C) 2026  kiminbaek — QQ notification feature

package handlers

import (
	"strconv"

	"magicmail/models"
	"magicmail/services"

	"github.com/gofiber/fiber/v2"
)

// QQNotificationHandler QQ 通知 API 处理器
type QQNotificationHandler struct {
	service *services.QQNotificationService
}

// NewQQNotificationHandler 创建 handler
func NewQQNotificationHandler(svc *services.QQNotificationService) *QQNotificationHandler {
	return &QQNotificationHandler{service: svc}
}

// GetConfig GET /api/v1/qq-notification/config
func (h *QQNotificationHandler) GetConfig(c *fiber.Ctx) error {
	uid := getUserIDFromContext(c)
	config, err := h.service.GetConfig(uid)
	if err != nil {
		return c.Status(500).JSON(fiber.Map{"error": "获取配置失败", "detail": err.Error()})
	}
	return c.JSON(config)
}

// SaveConfig POST /api/v1/qq-notification/config
func (h *QQNotificationHandler) SaveConfig(c *fiber.Ctx) error {
	uid := getUserIDFromContext(c)
	var req models.QQNotificationRequest
	if err := c.BodyParser(&req); err != nil {
		return c.Status(400).JSON(fiber.Map{"error": "请求格式错误", "detail": err.Error()})
	}
	config, err := h.service.SaveConfig(uid, &req)
	if err != nil {
		return c.Status(500).JSON(fiber.Map{"error": "保存配置失败", "detail": err.Error()})
	}
	return c.JSON(fiber.Map{"success": true, "config": config})
}

// Test POST /api/v1/qq-notification/test
func (h *QQNotificationHandler) Test(c *fiber.Ctx) error {
	uid := getUserIDFromContext(c)
	err := h.service.TestNotification(uid)
	if err != nil {
		return c.Status(500).JSON(fiber.Map{"success": false, "error": err.Error()})
	}
	return c.JSON(fiber.Map{"success": true, "message": "测试消息已发送，请检查 QQ"})
}

// GetLogs GET /api/v1/qq-notification/logs
func (h *QQNotificationHandler) GetLogs(c *fiber.Ctx) error {
	uid := getUserIDFromContext(c)
	limit := 50
	if l := c.Query("limit"); l != "" {
		if v, err := strconv.Atoi(l); err == nil {
			limit = v
		}
	}
	logs, err := h.service.GetLogs(uid, limit)
	if err != nil {
		return c.Status(500).JSON(fiber.Map{"error": "获取日志失败", "detail": err.Error()})
	}
	return c.JSON(logs)
}

// DeleteLog DELETE /api/v1/qq-notification/logs/:id
func (h *QQNotificationHandler) DeleteLog(c *fiber.Ctx) error {
	id, err := strconv.ParseUint(c.Params("id"), 10, 32)
	if err != nil {
		return c.Status(400).JSON(fiber.Map{"error": "无效的日志 ID"})
	}
	if err := h.service.DeleteLog(uint(id)); err != nil {
		return c.Status(500).JSON(fiber.Map{"error": "删除失败", "detail": err.Error()})
	}
	return c.JSON(fiber.Map{"success": true})
}

// getUserIDFromContext 从 Fiber context 获取 user_id（单用户模式兜底 1）
func getUserIDFromContext(c *fiber.Ctx) uint {
	if uid, ok := c.Locals("user_id").(uint); ok {
		return uid
	}
	// JWT 解析后可能是 float64
	if uid, ok := c.Locals("user_id").(float64); ok {
		return uint(uid)
	}
	return 1
}
