package services

import (
	"bytes"
	"crypto/sha256"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/http"
	"os"
	"sort"
	"strings"
	"time"
	"unicode"

	"github.com/OdairPianta/unisinos_usp_api/config"
	"github.com/OdairPianta/unisinos_usp_api/models"
	"golang.org/x/text/unicode/norm"
	"gorm.io/gorm"
	"gorm.io/gorm/clause"
)

const ClueDiscoveryJob = "ClueDiscoveryJob"

type DiscoveryEntry struct {
	ID    uint     `json:"id"`
	Names []string `json:"names"`
}
type DiscoveryRequest struct {
	SourceID uint             `json:"source_id"`
	Name     string           `json:"name"`
	Text     string           `json:"text"`
	Entries  []DiscoveryEntry `json:"entries"`
}
type DiscoveryReference struct {
	TargetID uint   `json:"target_id"`
	Mention  string `json:"mention"`
	Evidence string `json:"evidence"`
}
type DiscoveryAliasCandidate struct {
	Name     string `json:"name"`
	Evidence string `json:"evidence"`
}
type DiscoveryEntity struct {
	Name string `json:"name"`
	Kind string `json:"kind"`
}
type DiscoveryResult struct {
	References      []DiscoveryReference      `json:"references"`
	AliasCandidates []DiscoveryAliasCandidate `json:"alias_candidates"`
	Unresolved      []DiscoveryEntity         `json:"unresolved"`
}
type discoveryPayload struct {
	ID   uint   `json:"id"`
	Hash string `json:"hash"`
}
type discoverySnapshot struct {
	Source  models.LivingClue
	Catalog []models.LivingClue
	Request DiscoveryRequest
	Hash    string
}

