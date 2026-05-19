package handlers

import (
	"fmt"
	"net/http"
	"strconv"
	"strings"

	"github.com/Larguma/stuff/internal/models"
	"github.com/gin-gonic/gin"
	"gorm.io/gorm"
)

func (h *Handlers) getHouses(c *gin.Context) {
	user := h.currentUser(c)
	if user == nil {
		c.Redirect(http.StatusFound, "/login")
		return
	}

	var houses []models.House
	h.db.Joins("JOIN house_members hm ON hm.house_id = houses.id").
		Where("hm.user_id = ?", user.ID).
		Order("houses.name").
		Find(&houses)

	data := h.baseData(c)
	data["Title"] = "Houses"
	data["Houses"] = houses

	if house, ok := h.currentHouse(c); ok {
		data["CurrentHouse"] = house

		var members []models.User
		h.db.Joins("JOIN house_members hm ON hm.user_id = users.id").
			Where("hm.house_id = ?", house.ID).
			Order("users.username").
			Find(&members)
		data["Members"] = members

		var invites []models.Invite
		h.db.Where("house_id = ? AND used_at IS NULL", house.ID).
			Order("created_at DESC").
			Find(&invites)
		data["Invites"] = invites

		scheme := "http"
		if c.Request.TLS != nil {
			scheme = "https"
		}
		data["BaseURL"] = fmt.Sprintf("%s://%s", scheme, c.Request.Host)
	}

	c.HTML(http.StatusOK, "houses", data)
}

func (h *Handlers) postCreateHouse(c *gin.Context) {
	user := h.currentUser(c)
	if user == nil {
		c.Redirect(http.StatusFound, "/login")
		return
	}

	name := strings.TrimSpace(c.PostForm("name"))
	if name == "" {
		h.setFlashKey(c, "flash.house_name_required")
		c.Redirect(http.StatusFound, "/houses")
		return
	}

	house := models.House{Name: name}
	if err := h.db.Transaction(func(tx *gorm.DB) error {
		if err := tx.Create(&house).Error; err != nil {
			return err
		}
		member := models.HouseMember{UserID: user.ID, HouseID: house.ID}
		return tx.Create(&member).Error
	}); err != nil {
		h.setFlashKey(c, "flash.could_not_create_house")
		c.Redirect(http.StatusFound, "/houses")
		return
	}

	h.setSessionUser(c, user.ID, house.ID)
	h.setFlashKey(c, "flash.house_created")
	c.Redirect(http.StatusFound, "/houses")
}

func (h *Handlers) postSelectHouse(c *gin.Context) {
	user := h.currentUser(c)
	if user == nil {
		c.Redirect(http.StatusFound, "/login")
		return
	}

	houseID, err := strconv.ParseUint(c.PostForm("house_id"), 10, 64)
	if err != nil {
		h.setFlashKey(c, "flash.invalid_house")
		c.Redirect(http.StatusFound, "/houses")
		return
	}

	if !h.isHouseMember(user.ID, uint(houseID)) {
		h.setFlashKey(c, "flash.not_house_member")
		c.Redirect(http.StatusFound, "/houses")
		return
	}

	h.setSessionUser(c, user.ID, uint(houseID))
	c.Redirect(http.StatusFound, "/houses")
}

func (h *Handlers) postAddMember(c *gin.Context) {
	house, ok := h.requireHouse(c)
	if !ok {
		return
	}

	username := strings.TrimSpace(c.PostForm("username"))
	if username == "" {
		h.setFlashKey(c, "flash.enter_username")
		c.Redirect(http.StatusFound, "/houses")
		return
	}

	var user models.User
	if err := h.db.Where("username = ?", username).First(&user).Error; err != nil {
		h.setFlashKey(c, "flash.user_not_found")
		c.Redirect(http.StatusFound, "/houses")
		return
	}

	if err := h.addMemberToHouse(user.ID, house.ID); err != nil {
		h.setFlashKey(c, "flash.could_not_add_member")
		c.Redirect(http.StatusFound, "/houses")
		return
	}

	h.setFlashKey(c, "flash.member_added")
	c.Redirect(http.StatusFound, "/houses")
}
