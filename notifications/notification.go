package notifications

import "github.com/OdairPianta/unisinos_usp_api/models"

type Notification interface {
	GetNotifiable() models.User
	GetTitle() string
	GetContent() (string, error)
	GetNotifiers() []Notifier
	Send() error
}
