package goauth

import (
	"fmt"
	"net/http"

	"github.com/go-oauth2/oauth2/v4/errors"
	"github.com/kdjuwidja/aishoppercommon/logger"
	"golang.org/x/crypto/bcrypt"
	"gorm.io/gorm"
	dbmodel "netherealmstudio.com/m/v2/db"
)

type GoAuthHandler struct {
	dbConn *gorm.DB
}

func (h *GoAuthHandler) validateUser(email, password string) (string, error) {
	var user dbmodel.User
	result := h.dbConn.Where("email = ?", email).First(&user)
	if result.Error != nil {
		return "000000", result.Error
	}

	if result.RowsAffected == 0 {
		return "000000", errors.New("user not found")
	}

	err := bcrypt.CompareHashAndPassword([]byte(user.Password), []byte(password))
	if err != nil {
		return "000000", fmt.Errorf("invalid password")
	}

	logger.Debugf("ValidateUser %v successfully\n", user)
	return user.ID, nil
}

func (h *GoAuthHandler) userAuthorizationHandler(w http.ResponseWriter, r *http.Request) (userID string, err error) {
	email := r.PostFormValue("email")
	password := r.PostFormValue("password")
	logger.Tracef("email: %s, password: %s", email, password)

	//TODO: request scope from client, then verify scope with db
	userID, err = h.validateUser(email, password)
	if err != nil {
		return "", err
	}

	return userID, nil
}

func (h *GoAuthHandler) setInternalErrorHandler(err error) (re *errors.Response) {
	logger.Errorf("Internal Error: %s", err.Error())
	return
}

func (h *GoAuthHandler) setResponseErrorHandler(re *errors.Response) {
	logger.Errorf("Response Error: %s", re.Error.Error())
}
