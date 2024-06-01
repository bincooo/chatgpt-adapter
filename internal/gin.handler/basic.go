package handler

import (
	"fmt"
	"github.com/bincooo/chatgpt-adapter/internal/gin.handler/response"
	"github.com/bincooo/chatgpt-adapter/logger"
	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
	"net/http"
	"net/http/httputil"
	"os"
	"strconv"
	"strings"
	"github.com/joho/godotenv"
	"encoding/json"
	"go.mongodb.org/mongo-driver/bson"
	"go.mongodb.org/mongo-driver/bson/primitive"
	"go.mongodb.org/mongo-driver/mongo"
	"go.mongodb.org/mongo-driver/mongo/options"
	"log"
	"context"
)

type ModelConfig struct {
	Cookie     string `bson:"cookie"`
	Model      string `bson:"model"`
	Used       int    `bson:"used"`
    StartTime  int64  `bson:"start_time"`
	Lock       int    `bson:"lock"`
}

type OldConfig struct {
	ID     primitive.ObjectID       `bson:"_id"`
	Models map[string][]ModelConfig `bson:"models"`
}

type NewConfig struct {
	ID        primitive.ObjectID `bson:"_id"`
	ModelType string             `bson:"modelType"`
	Configs   []ModelConfig      `bson:"configs"`
}

type JSONConfig struct {
	Models map[string][]ModelConfig `json:"models"`
}

func readConfig() (JSONConfig, error) {
	var config JSONConfig
	configFile, err := os.ReadFile("config.json")
	if err != nil {
		return config, err
	}

	err = json.Unmarshal(configFile, &config)
	return config, err
}

func loadEnv() {
    if os.Getenv("HEROKU") == "" {
        err := godotenv.Load()
        if err != nil {
            log.Fatalf("Error loading .env file")
        }
    }
}

func connectToMongoDB() *mongo.Client {
    uri := os.Getenv("MONGODB_URI")
    client, err := mongo.Connect(context.TODO(), options.Client().ApplyURI(uri))
    if err != nil {
        log.Fatal(err)
    }
    return client
}

// func migrateData(client *mongo.Client) {
//     // Kiểm tra xem collection đã có dữ liệu chưa
//     collection := client.Database("coze").Collection("bot")
//     count, err := collection.CountDocuments(context.TODO(), bson.D{})
//     if err != nil {
//         log.Fatal(err)
//     }

//     // Nếu không có dữ liệu, tiến hành migration
//     if count == 0 {
//         var config Config // Đảm bảo bạn đã định nghĩa struct Config
//         configFile, err := os.ReadFile("config.json")
//         if err != nil {
//             log.Fatal(err)
//         }

//         err = json.Unmarshal(configFile, &config)
//         if err != nil {
//             log.Fatal(err)
//         }

//         // Chuyển đổi config thành BSON và insert vào MongoDB
//         _, err = collection.InsertOne(context.TODO(), config)
//         if err != nil {
//             log.Fatal(err)
//         }
//     }
// }

func migrateData(client *mongo.Client) error {
	collection := client.Database("coze").Collection("bot")
	ctx := context.TODO()

	// Read data from config.json
	jsonConfig, err := readConfig()
	if err != nil {
		return err
	}

	// Find all documents in the old structure
	cursor, err := collection.Find(ctx, bson.M{})
	if err != nil {
		return err
	}
	defer cursor.Close(ctx)

	var oldConfigs []OldConfig
	if err = cursor.All(ctx, &oldConfigs); err != nil {
		return err
	}

	if len(oldConfigs) == 0 {
		// No old configs found, insert new config directly from config.json
		for modelType, configs := range jsonConfig.Models {
			for _, config := range configs {
				newConfig := NewConfig{
					ID:        primitive.NewObjectID(),
					ModelType: modelType,
					Configs:   []ModelConfig{config}, // Chèn từng mảng config
				}
			
				_, err := collection.InsertOne(ctx, newConfig)
				if err != nil {
					return err
				}
			}
		}

		return nil
	}

	// // Convert old configs to new structure and insert them
	// for _, oldConfig := range oldConfigs {
	// 	for modelType, configs := range oldConfig.Models {
	// 		for _, config := range configs {
	// 			newConfig := NewConfig{
	// 				ID:        primitive.NewObjectID(),
	// 				ModelType: modelType,
	// 				Configs:   []ModelConfig{config}, // Chèn từng mảng config
	// 			}
			
	// 			_, err := collection.InsertOne(ctx, newConfig)
	// 			if err != nil {
	// 				return err
	// 			}
	// 		}
	// 	}

	// 	// Optionally, remove the old document
	// 	// _, err := collection.DeleteOne(ctx, bson.M{"_id": oldConfig.ID})
	// 	// if err != nil {
	// 	// 	return err
	// 	// }
	// }

	return nil
}