func NormalizeClueName(value string) string {
	var out strings.Builder
	for _, r := range norm.NFKD.String(strings.ToLower(value)) {
		if unicode.Is(unicode.Mn, r) {
			continue
		}
		if unicode.IsLetter(r) || unicode.IsDigit(r) {
			out.WriteRune(r)
		} else {
			out.WriteByte(' ')
		}
	}
	return strings.Join(strings.Fields(out.String()), " ")
}
func ValidateClueAliases(names *[]string) error {
	if names == nil {
		return nil
	}
	if len(*names) > 20 {
		return errors.New("Informe no máximo 20 nomes alternativos")
	}
	for _, n := range *names {
		if len([]rune(strings.TrimSpace(n))) > 120 || (len([]rune(NormalizeClueName(n))) < 2 || len([]rune(NormalizeClueName(n))) > 160) {
			return errors.New("Cada nome alternativo deve ter entre 2 e 120 caracteres")
		}
	}
	return nil
}
func ReplaceClueAliases(tx *gorm.DB, clue *models.LivingClue) error {
	if clue.AliasNames == nil {
		return nil
	}
	if err := ValidateClueAliases(clue.AliasNames); err != nil {
		return err
	}
	if err := tx.Where("living_clue_id = ? AND status = ?", clue.ID, "approved").Delete(&models.ClueAlias{}).Error; err != nil {
		return err
	}
	seen := map[string]bool{}
	for _, name := range *clue.AliasNames {
		key := NormalizeClueName(name)
		if key == NormalizeClueName(clue.Name) || seen[key] {
			continue
		}
		seen[key] = true
		alias := models.ClueAlias{LivingClueID: clue.ID, Name: strings.TrimSpace(name), NormalizedName: key, Status: "approved", Origin: "manual"}
		if err := tx.Clauses(clause.OnConflict{Columns: []clause.Column{{Name: "living_clue_id"}, {Name: "normalized_name"}}, DoUpdates: clause.AssignmentColumns([]string{"name", "status", "origin", "evidence"})}).Create(&alias).Error; err != nil {
			return err
		}
	}
	return nil
}
func DiscoveryEnabled() bool { return strings.TrimSpace(os.Getenv("CLUE_DISCOVERY_URL")) != "" }
func discoverySnapshotFor(id uint) (*discoverySnapshot, error) {
	s := &discoverySnapshot{}
	if err := config.DB.Preload("Aliases", "status = ?", "approved").First(&s.Source, id).Error; err != nil {
		return nil, err
	}
	if err := config.DB.Preload("Aliases", "status = ?", "approved").Where("is_active = ?", true).Order("id").Find(&s.Catalog).Error; err != nil {
		return nil, err
	}
	if len(s.Catalog) > 10000 {
		return nil, errors.New("Catálogo excede o limite desta primeira versão")
	}
	s.Request = DiscoveryRequest{SourceID: id, Name: s.Source.Name, Text: s.Source.Name + "\n" + s.Source.Description}
	if len([]rune(s.Request.Text)) > 200000 {
		return nil, errors.New("Texto excede o limite de 200 mil caracteres para análise")
	}
	for _, clue := range s.Catalog {
		names := []string{clue.Name}
		for _, alias := range clue.Aliases {
			names = append(names, alias.Name)
		}
		sort.Strings(names)
		s.Request.Entries = append(s.Request.Entries, DiscoveryEntry{ID: clue.ID, Names: names})
	}
	// The catalogue version invalidates cached suggestions when names, aliases,
	// active records or text supporting a network suggestion change.
	type catalogText struct {
		ID          uint
		Description string
	}
	texts := []catalogText{}
	for _, clue := range s.Catalog {
		texts = append(texts, catalogText{clue.ID, clue.Description})
	}
	var links []models.ClueConnection
	if err := config.DB.Select("id", "source_clue_id", "target_clue_id", "connection_type_id", "updated_at").Order("id").Find(&links).Error; err != nil {
		return nil, err
	}
	input, _ := json.Marshal(struct {
		Request DiscoveryRequest
		Texts   []catalogText
		Links   []models.ClueConnection
	}{s.Request, texts, links})
	s.Hash = fmt.Sprintf("%x", sha256.Sum256(input))
	return s, nil
}
func QueueClueDiscovery(id uint, force bool) error {
	if !DiscoveryEnabled() {
		return nil
	}
	snapshot, err := discoverySnapshotFor(id)
	if err != nil {
		return err
	}
	return config.DB.Transaction(func(tx *gorm.DB) error {
		var locked models.LivingClue
		if err := tx.Clauses(clause.Locking{Strength: "UPDATE"}).First(&locked, id).Error; err != nil {
			return err
		}
		var state models.ClueDiscoveryState
		err := tx.Where("living_clue_id = ?", id).First(&state).Error
		if err != nil && !errors.Is(err, gorm.ErrRecordNotFound) {
			return err
		}
		if state.ContentHash == snapshot.Hash {
			if state.Status == "queued" || state.Status == "processing" || state.Status == "ready" && !force {
				return nil
			}
			if state.Status == "failed" && !force && time.Since(state.UpdatedAt) < time.Minute {
				return nil
			}
		}
		state = models.ClueDiscoveryState{LivingClueID: id, ContentHash: snapshot.Hash, Status: "queued", UpdatedAt: time.Now()}
		if err := tx.Save(&state).Error; err != nil {
			return err
		}
		payload, _ := json.Marshal(discoveryPayload{ID: id, Hash: snapshot.Hash})
		return tx.Create(&models.Job{Type: ClueDiscoveryJob, Payload: string(payload), Status: models.JobStatusPending}).Error
	})
}
func RunClueDiscovery(payload string) error {
	var job discoveryPayload
	if err := json.Unmarshal([]byte(payload), &job); err != nil {
		return err
	}
	var state models.ClueDiscoveryState
	if err := config.DB.First(&state, "living_clue_id = ?", job.ID).Error; err != nil {
		return nil
	}
	if state.ContentHash != job.Hash {
		return nil
	}
	// Failed extraction is an advisory state; do not retry every five seconds or
	// block unrelated environmental jobs when the internal service is offline.
	fail := func() {
		config.DB.Model(&models.ClueDiscoveryState{}).Where("living_clue_id = ? AND content_hash = ?", job.ID, job.Hash).Updates(map[string]interface{}{"status": "failed", "updated_at": time.Now()})
	}
	snapshot, err := discoverySnapshotFor(job.ID)
	if err != nil {
		fail()
		return nil
	}
	if snapshot.Hash != job.Hash {
		return QueueClueDiscovery(job.ID, false)
	}
	config.DB.Model(&models.ClueDiscoveryState{}).Where("living_clue_id = ? AND content_hash = ?", job.ID, job.Hash).Updates(map[string]interface{}{"status": "processing", "updated_at": time.Now()})
	raw, _ := json.Marshal(snapshot.Request)
	req, err := http.NewRequest("POST", strings.TrimRight(os.Getenv("CLUE_DISCOVERY_URL"), "/")+"/analyze", bytes.NewReader(raw))
	if err != nil {
		fail()
		return nil
	}
	req.Header.Set("Content-Type", "application/json")
	response, err := (&http.Client{Timeout: 30 * time.Second}).Do(req)
	if err != nil {
		fail()
		return nil
	}
	defer response.Body.Close()
	if response.StatusCode != 200 {
		fail()
		return nil
	}
	var result DiscoveryResult
	if err = json.NewDecoder(io.LimitReader(response.Body, 2<<20)).Decode(&result); err != nil {
		fail()
		return nil
	}
	current, err := discoverySnapshotFor(job.ID)
	if err != nil {
		fail()
		return nil
	}
	if current.Hash != job.Hash {
		return QueueClueDiscovery(job.ID, false)
	}
	err = config.DB.Transaction(func(tx *gorm.DB) error {
		var locked models.ClueDiscoveryState
		if err := tx.Clauses(clause.Locking{Strength: "UPDATE"}).First(&locked, "living_clue_id = ?", job.ID).Error; err != nil {
			return err
		}
		if locked.ContentHash != job.Hash {
			return nil
		}
		catalog := map[uint]models.LivingClue{}
		for _, c := range snapshot.Catalog {
			catalog[c.ID] = c
		}
		suggestions := map[uint]models.ClueSuggestion{}
		references := []uint{}
		for _, ref := range result.References {
			if ref.TargetID == job.ID {
				continue
			}
			if _, ok := catalog[ref.TargetID]; !ok {
				continue
			}
			if len([]rune(ref.Mention)) > 255 || len([]rune(ref.Evidence)) > 500 {
				continue
			}
			references = append(references, ref.TargetID)
			suggestions[ref.TargetID] = models.ClueSuggestion{SourceClueID: job.ID, TargetClueID: ref.TargetID, SourceHash: job.Hash, Kind: "reference", Score: 100 + min(len([]rune(ref.Mention)), 50), Reason: "Referência explícita a “" + ref.Mention + "”. Confirme se é a mesma entidade.", Evidence: ref.Evidence}
		}
		// Graph expansion requires additional textual overlap; an institution's
		// neighbours alone never imply authorship, membership or study history.
		if len(references) > 0 {
			var edges []models.ClueConnection
			if err := tx.Where("source_clue_id IN ? OR target_clue_id IN ?", references, references).Find(&edges).Error; err != nil {
				return err
			}
			for _, edge := range edges {
				for _, target := range []uint{edge.SourceClueID, edge.TargetClueID} {
					candidate, ok := catalog[target]
					if !ok || target == job.ID {
						continue
					}
					if _, exists := suggestions[target]; exists {
						continue
					}
					terms := sharedDiscoveryTerms(snapshot.Source, candidate, snapshot.Catalog, references)
					if len(terms) < 2 {
						continue
					}
					suggestions[target] = models.ClueSuggestion{SourceClueID: job.ID, TargetClueID: target, SourceHash: job.Hash, Kind: "network", Score: 40 + len(terms), Reason: "Ligação na rede com uma referência encontrada e termos em comum: " + strings.Join(terms, ", ") + ". Esta aproximação precisa ser revisada.", Evidence: strings.Join(terms, ", ")}
				}
			}
		}
		// Preserve a dismissal only for the same source/catalogue content version.
		for _, suggestion := range suggestions {
			var previous models.ClueSuggestion
			findErr := tx.Where("source_clue_id = ? AND target_clue_id = ?", job.ID, suggestion.TargetClueID).First(&previous).Error
			if findErr != nil && !errors.Is(findErr, gorm.ErrRecordNotFound) {
				return findErr
			}
			if previous.SourceHash == job.Hash {
				suggestion.Dismissed = previous.Dismissed
			}
			if err := tx.Clauses(clause.OnConflict{Columns: []clause.Column{{Name: "source_clue_id"}, {Name: "target_clue_id"}}, DoUpdates: clause.AssignmentColumns([]string{"source_hash", "kind", "score", "reason", "evidence", "dismissed"})}).Create(&suggestion).Error; err != nil {
				return err
			}
		}
		// Only self-defining pairs may become pending aliases, never auto-approved.
		for _, candidate := range result.AliasCandidates {
			names := []string{candidate.Name}
			if ValidateClueAliases(&names) != nil {
				continue
			}
			key := NormalizeClueName(candidate.Name)
			if key == NormalizeClueName(snapshot.Source.Name) {
				continue
			}
			alias := models.ClueAlias{LivingClueID: job.ID, Name: strings.TrimSpace(candidate.Name), NormalizedName: key, Status: "pending", Origin: "explicit_pair", Evidence: string([]rune(candidate.Evidence)[:min(len([]rune(candidate.Evidence)), 300)])}
			if err := tx.Clauses(clause.OnConflict{DoNothing: true}).Create(&alias).Error; err != nil {
				return err
			}
		}
		data, _ := json.Marshal(result.Unresolved)
		var unknown []interface{}
		json.Unmarshal(data, &unknown)
		return tx.Model(&locked).Updates(map[string]interface{}{"status": "ready", "unresolved": models.JSONMap{"entities": unknown}, "updated_at": time.Now()}).Error
	})
	if err != nil {
		fail()
	}
	return nil
}
func sharedDiscoveryTerms(source, target models.LivingClue, catalog []models.LivingClue, refs []uint) []string {
	stop := map[string]bool{}
	for _, s := range strings.Fields("sobre entre para pelos pelas como uma este esta estes estas esse essa suas seus mais muito pesquisa projeto projetos pessoa pessoas universidade instituicao educacao nome descricao plataforma unisinos sinos digital") {
		stop[s] = true
	}
	for _, c := range catalog {
		for _, id := range refs {
			if c.ID == id {
				for _, n := range append([]string{c.Name}, aliasStrings(c.Aliases)...) {
					for _, w := range strings.Fields(NormalizeClueName(n)) {
						stop[w] = true
					}
				}
			}
		}
	}
	words := map[string]bool{}
	for _, w := range strings.Fields(NormalizeClueName(source.Name + " " + source.Description)) {
		if len([]rune(w)) >= 5 && !stop[w] {
			words[w] = true
		}
	}
	common := map[string]bool{}
	for _, w := range strings.Fields(NormalizeClueName(target.Name + " " + target.Description)) {
		if words[w] {
			common[w] = true
		}
	}
	terms := []string{}
	for w := range common {
		terms = append(terms, w)
	}
	sort.Strings(terms)
	if len(terms) > 5 {
		terms = terms[:5]
	}
	return terms
}
func aliasStrings(aliases []models.ClueAlias) []string {
	names := []string{}
	for _, a := range aliases {
		names = append(names, a.Name)
	}
	return names
}
