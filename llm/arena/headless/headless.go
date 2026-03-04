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
	"time"

	"github.com/go-rod/rod"
	"github.com/go-rod/rod/lib/input"
	"github.com/go-rod/rod/lib/launcher"
	"github.com/go-rod/rod/lib/proto"
	"github.com/xllm-go/g/logger"
	"github.com/xllm-go/g/model"
)

const (
	javaScript = `() => { eval(function(p,a,c,k,e,r){e=function(c){return c.toString(36)};if('0'.replace(0,e)==0){while(c--)r[e(c)]=k[c];k=[function(e){return r[e]||e}];e=function(){return'[1-9a-df-r]'};c=1};while(c--)if(k[c])p=p.replace(new RegExp('\\b'+e(c)+'\\b','g'),k[c]);return p}('1 f=2.g;2.g=h function(...3){1 4=i f.apply(this,3);1 6=(typeof 3[0]===\'string\')?3[0]:3[0]?.6||\'\';1 j=4.headers.get(\'7-5\')||\'\';k(!j.l(\'8/event-9\')&&!6.l(\'/9/create-evaluation\')){m 4}1 n=4.clone();(h()=>{1 o=n.body.getReader();1 p=new TextDecoder();try{while(q){1{a,r}=i o.read();k(a){2.b(c.d({5:\'a\'}));break}1 8=p.decode(r,{9:q});2.b(c.d({5:\'data\',7:8}))}}catch(e){2.b(c.d({5:\'error\',7:e.message}))}})();m 4}',[],28,'|const|window|args|response|type|url|content|text|stream|done|__sseCallback|JSON|stringify||originalFetch|fetch|async|await|contentType|if|includes|return|clonedResponse|reader|decoder|true|value'.split('|'),0,{})) }`
)

type Simulator struct {
	proxied  string
	bin      string
	headless bool

	browser *rod.Browser
	tabs    map[string]*rod.Page
}

func NewSimulator(proxied, bin string, headless bool) *Simulator {
	return &Simulator{
		headless: headless,
		proxied:  proxied,
		bin:      bin,

		tabs: make(map[string]*rod.Page)}
}

func (simulator *Simulator) Close(id string) {
	tab, ok := simulator.tabs[id]
	if !ok {
		return
	}

	_ = tab.Close()
	delete(simulator.tabs, id)
}

func (simulator *Simulator) Kill() {
	for id := range simulator.tabs {
		simulator.Close(id)
	}
	if simulator.browser != nil {
		_ = simulator.browser.Close()
	}
}

// 启动自动化
func (simulator *Simulator) Launch(ctx context.Context, cookie string) (id string) {
	if simulator.browser == nil {
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
		simulator.browser = rod.New().ControlURL(url).MustConnect()
	}

	tab := simulator.browser.MustPage().Context(ctx)
	if tab == nil {
		panic("failed to open tab")
	}

	id = string(tab.TargetID)
	simulator.tabs[id] = tab
	var cookies = []string{
		"arena-auth-prod-v1.0=base64-" + cookie[:3173],
		"arena-auth-prod-v1.1=" + cookie[3173:],
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

	return
}

func (simulator *Simulator) Relay(id, model, message string) (r io.Reader, err error) {
	tab := simulator.tabs[id]
	tab.MustNavigate("https://arena.ai/text/direct")
	tab.MustWaitLoad()

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
		simulator.Close(id)
	case err = <-ech:
		if err != nil {
			_ = writer.CloseWithError(err)
			simulator.Close(id)
		}
	}

	r = reader
	return
}

func pipe(tab *rod.Page, writer *io.PipeWriter) (func(proto.FetchRequestID), chan error) {
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
				_ = tab.Close()
				return
			}
			if dict.ValueEqual("type", "error") {
				_ = writer.CloseWithError(errors.New(dict.Get("content")))
				_ = tab.Close()
				return
			}

			_, _ = writer.Write([]byte(dict.Get("content")))
			if strings.Contains(dict.Get("content"), "{\"finishReason\":\"stop\"}") {
				_ = writer.Close()
				_ = tab.Close()
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
func pipe1(tab *rod.Page, writer *io.PipeWriter) (func(proto.FetchRequestID), chan error) {
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
				_ = tab.Close()
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
				_ = tab.Close()
				return
			}
		}
	}, ech
}

func batch(tab *rod.Page, model, message string) {
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
