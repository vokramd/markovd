package auth

import "github.com/vokramd/markovd/internal/models"

type Provider interface {
	Authenticate(username, password string) (*models.User, error)
	CreateUser(username, password string) (*models.User, error)
}
