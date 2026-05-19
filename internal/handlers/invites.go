package handlers

import (
	"errors"
	"fmt"
	"log"
	"net/http"
	"strings"
	"time"

	"github.com/Larguma/stuff/internal/auth"
	"github.com/Larguma/stuff/internal/models"
	"github.com/gin-gonic/gin"
	"gorm.io/gorm"
)

func (h *Handlers) getInvite(c *gin.Context) {
	code := strings.TrimSpace(c.Param("code"))
	data := h.baseData(c)
	data["Title"] = "Invite"
	if userID, ok := h.sessionUserID(c); ok {
		data["LoggedIn"] = true
		var user models.User
		if err := h.db.First(&user, userID).Error; err == nil {
			data["User"] = &user
		}
	}

	invite, house, err := h.loadInvite(code)
	if err != nil {
		data["Error"] = h.t(c, "invite.error_not_found")
		c.HTML(http.StatusNotFound, "invite", data)
		return
	}

	if invite.UsedAt != nil {
		data["Error"] = h.t(c, "invite.error_used")
		data["House"] = house
		c.HTML(http.StatusOK, "invite", data)
		return
	}

	data["Invite"] = invite
	data["House"] = house
	c.HTML(http.StatusOK, "invite", data)
}

func (h *Handlers) postInvite(c *gin.Context) {
	code := strings.TrimSpace(c.Param("code"))
	invite, house, err := h.loadInvite(code)
	if err != nil {
		h.setFlashKey(c, "flash.invite_not_found")
		c.Redirect(http.StatusFound, "/login")
		return
	}
	if invite.UsedAt != nil {
		h.setFlashKey(c, "flash.invite_used")
		c.Redirect(http.StatusFound, "/login")
		return
	}

	if h.isLoggedIn(c) {
		userID, ok := h.sessionUserID(c)
		if !ok {
			h.setFlashKey(c, "flash.login_again")
			c.Redirect(http.StatusFound, "/login")
			return
		}

		if err := h.addMemberToHouse(userID, invite.HouseID); err != nil {
			h.setFlashKey(c, "flash.failed_join")
			c.Redirect(http.StatusFound, "/houses")
			return
		}

		h.markInviteUsed(invite, &userID)
		h.setSessionUser(c, userID, invite.HouseID)
		h.setFlash(c, fmt.Sprintf(h.t(c, "flash.joined_house"), house.Name))
		c.Redirect(http.StatusFound, "/")
		return
	}

	username := strings.TrimSpace(c.PostForm("username"))
	password := c.PostForm("password")

	if username == "" || password == "" {
		h.setFlashKey(c, "flash.provide_credentials")
		c.Redirect(http.StatusFound, "/invite/"+code)
		return
	}

	if len(password) < 6 {
		h.setFlashKey(c, "flash.password_short")
		c.Redirect(http.StatusFound, "/invite/"+code)
		return
	}

	var existing int64
	h.db.Model(&models.User{}).Where("username = ?", username).Count(&existing)
	if existing > 0 {
		h.setFlashKey(c, "flash.username_exists")
		c.Redirect(http.StatusFound, "/login")
		return
	}

	hash, err := auth.HashPassword(password)
	if err != nil {
		h.setFlashKey(c, "flash.could_not_create_account")
		c.Redirect(http.StatusFound, "/invite/"+code)
		return
	}

	user := models.User{Username: username, PasswordHash: hash}
	if err := h.db.Transaction(func(tx *gorm.DB) error {
		if err := tx.Create(&user).Error; err != nil {
			return err
		}
		member := models.HouseMember{UserID: user.ID, HouseID: invite.HouseID}
		return tx.Create(&member).Error
	}); err != nil {
		h.setFlashKey(c, "flash.could_not_create_account")
		c.Redirect(http.StatusFound, "/invite/"+code)
		return
	}

	h.markInviteUsed(invite, &user.ID)
	h.setSessionUser(c, user.ID, invite.HouseID)
	c.Redirect(http.StatusFound, "/")
}

func (h *Handlers) postCreateInvite(c *gin.Context) {
	house, ok := h.requireHouse(c)
	if !ok {
		return
	}

	user := h.currentUser(c)
	if user == nil {
		c.Redirect(http.StatusFound, "/login")
		return
	}

	code, err := h.generateInviteCode()
	if err != nil {
		h.setFlashKey(c, "flash.could_not_create_invite")
		c.Redirect(http.StatusFound, "/houses")
		return
	}

	invite := models.Invite{
		HouseID:         house.ID,
		Code:            code,
		CreatedByUserID: user.ID,
		CreatedAt:       time.Now(),
	}

	if err := h.db.Create(&invite).Error; err != nil {
		h.setFlashKey(c, "flash.could_not_create_invite")
		c.Redirect(http.StatusFound, "/houses")
		return
	}

	h.setFlashKey(c, "flash.invite_created")
	c.Redirect(http.StatusFound, "/houses")
}

func (h *Handlers) loadInvite(code string) (*models.Invite, *models.House, error) {
	var invite models.Invite
	if err := h.db.Where("code = ?", code).First(&invite).Error; err != nil {
		return nil, nil, err
	}

	var house models.House
	if err := h.db.First(&house, invite.HouseID).Error; err != nil {
		return nil, nil, err
	}

	return &invite, &house, nil
}

func (h *Handlers) markInviteUsed(invite *models.Invite, userID *uint) {
	now := time.Now()
	updates := map[string]any{
		"used_at": now,
	}
	if userID != nil {
		updates["used_by_user_id"] = *userID
	}

	if err := h.db.Model(invite).Updates(updates).Error; err != nil {
		log.Printf("failed to mark invite used: %v", err)
	}
}

func (h *Handlers) generateInviteCode() (string, error) {
	for i := 0; i < 5; i++ {
		code, err := randomCode(10)
		if err != nil {
			return "", err
		}

		var count int64
		h.db.Model(&models.Invite{}).Where("code = ?", code).Count(&count)
		if count == 0 {
			return code, nil
		}
	}
	return "", errors.New("could not generate invite code")
}
