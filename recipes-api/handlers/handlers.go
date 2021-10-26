package handlers

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"time"

	"github.com/disharjayanth/recepis-api-gin/models"
	"github.com/gin-gonic/gin"
	"github.com/go-redis/redis/v8"
	"go.mongodb.org/mongo-driver/bson"
	"go.mongodb.org/mongo-driver/bson/primitive"
	"go.mongodb.org/mongo-driver/mongo"
)

type RecipeHandler struct {
	collection  *mongo.Collection
	ctx         context.Context
	redisClient *redis.Client
}

func NewRecipesHandler(ctx context.Context, collection *mongo.Collection, redisClient *redis.Client) *RecipeHandler {
	return &RecipeHandler{
		collection:  collection,
		ctx:         ctx,
		redisClient: redisClient,
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
	val, err := handler.redisClient.Get(handler.ctx, "recipes").Result()
	if err == redis.Nil {
		fmt.Println("Request forwarded to mongoDB since key isnt present in redis")
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

		sliceOfJSON, err := json.Marshal(recipes)
		if err != nil {
			fmt.Println("Error while marshalling list of recipes to array of JSON in ListRecipeHandler:", err)
			return
		}

		handler.redisClient.Set(handler.ctx, "recipes", string(sliceOfJSON), 0)
		c.JSON(http.StatusOK, recipes)
	} else if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{
			"error": err.Error(),
		})
	} else {
		fmt.Println("Response coming from Redis")
		recipes := make([]models.Recipe, 0)
		json.Unmarshal([]byte(val), &recipes)
		c.JSON(http.StatusOK, recipes)
	}
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

	fmt.Println("Delete `recipes` key from redis since new recipe is added to list and it's outdated")
	handler.redisClient.Del(handler.ctx, "recipes")

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
func (handler *RecipeHandler) UpdateRecipeHandler(c *gin.Context) {
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

	_, err = handler.collection.UpdateOne(handler.ctx, bson.M{"_id": objectID}, bson.D{{"$set", bson.D{
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

	fmt.Println("Delete `recipes` key from redis since a recipe has been updated to list and it's outdated")
	handler.redisClient.Del(handler.ctx, "recipes")

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

	fmt.Println("Delete `recipes` key from redis since a recipe has been deleted from list and it's outdated")
	handler.redisClient.Del(handler.ctx, "recipes")

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
