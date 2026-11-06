package services

import (
	"errors"
	"strings"
	"sync"
	"time"
	"unicode"
	"unicode/utf8"

	"github.com/OdairPianta/unisinos_usp_api/config"
	"github.com/OdairPianta/unisinos_usp_api/models"
	"github.com/google/uuid"
	"gorm.io/gorm"
	"gorm.io/gorm/clause"
	"gorm.io/gorm/logger"
)

var ErrParticipantText = errors.New("Use de 1 a 2.000 caracteres de texto ou emojis.")
var ErrParticipantRetry = errors.New("Essa tentativa já foi utilizada para outra mensagem.")
var ErrParticipantRate = errors.New("Você enviou muitas mensagens. Aguarde um minuto para continuar.")
var ErrParticipantBlocked = errors.New("Esta conversa está bloqueada.")
var ErrParticipantDescription = errors.New("Descrição inválida; use até 2.000 caracteres.")

func ParticipantBlockStatus(c models.ParticipantConversation) (uint, error) {
	var block models.ParticipantConversationBlock
	err := config.DB.Where("conversation_id = ?", c.ID).First(&block).Error
	if errors.Is(err, gorm.ErrRecordNotFound) {
		return 0, nil
	}
	return block.UserID, err
}

func ReportParticipantConversation(c models.ParticipantConversation, user uint, description string) error {
	if user != c.UserA && user != c.UserB {
		return gorm.ErrRecordNotFound
	}
	description = strings.TrimSpace(description)
	if !utf8.ValidString(description) || utf8.RuneCountInString(description) > 2000 {
		return ErrParticipantDescription
	}
	for _, r := range description {
		if unicode.IsControl(r) && r != '\n' && r != '\t' {
			return ErrParticipantDescription
		}
	}
	var optional *string
	if description != "" {
		optional = &description
	}
	return config.DB.Create(&models.ParticipantConversationReport{UserID: user, ConversationID: c.ID, Description: optional}).Error
}

func BlockParticipantConversation(c models.ParticipantConversation, user uint) error {
	if user != c.UserA && user != c.UserB {
		return gorm.ErrRecordNotFound
	}
	return config.DB.Clauses(clause.OnConflict{DoNothing: true}).Create(&models.ParticipantConversationBlock{UserID: user, ConversationID: c.ID}).Error
}

func UnblockParticipantConversation(c models.ParticipantConversation, user uint) error {
	if user != c.UserA && user != c.UserB {
		return gorm.ErrRecordNotFound
	}
	return config.DB.Where("conversation_id = ? AND user_id = ?", c.ID, user).Delete(&models.ParticipantConversationBlock{}).Error
}

type ParticipantMessageView struct {
	ID             uint64    `json:"id"`
	ConversationID string    `json:"conversation_id"`
	Text           string    `json:"text"`
	Own            bool      `json:"own"`
	CreatedAt      time.Time `json:"created_at"`
}
type ParticipantMessagePage struct {
	Messages []ParticipantMessageView `json:"messages"`
	HasMore  bool                     `json:"has_more"`
}

func ParticipantMessageDTO(m models.ParticipantMessage, user uint) ParticipantMessageView {
	return ParticipantMessageView{m.ID, m.ConversationID, m.Text, m.SenderID == user, m.CreatedAt}
}
func FindParticipantConversation(id string, user uint) (models.ParticipantConversation, error) {
	var c models.ParticipantConversation
	if _, err := uuid.Parse(id); err != nil {
		return c, gorm.ErrRecordNotFound
	}
	err := config.DB.Where("id = ? AND (user_a = ? OR user_b = ?)", id, user, user).First(&c).Error
	return c, err
}
func OpenParticipantConversation(user, peer uint) (models.ParticipantConversation, error) {
	var c models.ParticipantConversation
	if user == peer || user == 0 || peer == 0 {
		return c, gorm.ErrRecordNotFound
	}
	var target models.User
	if err := config.DB.First(&target, peer).Error; err != nil {
		return c, err
	}
	a, b := user, peer
	if a > b {
		a, b = b, a
	}
	c = models.ParticipantConversation{ID: uuid.NewString(), UserA: a, UserB: b}
	if err := config.DB.Clauses(clause.OnConflict{DoNothing: true}).Create(&c).Error; err != nil {
		return c, err
	}
	c = models.ParticipantConversation{}
	err := config.DB.Where("user_a = ? AND user_b = ?", a, b).First(&c).Error
	return c, err
}
func ParticipantMessages(c models.ParticipantConversation, user uint, before uint64) (ParticipantMessagePage, error) {
	query := config.DB.Where("conversation_id = ?", c.ID)
	if before > 0 {
		query = query.Where("id < ?", before)
	}
	var rows []models.ParticipantMessage
	err := query.Order("id DESC").Limit(31).Find(&rows).Error
	page := ParticipantMessagePage{Messages: []ParticipantMessageView{}, HasMore: len(rows) > 30}
	if len(rows) > 30 {
		rows = rows[:30]
	}
	for i := len(rows) - 1; i >= 0; i-- {
		page.Messages = append(page.Messages, ParticipantMessageDTO(rows[i], user))
	}
	return page, err
}

