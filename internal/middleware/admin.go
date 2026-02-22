package middleware

import "supervisor/internal/repository"

func IsGroupAdmin(repo *repository.Repository, groupID uint, tgUserID int64) bool {
	ok, err := repo.CheckAdmin(groupID, tgUserID)
	if err != nil {
		return false
	}
	return ok
}
