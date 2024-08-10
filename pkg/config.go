package pkg

import (
	"bytes"
	"chatgpt-adapter/logger"
	"crypto/cipher"
	"encoding/binary"
	"encoding/hex"
	"encoding/json"
	"github.com/spf13/viper"
	"os"

	"crypto/aes"
)

var (
	Config *viper.Viper

	keys = []string{
		"white-addr",
		"you.cookies",
		"claude.cookies",
		"coze.websdk.accounts",
		"llm.token",
	}
)

func InitConfig() {
	//time.Sleep(3 * time.Second)
	config, err := LoadConfig()
	if err != nil {
		panic(err)
	}
	Config = config
}

func LoadConfig() (*viper.Viper, error) {
	data, err := os.ReadFile("config.yaml")
	if err != nil {
		return nil, err
	}

	vip := viper.New()
	vip.SetConfigType("yaml")
	if err = vip.ReadConfig(bytes.NewReader(data)); err != nil {
		return nil, err
	}

	c := os.Getenv("CIPHER")
	if c == "" {
		return vip, nil
	}

	newCipher, err := aes.NewCipher([]byte(c))
	if err != nil {
		return nil, err
	}

	for _, key := range keys {
		content := vip.GetString(key)
		if content != "" {
			var d any
			d, err = decrypt(newCipher, content)
			if err != nil {
				logger.Infof("[%s]解码失败", key)
				d = content
			}
			vip.Set(key, d)
		}
	}

	return vip, nil
}

func decrypt(block cipher.Block, content string) (data any, err error) {
	db, err := hex.DecodeString(content)
	if err != nil {
		return
	}

	bToI := func(b []byte) int {
		buffer := bytes.NewBuffer(b)
		var x int32
		_ = binary.Read(buffer, binary.BigEndian, &x)
		return int(x)
	}

	iv := db[:aes.BlockSize]
	contentL := bToI(db[len(db)-4:])
	ctx := db[aes.BlockSize:]
	cipher.NewCBCDecrypter(block, iv).CryptBlocks(ctx, ctx[:len(ctx)-4])
	ctx = ctx[:contentL]

	if json.Unmarshal(ctx, &data) != nil {
		return string(ctx), nil
	}
	return
}
