package auth

import (
	"chat/globals"
	"chat/utils"
	"database/sql"
	"fmt"
	"github.com/gin-gonic/gin"
	"github.com/spf13/viper"
	"net/http"
	"strings"
)

func MgccLoginAPI(c *gin.Context) {
	// 换token
	var form DeepLoginForm
	c.ShouldBind(&form)
	if form.Token == "" {
		LoginAPI(c)
		return
	}
	tokenRep, _ := utils.Get(fmt.Sprintf("%s/oauth/token?client_id=%s&client_secret=%s&code=%s",
		viper.GetString("mgcc.idphost"), viper.GetString("mgcc.idpid"), viper.GetString("mgcc.idpsecret"), form.Token), map[string]string{
		"Content-Type": "application/json",
	})
	tokenMap := tokenRep.(map[string]interface{})
	accessToken := tokenMap["access_token"].(string)
	userRep, _ := utils.Get(fmt.Sprintf("%s/userinfo?access_token=%s", viper.GetString("mgcc.idphost"), accessToken), map[string]string{
		"Content-Type": "application/json",
	})
	userRepMap := userRep.(map[string]interface{})
	if userRepMap["code"].(float64) != 200 {
		c.JSON(http.StatusOK, gin.H{
			"status": false,
			"error":  "user not found",
		})
		return
	}
	detailMap := userRepMap["data"].(map[string]interface{})["detail"].(map[string]interface{})
	email := detailMap["email"].(string)
	db := utils.GetDBFromContext(c)
	username := strings.Split(email, "@")[0]
	user := GetUserPwdByEmail(db, email)
	if user == nil {
		// 创建用户
		password := utils.Sha2Encrypt(utils.GenerateChar(64))
		_, err := globals.QueryDb(db, "INSERT INTO auth (username,email, token, password) VALUES (?, ?, ?, ?)",
			username, email, utils.Extract(accessToken, 255, ""), password)
		if err != nil {
			c.JSON(http.StatusOK, gin.H{
				"status": false,
				"error":  "register failed",
			})
			return
		}

		user = &User{
			Username: username,
			Password: password,
		}
		user.CreateInitialQuota(db)
		user.SetQuota(db, 2000) //初始给2000点
	}
	utk, _ := user.GenerateToken()
	c.JSON(http.StatusOK, gin.H{
		"status": true,
		"token":  utk,
	})
}

func GetUserPwdByEmail(db *sql.DB, email string) *User {
	var user User
	if err := globals.QueryRowDb(db, "SELECT id, username, password FROM auth WHERE email = ?", email).Scan(&user.ID, &user.Username, &user.Password); err != nil {
		return nil
	}
	return &user
}