func Bind(port int, version, proxies string) {
	gin.SetMode(gin.ReleaseMode)
	route := gin.Default()

	route.Use(crosHandler)
	route.Use(panicHandler)
	route.Use(tokenHandler)
	route.Use(proxiesHandler(proxies))
	route.Use(func(ctx *gin.Context) {
		ctx.Set("port", port)
	})

	route.GET("/", welcome(version))
	route.POST("/v1/chat/completions", completions)
	route.POST("/v1/object/completions", completions)
	route.POST("/proxies/v1/chat/completions", completions)
	route.POST("v1/images/generations", generations)
	route.POST("v1/object/generations", generations)
	route.POST("proxies/v1/images/generations", generations)
	route.GET("/proxies/v1/models", models)
	route.GET("/v1/models", models)
	route.Static("/file/tmp/", "tmp")

	addr := ":" + strconv.Itoa(port)
	logger.Info(fmt.Sprintf("server start by http://0.0.0.0%s/v1", addr))
	loadEnv()
    client := connectToMongoDB()
    migrateData(client)
	if err := route.Run(addr); err != nil {
		logger.Error(err)
		os.Exit(1)
	}
}

func proxiesHandler(proxies string) gin.HandlerFunc {
	return func(ctx *gin.Context) {
		if proxies != "" {
			ctx.Set("proxies", proxies)
		}
	}
}

func tokenHandler(ctx *gin.Context) {
	token := ctx.Request.Header.Get("X-Api-Key")
	if token == "" {
		token = strings.TrimPrefix(ctx.Request.Header.Get("Authorization"), "Bearer ")
	}

	if token != "" {
		ctx.Set("token", token)
	}
}

func crosHandler(context *gin.Context) {
	method := context.Request.Method
	context.Writer.Header().Set("Access-Control-Allow-Origin", "*")
	context.Header("Access-Control-Allow-Origin", "*") // 设置允许访问所有域
	context.Header("Access-Control-Allow-Methods", "POST, GET, OPTIONS, PUT, DELETE,UPDATE")
	context.Header("Access-Control-Allow-Headers", "*")
	context.Header("Access-Control-Expose-Headers", "*")
	context.Header("Access-Control-Max-Age", "172800")
	context.Header("Access-Control-Allow-Credentials", "false")
	context.Set("content-type", "application/json")

	if method == "OPTIONS" {
		context.Status(http.StatusOK)
		return
	}

	uid := uuid.NewString()
	// 请求打印
	data, err := httputil.DumpRequest(context.Request, false)
	if err != nil {
		logger.Error(err)
	} else {
		logger.Infof("\n------ START REQUEST %s ---------\n%s", uid, data)
	}

	//处理请求
	context.Next()

	// 结束处理
	logger.Infof("\n------ END REQUEST %s ---------", uid)
}

func panicHandler(ctx *gin.Context) {
	defer func() {
		if r := recover(); r != nil {
			logger.Errorf("response error: %v", r)
			response.Error(ctx, -1, fmt.Sprintf("%v", r))
		}
	}()

	//处理请求
	ctx.Next()
}

func welcome(version string) gin.HandlerFunc {
	return func(context *gin.Context) {
		w := context.Writer
		str := strings.ReplaceAll(html, "VERSION", version)
		str = strings.ReplaceAll(str, "HOST", context.Request.Host)
		_, _ = w.WriteString(str)
	}
}

func models(ctx *gin.Context) {
	ctx.JSON(200, gin.H{
		"object": "list",
		"data":   GlobalExtension.Models(),
	})
}