type participantWindow struct {
	Start time.Time
	Count int
}

var participantRates = struct {
	sync.Mutex
	Users map[uint]participantWindow
}{Users: map[uint]participantWindow{}}

func participantCanSend(user uint) bool {
	participantRates.Lock()
	defer participantRates.Unlock()
	now := time.Now()
	for id, w := range participantRates.Users {
		if now.Sub(w.Start) >= time.Minute {
			delete(participantRates.Users, id)
		}
	}
	w := participantRates.Users[user]
	if w.Count >= 20 {
		return false
	}
	if w.Count == 0 {
		w.Start = now
	}
	w.Count++
	participantRates.Users[user] = w
	return true
}
func SendParticipantMessage(c models.ParticipantConversation, user uint, text, clientID string) (models.ParticipantMessage, bool, error) {
	var message models.ParticipantMessage
	if user != c.UserA && user != c.UserB {
		return message, false, gorm.ErrRecordNotFound
	}
	blockedBy, err := ParticipantBlockStatus(c)
	if err != nil {
		return message, false, err
	}
	if blockedBy != 0 {
		return message, false, ErrParticipantBlocked
	}
	text = strings.TrimSpace(text)
	if !utf8.ValidString(text) || utf8.RuneCountInString(text) < 1 || utf8.RuneCountInString(text) > 2000 {
		return message, false, ErrParticipantText
	}
	for _, r := range text {
		if unicode.IsControl(r) && r != '\n' && r != '\t' {
			return message, false, ErrParticipantText
		}
	}
	if _, err := uuid.Parse(clientID); err != nil || len(clientID) != 36 {
		return message, false, ErrParticipantRetry
	}
	existing := config.DB.Where("conversation_id = ? AND sender_id = ? AND client_id = ?", c.ID, user, clientID).First(&message)
	if existing.Error == nil {
		if message.Text != text {
			return message, false, ErrParticipantRetry
		}
		return message, false, nil
	}
	if !errors.Is(existing.Error, gorm.ErrRecordNotFound) {
		return message, false, existing.Error
	}
	peer := c.UserA
	if peer == user {
		peer = c.UserB
	}
	var target models.User
	if err := config.DB.First(&target, peer).Error; err != nil {
		return message, false, err
	}
	if !participantCanSend(user) {
		return message, false, ErrParticipantRate
	}
	message = models.ParticipantMessage{ConversationID: c.ID, SenderID: user, ClientID: clientID, Text: text}
	created := false
	err = config.DB.Session(&gorm.Session{Logger: logger.Default.LogMode(logger.Silent)}).Transaction(func(tx *gorm.DB) error {
		result := tx.Clauses(clause.OnConflict{DoNothing: true}).Create(&message)
		if result.Error != nil {
			return result.Error
		}
		created = result.RowsAffected > 0
		if !created {
			message = models.ParticipantMessage{}
			if err := tx.Where("conversation_id = ? AND sender_id = ? AND client_id = ?", c.ID, user, clientID).First(&message).Error; err != nil {
				return err
			}
			if message.Text != text {
				return ErrParticipantRetry
			}
			return nil
		}
		column := "read_a"
		if c.UserB == user {
			column = "read_b"
		}
		return tx.Model(&models.ParticipantConversation{}).Where("id = ?", c.ID).Updates(map[string]interface{}{"updated_at": message.CreatedAt, column: gorm.Expr("GREATEST("+column+", ?)", message.ID)}).Error
	})
	return message, created, err
}
func ReadParticipantConversation(c models.ParticipantConversation, user uint, through uint64) error {
	var latest models.ParticipantMessage
	if err := config.DB.Where("conversation_id = ? AND id <= ?", c.ID, through).Order("id DESC").First(&latest).Error; err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil
		}
		return err
	}
	column := "read_a"
	if c.UserB == user {
		column = "read_b"
	}
	return config.DB.Model(&models.ParticipantConversation{}).Where("id = ?", c.ID).UpdateColumn(column, gorm.Expr("GREATEST("+column+", ?)", latest.ID)).Error
}
