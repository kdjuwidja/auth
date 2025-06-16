package bizregister

import (
	"context"
	"fmt"
	"strings"

	"github.com/google/uuid"
	"github.com/kdjuwidja/aishoppercommon/logger"
	"golang.org/x/crypto/bcrypt"
	"gorm.io/gorm"
	"netherealmstudio.com/m/v2/db"
)

type RegistrationManager struct {
	maxRetry   int
	dbConn     *gorm.DB
	userRoleID int
}

func NewRegistrationManager(dbConn *gorm.DB, maxRetry int, userRoleID int) *RegistrationManager {
	return &RegistrationManager{
		dbConn:     dbConn,
		maxRetry:   maxRetry,
		userRoleID: userRoleID,
	}
}

func (r *RegistrationManager) GetRegistrationCode(ctx context.Context) (string, error) {
	success := false
	retry := 0
	for !success && retry < r.maxRetry {
		code, err := generateRegistrationCode()
		if err != nil {
			retry++
			continue
		}

		dbCode := db.RegistrationCode{
			Code: code,
		}

		// Create fails if the code already exists
		err = r.dbConn.Create(&dbCode).Error
		if err != nil {
			retry++
			continue
		}

		success = true
		return code, nil
	}

	return "", fmt.Errorf("failed to generate registration code after %d retries", r.maxRetry)
}

// RegisterUser registers a new user with the given code(single use), email, and password
func (r *RegistrationManager) RegisterUser(ctx context.Context, code string, email string, password string) error {
	tx := r.dbConn.WithContext(ctx).Begin()

	// Ensure code exists before creating user. The code is consumed in the process of registration within a single transaction so that no locking is required.
	result := tx.Unscoped().Delete(&db.RegistrationCode{}, "code = ?", code)
	if result.Error != nil {
		tx.Rollback()
		return result.Error
	}

	if result.RowsAffected == 0 {
		tx.Rollback()
		return fmt.Errorf("registration code not found")
	}

	hashedPassword, err := bcrypt.GenerateFromPassword([]byte(password), bcrypt.DefaultCost)
	if err != nil {
		logger.Errorf("failed to generate hashed password: %v", err)
		tx.Rollback()
		return err
	}

	user := db.User{
		ID:       strings.ReplaceAll(uuid.New().String(), "-", ""),
		Email:    email,
		Password: string(hashedPassword),
	}

	err = tx.Create(&user).Error
	if err != nil {
		tx.Rollback()
		return err
	}

	userRole := db.UserRole{
		UserID: user.ID,
		RoleID: r.userRoleID,
	}

	err = tx.Create(&userRole).Error
	if err != nil {
		tx.Rollback()
		return err
	}

	tx.Commit()
	return nil
}
