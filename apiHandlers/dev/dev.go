package apiHandlersdev

import (
	"net/http"

	"github.com/gin-gonic/gin"
	"golang.org/x/crypto/bcrypt"
)

type DevHandler struct {
}

func InitializeDevHandler() *DevHandler {
	return &DevHandler{}
}

func (h *DevHandler) GetBCryptHash(c *gin.Context) {
	password := c.Query("text")
	hash, err := bcrypt.GenerateFromPassword([]byte(password), bcrypt.DefaultCost)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}
	c.JSON(http.StatusOK, gin.H{"hash": string(hash)})
}
