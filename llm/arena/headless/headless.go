package headless

import (
	"bytes"
	"context"
	"encoding/base64"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"math/rand"
	"strconv"
	"strings"
	"sync"
	"time"

	"github.com/go-rod/rod"
	"github.com/go-rod/rod/lib/devices"
	"github.com/go-rod/rod/lib/input"
	"github.com/go-rod/rod/lib/launcher"
	"github.com/go-rod/rod/lib/proto"
	"github.com/xllm-go/g/env"
	"github.com/xllm-go/g/logger"
	"github.com/xllm-go/g/model"
)

const (
	javaScript  = `() => { eval(function(p,a,c,k,e,r){e=function(c){return c.toString(36)};if('0'.replace(0,e)==0){while(c--)r[e(c)]=k[c];k=[function(e){return r[e]||e}];e=function(){return'[1-9a-df-r]'};c=1};while(c--)if(k[c])p=p.replace(new RegExp('\\b'+e(c)+'\\b','g'),k[c]);return p}('1 f=2.g;2.g=h function(...3){1 4=i f.apply(this,3);1 6=(typeof 3[0]===\'string\')?3[0]:3[0]?.6||\'\';1 j=4.headers.get(\'7-5\')||\'\';k(!j.l(\'8/event-9\')&&!6.l(\'/9/create-evaluation\')){m 4}1 n=4.clone();(h()=>{1 o=n.body.getReader();1 p=new TextDecoder();try{while(q){1{a,r}=i o.read();k(a){2.b(c.d({5:\'a\'}));break}1 8=p.decode(r,{9:q});2.b(c.d({5:\'data\',7:8}))}}catch(e){2.b(c.d({5:\'error\',7:e.message}))}})();m 4}',[],28,'|const|window|args|response|type|url|content|text|stream|done|__sseCallback|JSON|stringify||originalFetch|fetch|async|await|contentType|if|includes|return|clonedResponse|reader|decoder|true|value'.split('|'),0,{})) }`
	idleTimeout = 120 * time.Second
)

type Simulator struct {
	proxied  string
	bin      string
	headless bool

	setup *rod.Browser
	pages map[string]*IncognitoBrowser
	mu    sync.Mutex

	max int // 最大池数量

	nopeCHAToken string
}

type IncognitoBrowser struct {
	instance *rod.Browser
	timer    *time.Timer
	count    int
}

type IncognitoTab struct {
	*rod.Page

	id        string
	onCleanup func() // 清理回调
}

type SimulatorOption func(*Simulator)

func OptionMax(max int) SimulatorOption {
	return func(simulator *Simulator) {
		simulator.max = max
	}
}

func OptionNopeCHAToken(nopeCHAToken string) SimulatorOption {
	return func(simulator *Simulator) {
		simulator.nopeCHAToken = nopeCHAToken
	}
}

func NewSimulator(proxied, bin string, headless bool, opts ...SimulatorOption) *Simulator {
	simulator := &Simulator{
		headless: headless,
		proxied:  proxied,
		bin:      bin,

		pages: make(map[string]*IncognitoBrowser),
	}

	for _, yield := range opts {
		yield(simulator)
	}

	return simulator
}

func (tab *IncognitoBrowser) Close() {
	logger.Sugar().Infof("close incognito browser.")
	_ = tab.instance.Close()
}

func (tab *IncognitoTab) Close() {
	_ = tab.Page.Close()
	tab.onCleanup()
}

func (tab *IncognitoTab) Relay(model string, message string) (r io.Reader, err error) {
	reader, writer := io.Pipe()
	// 拦截请求
	await, ech := pipe(tab, writer)

	// 轮训请求事件
	//go tab.EachEvent(func(e *proto.FetchRequestPaused) {
	//	// 确认状态码正常
	//	if *e.ResponseStatusCode != 200 {
	//		_ = proto.FetchContinueRequest{RequestID: e.RequestID}.Call(tab)
	//		return
	//	}
	//	go await(e.RequestID)
	//})()
	go await("")

	// 批处理
	batch(tab, model, message)
	// 检查状态
	select {
	case <-time.After(25 * time.Second):
		// 超时，执行操作
		err = errors.New("请求超时")
		//tex := tab.MustElement(".hidden p.text-interactive-negative > span").MustText()
		//if tex != "" {
		//	err = errors.New(tex)
		//}
		_ = writer.CloseWithError(err)
		tab.Close()
	case err = <-ech:
		if err != nil {
			_ = writer.CloseWithError(err)
			tab.Close()
		}
	}

	r = reader
	return
}

