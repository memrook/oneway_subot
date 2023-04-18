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
	ID        primitive.ObjectID `bson:"_id"`
	From      string             `bson:"from"`
	Text      string             `bson:"text"`
	CreatedAt time.Time          `bson:"created_at"`
	UpdatedAt time.Time          `bson:"updated_at"`
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
	Messages  []interface{}      `bson:"messages"`
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

func (ch Chat) FindChatIDByUserID(userID int64) int {
	filter := bson.D{{"user_id", userID}, {"isActive", true}}
	opts := options.FindOne().
		SetSort(bson.D{{"created_at", -1}}).
		SetProjection(bson.D{{"chat_id", 1}})
	err := chats.FindOne(ctx, filter, opts).Decode(&ch)
	if err != nil {
		if err == mongo.ErrNoDocuments {
			// This error means your query did not match any documents.
			return 0
		}
		color.Red.Printf("failed db query chats.FindOne for userID: ", userID)
	}
	color.LightGreen.Println("ChatID is ", ch.ChatID)
	return ch.ChatID
}

func UpdateChatID(postID int, chatID int) error {
	//opts := options.Update().SetUpsert(true)
	res, err := chats.UpdateOne(ctx,
		bson.M{"post_id": postID},
		bson.M{"$set": bson.M{"chat_id": chatID}})
	if err != nil {
		if err == mongo.ErrNoDocuments {
			// This error means your query did not match any documents.
			return err
		}
		return err
	}
	color.Cyan.Println(res)
	return nil
}

func (u User) New(user *telego.User) error {
	chatsArray := make([]primitive.ObjectID, 0)
	u.ID = primitive.NewObjectID()
	u.Username = user.Username
	u.UserID = user.ID
	u.FirstName = user.FirstName
	u.LastName = user.LastName
	u.Username = user.Username
	u.LanguageCode = user.LanguageCode
	u.IsPremium = user.IsPremium
	u.ChatsID = chatsArray
	u.CreatedAt = time.Now()
	u.UpdatedAt = time.Now()

	if _, err := users.InsertOne(ctx, u); err != nil {
		return err
	}
	return nil
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

func NewChat(messID int, message *telego.Message) error {
	//Close all active chats before creating a new one
	err := CloseAllChats(message.From.ID)
	if err != nil {
		log.Println("failed to close all chats due: ", err)
	}
	messages := make([]interface{}, 0)
	ch := &Chat{
		ID:        primitive.NewObjectID(),
		PostID:    messID,
		ChatID:    0,
		UserID:    message.From.ID,
		FirstName: message.From.FirstName,
		LastName:  message.From.LastName,
		Username:  message.From.Username,
		Type:      message.Chat.Type,
		IsActive:  true,
		Messages:  messages,
		CreatedAt: time.Now(),
		UpdatedAt: time.Now(),
	}

	res, err := chats.InsertOne(ctx, ch)
	log.Println(res)
	if err != nil {
		log.Printf("failed to insert chat ID %v due to err:%s", messID, err)
		return err
	}

	_, err = users.UpdateOne(ctx, bson.M{"user_id": message.From.ID}, bson.M{"$addToSet": bson.M{"chats": ch.ID}})
	if err != nil {
		color.Red.Println("failed to Update chatID: ", err)
	}
	return nil
}

func AddMessage(chatID int, message *telego.Message) error {
	m := Message{
		ID:        primitive.NewObjectID(),
		From:      message.From.Username,
		Text:      message.Text,
		CreatedAt: time.Now(),
		UpdatedAt: time.Now(),
	}
	res, err := chats.UpdateOne(ctx,
		bson.M{"chat_id": chatID},
		bson.M{"$addToSet": bson.M{"messages": m}})
	if err != nil {
		log.Printf("failed to insert new message %v due to err:%s", m.Text, err)
	}
	color.LightGreen.Println("Add new message: ", res)
	return nil
}

func GetUserByChatID(chatID int) (int64, error) {
	var user User
	filter := bson.M{"chat_id": chatID}
	opts := options.FindOne().SetProjection(bson.D{{"user_id", 1}})
	err := chats.FindOne(ctx, filter, opts).Decode(&user)
	if err != nil {
		if err == mongo.ErrNilDocument {
			// This error means your query did not match any documents.
			color.Red.Println("can't Find User by chat ID: ", err)
			return 0, err
		}
		color.Red.Println("failed to Find User by chat ID: ", err)
		return 0, err
	}
	userID := user.UserID
	color.LightGreen.Println("Found User by Chat ID: ", userID)
	return userID, nil
}

func CloseRequest(threadID int) error {
	res, err := chats.UpdateOne(ctx,
		bson.M{"chat_id": threadID},
		bson.M{"$set": bson.M{"isActive": false}})
	if err != nil {
		if err == mongo.ErrNoDocuments {
			// This error means your query did not match any documents.
			color.Red.Println("no one chat status found and updated: ", err)
			return err
		}
		color.Red.Println("failed to update chat status: ", err)
		return err
	}
	color.LightGreen.Println("Chat status updated to 'false' : ", res)
	return nil
}

func CloseAllChats(userID int64) error {
	//opts := options.Update().SetUpsert(true)
	res, err := chats.UpdateMany(ctx,
		bson.M{"user_id": userID},
		bson.M{"$set": bson.M{"isActive": false}})
	if err != nil {
		if err == mongo.ErrNoDocuments {
			// This error means your query did not match any documents.
			color.Red.Println("no one chat status found and updated: ", err)
			return err
		}
		color.Red.Println("failed to update chats status: ", err)
		return err
	}
	color.LightGreen.Println("Chats statuses updated to 'false' : ", res)
	return nil
}

func GetThreadIDbyUsername(username string) int {
	var chat Chat
	filter := bson.D{{"username", username}, {"isActive", true}}
	opts := options.FindOne().
		SetSort(bson.D{{"created_at", -1}}).
		SetProjection(bson.D{{"chat_id", 1}})
	err := chats.FindOne(ctx, filter, opts).Decode(&chat)
	if err != nil {
		if err == mongo.ErrNoDocuments {
			// This error means your query did not match any documents.
			return 0
		}
		color.Red.Printf("failed db query chats.FindOne for username: ", username, err)
	}
	color.LightGreen.Println("ThreadID is ", chat.ChatID)
	return chat.ChatID
}
