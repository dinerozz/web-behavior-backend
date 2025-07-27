package user

import (
	"github.com/dinerozz/web-behavior-backend/internal/model/request"
	"github.com/dinerozz/web-behavior-backend/internal/model/response"
	"github.com/dinerozz/web-behavior-backend/internal/repository"
	"github.com/gofrs/uuid"
)

type UserService struct {
	Repo *repository.UserRepository
}

func NewUserService(repo *repository.UserRepository) *UserService {
	return &UserService{Repo: repo}
}

func (s *UserService) CheckIfUserExistsByUsername(username string) bool {
	user, err := s.Repo.GetUserByUsername(username)
	if err != nil {
		return false
	}
	return user.ID != uuid.Nil
}

func (s *UserService) GetUserById(userID uuid.UUID) (response.User, error) {
	return s.Repo.GetUserById(userID)
}

func (s *UserService) GetUserByUsername(username string) (response.User, error) {
	return s.Repo.GetUserByUsername(username)
}

func (s *UserService) CreateUserWithPassword(user *request.CreateUserWithPassword) (response.User, error) {
	return s.Repo.CreateUserWithPassword(user)
}

func (s *UserService) GetAllUsers() ([]response.User, error) {
	return s.Repo.GetAllUsers()
}

// Deprecated
func (s *UserService) CreateOrAuthenticateUserWithPassword(user *request.CreateUserWithPassword) (response.User, error) {
	return s.Repo.CreateOrAuthenticateUserWithPassword(user)
}
