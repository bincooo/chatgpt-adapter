package emulate

import (
	"bypass/jinja"
	"bytes"
	"crypto/sha3"
	"encoding/base64"
	"encoding/hex"
	"encoding/json"
	"errors"
	"math"
	"math/rand"
	"net/http"
	"net/url"
	"strconv"
	"time"

	"github.com/PuerkitoBio/goquery"
	"github.com/google/uuid"
	"github.com/xllm-go/g/model"
)

var (
	domain, _ = url.Parse("https://chatgpt.com")

	emulator = newEmulator()
)

type Emulator struct {
	userAgent string

	proof string

	scripts []string

	cachedId, cachedSid string

	cachedCore, cachedHardware int

	startTime time.Time
}

func newEmulator() *Emulator {
	cores := []int{8, 12, 16, 24}
	screens := []int{3000, 4000, 6000}
	return &Emulator{
		userAgent:      "Mozilla/5.0 (X11; Linux x86_64) AppleWebKit/537.36 (KHTML, like Gecko) Chrome/133.0.0.0 Safari/537.36",
		cachedCore:     cores[rand.Intn(4)],
		cachedHardware: screens[rand.Intn(3)],
		startTime:      time.Now(),
	}
}

func GetEmulator() *Emulator {
	return emulator
}

func (emulator *Emulator) Equi() string {
	return emulator.userAgent
}

type chatRequire struct {
	Token string `json:"token"`
	Proof struct {
		Difficulty string `json:"difficulty,omitempty"`
		Required   bool   `json:"required"`
		Seed       string `json:"seed,omitempty"`
	} `json:"proofofwork,omitempty"`
	Turnstile struct {
		Required bool   `json:"required"`
		DX       string `json:"dx,omitempty"`
	} `json:"turnstile"`
	ForceLogin bool `json:"force_login,omitempty"`
}

func (require chatRequire) calcProofToken() string {
	return "gAAAAAB" + generateAnswer(require.Proof.Seed, require.Proof.Difficulty)
}

func ConvertAPIRequest(completion *model.Completion) (request *RequestStructure, err error) {
	require, err := checkRequire()
	if err != nil {
		return
	}

	request = newChatGPTRequest()
	request.Model = "gpt-5.2"
	request.Token = require.Token
	if require.Proof.Required {
		request.ProofToken = require.calcProofToken()
	}

	system := ""
	if len(completion.Messages) > 0 && completion.Messages[0].ValueEqual("role", "system") {
		message := completion.Messages[0]
		system = model.JustValue[string, string](message, "content")
	}
	system, err = model.ContextJinjaMessage(jinja.ChatGPTSystemMessage, map[string]interface{}{
		"system": system,
		"tools":  completion.Tools,
	})
	if err != nil {
		return
	}

	request.addMessage("critic", system)
	for i, message := range completion.Messages {
		if i == 0 {
			continue
		}

		var content string
		content, err = model.ContextJinjaMessage(jinja.ChatGPTConversionMessage, map[string]interface{}{
			"message": message,
		})
		if err != nil {
			return
		}
		role := model.JustValue[string, string](message, "role")
		if role == "tool" || role == "system" {
			role = "critic"
		}

		request.addMessage(role, content)
	}

	return
}

func checkRequire() (require *chatRequire, err error) {
	setDeviceId()

	if emulator.proof == "" {
		emulator.proof = "gAAAAAC" + generateAnswer(strconv.FormatFloat(rand.Float64(), 'f', -1, 64), "0")
	}
	body := bytes.NewBuffer([]byte(`{"p":"` + emulator.proof + `"}`))
	request, err := http.NewRequest(http.MethodPost, "https://chatgpt.com/backend-anon/sentinel/chat-requirements", body)
	if err != nil {
		return
	}

	request.Header.Set("User-Agent", emulator.userAgent)
	request.Header.Set("Accept", "*/*")
	request.Header.Set("Oai-Device-Id", emulator.GetDeviceId())
	request.Header.Set("Oai-Language", "en-US")
	request.Header.Set("Content-Type", "application/json")
	response, err := http.DefaultClient.Do(request)
	if err != nil {
		return
	}

	defer response.Body.Close()
	require = new(chatRequire)
	err = json.NewDecoder(response.Body).Decode(&require)
	if err != nil {
		return
	}

	if require.ForceLogin {
		err = errors.New("force login")
	}

	return
}

