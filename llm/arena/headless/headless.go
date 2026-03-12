package headless

import (
	"bytes"
	"context"
	"encoding/base64"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"strconv"
	"strings"
	"sync"
	"time"

	"github.com/go-rod/rod"
	"github.com/go-rod/rod/lib/devices"
	"github.com/go-rod/rod/lib/input"
	"github.com/go-rod/rod/lib/launcher"
	"github.com/go-rod/rod/lib/proto"
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

func NewSimulator(proxied, bin string, headless bool, max int) *Simulator {
	return &Simulator{
		headless: headless,
		proxied:  proxied,
		bin:      bin,

		pages: make(map[string]*IncognitoBrowser),
		max:   max,
	}
}

func (tab *IncognitoBrowser) Close() {
	logger.Sugar().Debugf("close incognito browser.")
	_ = tab.instance.Close()
}

func (tab *IncognitoTab) Close() {
	logger.Sugar().Debugf("close incognito tab.")
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
	case <-time.After(12 * time.Second):
		// 超时，执行操作
		err = errors.New("请求超时")
		tex := tab.MustElement(".hidden p.text-interactive-negative > span").MustText()
		if tex != "" {
			err = errors.New(tex)
		}
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
		url := launcher.New().
			Bin(simulator.bin).
			Proxy(simulator.proxied).
			HeadlessNew(simulator.headless). // 无头模式
			Devtools(false). // 是否打开开发者工具

			Delete("disable-site-isolation-trials"). // 禁用站点隔离试验
			Delete("enable-automation"). // 启用自动化标记

			Set("disable-extensions").
			Set("disable-gpu").
			Set("disable-css-animations").
			Set("hide-scrollbars").
			Set("no-default-browser-check").
			Set("safebrowsing-disable-auto-update").
			Set("window-size", "800,600").
			Set("disable-extensions").
			Set("disable-default-apps").
			Set("disable-popup-blocking").
			Set("disable-images").
			Set("fingerprint", strconv.FormatInt(time.Now().Unix(), 10)).
			Set("fingerprint-brand", "Edge").
			Set("fingerprint-platform", "macos").
			Set("fingerprint-platform-version", "15.2.0").
			MustLaunch()
		simulator.setup = rod.New().ControlURL(url).MustConnect()
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
		if p := incognito.MustPage("about:blank"); p == nil {
			panic("failed to open tab")
		}
		page = &IncognitoBrowser{
			instance: incognito,
		}
		simulator.pages[accessToken] = page
	}

	tab := page.instance.MustPage().Context(ctx)
	if tab == nil {
		panic("failed to open tab")
	}

	tab.MustEmulate(devices.Pixel2)
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

func onCleanup(simulator *Simulator, page *IncognitoBrowser, id string) func() {
	page.count++
	if page.timer != nil {
		page.timer.Stop()
		page.timer = nil
	}

	return func() {
		logger.Sugar().Debugf("running onCleanup.")

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

	// 2. 监听 JS 回调
	go tab.EachEvent(func(e *proto.RuntimeBindingCalled) {
		if e.Name == callbackName {
			var dict model.Record[string, string]
			_ = json.Unmarshal([]byte(e.Payload), &dict)
			if dict.ValueEqual("type", "done") {
				_ = writer.Close()
				tab.Close()
				return
			}
			if dict.ValueEqual("type", "error") {
				_ = writer.CloseWithError(errors.New(dict.Get("content")))
				tab.Close()
				return
			}

			_, _ = writer.Write([]byte(dict.Get("content")))
			if strings.Contains(dict.Get("content"), "{\"finishReason\":\"stop\"}") {
				_ = writer.Close()
				tab.Close()
				return
			}
		}
	})()

	ech := make(chan error, 1)
	return func(id proto.FetchRequestID) {
		// 3. 注入 JS，Hook fetch
		_, err := tab.Evaluate(rod.Eval(fmt.Sprintf(javaScript)))
		if err != nil {
			ech <- fmt.Errorf("构建流失败: %v", err)
			return
		}
		ech <- nil
	}, ech

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

	tab.MustElement("#chat-area .border-t button.whitespace-nowrap:not([role])").
		MustEval(`() => this.click()`)
	time.Sleep(200 * time.Millisecond)
	div := tab.MustElement("div[data-radix-scroll-area-viewport]")
	div.MustElementX(fmt.Sprintf(`.//*[normalize-space(text())='%s']`, model)).
		MustEval(`() => this.click()`)
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
