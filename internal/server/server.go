package server

import (
	"html/template"
	"net/http"

	"github.com/Larguma/stuff/internal/config"
	"github.com/Larguma/stuff/internal/handlers"
	"github.com/Larguma/stuff/internal/i18n"
	"github.com/Larguma/stuff/internal/search"
	"github.com/gin-contrib/multitemplate"
	"github.com/gin-contrib/sessions"
	"github.com/gin-contrib/sessions/cookie"
	"github.com/gin-gonic/gin"
	"gorm.io/gorm"
)

func NewRouter(db *gorm.DB, cfg config.Config, index *search.Index) *gin.Engine {
	router := gin.Default()
	router.MaxMultipartMemory = 32 << 20

	router.HTMLRender = newRenderer()
	router.Static("/static", "web/static")
	router.Static("/uploads", cfg.UploadDir)

	store := cookie.NewStore([]byte(cfg.SessionSecret))
	store.Options(sessions.Options{
		Path:     "/",
		HttpOnly: true,
		SameSite: http.SameSiteLaxMode,
	})
	router.Use(sessions.Sessions("stuff_session", store))

	h := handlers.New(db, cfg, index)
	h.RegisterRoutes(router)

	return router
}

func newRenderer() multitemplate.Renderer {
	renderer := multitemplate.NewRenderer()
	funcs := template.FuncMap{
		"t": func(lang string, key string) string {
			return i18n.T(lang, key)
		},
	}
	base := "web/templates/base.html"

	renderer.AddFromFilesFuncs("login", funcs, base, "web/templates/login.html")
	renderer.AddFromFilesFuncs("setup", funcs, base, "web/templates/setup.html")
	renderer.AddFromFilesFuncs("items", funcs, base, "web/templates/items.html")
	renderer.AddFromFilesFuncs("item_new", funcs, base, "web/templates/item_new.html")
	renderer.AddFromFilesFuncs("item_edit", funcs, base, "web/templates/item_edit.html")
	renderer.AddFromFilesFuncs("item_view", funcs, base, "web/templates/item_view.html")
	renderer.AddFromFilesFuncs("search", funcs, base, "web/templates/search.html")
	renderer.AddFromFilesFuncs("houses", funcs, base, "web/templates/houses.html")
	renderer.AddFromFilesFuncs("invite", funcs, base, "web/templates/invite.html")

	return renderer
}
