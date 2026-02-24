package middleware

import "supervisor/internal/repository"

// IsGroupAdmin 检查用户是否为群组管理员。
func IsGroupAdmin(repo *repository.Repository, groupID uint, tgUserID int64) bool {
	ok, err := repo.CheckAdmin(groupID, tgUserID)
	if err != nil {
		return false
	}
	return ok
}
