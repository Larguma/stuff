package handlers

import (
	"net/http"
	"strings"

	"github.com/Larguma/stuff/internal/models"
	"github.com/gin-gonic/gin"
)

func (h *Handlers) getSearch(c *gin.Context) {
	house, ok := h.requireHouse(c)
	if !ok {
		return
	}

	query := strings.TrimSpace(c.Query("q"))
	data := h.baseData(c)
	data["Title"] = "Search"
	data["Query"] = query

	if query != "" {
		itemIDs, err := h.index.SearchItemIDs(house.ID, query)
		if err == nil {
			items, err := h.loadItemsByIDs(itemIDs)
			if err == nil {
				data["Items"] = items
			}
		}

		tagIDs, err := h.index.SearchTagIDs(house.ID, query)
		if err == nil {
			tags, err := h.loadTagsByIDs(tagIDs)
			if err == nil {
				data["Tags"] = tags
			}
		}

		locationIDs, err := h.index.SearchLocationIDs(house.ID, query)
		if err == nil {
			locations, err := h.loadLocationsByIDs(locationIDs)
			if err == nil {
				data["Locations"] = locations
			}
		}
	}

	c.HTML(http.StatusOK, "search", data)
}

func (h *Handlers) loadItemsByIDs(ids []uint) ([]models.Item, error) {
	if len(ids) == 0 {
		return nil, nil
	}

	var items []models.Item
	if err := h.db.Preload("Location").Preload("Tags").Preload("Images").Find(&items, ids).Error; err != nil {
		return nil, err
	}

	byID := make(map[uint]models.Item, len(items))
	for _, item := range items {
		byID[item.ID] = item
	}

	ordered := make([]models.Item, 0, len(ids))
	for _, id := range ids {
		if item, ok := byID[id]; ok {
			ordered = append(ordered, item)
		}
	}

	return ordered, nil
}

func (h *Handlers) loadTagsByIDs(ids []uint) ([]models.Tag, error) {
	if len(ids) == 0 {
		return nil, nil
	}

	var tags []models.Tag
	if err := h.db.Find(&tags, ids).Error; err != nil {
		return nil, err
	}

	byID := make(map[uint]models.Tag, len(tags))
	for _, tag := range tags {
		byID[tag.ID] = tag
	}

	ordered := make([]models.Tag, 0, len(ids))
	for _, id := range ids {
		if tag, ok := byID[id]; ok {
			ordered = append(ordered, tag)
		}
	}

	return ordered, nil
}

func (h *Handlers) loadLocationsByIDs(ids []uint) ([]models.Location, error) {
	if len(ids) == 0 {
		return nil, nil
	}

	var locations []models.Location
	if err := h.db.Find(&locations, ids).Error; err != nil {
		return nil, err
	}

	byID := make(map[uint]models.Location, len(locations))
	for _, location := range locations {
		byID[location.ID] = location
	}

	ordered := make([]models.Location, 0, len(ids))
	for _, id := range ids {
		if location, ok := byID[id]; ok {
			ordered = append(ordered, location)
		}
	}

	return ordered, nil
}