func generateAnswer(seed string, diff string) string {
	getDpl()
	timeStart := time.Now()
	config := getConfig()
	diffLen := len(diff)
	hasher := sha3.New512()
	for i := 0; i < 500000; i++ {
		config[3] = i
		config[9] = math.Round((float64(time.Since(timeStart).Nanoseconds())) / 1e6)
		chunk, _ := json.Marshal(config)
		base := base64.StdEncoding.EncodeToString(chunk)
		hasher.Write([]byte(seed + base))
		hash := hasher.Sum(nil)
		hasher.Reset()
		if hex.EncodeToString(hash[:diffLen])[:diffLen] <= diff {
			return base
		}
	}
	return "wQ8Lk5FbGpA2NcR9dShT6gYjU7VxZ4D" + base64.StdEncoding.EncodeToString([]byte(`"`+seed+`"`))
}

func getDpl() {
	if emulator.cachedId != "" {
		return
	}

	emulator.cachedId = "prod-a696433ddfe0489db6696cae8c5778c2128f26e8"
	request, err := http.NewRequest(http.MethodGet, "https://chatgpt.com/?oai-dm=1", nil)
	if err != nil {
		return
	}

	request.Header.Set("User-Agent", emulator.userAgent)
	request.Header.Set("Accept", "*/*")
	response, err := http.DefaultClient.Do(request)
	if err != nil {
		return
	}

	defer response.Body.Close()
	doc, _ := goquery.NewDocumentFromReader(response.Body)
	var scripts []string
	doc.Find("script[src]").Each(func(i int, s *goquery.Selection) {
		src, exists := s.Attr("src")
		if exists {
			scripts = append(scripts, src)
		}
	})
	doc.Find("html").Each(func(i int, s *goquery.Selection) {
		id, _ := s.Attr("data-build")
		emulator.cachedId = id
	})

	if len(scripts) != 0 {
		emulator.scripts = scripts
	}
}
func getConfig() []interface{} {
	rand.New(rand.NewSource(time.Now().UnixNano()))
	var script interface{}
	if len(emulator.scripts) > 0 {
		script = emulator.scripts[rand.Intn(len(emulator.scripts))]
	} else {
		script = nil
	}
	timeNum := (float64(time.Since(emulator.startTime).Nanoseconds()) + rand.Float64()) / 1e6
	return []interface{}{emulator.cachedHardware, getLocationTime(), int64(4294705152), 0, emulator.userAgent, script, emulator.cachedId, "en-US", "en-US", 0, "webkitGetUserMedia−function webkitGetUserMedia() { [native code] }", "location", "ontransitionend", timeNum, emulator.cachedSid, "", emulator.cachedCore, float64(emulator.startTime.UnixMicro()) / 1e3}
}

var (
	timeLocation, _ = time.LoadLocation("Asia/Shanghai")
	timeLayout      = "Mon Jan 2 2006 15:04:05"
)

func getLocationTime() string {
	now := time.Now()
	now = now.In(timeLocation)
	return now.Format(timeLayout) + " GMT+0800 (中国标准时间)"
}

func setDeviceId() {
	cookies := http.DefaultClient.Jar.Cookies(domain)
	var cookie *http.Cookie
	for _, c := range cookies {
		if c.Name == "oai-did" {
			cookie = c
			break
		}
	}

	if cookie == nil {
		cookie = &http.Cookie{
			Name:    "oai-did",
			Expires: time.Now().Add(1 * time.Hour),
		}
		cookie.Value = uuid.NewString()
		cookies = append(cookies, cookie)
	}

	http.DefaultClient.Jar.SetCookies(domain, cookies)
}

func (emulator *Emulator) GetDeviceId() (did string) {
	cookies := http.DefaultClient.Jar.Cookies(domain)
	for _, c := range cookies {
		if c.Name == "oai-did" {
			return c.Value
		}
	}
	return
}

func deleteDeviceId() {
	cookies := http.DefaultClient.Jar.Cookies(domain)
	for i := len(cookies) - 1; i >= 0; i-- {
		cookie := cookies[i]
		if cookie.Name == "oai-did" {
			cookies = append(cookies[:i], cookies[i+1:]...)
		}
	}
}
