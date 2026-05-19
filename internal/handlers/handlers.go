package handlers

import (
	"github.com/Larguma/stuff/internal/config"
	"github.com/Larguma/stuff/internal/search"
	"github.com/gin-gonic/gin"
	"gorm.io/gorm"
)

type Handlers struct {
	db    *gorm.DB
	cfg   config.Config
	index *search.Index
}

func New(db *gorm.DB, cfg config.Config, index *search.Index) *Handlers {
	return &Handlers{db: db, cfg: cfg, index: index}
}

func (h *Handlers) RegisterRoutes(router *gin.Engine) {
	router.GET("/setup", h.getSetup)
	router.POST("/setup", h.postSetup)
	router.GET("/login", h.getLogin)
	router.POST("/login", h.postLogin)
	router.GET("/logout", h.getLogout)
	router.GET("/invite/:code", h.getInvite)
	router.POST("/invite/:code", h.postInvite)

	authenticated := router.Group("/")
	authenticated.Use(h.authRequired())
	{
		authenticated.GET("/", h.getItems)
		authenticated.GET("/items/new", h.getNewItem)
		authenticated.POST("/items", h.postNewItem)
		authenticated.GET("/items/:id/edit", h.getEditItem)
		authenticated.GET("/items/:id", h.getItem)
		authenticated.POST("/items/:id", h.postEditItem)
		authenticated.POST("/items/:id/delete", h.postDeleteItem)
		authenticated.GET("/search", h.getSearch)
		authenticated.GET("/houses", h.getHouses)
		authenticated.POST("/houses", h.postCreateHouse)
		authenticated.POST("/houses/select", h.postSelectHouse)
		authenticated.POST("/members", h.postAddMember)
		authenticated.POST("/invites", h.postCreateInvite)
	}
}
