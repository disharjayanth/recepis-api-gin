// Recipes API
//
// This is a sample recipes API. You can find out more about this API at
// https://github.com/disharjayanth/recepis-api-gin
//
// Schemes: http
// Host: localhost:3000
// BasePath: /
// Version: 1.0.0
// Contact: Dishar Jayantha<dishuj15@gmail.com> https://www.jayantha.in
//
// Consumes:
// - application/json
//
// Produces:
// - application/json
// swagger:meta
package main

import (
	"context"
	"fmt"
	"log"
	"os"
	"time"

	"github.com/disharjayanth/recepis-api-gin/handlers"
	"github.com/gin-gonic/gin"
	"go.mongodb.org/mongo-driver/bson/primitive"
	"go.mongodb.org/mongo-driver/mongo"
	"go.mongodb.org/mongo-driver/mongo/options"
	"go.mongodb.org/mongo-driver/mongo/readpref"
)

type Recipe struct {
	ID           primitive.ObjectID `json:"id" bson:"_id"`
	Name         string             `json:"name" bson:"name"`
	Tags         []string           `json:"tags" bson:"tags`
	Ingredients  []string           `json:"ingredients" bson:"ingredients`
	Instructions []string           `json:"instructions" bson:"instructions"`
	PublishedAt  time.Time          `json:"publishedAt" bson:"publishedAt`
}

var err error

var client *mongo.Client
var collection *mongo.Collection

var recipesHandler *handlers.RecipeHandler

func init() {
	ctx := context.Background()
	client, err = mongo.Connect(ctx, options.Client().ApplyURI(os.Getenv("MONGO_URI")))
	if err != nil {
		fmt.Println("Error connecting to mongodb server:", err)
		return
	}

	if err = client.Ping(context.TODO(), readpref.Primary()); err != nil {
		log.Fatal(err)
		return
	}

	fmt.Println("Connected to mongodb..")

	collection = client.Database(os.Getenv("MONGO_DATABASE")).Collection("recipes")
	recipesHandler = handlers.NewRecipesHandler(ctx, collection)
}

func main() {
	router := gin.Default()

	router.POST("/recipes", recipesHandler.NewRecipeHandler)
	router.GET("/recipes", recipesHandler.ListRecipesHandler)
	router.PUT("/recipes/:id", recipesHandler.UpdateRecipeHandler)
	router.DELETE("/recipes/:id", recipesHandler.DeleteRecipeHandler)
	router.GET("/recipes/search", recipesHandler.SearchRecipesHandler)

	router.Run(":3000")
}