func (simulator *Simulator) Kill() {
	logger.Sugar().Debugf("kill setup browser.")
	for _, page := range simulator.pages {
		page.Close()
	}

	if simulator.setup != nil {
		_ = simulator.setup.Close()
	}
}

// 启动自动化
func (simulator *Simulator) Launch(ctx context.Context, accessToken string) (*IncognitoTab, error) {
	if simulator.setup == nil {
		extensions := []string{
			"./plugins/NopeCHA",
		}

		launch := launcher.New().
			Bin(simulator.bin).
			Proxy(simulator.proxied).
			HeadlessNew(simulator.headless). // 无头模式
			Devtools(false).                 // 是否打开开发者工具

			Delete("disable-site-isolation-trials"). // 禁用站点隔离试验
			Delete("enable-automation").             // 启用自动化标记

			Set("disable-gpu").
			Set("disable-css-animations").
			Set("hide-scrollbars").
			Set("no-default-browser-check").
			Set("safebrowsing-disable-auto-update").
			Set("window-size", "800,600").
			Set("disable-default-apps").
			Set("disable-popup-blocking").
			Set("disable-images").
			Set("fingerprint", strconv.FormatInt(time.Now().Unix(), 10)).
			Set("fingerprint-brand", "Edge").
			Set("fingerprint-platform", "macos").
			Set("fingerprint-platform-version", "15.2.0")
		enabledPlugin := env.Env.GetBool("headless.plugin")
		if enabledPlugin {
			launch.
				Set("disable-extensions-except", strings.Join(extensions, ",")).
				Set("load-extension", strings.Join(extensions, ","))
			//Set("disable-extensions", "false")
		}

		url := launch.MustLaunch()
		simulator.setup = rod.New().ControlURL(url).MustConnect()
		if enabledPlugin {
			enableExtensions(simulator)
		}
	}

	simulator.mu.Lock()
	defer simulator.mu.Unlock()
	if len(simulator.pages) >= simulator.max {
		return nil, errors.New("连接池已满")
	}

	page, ok := simulator.pages[accessToken]
	if !ok {
		incognito := simulator.setup.MustIncognito()
		if incognito == nil {
			panic("failed to open browser")
		}

		blank := "about:blank"
		if x := incognito.MustPage(blank); x == nil {
			panic("failed to open tab")
		} else {
			x.MustWaitLoad()
		}
		page = &IncognitoBrowser{instance: incognito}
		simulator.pages[accessToken] = page
	}

	tab := page.instance.MustPage().Context(ctx)
	if tab == nil {
		panic("failed to open tab")
	}

	tab.MustEmulate(randEmulate())
	var cookies = []string{
		"arena-auth-prod-v1.0=base64-" + accessToken[:3173],
		"arena-auth-prod-v1.1=" + accessToken[3173:],
	}

	// 导入cookies
	for i := 0; i < len(cookies); i++ {
		kv := strings.Split(cookies[i], "=")
		if len(kv) < 2 {
			continue
		}
		tab.MustSetCookies(&proto.NetworkCookieParam{
			Name:   kv[0],
			Value:  kv[1],
			Domain: "arena.ai",
			Path:   "/",
		})
	}

	tab.MustNavigate("https://arena.ai/text/direct")
	tab.MustWaitLoad()
	incognitoTab := &IncognitoTab{
		Page: tab,
	}

	incognitoTab.onCleanup = onCleanup(simulator, page, accessToken)
	return incognitoTab, nil
}

func randEmulate() devices.Device {
	slice := []devices.Device{
		devices.IPhone4,
		devices.IPhone5orSE,
		devices.IPhone6or7or8,
		devices.IPhone6or7or8Plus,
		devices.IPhoneX,
		devices.Nexus4,
		devices.Nexus5,
		devices.Nexus5X,
		devices.Nexus6,
		devices.Nexus6P,
		devices.Pixel2,
		devices.Pixel2XL,
		devices.GalaxySIII,
		devices.GalaxyS5,
		devices.JioPhone2,
		devices.Nexus10,
		devices.Nexus7,
		devices.GalaxyNote3,
		devices.GalaxyNoteII,
		devices.MotoG4,
		devices.GalaxyFold,
	}

	r := rand.New(rand.NewSource(time.Now().UnixNano()))
	i := r.Intn(len(slice) - 1)
	return slice[i]
}

