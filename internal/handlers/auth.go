package handlers

import (
	"fmt"
	"net/http"
	"strings"

	"github.com/Larguma/stuff/internal/auth"
	"github.com/Larguma/stuff/internal/models"
	"github.com/gin-gonic/gin"
	"gorm.io/gorm"
)

func (h *Handlers) getSetup(c *gin.Context) {
	if h.anyUsers() {
		c.Redirect(http.StatusFound, "/login")
		return
	}

	data := h.baseData(c)
	data["Title"] = "Setup"
	c.HTML(http.StatusOK, "setup", data)
}

func (h *Handlers) postSetup(c *gin.Context) {
	if h.anyUsers() {
		c.Redirect(http.StatusFound, "/login")
		return
	}

	username := strings.TrimSpace(c.PostForm("username"))
	password := c.PostForm("password")
	houseName := strings.TrimSpace(c.PostForm("house"))

	if username == "" || password == "" || houseName == "" {
		h.setFlashKey(c, "flash.fill_all_fields")
		c.Redirect(http.StatusFound, "/setup")
		return
	}

	if len(password) < 6 {
		h.setFlashKey(c, "flash.password_short")
		c.Redirect(http.StatusFound, "/setup")
		return
	}

	var existing int64
	h.db.Model(&models.User{}).Where("username = ?", username).Count(&existing)
	if existing > 0 {
		h.setFlashKey(c, "flash.username_taken")
		c.Redirect(http.StatusFound, "/setup")
		return
	}

	hash, err := auth.HashPassword(password)
	if err != nil {
		h.setFlashKey(c, "flash.failed_create_user")
		c.Redirect(http.StatusFound, "/setup")
		return
	}

	user := models.User{Username: username, PasswordHash: hash}
	house := models.House{Name: houseName}

	if err := h.db.Transaction(func(tx *gorm.DB) error {
		if err := tx.Create(&user).Error; err != nil {
			return err
		}
		if err := tx.Create(&house).Error; err != nil {
			return err
		}
		member := models.HouseMember{UserID: user.ID, HouseID: house.ID}
		return tx.Create(&member).Error
	}); err != nil {
		h.setFlashKey(c, "flash.failed_setup")
		c.Redirect(http.StatusFound, "/setup")
		return
	}

	h.setSessionUser(c, user.ID, house.ID)
	h.setFlash(c, fmt.Sprintf(h.t(c, "flash.welcome"), h.t(c, "app.name")))
	c.Redirect(http.StatusFound, "/")
}

func (h *Handlers) getLogin(c *gin.Context) {
	if h.isLoggedIn(c) {
		c.Redirect(http.StatusFound, "/")
		return
	}

	data := h.baseData(c)
	data["Title"] = "Login"
	c.HTML(http.StatusOK, "login", data)
}

func (h *Handlers) postLogin(c *gin.Context) {
	username := strings.TrimSpace(c.PostForm("username"))
	password := c.PostForm("password")

	if username == "" || password == "" {
		h.setFlashKey(c, "flash.enter_credentials")
		c.Redirect(http.StatusFound, "/login")
		return
	}

	var user models.User
	if err := h.db.Where("username = ?", username).First(&user).Error; err != nil {
		h.setFlashKey(c, "flash.invalid_credentials")
		c.Redirect(http.StatusFound, "/login")
		return
	}

	if !auth.CheckPassword(user.PasswordHash, password) {
		h.setFlashKey(c, "flash.invalid_credentials")
		c.Redirect(http.StatusFound, "/login")
		return
	}

	houseID := h.firstHouseID(user.ID)
	h.setSessionUser(c, user.ID, houseID)
	c.Redirect(http.StatusFound, "/")
}

func (h *Handlers) getLogout(c *gin.Context) {
	h.clearSession(c)
	c.Redirect(http.StatusFound, "/login")
}

func (h *Handlers) authRequired() gin.HandlerFunc {
	return func(c *gin.Context) {
		userID, ok := h.sessionUserID(c)
		if !ok {
			c.Redirect(http.StatusFound, "/login")
			c.Abort()
			return
		}

		var user models.User
		if err := h.db.First(&user, userID).Error; err != nil {
			h.clearSession(c)
			c.Redirect(http.StatusFound, "/login")
			c.Abort()
			return
		}

		c.Set("user", &user)
		if house, err := h.ensureHouse(c, user.ID); err == nil {
			c.Set("house", house)
		}

		c.Next()
	}
}
