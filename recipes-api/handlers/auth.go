package handlers

import (
	"context"
	"fmt"
	"net/http"
	"os"
	"time"

	"github.com/disharjayanth/recepis-api-gin/models"
	"github.com/gin-contrib/sessions"
	"github.com/gin-gonic/gin"
	"github.com/golang-jwt/jwt"
	"github.com/rs/xid"
	"go.mongodb.org/mongo-driver/bson"
	"go.mongodb.org/mongo-driver/mongo"
	"golang.org/x/crypto/bcrypt"
)

type AuthHandler struct {
	collection *mongo.Collection
	ctx        context.Context
}

type Claims struct {
	Username string `json:"username"`
	jwt.StandardClaims
}

type JWTOutput struct {
	Token   string    `json:"token"`
	Expires time.Time `json:"expires"`
}

func NewAuthHandler(ctx context.Context, collection *mongo.Collection) *AuthHandler {
	return &AuthHandler{
		collection: collection,
		ctx:        ctx,
	}
}

func (handler *AuthHandler) SignUpHandler(c *gin.Context) {
	var user models.User
	if err := c.ShouldBindJSON(&user); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{
			"error": err.Error(),
		})
		return
	}

	bytes, err := bcrypt.GenerateFromPassword([]byte(user.Password), 14)
	if err != nil {
		fmt.Println("Error generating hash for password:", err)
		return
	}

	_, err = handler.collection.InsertOne(handler.ctx, bson.M{
		"username": user.Username,
		"password": string(bytes),
	})
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{
			"error": err.Error(),
		})
		return
	}

	expirationTime := time.Now().Add(10 * time.Minute)
	claims := &Claims{
		Username: user.Username,
		StandardClaims: jwt.StandardClaims{
			ExpiresAt: expirationTime.Unix(),
		},
	}

	token := jwt.NewWithClaims(jwt.SigningMethodHS256, claims)
	tokenString, err := token.SignedString([]byte(os.Getenv("JWT_SECRET")))
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{
			"error": err.Error(),
		})
		return
	}

	jwtToken := JWTOutput{
		Token:   tokenString,
		Expires: expirationTime,
	}

	c.JSON(http.StatusOK, jwtToken)
}

// SignIn handler
func (handler *AuthHandler) SignInHandler(c *gin.Context) {
	var user models.User
	var getUser bson.M
	if err := c.ShouldBindJSON(&user); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{
			"error": err.Error(),
		})
		return
	}

	if err := handler.collection.FindOne(handler.ctx, bson.M{"username": user.Username}).Decode(&getUser); err != nil {
		c.JSON(http.StatusUnauthorized, gin.H{
			"error": "Invalid username or password " + err.Error(),
		})
		return
	}

	if err := bcrypt.CompareHashAndPassword([]byte(getUser["password"].(string)), []byte(user.Password)); err != nil {
		c.JSON(http.StatusUnauthorized, gin.H{
			"error": "Invalid username or password " + err.Error(),
		})
		return
	}

	sessionToken := xid.New().String()
	session := sessions.Default(c)
	session.Set("username", user.Username)
	session.Set("token", sessionToken)
	if err := session.Save(); err != nil {
		fmt.Println("Error saving cookie session in request:", err)
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"message": "User signed in",
	})
}

func (handler *AuthHandler) RefreshHandler(c *gin.Context) {
	session := sessions.Default(c)
	sessionUser := session.Get("username")
	sessionToken := session.Get("token")
	if sessionToken == nil {
		c.JSON(http.StatusUnauthorized, gin.H{
			"error": "Invalid session cookie",
		})
		return
	}

	sessionToken = xid.New().String()
	session.Set("username", sessionUser.(string))
	session.Set("token", sessionToken)
	if err := session.Save(); err != nil {
		fmt.Println("Error saving cookie session in request:", err)
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"message": "New session issued",
	})
}

// Auth Middleware
func (handler *AuthHandler) AuthMiddleware() gin.HandlerFunc {
	return func(c *gin.Context) {
		session := sessions.Default(c)
		sessionToken := session.Get("token")
		if sessionToken == nil {
			c.JSON(http.StatusForbidden, gin.H{
				"message": "Not logged in",
			})
			c.Abort()
		}
		c.Next()
	}
}

func (handler *AuthHandler) SignOutHandler(c *gin.Context) {
	session := sessions.Default(c)
	session.Clear()
	if err := session.Save(); err != nil {
		fmt.Println("Error saving cookie session in request:", err)
		return
	}
	c.JSON(http.StatusOK, gin.H{
		"message": "Signed out....",
	})
}
