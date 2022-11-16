package mongodb

import (
	"context"
	"github.com/mymmrac/telego"
	"go.mongodb.org/mongo-driver/bson"
	"go.mongodb.org/mongo-driver/bson/primitive"
	"go.mongodb.org/mongo-driver/mongo"
	"go.mongodb.org/mongo-driver/mongo/options"
	"log"
	"time"
)

var users, chats *mongo.Collection
var ctx = context.TODO()

// Chat mongo structure
type Chat struct {
	ID        primitive.ObjectID `bson:"_id"`
	ChatID    int64              `bson:"chat_id"`
	UserID    primitive.ObjectID `bson:"user_id"`
	FirstName string             `bson:"first_name"`
	LastName  string             `bson:"last_name"`
	Username  string             `bson:"username"`
	Type      string             `bson:"type"`
	isActive  bool               `bson:"is_active"`
	CreatedAt time.Time          `bson:"created_at"`
	UpdatedAt time.Time          `bson:"updated_at"`
}

// User mongo structure with array of Chat
type User struct {
	ID            primitive.ObjectID   `bson:"_id"`
	UserID        int64                `bson:"user_id"`
	FirstName     string               `bson:"first_name"`
	LastName      string               `bson:"last_name"`
	Username      string               `bson:"username"`
	LanguageCode  string               `bson:"language_code"`
	IsPremium     bool                 `bson:"is_premium"`
	ChatsID       []primitive.ObjectID `bson:"chats"`
	isActiveChats bool                 `bson:"is_active_chats"`
	CreatedAt     time.Time            `bson:"created_at"`
	UpdatedAt     time.Time            `bson:"updated_at"`
}

func init() {
	clientOptions := options.Client().ApplyURI("mongodb://root:kfgecbr@localhost:27017/")
	client, err := mongo.Connect(ctx, clientOptions)
	if err != nil {
		log.Fatal(err)
	}

	err = client.Ping(ctx, nil)
	if err != nil {
		log.Fatal(err)
	}

	users = client.Database("onewaySuBot").Collection("users")
	chats = client.Database("onewaySuBot").Collection("chats")
}

func IsFirstMessage(user *User) bool {
	//opts := options.Find()

	return true
}

func AddUser(user *telego.User) error {
	u := &User{
		ID:            primitive.NewObjectID(),
		UserID:        user.ID,
		FirstName:     user.FirstName,
		LastName:      user.LastName,
		Username:      user.Username,
		LanguageCode:  user.LanguageCode,
		IsPremium:     user.IsPremium,
		ChatsID:       nil,
		isActiveChats: false,
		CreatedAt:     time.Now(),
		UpdatedAt:     time.Now(),
	}
	_, err := users.InsertOne(ctx, u)
	return err
}

func GetUser(id int64) (*User, error) {
	filter := bson.D{{"user_id", id}}
	var User *User
	//opts := options.FindOne().SetProjection(bson.D{{"user_id", 1}, {"first_name", 1}})
	err := users.FindOne(ctx, filter).Decode(&User)
	if err != nil {
		if err == mongo.ErrNoDocuments {
			// This error means your query did not match any documents.
			return nil, err
		}
		panic(err)
	}
	return User, nil
}
