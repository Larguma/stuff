package search

import (
	"fmt"
	"strings"
	"unicode"

	"github.com/Larguma/stuff/internal/models"
	"gorm.io/gorm"
)

type Index struct {
	db *gorm.DB
}

func NewIndex(db *gorm.DB) *Index {
	return &Index{db: db}
}

func (i *Index) EnsureTables() error {
	statements := []string{
		"CREATE VIRTUAL TABLE IF NOT EXISTS item_search USING fts5(item_id UNINDEXED, house_id UNINDEXED, name, notes, location, tags);",
		"CREATE VIRTUAL TABLE IF NOT EXISTS tag_search USING fts5(tag_id UNINDEXED, house_id UNINDEXED, name);",
		"CREATE VIRTUAL TABLE IF NOT EXISTS location_search USING fts5(location_id UNINDEXED, house_id UNINDEXED, name);",
	}

	for _, stmt := range statements {
		if err := i.db.Exec(stmt).Error; err != nil {
			if isMissingFTS5(err) {
				return fmt.Errorf("fts5 unavailable in sqlite build; recompile with -tags \"fts5\": %w", err)
			}
			return err
		}
	}

	return nil
}

func (i *Index) Rebuild() error {
	if err := i.db.Exec("DELETE FROM item_search;").Error; err != nil {
		return err
	}
	if err := i.db.Exec("DELETE FROM tag_search;").Error; err != nil {
		return err
	}
	if err := i.db.Exec("DELETE FROM location_search;").Error; err != nil {
		return err
	}

	var tags []models.Tag
	if err := i.db.Find(&tags).Error; err != nil {
		return err
	}
	for idx := range tags {
		if err := i.IndexTag(&tags[idx]); err != nil {
			return err
		}
	}

	var locations []models.Location
	if err := i.db.Find(&locations).Error; err != nil {
		return err
	}
	for idx := range locations {
		if err := i.IndexLocation(&locations[idx]); err != nil {
			return err
		}
	}

	var items []models.Item
	if err := i.db.Preload("Tags").Preload("Location").Find(&items).Error; err != nil {
		return err
	}
	for idx := range items {
		if err := i.IndexItem(&items[idx]); err != nil {
			return err
		}
	}

	return nil
}

func (i *Index) IndexItem(item *models.Item) error {
	location := ""
	if item.LocationID != nil {
		location = item.Location.Name
	}

	tagNames := make([]string, 0, len(item.Tags))
	for _, tag := range item.Tags {
		tagNames = append(tagNames, tag.Name)
	}

	if err := i.db.Exec("DELETE FROM item_search WHERE item_id = ?", item.ID).Error; err != nil {
		return err
	}

	return i.db.Exec(
		"INSERT INTO item_search (item_id, house_id, name, notes, location, tags) VALUES (?, ?, ?, ?, ?, ?)",
		item.ID,
		item.HouseID,
		item.Name,
		item.Notes,
		location,
		strings.Join(tagNames, " "),
	).Error
}

func (i *Index) DeleteItem(itemID uint) error {
	return i.db.Exec("DELETE FROM item_search WHERE item_id = ?", itemID).Error
}

func (i *Index) IndexTag(tag *models.Tag) error {
	if err := i.db.Exec("DELETE FROM tag_search WHERE tag_id = ?", tag.ID).Error; err != nil {
		return err
	}

	return i.db.Exec(
		"INSERT INTO tag_search (tag_id, house_id, name) VALUES (?, ?, ?)",
		tag.ID,
		tag.HouseID,
		tag.Name,
	).Error
}

func (i *Index) IndexLocation(location *models.Location) error {
	if err := i.db.Exec("DELETE FROM location_search WHERE location_id = ?", location.ID).Error; err != nil {
		return err
	}

	return i.db.Exec(
		"INSERT INTO location_search (location_id, house_id, name) VALUES (?, ?, ?)",
		location.ID,
		location.HouseID,
		location.Name,
	).Error
}

func (i *Index) SearchItemIDs(houseID uint, query string) ([]uint, error) {
	ftsQuery := buildFTSQuery(query)
	if ftsQuery == "" {
		return nil, nil
	}

	rows, err := i.db.Raw(
		"SELECT item_id FROM item_search WHERE item_search MATCH ? AND house_id = ? ORDER BY bm25(item_search)",
		ftsQuery,
		houseID,
	).Rows()
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var ids []uint
	for rows.Next() {
		var id uint
		if err := rows.Scan(&id); err != nil {
			return nil, err
		}
		ids = append(ids, id)
	}

	return ids, nil
}

func (i *Index) SearchTagIDs(houseID uint, query string) ([]uint, error) {
	ftsQuery := buildFTSQuery(query)
	if ftsQuery == "" {
		return nil, nil
	}

	rows, err := i.db.Raw(
		"SELECT tag_id FROM tag_search WHERE tag_search MATCH ? AND house_id = ? ORDER BY bm25(tag_search)",
		ftsQuery,
		houseID,
	).Rows()
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var ids []uint
	for rows.Next() {
		var id uint
		if err := rows.Scan(&id); err != nil {
			return nil, err
		}
		ids = append(ids, id)
	}

	return ids, nil
}

func (i *Index) SearchLocationIDs(houseID uint, query string) ([]uint, error) {
	ftsQuery := buildFTSQuery(query)
	if ftsQuery == "" {
		return nil, nil
	}

	rows, err := i.db.Raw(
		"SELECT location_id FROM location_search WHERE location_search MATCH ? AND house_id = ? ORDER BY bm25(location_search)",
		ftsQuery,
		houseID,
	).Rows()
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var ids []uint
	for rows.Next() {
		var id uint
		if err := rows.Scan(&id); err != nil {
			return nil, err
		}
		ids = append(ids, id)
	}

	return ids, nil
}

func buildFTSQuery(input string) string {
	tokens := strings.FieldsFunc(input, func(r rune) bool {
		return !unicode.IsLetter(r) && !unicode.IsNumber(r)
	})

	if len(tokens) == 0 {
		return ""
	}

	for idx, token := range tokens {
		tokens[idx] = token + "*"
	}

	return strings.Join(tokens, " ")
}

func isMissingFTS5(err error) bool {
	if err == nil {
		return false
	}
	return strings.Contains(err.Error(), "no such module: fts5")
}
