package handlers

import (
	"context"
	"fmt"
	"net/http"
	"time"

	"github.com/disharjayanth/recepis-api-gin/models"
	"github.com/gin-gonic/gin"
	"go.mongodb.org/mongo-driver/bson"
	"go.mongodb.org/mongo-driver/bson/primitive"
	"go.mongodb.org/mongo-driver/mongo"
)

type RecipeHandler struct {
	collection *mongo.Collection
	ctx        context.Context
}

func NewRecipesHandler(ctx context.Context, collection *mongo.Collection) *RecipeHandler {
	return &RecipeHandler{
		collection: collection,
		ctx:        ctx,
	}
}

// swagger:operation GET /recipes recipes listRecipes
// Returns list of recipes
// ---
//  Produces:
// -application/json
// responses:
// 	'200':
// 		description: Successful operation
func (handler *RecipeHandler) ListRecipesHandler(c *gin.Context) {
	// Find method returns cursor to interate over collection of objects
	cur, err := handler.collection.Find(handler.ctx, bson.M{})
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{
			"error": err,
		})
		return
	}
	defer cur.Close(handler.ctx)

	recipes := make([]models.Recipe, 0)
	for cur.Next(handler.ctx) {
		var recipe models.Recipe
		cur.Decode(&recipe)
		recipes = append(recipes, recipe)
	}

	c.JSON(http.StatusOK, recipes)
}

func (handler *RecipeHandler) NewRecipeHandler(c *gin.Context) {
	var recipe models.Recipe
	if err := c.ShouldBindJSON(&recipe); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{
			"error": err.Error(),
		})
		return
	}

	recipe.ID = primitive.NewObjectID()
	recipe.PublishedAt = time.Now()
	_, err := handler.collection.InsertOne(handler.ctx, recipe)
	if err != nil {
		fmt.Println("Errror inserting new recipe into recipe collection:", err)
		c.JSON(http.StatusInternalServerError, gin.H{
			"error": "Error while inserting a new recipe",
		})
		return
	}

	c.JSON(http.StatusOK, recipe)
}

// swagger:operation PUT /recipes/{id} recipes updateRecipe
// Update an existing recipe
// ---
// parameters:
// - name: id
//   in: path
//   description: ID of the recipe
//   required: true
//   type: string
// produces:
// - application/json
// responses:
//     '200':
//         description: Successful operation
//     '400':
//         description: Invalid input
//     '404':
//         description: Invalid recipe ID
func (handlers *RecipeHandler) UpdateRecipeHandler(c *gin.Context) {
	id := c.Param("id")
	var recipe models.Recipe
	if err := c.ShouldBindJSON(&recipe); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{
			"error": err.Error(),
		})
		return
	}

	objectID, err := primitive.ObjectIDFromHex(id)
	if err != nil {
		fmt.Println("Error converting id to objectid:", err)
		return
	}

	_, err = handlers.collection.UpdateOne(handlers.ctx, bson.M{"_id": objectID}, bson.D{{"$set", bson.D{
		{"name", recipe.Name},
		{"instructions", recipe.Instructions},
		{"ingredients", recipe.Ingredients},
		{"tags", recipe.Tags},
	}}})

	if err != nil {
		fmt.Println("Error updating document in collections:", err)
		c.JSON(http.StatusInternalServerError, gin.H{
			"error": err.Error(),
		})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"message": "Recipe has been updated",
	})
}

func (handler *RecipeHandler) DeleteRecipeHandler(c *gin.Context) {
	id := c.Param("id")

	objectID, err := primitive.ObjectIDFromHex(id)
	if err != nil {
		fmt.Println("Cannot get id from query string for delete operation:", err)
		return
	}

	res, err := handler.collection.DeleteOne(handler.ctx, bson.M{"_id": objectID})
	if err != nil {
		fmt.Println("Error deleting document from collection with id:", id, err)
		c.JSON(http.StatusInternalServerError, gin.H{
			"message": "error while deleting",
		})
		return
	}

	if res.DeletedCount == 0 {
		c.JSON(http.StatusOK, gin.H{
			"message": "Recipe not present with given id for deletion " + id,
		})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"message": "recipe deleted with id " + id,
	})
}

func (handler *RecipeHandler) SearchRecipesHandler(c *gin.Context) {
	tag := c.Query("tag")
	listOfRecipes := make([]models.Recipe, 0)

	cur, err := handler.collection.Find(handler.ctx, bson.M{"tags": tag})
	if err != nil {
		fmt.Println("Error finding recipes with particular tag:", tag, err)
		return
	}

	for cur.Next(handler.ctx) {
		var recipe models.Recipe
		cur.Decode(&recipe)
		listOfRecipes = append(listOfRecipes, recipe)
	}

	if len(listOfRecipes) == 0 {
		c.JSON(http.StatusOK, gin.H{
			"message": "Recipes not present with tag " + tag,
		})
		return
	}

	c.JSON(http.StatusOK, listOfRecipes)
}
