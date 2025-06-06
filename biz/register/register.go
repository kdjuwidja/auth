package bizregister

import (
	"context"
	"fmt"

	"gorm.io/gorm"
	"netherealmstudio.com/m/v2/db"
)

type RegistrationManager struct {
	maxRetry int
	dbConn   *gorm.DB
}

func NewRegistrationManager(dbConn *gorm.DB, maxRetry int) *RegistrationManager {
	return &RegistrationManager{
		dbConn:   dbConn,
		maxRetry: maxRetry,
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
	err := tx.Delete(&db.RegistrationCode{}, "code = ?", code).Unscoped().Error
	if err != nil {
		tx.Rollback()
		return err
	}

	user := db.User{
		Email:    email,
		Password: password,
	}

	err = tx.Create(&user).Error
	if err != nil {
		tx.Rollback()
		return err
	}

	tx.Commit()
	return nil
}