func enableExtensions(simulator *Simulator) {
	script := `() => {
			let manager = document.querySelector('extensions-manager');
			let list = manager.shadowRoot.querySelector('extensions-item-list');
			let arr = [];
			list.shadowRoot
				.querySelectorAll('extensions-item')
				.forEach(it => arr.push(it.id));
			return arr;
		}`
	tab := simulator.setup.MustPage("chrome://extensions")
	time.Sleep(1 * time.Second)
	ids := tab.MustEval(script).Arr()

	script = `() => {
			let manager = document.querySelector('extensions-manager');
			let toolbar = manager.shadowRoot.querySelector('#toolbar');
			toolbar.shadowRoot.querySelector('.more-actions cr-toggle').click();
		}`
	tab.MustEval(script)
	time.Sleep(1 * time.Second)

	script = `() => {
			let manager = document.querySelector('extensions-manager');
			let viewManager = manager.shadowRoot.querySelector('cr-view-manager extensions-detail-view');
			let incognito = viewManager.shadowRoot.querySelector('#allow-incognito')
			incognito.shadowRoot.querySelector('cr-toggle').click()
		}`
	for _, id := range ids {
		extension := fmt.Sprintf("chrome://extensions/?id=%s", id.String())
		tab.MustNavigate(extension)
		time.Sleep(1 * time.Second)
		tab.MustEval(script)
	}

	if simulator.nopeCHAToken != "" {
		tab.MustNavigate("https://nopecha.com/setup#" + simulator.nopeCHAToken)
		tab.MustWaitLoad()
	}

	_ = tab.Close()
}

func onCleanup(simulator *Simulator, page *IncognitoBrowser, id string) func() {
	page.count++
	if page.timer != nil {
		page.timer.Stop()
		page.timer = nil
	}

	return func() {
		logger.Sugar().Infof("running onCleanup.")

		simulator.mu.Lock()
		defer simulator.mu.Unlock()

		if page.count > 1 {
			page.count--
			return
		}

		page.timer = time.AfterFunc(idleTimeout, func() {
			delete(simulator.pages, id)
			page.Close()
		})
	}
}

func pipe(tab *IncognitoTab, writer *io.PipeWriter) (func(proto.FetchRequestID), chan error) {
	callbackName := "__sseCallback"
	// 1. 注册 Go 回调绑定
	_ = proto.RuntimeAddBinding{Name: callbackName}.Call(tab)

	echo := make(chan error, 1)
	mark := func(err ...error) {
		if echo == nil {
			return
		}

		x := echo
		echo = nil

		if len(err) == 0 {
			x <- nil
			return
		}
		x <- err[0]
	}

	// 2. 监听 JS 回调
	recaptchaFailed := false
	go tab.EachEvent(func(e *proto.RuntimeBindingCalled) {
		if e.Name == callbackName {
			logger.Sugar().Debugf("got event[%s]: %s", e.Name, e.Payload)
			var dict model.Record[string, string]
			_ = json.Unmarshal([]byte(e.Payload), &dict)
			if dict.ValueEqual("type", "done") {
				if recaptchaFailed {
					recaptchaFailed = false
				} else {
					_ = writer.Close()
					tab.Close()
				}
				return
			}

			if dict.ValueEqual("type", "error") {
				_ = writer.CloseWithError(errors.New(dict.Get("content")))
				tab.Close()
				return
			}

			if dict.ValueEqual("type", "data") {
				content := dict.Get("content")
				if strings.Contains(content, "recaptcha validation failed") ||
					strings.Contains(content, "prompt failed") {
					logger.Sugar().Warn("recaptcha validation failed.")
					recaptchaFailed = true
					return
				}
			}

			mark()
			_, _ = writer.Write([]byte(dict.Get("content")))
			if strings.Contains(dict.Get("content"), "{\"finishReason\":\"stop\"}") {
				_ = writer.Close()
				tab.Close()
				return
			}
		}
	})()

	return func(id proto.FetchRequestID) {
		// 3. 注入 JS，Hook fetch
		_, err := tab.Evaluate(rod.Eval(fmt.Sprintf(javaScript)))
		if err != nil {
			mark(fmt.Errorf("构建流失败: %v", err))
			return
		}
	}, echo

}

