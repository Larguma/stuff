package handlers

import (
	"errors"
	"fmt"
	"log"
	"net/http"
	"os"
	"path/filepath"
	"strconv"
	"strings"

	"github.com/Larguma/stuff/internal/models"
	"github.com/Larguma/stuff/internal/utils"
	"github.com/disintegration/imaging"
	"github.com/gin-gonic/gin"
	"gorm.io/gorm"
)

func (h *Handlers) getItems(c *gin.Context) {
	house, ok := h.requireHouse(c)
	if !ok {
		return
	}

	selectedLocationID := strings.TrimSpace(c.Query("location_id"))
	query := h.db.Where("house_id = ?", house.ID).
		Preload("Location").
		Preload("Tags").
		Preload("Images")
	if selectedLocationID != "" {
		if selectedLocationID == "0" {
			query = query.Where("location_id IS NULL")
		} else if locationID, err := strconv.ParseUint(selectedLocationID, 10, 64); err == nil {
			query = query.Where("location_id = ?", uint(locationID))
		} else {
			selectedLocationID = ""
		}
	}

	var items []models.Item
	if err := query.Order("created_at DESC").Limit(100).Find(&items).Error; err != nil {
		h.setFlashKey(c, "flash.failed_load_items")
		c.Redirect(http.StatusFound, "/houses")
		return
	}

	var locations []models.Location
	if err := h.db.Where("house_id = ?", house.ID).Order("name").Find(&locations).Error; err != nil {
		log.Printf("failed to load locations: %v", err)
	}

	viewMode := strings.ToLower(strings.TrimSpace(c.Query("view")))
	if viewMode != "cards" {
		viewMode = "list"
	}

	locationQuery := ""
	if selectedLocationID != "" {
		locationQuery = "&location_id=" + selectedLocationID
	}

	data := h.baseData(c)
	data["Title"] = "Items"
	data["Items"] = items
	data["ViewMode"] = viewMode
	data["Locations"] = locations
	data["SelectedLocationID"] = selectedLocationID
	data["LocationQuery"] = locationQuery
	c.HTML(http.StatusOK, "items", data)
}

func (h *Handlers) getNewItem(c *gin.Context) {
	house, ok := h.requireHouse(c)
	if !ok {
		return
	}

	var locations []models.Location
	if err := h.db.Where("house_id = ?", house.ID).Order("name").Find(&locations).Error; err != nil {
		log.Printf("failed to load locations: %v", err)
	}

	var tags []models.Tag
	if err := h.db.Where("house_id = ?", house.ID).Order("name").Find(&tags).Error; err != nil {
		log.Printf("failed to load tags: %v", err)
	}

	data := h.baseData(c)
	data["Title"] = "Add item"
	data["Locations"] = locations
	data["Tags"] = tags
	c.HTML(http.StatusOK, "item_new", data)
}

