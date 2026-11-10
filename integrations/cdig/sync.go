package cdig

import (
	"context"
	"errors"
	"fmt"
	"time"

	"github.com/OdairPianta/unisinos_usp_api/models"
	"github.com/OdairPianta/unisinos_usp_api/repository"
	"gorm.io/gorm"
)

type Action struct {
	Kind   string `json:"kind"`
	ID     string `json:"external_id"`
	ClueID uint   `json:"clue_id,omitempty"`
	Action string `json:"action"`
	Reason string `json:"reason,omitempty"`
}
type Report struct {
	Enabled   bool           `json:"enabled"`
	Committed bool           `json:"committed"`
	DryRun    bool           `json:"dry_run"`
	Source    string         `json:"source,omitempty"`
	Actions   []Action       `json:"actions"`
	Counts    map[string]int `json:"counts"`
}
type Sync struct {
	Config Config
	Client *Client
	DB     *gorm.DB
}

func (s *Sync) Run(ctx context.Context, dry bool) (report Report, err error) {
	report = Report{Enabled: s.Config.Enabled, DryRun: dry, Actions: []Action{}, Counts: map[string]int{}}
	if !s.Config.Enabled {
		return report, nil
	}
	report.Source = s.Config.SourceKey
	if s.DB == nil || s.Client == nil {
		return report, errors.New("CDIG sync dependencies are missing")
	}
	// Fetch/validate the entire catalog and revoke a temporary token BEFORE any writes.
	catalog, e := s.Client.Fetch(ctx)
	if e != nil {
		return report, e
	}
	items, e := s.Config.Items(catalog)
	if e != nil {
		return report, e
	}
	err = s.DB.WithContext(ctx).Connection(func(db *gorm.DB) error {
		// Connection-scoped MySQL advisory lock protects multiple workers/processes.
		var acquired int
		key := "cdig:" + s.Config.SourceID()[:48]
		if e := db.Raw("SELECT GET_LOCK(?, 0)", key).Scan(&acquired).Error; e != nil || acquired != 1 {
			return errors.New("CDIG synchronization is already running or lock is unavailable")
		}
		defer db.Exec("SELECT RELEASE_LOCK(?)", key)
		if !dry {
			if e := db.AutoMigrate(&models.CDIGMapping{}, &models.CDIGMembership{}); e != nil {
				return errors.New("CDIG tracking migration failed")
			}
		}
		return db.Transaction(func(tx *gorm.DB) error { return s.apply(tx, items, dry, &report) })
	})
	if err != nil {
		return report, errors.New("CDIG sync transaction failed or another run holds the lock")
	}
	report.Committed = !dry
	return
}
func category(tx *gorm.DB, name string, dry bool) (uint, error) {
	var rows []models.Category
	if err := tx.Where("name = ?", name).Find(&rows).Error; err != nil {
		return 0, err
	}
	if len(rows) > 1 {
		return 0, errors.New("duplicate integration category")
	}
	if len(rows) == 1 {
		return rows[0].ID, nil
	}
	if dry {
		return 0, nil
	}
	categoryType := "Pista de Dados"
	if name == "Coleção temática" {
		categoryType = "Pista Online"
	}
	c := models.Category{Name: name, Type: categoryType, ColorHex: "#21B6A8", MetadataTags: "acervo,fonte,recurso_original", Description: "Conteúdos referenciados da Cidadania Digital."}
	if err := tx.Create(&c).Error; err != nil {
		return 0, err
	}
	return c.ID, nil
}
func (s *Sync) apply(tx *gorm.DB, items []Item, dry bool, report *Report) error {
	var actor models.User
	if err := tx.First(&actor, s.Config.ActorID).Error; err != nil {
		return errors.New("CDIG import user does not exist")
	}
	var clues []models.LivingClue
	if err := tx.Unscoped().Find(&clues).Error; err != nil {
		return err
	}
	var mappings []models.CDIGMapping
	if tx.Migrator().HasTable(&models.CDIGMapping{}) {
		if err := tx.Where("source = ?", s.Config.SourceID()).Find(&mappings).Error; err != nil {
			return err
		}
	}
	byIdentity := map[string]*models.CDIGMapping{}
	byClue := map[uint]*models.LivingClue{}
	for i := range clues {
		byClue[clues[i].ID] = &clues[i]
	}
	for i := range mappings {
		byIdentity[mappings[i].Kind+":"+mappings[i].ExternalID] = &mappings[i]
	}
	cat, err := category(tx, "Acervo Cidadania Digital", dry)
	if err != nil {
		return err
	}
	collectionCat, err := category(tx, "Coleção temática", dry)
	if err != nil {
		return err
	}
	seen := map[string]bool{}
	dest := map[string]uint{}
	writableSeries := map[string]bool{}
	add := func(item Item, action, reason string, id uint) {
		report.Actions = append(report.Actions, Action{item.Kind, item.ID, id, action, reason})
		report.Counts[action]++
	}
	for _, item := range items {
		key := item.Kind + ":" + item.ID
		if seen[key] {
			return errors.New("duplicate CDIG source identity")
		}
		seen[key] = true
		mapping := byIdentity[key]
		var clue *models.LivingClue
		action := ""
		if mapping != nil {
			clue = byClue[mapping.ClueID]
			if mapping.Suppressed || clue == nil || clue.DeletedAt.Valid {
				add(item, "suppressed", "deleted or explicitly suppressed locally", mapping.ClueID)
				if !dry && !mapping.Suppressed {
					if e := tx.Model(mapping).Update("suppressed", true).Error; e != nil {
						return e
					}
				}
				continue
			}
			if mapping.RemoteHash == item.Hash && mapping.Missing {
				if data, ok := clue.Metadata["cdig"].(map[string]any); ok {
					data["state"] = "available"
				}
				mapping.Missing = false
				mapping.LastSeenAt = time.Now()
				if !dry {
					if e := tx.Model(clue).Update("metadata", clue.Metadata).Error; e != nil {
						return e
					}
					if e := tx.Save(mapping).Error; e != nil {
						return e
					}
				}
				dest[key] = clue.ID
				add(item, "restored", "source available again; presentation retained", clue.ID)
				continue
			}
			if mapping.RemoteHash == item.Hash && !mapping.Missing {
				if !dry {
					if e := tx.Model(mapping).Update("last_seen_at", time.Now()).Error; e != nil {
						return e
					}
				}
				dest[key] = clue.ID
				add(item, "unchanged", "", clue.ID)
				continue
			}
			if mapping.AppliedHash != ownedHash(*clue) {
				add(item, "conflict", "managed content was edited locally", clue.ID)
				continue
			}
			action = "updated"
		} else {
			candidates := []*models.LivingClue{}
			mismatch := false
			for _, candidate := range clues {
				if candidate.Metadata["acervo"] != "Cidadania Digital" || candidate.Metadata["cdig"] != nil {
					continue
				}
				legacyID, _ := candidate.Metadata["identificador_origem"].(string)
				origin, _ := candidate.Metadata["recurso_original"].(string)
				if legacyID == item.Kind+"-"+item.ID {
					if canonical(origin) != "" && canonical(origin) == canonical(item.URL) {
						candidates = append(candidates, byClue[candidate.ID])
					} else {
						mismatch = true
					}
				}
			}
			if len(candidates) > 1 || (len(candidates) == 0 && mismatch) {
				add(item, "conflict", "legacy identity requires source URL verification", 0)
				continue
			}
			mapping = &models.CDIGMapping{Source: s.Config.SourceID(), Kind: item.Kind, ExternalID: item.ID}
			if len(candidates) == 1 {
				clue = candidates[0]
				mapping.ClueID = clue.ID
				if clue.DeletedAt.Valid {
					mapping.Suppressed = true
					if !dry {
						if e := tx.Create(mapping).Error; e != nil {
							return e
						}
					}
					add(item, "suppressed", "legacy import was deleted locally", clue.ID)
					continue
				}
				// Adopt the verified identity but preserve its current presentation and cover.
				action = "linked"
			} else {
				clue = &models.LivingClue{CategoryID: cat, IsActive: true}
				if item.Kind == "serie" {
					clue.CategoryID = collectionCat
				}
				action = "created"
			}
		}
		if clue.Metadata == nil {
			clue.Metadata = models.JSONMap{}
		}
		clue.Metadata["cdig"] = item.Data
		if action != "linked" {
			clue.Name = item.Name
			clue.Description = item.Description
			clue.DescriptionDelta = delta(item.Description)
			clue.Latitude = item.Lat
			clue.Longitude = item.Lng
			if item.Image != "" {
				clue.ImageURL = item.Image
			}
		}
		if action == "created" {
			clue.Metadata["acervo"] = "Cidadania Digital"
			clue.Metadata["fonte"] = item.URL
			clue.Metadata["recurso_original"] = item.URL
			clue.Metadata["identificador_origem"] = item.Kind + "-" + item.ID
		}
		if !dry {
			if action == "created" {
				if e := tx.Create(clue).Error; e != nil {
					return e
				}
			} else {
				if e := tx.Model(clue).Select("name", "description", "description_delta", "latitude", "longitude", "image_url", "metadata").Updates(clue).Error; e != nil {
					return e
				}
			}
			if e := repository.SyncImportedCluePreviews(tx, clue); e != nil {
				return e
			}
			if e := tx.Create(&models.ClueEdit{LivingClueID: clue.ID, UserID: s.Config.ActorID, EventType: map[bool]string{true: "created", false: "updated"}[action == "created"]}).Error; e != nil {
				return e
			}
		} else if clue.ID == 0 {
			clue.ID = uint(4000000000 + len(report.Actions))
		}
		mapping.ClueID = clue.ID
		mapping.RemoteHash = item.Hash
		mapping.AppliedHash = ownedHash(*clue)
		mapping.LastSeenAt = time.Now()
		mapping.Missing = false
		if !dry {
			if e := tx.Save(mapping).Error; e != nil {
				return e
			}
		}
		byIdentity[key] = mapping
		byClue[clue.ID] = clue
		dest[key] = clue.ID
		add(item, action, "", clue.ID)
		if item.Kind == "serie" {
			writableSeries[item.ID] = true
		}
	}
	for _, item := range items {
		if item.Kind == "serie" && writableSeries[item.ID] {
			if e := s.memberships(tx, item, byClue[dest["serie:"+item.ID]], dest, dry); e != nil {
				return e
			}
			m := byIdentity["serie:"+item.ID]
			m.AppliedHash = ownedHash(*byClue[m.ClueID])
			if !dry {
				if e := tx.Save(m).Error; e != nil {
					return e
				}
			}
		}
	}
	for _, m := range mappings {
		key := m.Kind + ":" + m.ExternalID
		if seen[key] || m.Suppressed || m.Missing {
			continue
		}
		clue := byClue[m.ClueID]
		add(Item{Kind: m.Kind, ID: m.ExternalID}, "unavailable", "absent from a complete source snapshot; retained locally", m.ClueID)
		if dry {
			continue
		}
		if clue == nil || clue.DeletedAt.Valid {
			continue
		}
		// A missing source is a marker, never a deletion of local history or links.
		data, ok := clue.Metadata["cdig"].(map[string]any)
		if ok {
			data["state"] = "unavailable"
			if e := tx.Model(clue).Update("metadata", clue.Metadata).Error; e != nil {
				return e
			}
		}
		m.Missing = true
		if e := tx.Save(&m).Error; e != nil {
			return e
		}
	}
	return nil
}
func (s *Sync) memberships(tx *gorm.DB, item Item, collection *models.LivingClue, dest map[string]uint, dry bool) error {
	order := []uint{}
	for _, v := range item.Members {
		id := dest["video:"+v.ID]
		if id == 0 {
			return fmt.Errorf("series %s has an unresolved episode; transaction aborted", item.ID)
		}
		order = append(order, id)
	}
	if dry {
		collection.Metadata["colecao_ordem_itens_v1"] = order
		return nil
	}
	var types []models.ConnectionType
	if e := tx.Where("name = ?", "Coleção contém item").Find(&types).Error; e != nil {
		return e
	}
	if len(types) > 1 {
		return errors.New("duplicate collection connection types")
	}
	relation := models.ConnectionType{Name: "Coleção contém item", IsDirectional: true, StrokeStyle: "solid"}
	if len(types) == 1 {
		relation = types[0]
		if !relation.IsDirectional {
			return errors.New("collection relation must be directional")
		}
	} else {
		if e := tx.Create(&relation).Error; e != nil {
			return e
		}
	}
	var old []models.CDIGMembership
	if e := tx.Where("source = ? AND serie_id = ?", s.Config.SourceID(), item.ID).Find(&old).Error; e != nil {
		return e
	}
	wanted := map[string]bool{}
	for _, v := range item.Members {
		wanted[v.ID] = true
		var membership *models.CDIGMembership
		for i := range old {
			if old[i].VideoID == v.ID {
				membership = &old[i]
			}
		}
		if membership != nil {
			var count int64
			tx.Model(&models.ClueConnection{}).Where("id = ?", membership.ConnectionID).Count(&count)
			if count == 0 {
				return errors.New("an imported collection link was removed locally; review required")
			}
			continue
		}
		var existing models.ClueConnection
		err := tx.Where("source_clue_id = ? AND target_clue_id = ? AND connection_type_id = ?", collection.ID, dest["video:"+v.ID], relation.ID).First(&existing).Error
		if err == nil {
			continue
		}
		if !errors.Is(err, gorm.ErrRecordNotFound) {
			return err
		}
		connection := models.ClueConnection{SourceClueID: collection.ID, TargetClueID: dest["video:"+v.ID], ConnectionTypeID: relation.ID, Strength: 1, Evidence: "Curadoria CDIG"}
		if e := tx.Create(&connection).Error; e != nil {
			return e
		}
		m := models.CDIGMembership{Source: s.Config.SourceID(), SerieID: item.ID, VideoID: v.ID, ConnectionID: connection.ID}
		if e := tx.Create(&m).Error; e != nil {
			return e
		}
	}
	for _, m := range old {
		if !wanted[m.VideoID] {
			if e := tx.Delete(&models.ClueConnection{}, m.ConnectionID).Error; e != nil {
				return e
			}
			if e := tx.Delete(&m).Error; e != nil {
				return e
			}
		}
	}
	// Append local collection members instead of silently losing a curator's order.
	var connections []models.ClueConnection
	if e := tx.Where("source_clue_id = ? AND connection_type_id = ?", collection.ID, relation.ID).Order("id").Find(&connections).Error; e != nil {
		return e
	}
	included := map[uint]bool{}
	for _, id := range order {
		included[id] = true
	}
	for _, connection := range connections {
		if !included[connection.TargetClueID] {
			order = append(order, connection.TargetClueID)
			included[connection.TargetClueID] = true
		}
	}
	collection.Metadata["colecao_ordem_itens_v1"] = order
	return tx.Model(collection).Update("metadata", collection.Metadata).Error
}