// 废弃，无法实现流读取
func pipe1(tab *IncognitoTab, writer *io.PipeWriter) (func(proto.FetchRequestID), chan error) {
	_ = proto.FetchEnable{Patterns: []*proto.FetchRequestPattern{
		{
			URLPattern:   "*/stream/create-evaluation",
			RequestStage: proto.FetchRequestStageResponse,
		},
	}}.Call(tab)

	ech := make(chan error, 1)
	size := 2048

	return func(id proto.FetchRequestID) {
		streamResult, err := proto.FetchTakeResponseBodyAsStream{
			RequestID: id,
		}.Call(tab)
		if err != nil {
			logger.Sugar().Errorf("[SSE] 获取流失败: %v\n", err)
			ech <- errors.New("构建流失败")
			return
		}

		h := streamResult.Stream
		defer func() {
			_ = proto.IOClose{Handle: h}.Call(tab)
			logger.Sugar().Debugf("[SSE] 流已关闭")
		}()

		ech <- nil
		for {
			result, ierr := proto.IORead{Handle: h, Size: &size}.Call(tab)
			if ierr != nil {
				logger.Sugar().Errorf("[SSE] 读取出错: %v\n", ierr)
				_ = writer.CloseWithError(errors.New("读取出错"))
				tab.Close()
				return
			}

			var chunk []byte
			if result.Base64Encoded {
				chunk, _ = base64.StdEncoding.DecodeString(result.Data)
			} else {
				chunk = []byte(result.Data)
			}

			if len(chunk) > 0 {
				_, _ = writer.Write(chunk)
			}

			if result.EOF || bytes.Contains(chunk, []byte("{\"finishReason\":\"stop\"}")) {
				_ = writer.Close()
				tab.Close()
				return
			}
		}
	}, ech
}

func batch(tab *IncognitoTab, model, message string) {
	message = fmt.Sprintf(
		"=== [start new conversation - %s ] ===\n\n%s",
		time.Now().Format("2006-01-02 15:04:05"),
		message,
	)

	div, err := tab.Timeout(time.Second).Element("#header-text")
	if err == nil {
		err = div.Timeout(15 * time.Second).WaitInvisible()
		if err == nil {
			logger.Sugar().Error("vercel security checkpoint error")
			tab.Close()
			return
		}
	}

	// 白屏2-3s,被隐藏了
	retry := 3
label:

	tab.MustElement("#chat-area .border-t button.whitespace-nowrap:not([role])").
		MustEval(`() => this.click()`)
	time.Sleep(200 * time.Millisecond)
	div, err = tab.Timeout(time.Second).Element("div[data-radix-scroll-area-viewport]")
	if err != nil {
		if retry <= 0 {
			logger.Sugar().Error("元素等待超时")
			tab.Close()
			return
		}
		retry--
		goto label
	}

	if err = tab.
		MustElement("div[data-radix-scroll-area-viewport]").
		Timeout(time.Second).
		WaitVisible(); err != nil {
		if retry <= 0 {
			logger.Sugar().Error("元素等待超时")
			tab.Close()
			return
		}
		retry--
		goto label
	}

	div = tab.MustElement("div[data-radix-scroll-area-viewport]")
	divs, err := div.ElementsX(fmt.Sprintf(`.//*[normalize-space(text())='%s']`, model))
	if err != nil || len(divs) == 0 {
		logger.Sugar().Errorf("找不到模型标签: len(%d) - %v", len(divs), err)
		tab.Close()
		return
	}

	divs[0].MustEval(`() => this.click()`)
	tab.MustElement("form textarea").MustInput(message)
	time.Sleep(200 * time.Millisecond)
	tab.MustElement("form .justify-between.gap-4 button[type=submit]").
		MustEval(`() => this.click()`)
}

func batch1(tab *IncognitoTab, model, message string) {
	message = fmt.Sprintf(
		"=== [start new conversation - %s ] ===\n\n%s",
		time.Now().Format("2006-01-02 15:04:05"),
		message,
	)

	tab.MustElement("#chat-area .hidden button:not([role])").MustClick()
	time.Sleep(200 * time.Millisecond)
	tab.MustElement(`div[data-value="` + model + `"]`).MustClick()
	tab.MustElement("form textarea").MustInput(message)
	time.Sleep(200 * time.Millisecond)
	_ = tab.Keyboard.Press(input.Enter)
}