func (h *Handlers) postNewItem(c *gin.Context) {
	user := h.currentUser(c)
	if user == nil {
		c.Redirect(http.StatusFound, "/login")
		return
	}

	house, ok := h.requireHouse(c)
	if !ok {
		return
	}

	name := strings.TrimSpace(c.PostForm("name"))
	if name == "" {
		h.setFlashKey(c, "flash.item_name_required")
		c.Redirect(http.StatusFound, "/items/new")
		return
	}

	quantity, err := parseQuantity(c.PostForm("quantity"))
	if err != nil {
		h.setFlashKey(c, "flash.invalid_quantity")
		c.Redirect(http.StatusFound, "/items/new")
		return
	}

	locationName := strings.TrimSpace(c.PostForm("location"))
	link := strings.TrimSpace(c.PostForm("link"))
	item := models.Item{
		HouseID:         house.ID,
		Name:            name,
		Notes:           strings.TrimSpace(c.PostForm("notes")),
		Quantity:        quantity,
		Link:            link,
		CreatedByUserID: user.ID,
	}

	var location *models.Location
	if locationName != "" {
		location, err = h.findOrCreateLocation(house.ID, locationName)
		if err != nil {
			h.setFlashKey(c, "flash.could_not_save_location")
			c.Redirect(http.StatusFound, "/items/new")
			return
		}
		item.LocationID = &location.ID
	}

	if err := h.db.Create(&item).Error; err != nil {
		h.setFlashKey(c, "flash.could_not_save_item")
		c.Redirect(http.StatusFound, "/items/new")
		return
	}

	tagNames := utils.SplitTags(c.PostForm("tags"))
	if len(tagNames) > 0 {
		tags := make([]models.Tag, 0, len(tagNames))
		for _, tagName := range tagNames {
			tag, err := h.findOrCreateTag(house.ID, tagName)
			if err != nil {
				h.setFlashKey(c, "flash.could_not_save_tags")
				c.Redirect(http.StatusFound, "/items/new")
				return
			}
			tags = append(tags, *tag)
		}
		if err := h.db.Model(&item).Association("Tags").Replace(&tags); err != nil {
			h.setFlashKey(c, "flash.could_not_save_tags")
			c.Redirect(http.StatusFound, "/items/new")
			return
		}
	}

	images, err := h.saveUploadedImages(c, item.ID)
	if err != nil {
		h.setFlashKey(c, "flash.could_not_upload_images")
		c.Redirect(http.StatusFound, "/items/new")
		return
	}
	if len(images) > 0 {
		if err := h.db.Create(&images).Error; err != nil {
			h.setFlashKey(c, "flash.could_not_save_images")
			c.Redirect(http.StatusFound, "/items/new")
			return
		}
	}

	if err := h.db.Preload("Tags").Preload("Location").First(&item, item.ID).Error; err == nil {
		if err := h.index.IndexItem(&item); err != nil {
			log.Printf("search index update failed: %v", err)
		}
	}

	h.setFlashKey(c, "flash.item_added")
	c.Redirect(http.StatusFound, "/")
}

func (h *Handlers) getItem(c *gin.Context) {
	house, ok := h.requireHouse(c)
	if !ok {
		return
	}

	itemID, err := strconv.ParseUint(c.Param("id"), 10, 64)
	if err != nil {
		h.setFlashKey(c, "flash.item_not_found")
		c.Redirect(http.StatusFound, "/")
		return
	}

	var item models.Item
	if err := h.db.Where("house_id = ? AND id = ?", house.ID, uint(itemID)).
		Preload("Location").
		Preload("Tags").
		Preload("Images").
		First(&item).Error; err != nil {
		h.setFlashKey(c, "flash.item_not_found")
		c.Redirect(http.StatusFound, "/")
		return
	}

	data := h.baseData(c)
	data["Title"] = item.Name
	data["Item"] = item
	c.HTML(http.StatusOK, "item_view", data)
}

func (h *Handlers) getEditItem(c *gin.Context) {
	house, ok := h.requireHouse(c)
	if !ok {
		return
	}

	itemID, err := strconv.ParseUint(c.Param("id"), 10, 64)
	if err != nil {
		h.setFlashKey(c, "flash.item_not_found")
		c.Redirect(http.StatusFound, "/")
		return
	}

	var item models.Item
	if err := h.db.Where("house_id = ? AND id = ?", house.ID, uint(itemID)).
		Preload("Location").
		Preload("Tags").
		Preload("Images").
		First(&item).Error; err != nil {
		h.setFlashKey(c, "flash.item_not_found")
		c.Redirect(http.StatusFound, "/")
		return
	}

	locationName := ""
	if item.LocationID != nil {
		locationName = item.Location.Name
	}

	var locations []models.Location
	if err := h.db.Where("house_id = ?", house.ID).Order("name").Find(&locations).Error; err != nil {
		log.Printf("failed to load locations: %v", err)
	}

	var tags []models.Tag
	if err := h.db.Where("house_id = ?", house.ID).Order("name").Find(&tags).Error; err != nil {
		log.Printf("failed to load tags: %v", err)
	}

	data := h.baseData(c)
	data["Title"] = "Edit " + item.Name
	data["Item"] = item
	data["TagList"] = joinTagNames(item.Tags)
	data["LocationName"] = locationName
	data["Locations"] = locations
	data["Tags"] = tags
	c.HTML(http.StatusOK, "item_edit", data)
}

