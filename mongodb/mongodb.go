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

// Chat mongo structure
type Chat struct {
	ID        primitive.ObjectID `bson:"_id"`
	ChatID    int64              `bson:"chat_id"`
	UserID    int64              `bson:"user_id"`
	FirstName string             `bson:"first_name"`
	LastName  string             `bson:"last_name"`
	Username  string             `bson:"username"`
	Type      string             `bson:"type"`
	isActive  bool               `bson:"is_active"`
	CreatedAt time.Time          `bson:"created_at"`
	UpdatedAt time.Time          `bson:"updated_at"`
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

func IsFirstMessage(id int64) bool {
	var dbUser *User
	filter := bson.D{{"user_id", id}}
	opts := options.FindOne().SetProjection(bson.D{{"user_id", 1}, {"chats", 1}})
	err := users.FindOne(ctx, filter, opts).Decode(&dbUser)
	if err != nil {
		log.Println("failed find a user: ", id)
	}
	log.Print(dbUser.ChatsID)
	return len(dbUser.ChatsID) == 0
}

func AddUser(user *telego.User) error {
	chatsArray := make([]primitive.ObjectID, 0)
	u := &User{
		ID:            primitive.NewObjectID(),
		UserID:        user.ID,
		FirstName:     user.FirstName,
		LastName:      user.LastName,
		Username:      user.Username,
		LanguageCode:  user.LanguageCode,
		IsPremium:     user.IsPremium,
		ChatsID:       chatsArray,
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

func NewChat(id int64, chat *telego.Chat) error {
	ch := &Chat{
		ID:        primitive.NewObjectID(),
		ChatID:    chat.ID,
		UserID:    id,
		FirstName: chat.FirstName,
		LastName:  chat.LastName,
		Username:  chat.Username,
		Type:      chat.Type,
		isActive:  true,
		CreatedAt: time.Now(),
		UpdatedAt: time.Now(),
	}
	res, err := chats.InsertOne(ctx, ch)
	log.Println(res)
	if err != nil {
		log.Printf("failed to insert chat ID %v due to err:%s", chat.ID, err)
		return err
	}

	res2, err := users.UpdateOne(ctx, bson.M{"user_id": id}, bson.M{"$addToSet": bson.M{"chats": chat.ID}})
	log.Println(res2)

	return nil
}
