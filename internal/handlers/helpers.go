package handlers

import (
	"crypto/rand"
	"log"
	"strconv"
	"strings"

	"github.com/Larguma/stuff/internal/i18n"
	"github.com/Larguma/stuff/internal/models"
	"github.com/gin-contrib/sessions"
	"github.com/gin-gonic/gin"
)

type LangOption struct {
	Code  string
	Label string
}

func (h *Handlers) baseData(c *gin.Context) gin.H {
	lang := h.language(c)
	data := gin.H{
		"AppName":     i18n.T(lang, "app.name"),
		"Flash":       h.popFlash(c),
		"Lang":        lang,
		"LangOptions": h.langOptions(),
	}

	if user, ok := c.Get("user"); ok {
		data["User"] = user
	}
	if house, ok := c.Get("house"); ok {
		data["House"] = house
	}

	return data
}

func (h *Handlers) language(c *gin.Context) string {
	lang := strings.ToLower(strings.TrimSpace(c.Query("lang")))
	if lang != "" && i18n.IsSupported(lang) {
		session := sessions.Default(c)
		session.Set("lang", lang)
		if err := session.Save(); err != nil {
			log.Printf("lang save failed: %v", err)
		}
		return lang
	}

	if value := sessions.Default(c).Get("lang"); value != nil {
		if typed, ok := value.(string); ok {
			typed = strings.ToLower(typed)
			if i18n.IsSupported(typed) {
				return typed
			}
		}
	}

	return "en"
}

func (h *Handlers) langOptions() []LangOption {
	langs := i18n.Supported()
	options := make([]LangOption, 0, len(langs))
	for _, lang := range langs {
		options = append(options, LangOption{Code: lang.Code, Label: lang.Label})
	}
	return options
}

func (h *Handlers) t(c *gin.Context, key string) string {
	return i18n.T(h.language(c), key)
}

func (h *Handlers) setFlashKey(c *gin.Context, key string) {
	h.setFlash(c, h.t(c, key))
}

func (h *Handlers) anyUsers() bool {
	var count int64
	h.db.Model(&models.User{}).Count(&count)
	return count > 0
}

func (h *Handlers) currentUser(c *gin.Context) *models.User {
	if user, ok := c.Get("user"); ok {
		if typed, ok := user.(*models.User); ok {
			return typed
		}
	}
	return nil
}

func (h *Handlers) currentHouse(c *gin.Context) (*models.House, bool) {
	if house, ok := c.Get("house"); ok {
		if typed, ok := house.(*models.House); ok {
			return typed, true
		}
	}
	return nil, false
}

func (h *Handlers) requireHouse(c *gin.Context) (*models.House, bool) {
	house, ok := h.currentHouse(c)
	if !ok || house == nil {
		h.setFlashKey(c, "flash.house_required")
		c.Redirect(302, "/houses")
		return nil, false
	}
	return house, true
}

func (h *Handlers) isLoggedIn(c *gin.Context) bool {
	_, ok := h.sessionUserID(c)
	return ok
}

func (h *Handlers) sessionUserID(c *gin.Context) (uint, bool) {
	value := sessions.Default(c).Get("user_id")
	return sessionUint(value)
}

func (h *Handlers) setSessionUser(c *gin.Context, userID uint, houseID uint) {
	session := sessions.Default(c)
	session.Set("user_id", userID)
	if houseID > 0 {
		session.Set("house_id", houseID)
	} else {
		session.Delete("house_id")
	}
	if err := session.Save(); err != nil {
		log.Printf("session save failed: %v", err)
	}
}

func (h *Handlers) clearSession(c *gin.Context) {
	session := sessions.Default(c)
	session.Clear()
	if err := session.Save(); err != nil {
		log.Printf("session clear failed: %v", err)
	}
}

func (h *Handlers) setFlash(c *gin.Context, message string) {
	session := sessions.Default(c)
	session.Set("flash", message)
	if err := session.Save(); err != nil {
		log.Printf("flash save failed: %v", err)
	}
}

func (h *Handlers) popFlash(c *gin.Context) string {
	session := sessions.Default(c)
	value := session.Get("flash")
	if value == nil {
		return ""
	}
	session.Delete("flash")
	if err := session.Save(); err != nil {
		log.Printf("flash delete failed: %v", err)
	}

	if message, ok := value.(string); ok {
		return message
	}
	return ""
}

func (h *Handlers) ensureHouse(c *gin.Context, userID uint) (*models.House, error) {
	session := sessions.Default(c)
	if value := session.Get("house_id"); value != nil {
		if houseID, ok := sessionUint(value); ok {
			if h.isHouseMember(userID, houseID) {
				var house models.House
				if err := h.db.First(&house, houseID).Error; err == nil {
					return &house, nil
				}
			}
		}
	}

	var house models.House
	if err := h.db.Joins("JOIN house_members hm ON hm.house_id = houses.id").
		Where("hm.user_id = ?", userID).
		Order("houses.created_at").
		First(&house).Error; err != nil {
		return nil, err
	}

	session.Set("house_id", house.ID)
	if err := session.Save(); err != nil {
		log.Printf("session save failed: %v", err)
	}

	return &house, nil
}

func (h *Handlers) isHouseMember(userID uint, houseID uint) bool {
	var count int64
	h.db.Model(&models.HouseMember{}).
		Where("user_id = ? AND house_id = ?", userID, houseID).
		Count(&count)
	return count > 0
}

func (h *Handlers) firstHouseID(userID uint) uint {
	var house models.House
	if err := h.db.Joins("JOIN house_members hm ON hm.house_id = houses.id").
		Where("hm.user_id = ?", userID).
		Order("houses.created_at").
		First(&house).Error; err != nil {
		return 0
	}
	return house.ID
}

func (h *Handlers) addMemberToHouse(userID uint, houseID uint) error {
	if h.isHouseMember(userID, houseID) {
		return nil
	}
	member := models.HouseMember{UserID: userID, HouseID: houseID}
	return h.db.Create(&member).Error
}

func randomCode(length int) (string, error) {
	const alphabet = "ABCDEFGHJKLMNPQRSTUVWXYZ23456789"
	bytes := make([]byte, length)
	if _, err := rand.Read(bytes); err != nil {
		return "", err
	}

	for i, b := range bytes {
		bytes[i] = alphabet[int(b)%len(alphabet)]
	}

	return string(bytes), nil
}

func sessionUint(value any) (uint, bool) {
	switch typed := value.(type) {
	case uint:
		return typed, true
	case int:
		if typed < 0 {
			return 0, false
		}
		return uint(typed), true
	case int64:
		if typed < 0 {
			return 0, false
		}
		return uint(typed), true
	case float64:
		if typed < 0 {
			return 0, false
		}
		return uint(typed), true
	case string:
		parsed, err := strconv.ParseUint(typed, 10, 64)
		if err != nil {
			return 0, false
		}
		return uint(parsed), true
	default:
		return 0, false
	}
}