func (h *Handlers) postEditItem(c *gin.Context) {
	house, ok := h.requireHouse(c)
	if !ok {
		return
	}

	itemID, err := strconv.ParseUint(c.Param("id"), 10, 64)
	if err != nil {
		h.setFlashKey(c, "flash.item_not_found")
		c.Redirect(http.StatusFound, "/")
		return
	}

	var item models.Item
	if err := h.db.Where("house_id = ? AND id = ?", house.ID, uint(itemID)).
		Preload("Tags").
		Preload("Location").
		Preload("Images").
		First(&item).Error; err != nil {
		h.setFlashKey(c, "flash.item_not_found")
		c.Redirect(http.StatusFound, "/")
		return
	}

	editPath := fmt.Sprintf("/items/%d/edit", item.ID)
	name := strings.TrimSpace(c.PostForm("name"))
	if name == "" {
		h.setFlashKey(c, "flash.item_name_required")
		c.Redirect(http.StatusFound, editPath)
		return
	}

	quantity, err := parseQuantity(c.PostForm("quantity"))
	if err != nil {
		h.setFlashKey(c, "flash.invalid_quantity")
		c.Redirect(http.StatusFound, editPath)
		return
	}

	locationName := strings.TrimSpace(c.PostForm("location"))
	var locationID *uint
	if locationName != "" {
		location, err := h.findOrCreateLocation(house.ID, locationName)
		if err != nil {
			h.setFlashKey(c, "flash.could_not_save_location")
			c.Redirect(http.StatusFound, editPath)
			return
		}
		locationID = &location.ID
	}

	updates := map[string]any{
		"name":        name,
		"notes":       strings.TrimSpace(c.PostForm("notes")),
		"quantity":    quantity,
		"link":        strings.TrimSpace(c.PostForm("link")),
		"location_id": locationID,
	}
	if err := h.db.Model(&item).Updates(updates).Error; err != nil {
		h.setFlashKey(c, "flash.could_not_update_item")
		c.Redirect(http.StatusFound, editPath)
		return
	}

	tagNames := utils.SplitTags(c.PostForm("tags"))
	if len(tagNames) == 0 {
		if err := h.db.Model(&item).Association("Tags").Clear(); err != nil {
			h.setFlashKey(c, "flash.could_not_update_tags")
			c.Redirect(http.StatusFound, editPath)
			return
		}
	} else {
		tags := make([]models.Tag, 0, len(tagNames))
		for _, tagName := range tagNames {
			tag, err := h.findOrCreateTag(house.ID, tagName)
			if err != nil {
				h.setFlashKey(c, "flash.could_not_update_tags")
				c.Redirect(http.StatusFound, editPath)
				return
			}
			tags = append(tags, *tag)
		}
		if err := h.db.Model(&item).Association("Tags").Replace(&tags); err != nil {
			h.setFlashKey(c, "flash.could_not_update_tags")
			c.Redirect(http.StatusFound, editPath)
			return
		}
	}

	deleteIDs := c.PostFormArray("delete_images")
	if len(deleteIDs) > 0 {
		imagesByID := make(map[uint]models.ItemImage, len(item.Images))
		for _, image := range item.Images {
			imagesByID[image.ID] = image
		}

		for _, idStr := range deleteIDs {
			parsed, err := strconv.ParseUint(idStr, 10, 64)
			if err != nil {
				continue
			}
			image, ok := imagesByID[uint(parsed)]
			if !ok {
				continue
			}

			if err := h.db.Delete(&models.ItemImage{}, "id = ? AND item_id = ?", image.ID, item.ID).Error; err != nil {
				h.setFlashKey(c, "flash.could_not_remove_images")
				c.Redirect(http.StatusFound, editPath)
				return
			}

			path := filepath.Join(h.cfg.UploadDir, filepath.FromSlash(image.Path))
			if err := os.Remove(path); err != nil && !errors.Is(err, os.ErrNotExist) {
				log.Printf("failed to remove image %s: %v", path, err)
			}
			if image.ThumbnailPath != "" {
				thumbPath := filepath.Join(h.cfg.UploadDir, filepath.FromSlash(image.ThumbnailPath))
				if err := os.Remove(thumbPath); err != nil && !errors.Is(err, os.ErrNotExist) {
					log.Printf("failed to remove thumbnail %s: %v", thumbPath, err)
				}
			}
		}
	}

	images, err := h.saveUploadedImages(c, item.ID)
	if err != nil {
		h.setFlashKey(c, "flash.could_not_upload_images")
		c.Redirect(http.StatusFound, editPath)
		return
	}
	if len(images) > 0 {
		if err := h.db.Create(&images).Error; err != nil {
			h.setFlashKey(c, "flash.could_not_save_images")
			c.Redirect(http.StatusFound, editPath)
			return
		}
	}

	if err := h.db.Preload("Tags").Preload("Location").First(&item, item.ID).Error; err == nil {
		if err := h.index.IndexItem(&item); err != nil {
			log.Printf("search index update failed: %v", err)
		}
	}

	h.setFlashKey(c, "flash.item_updated")
	c.Redirect(http.StatusFound, fmt.Sprintf("/items/%d", item.ID))
}

