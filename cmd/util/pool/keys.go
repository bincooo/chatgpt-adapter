package pool

import (
	"context"
	"errors"
	"fmt"
	cmdvars "github.com/bincooo/AutoAI/cmd/vars"
	"github.com/bincooo/claude-api"
	"github.com/bincooo/claude-api/util"
	clvars "github.com/bincooo/claude-api/vars"
	"github.com/joho/godotenv"
	"github.com/sirupsen/logrus"
	"os"
	"regexp"
	"strings"
	"sync"
	"time"
)

const (
	MINIMUM_SURVIVAL = 3 // 最小存活数
)

var (
	keys      []*Key // session池
	currIndex int    = -1
	IsLocal          = false
	mu        sync.Mutex
)

type Key struct {
	Token string
	IsDie bool
	Error error
}

func init() {
	_ = godotenv.Load()
	k := strings.TrimSpace(LoadEnvVar("KEYS", ""))
	keys = make([]*Key, 0)
	if k != "" {
		split := strings.Split(k, ",")
		for _, key := range split {
			keys = append(keys, &Key{strings.TrimSpace(key), false, nil})
			IsLocal = true
		}
	}

	// 开启线程自检
	if cmdvars.EnablePool && !IsLocal {
		ctoken := LoadEnvVar("CACHE_KEY", "")
		if ctoken != "" {
			keys = append(keys, &Key{ctoken, false, nil})
		}
		go func() {
			time.Sleep(3 * time.Second)
			for {
				mu.Lock()
				if cmdvars.Gen {
					mu.Unlock()
					return
				}
				// 删除
				if len(keys) > 0 {
					for index, key := range keys {
						if key.IsDie {
							logrus.Warn("发现缓存池sessionKey已失效: " + key.Token)
							logrus.Warn("删除失效的缓存池sessionKey: ", key.Error)
							keys = append(keys[:index], keys[index+1:]...)
						}
					}
				}
				mu.Unlock()

				// 新增
				if len(keys) < MINIMUM_SURVIVAL {
					_, token, err := GenerateSessionKey()
					if err == nil {
						mu.Lock()
						logrus.Info("新增缓存池sessionKey: " + token)
						keys = append(keys, &Key{token, false, nil})
						mu.Unlock()
						CacheKey("CACHE_KEY", token)
					} else {
						logrus.Warn("自动获取新的缓存池sessionKey失败:", err)
					}
				}
				time.Sleep(5 * time.Second)
			}
		}()
	}
}

func GetKey() (string, error) {
	if IsLocal {
		return getLocalKey()
	} else {
		return getSmailKey()
	}
}

// 本地sessionKey池
func getLocalKey() (string, error) {
	mu.Lock()
	defer mu.Unlock()
	l := len(keys)
	if l == 0 {
		return "", errors.New("本地连接池sessionKey为空")
	}

	var err error
	for index := 0; index < l; index++ {
		currIndex++
		if currIndex >= l {
			currIndex = 0
		}
		key := keys[currIndex]
		if key.IsDie {
			err = key.Error
			continue
		}
		// 测试是否可用
		if err = TestMessage(key.Token); err != nil {
			key.IsDie = true
			key.Error = err
			continue
		} else {
			return key.Token, nil
		}
	}

	return "", errors.New("本地所有sessionKey均已失效：" + err.Error())
}

// 联网缓存smail获取到的sessionKey
func getSmailKey() (string, error) {
	l := len(keys)
	var err error

	mu.Lock()
	defer mu.Unlock()

	if currIndex > -1 {
		key := keys[currIndex]
		if err = TestMessage(key.Token); err != nil {
			key.IsDie = true
			key.Error = err
		} else {
			return key.Token, nil
		}
	}

	for index := 0; index < l; index++ {
		currIndex++
		if currIndex >= l {
			currIndex = 0
		}
		key := keys[currIndex]
		if key.IsDie {
			err = key.Error
			continue
		}
		// 测试是否可用
		if err = TestMessage(key.Token); err != nil {
			key.IsDie = true
			key.Error = err
			continue
		} else {
			return key.Token, nil
		}
	}

	// 缓存池中都失效了，尝试一下获取新的
	_, token, err := GenerateSessionKey()
	if err != nil {
		return token, errors.New("缓存池内所有sessionKey均已失效：" + err.Error())
	}
	keys = append(keys, &Key{token, false, nil})
	return token, err
}

func GenerateSessionKey() (email, token string, err error) {
	var cnt = 2 // 重试次数

label:
	email, token, err = util.LoginFor(cmdvars.Bu, cmdvars.Suffix, cmdvars.Proxy)
	if err != nil {
		logrus.Error(cmdvars.I18n("FAILED_GENERATE_SESSION_KEY")+"： email --- "+email, err)
		return email, token, err
	}

	err = TestMessage(token)
	if err != nil {
		if cnt > 0 {
			cnt--
			goto label
		}
	}
	return email, token, err
}

// 测试sessionKey是否可用
func TestMessage(token string) error {
	options := claude.NewDefaultOptions(token, "", clvars.Model4WebClaude2)
	options.Agency = cmdvars.Proxy
	chat, err := claude.New(options)
	if err != nil {
		return err
	}
	prompt := "I say ping, You say pong"
	partialResponse, err := chat.Reply(context.Background(), prompt, nil)
	if err != nil {
		return err
	}
	defer chat.Delete()
	for {
		message, ok := <-partialResponse
		if !ok {
			return nil
		}

		if message.Error != nil {
			return message.Error
		}
	}
}

// 缓存CACHE_KEY
func CacheKey(key, value string) {
	// 文件不存在...   就创建吧
	if _, err := os.Lstat(".env"); os.IsNotExist(err) {
		if _, e := os.Create(".env"); e != nil {
			fmt.Println("Error: ", e)
			return
		}
	}

	bytes, err := os.ReadFile(".env")
	if err != nil {
		fmt.Println("Error: ", err)
	}
	tmp := string(bytes)
	compileRegex := regexp.MustCompile(`(\n|^)` + key + `\s*=[^\n]*`)
	matchSlice := compileRegex.FindStringSubmatch(tmp)
	if len(matchSlice) > 0 {
		str := matchSlice[0]
		if strings.HasPrefix(str, "\n") {
			str = str[1:]
		}
		tmp = strings.Replace(tmp, str, key+"=\""+value+"\"", -1)
	} else {
		delimiter := ""
		if len(tmp) > 0 && !strings.HasSuffix(tmp, "\n") {
			delimiter = "\n"
		}
		tmp += delimiter + key + "=\"" + value + "\""
	}
	err = os.WriteFile(".env", []byte(tmp), 0664)
	if err != nil {
		fmt.Println("Error: ", err)
	}
}

func LoadEnvVar(key, defaultValue string) string {
	value, exists := os.LookupEnv(key)
	if !exists {
		value = defaultValue
	}
	return value
}
