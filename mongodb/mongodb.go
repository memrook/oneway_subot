package mongodb

import (
	"context"
	"github.com/mymmrac/telego"
	"go.mongodb.org/mongo-driver/bson"
	"go.mongodb.org/mongo-driver/bson/primitive"
	"go.mongodb.org/mongo-driver/mongo"
	"go.mongodb.org/mongo-driver/mongo/options"
	"gopkg.in/gookit/color.v1"
	"log"
	"time"
)

var users, chats *mongo.Collection
var ctx = context.TODO()

// User mongo structure with array of Chat
type User struct {
	ID           primitive.ObjectID   `bson:"_id"`
	UserID       int64                `bson:"user_id"`
	FirstName    string               `bson:"first_name"`
	LastName     string               `bson:"last_name"`
	Username     string               `bson:"username"`
	LanguageCode string               `bson:"language_code"`
	IsPremium    bool                 `bson:"is_premium"`
	ChatsID      []primitive.ObjectID `bson:"chats"`
	CreatedAt    time.Time            `bson:"created_at"`
	UpdatedAt    time.Time            `bson:"updated_at"`
}

type Message struct {
	Text      string    `bson:"text"`
	CreatedAt time.Time `bson:"created_at"`
	UpdatedAt time.Time `bson:"updated_at"`
}

// Chat mongo structure
type Chat struct {
	ID        primitive.ObjectID `bson:"_id"`
	PostID    int                `bson:"post_id"`
	ChatID    int                `bson:"chat_id"`
	UserID    int64              `bson:"user_id"`
	FirstName string             `bson:"first_name"`
	LastName  string             `bson:"last_name"`
	Username  string             `bson:"username"`
	Type      string             `bson:"type"`
	IsActive  bool               `bson:"isActive"`
	Messages  []Message          `bson:"messages"`
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

//func IsFirstMessage(id int64) bool {
//	var dbUser *User
//	filter := bson.D{{"user_id", id}}
//	opts := options.FindOne().SetProjection(bson.D{{"user_id", 1}, {"chats", 1}})
//	err := users.FindOne(ctx, filter, opts).Decode(&dbUser)
//	if err != nil {
//		log.Println("failed find a user: ", id)
//	}
//	log.Print(dbUser.ChatsID)
//	return len(dbUser.ChatsID) == 0
//}

func FindChatID(id int64) int {
	var chat Chat
	filter := bson.D{{"user_id", id}, {"isActive", true}}
	opts := options.FindOne().SetProjection(bson.D{{"chat_id", 1}})
	err := chats.FindOne(ctx, filter, opts).Decode(&chat)
	if err != nil {
		if err == mongo.ErrNoDocuments {
			// This error means your query did not match any documents.
			return 0
		}
		color.Red.Printf("failed db query chats.FindOne for userID: ", id)
	}
	color.LightGreen.Println("ChatID is ", chat.ChatID)
	return chat.ChatID
}

func UpdateChatID(postID int, chatID int) (bool, error) {
	//opts := options.Update().SetUpsert(true)
	//filter := bson.D{{"post_id", postID}}
	//update := bson.D{{"$set", bson.D{{"chat_id", chatID}}}}
	//res, err := chats.UpdateOne(context.TODO(), filter, update, opts)
	res, err := chats.UpdateOne(ctx, bson.M{"post_id": postID}, bson.M{"$set": bson.M{"chat_id": chatID}})
	if err != nil {
		if err == mongo.ErrNoDocuments {
			// This error means your query did not match any documents.
			return false, err
		}
		return false, err
	}
	color.Cyan.Println(res)
	return true, nil
}

func AddUser(user *telego.User) (*User, error) {
	chatsArray := make([]primitive.ObjectID, 0)
	u := &User{
		ID:           primitive.NewObjectID(),
		UserID:       user.ID,
		FirstName:    user.FirstName,
		LastName:     user.LastName,
		Username:     user.Username,
		LanguageCode: user.LanguageCode,
		IsPremium:    user.IsPremium,
		ChatsID:      chatsArray,
		CreatedAt:    time.Now(),
		UpdatedAt:    time.Now(),
	}
	if _, err := users.InsertOne(ctx, u); err != nil {
		return nil, err
	}
	return u, nil
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
		return nil, err
	}
	return User, nil
}

func NewChat(userID int64, messID int, chat *telego.Chat) error {
	messages := make([]Message, 0)
	ch := &Chat{
		ID:        primitive.NewObjectID(),
		PostID:    messID,
		ChatID:    0,
		UserID:    userID,
		FirstName: chat.FirstName,
		LastName:  chat.LastName,
		Username:  chat.Username,
		Type:      chat.Type,
		IsActive:  true,
		Messages:  messages,
		CreatedAt: time.Now(),
		UpdatedAt: time.Now(),
	}
	res, err := chats.InsertOne(ctx, ch)
	log.Println(res)
	if err != nil {
		log.Printf("failed to insert chat ID %v due to err:%s", chat.ID, err)
		return err
	}

	res2, err := users.UpdateOne(ctx, bson.M{"user_id": userID}, bson.M{"$addToSet": bson.M{"chats": ch.ID}})
	log.Println("res2: ", res2)

	return nil
}