func (h *Handlers) postDeleteItem(c *gin.Context) {
	house, ok := h.requireHouse(c)
	if !ok {
		return
	}

	itemID, err := strconv.ParseUint(c.Param("id"), 10, 64)
	if err != nil {
		h.setFlashKey(c, "flash.item_not_found")
		c.Redirect(http.StatusFound, "/")
		return
	}

	var item models.Item
	if err := h.db.Where("house_id = ? AND id = ?", house.ID, uint(itemID)).
		Preload("Images").
		First(&item).Error; err != nil {
		h.setFlashKey(c, "flash.item_not_found")
		c.Redirect(http.StatusFound, "/")
		return
	}

	if err := h.db.Transaction(func(tx *gorm.DB) error {
		if err := tx.Model(&item).Association("Tags").Clear(); err != nil {
			return err
		}
		if err := tx.Delete(&models.ItemImage{}, "item_id = ?", item.ID).Error; err != nil {
			return err
		}
		return tx.Delete(&item).Error
	}); err != nil {
		h.setFlashKey(c, "flash.could_not_delete_item")
		c.Redirect(http.StatusFound, "/")
		return
	}

	for _, image := range item.Images {
		path := filepath.Join(h.cfg.UploadDir, filepath.FromSlash(image.Path))
		if err := os.Remove(path); err != nil && !errors.Is(err, os.ErrNotExist) {
			log.Printf("failed to remove image %s: %v", path, err)
		}
		if image.ThumbnailPath != "" {
			thumbPath := filepath.Join(h.cfg.UploadDir, filepath.FromSlash(image.ThumbnailPath))
			if err := os.Remove(thumbPath); err != nil && !errors.Is(err, os.ErrNotExist) {
				log.Printf("failed to remove thumbnail %s: %v", thumbPath, err)
			}
		}
	}

	if err := h.index.DeleteItem(item.ID); err != nil {
		log.Printf("search index delete failed: %v", err)
	}

	h.setFlashKey(c, "flash.item_deleted")
	c.Redirect(http.StatusFound, "/")
}

func (h *Handlers) findOrCreateLocation(houseID uint, name string) (*models.Location, error) {
	normalized := utils.NormalizeName(name)
	var location models.Location
	if err := h.db.Where("house_id = ? AND name_normalized = ?", houseID, normalized).
		First(&location).Error; err == nil {
		return &location, nil
	} else if !errors.Is(err, gorm.ErrRecordNotFound) {
		return nil, err
	}

	location = models.Location{
		HouseID:        houseID,
		Name:           name,
		NameNormalized: normalized,
	}
	if err := h.db.Create(&location).Error; err != nil {
		return nil, err
	}

	if err := h.index.IndexLocation(&location); err != nil {
		log.Printf("location index failed: %v", err)
	}

	return &location, nil
}

func (h *Handlers) findOrCreateTag(houseID uint, name string) (*models.Tag, error) {
	normalized := utils.NormalizeName(name)
	var tag models.Tag
	if err := h.db.Where("house_id = ? AND name_normalized = ?", houseID, normalized).
		First(&tag).Error; err == nil {
		return &tag, nil
	} else if !errors.Is(err, gorm.ErrRecordNotFound) {
		return nil, err
	}

	tag = models.Tag{
		HouseID:        houseID,
		Name:           name,
		NameNormalized: normalized,
	}
	if err := h.db.Create(&tag).Error; err != nil {
		return nil, err
	}

	if err := h.index.IndexTag(&tag); err != nil {
		log.Printf("tag index failed: %v", err)
	}

	return &tag, nil
}

func (h *Handlers) saveUploadedImages(c *gin.Context, itemID uint) ([]models.ItemImage, error) {
	form, err := c.MultipartForm()
	if err != nil {
		return nil, nil
	}
	files := form.File["images"]
	if len(files) == 0 {
		return nil, nil
	}

	dir := filepath.Join(h.cfg.UploadDir, fmt.Sprintf("item_%d", itemID))
	if err := os.MkdirAll(dir, 0o755); err != nil {
		return nil, err
	}

	images := make([]models.ItemImage, 0, len(files))
	for _, file := range files {
		code, err := randomCode(12)
		if err != nil {
			return nil, err
		}

		filename := code + ".jpg"
		thumbName := code + "_thumb.jpg"

		relPath := filepath.ToSlash(filepath.Join(fmt.Sprintf("item_%d", itemID), filename))
		relThumbPath := filepath.ToSlash(filepath.Join(fmt.Sprintf("item_%d", itemID), thumbName))

		fullPath := filepath.Join(h.cfg.UploadDir, relPath)
		fullThumbPath := filepath.Join(h.cfg.UploadDir, relThumbPath)

		src, err := file.Open()
		if err != nil {
			return nil, err
		}
		defer src.Close()

		img, err := imaging.Decode(src, imaging.AutoOrientation(true))
		if err != nil {
			return nil, err
		}

		mainImg := img
		if img.Bounds().Dx() > 1200 || img.Bounds().Dy() > 1200 {
			mainImg = imaging.Fit(img, 1200, 1200, imaging.Lanczos)
		}

		thumbImg := imaging.Fit(img, 400, 400, imaging.Lanczos)

		if err := imaging.Save(mainImg, fullPath, imaging.JPEGQuality(85)); err != nil {
			return nil, err
		}
		if err := imaging.Save(thumbImg, fullThumbPath, imaging.JPEGQuality(80)); err != nil {
			return nil, err
		}

		images = append(images, models.ItemImage{
			ItemID:        itemID,
			Path:          relPath,
			ThumbnailPath: relThumbPath,
			OriginalName:  file.Filename,
		})
	}

	return images, nil
}

func joinTagNames(tags []models.Tag) string {
	if len(tags) == 0 {
		return ""
	}

	names := make([]string, 0, len(tags))
	for _, tag := range tags {
		names = append(names, tag.Name)
	}

	return strings.Join(names, ", ")
}

func parseQuantity(value string) (int, error) {
	trimmed := strings.TrimSpace(value)
	if trimmed == "" {
		return 1, nil
	}

	quantity, err := strconv.Atoi(trimmed)
	if err != nil || quantity < 0 {
		return 0, errors.New("invalid quantity")
	}

	return quantity, nil
}
