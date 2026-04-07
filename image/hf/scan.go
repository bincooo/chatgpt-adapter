package hf

import (
	"bufio"
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"math/rand"
	"net/http"

	"github.com/RomiChan/websocket"
)

type JoinEvent struct {
	Msg     string      `json:"msg"`
	EventId string      `json:"event_id"`
	Success bool        `json:"success"`
	Output  *joinOutput `json:"output"`

	InitialBytes []byte `json:"-"`
}

type joinOutput struct {
	Generating      bool    `json:"is_generating"`
	Duration        float64 `json:"duration"`
	AverageDuration float64 `json:"average_duration"`

	Data []interface{} `json:"data"`
}

type Scanner struct {
	response *http.Response
	conn     *websocket.Conn
	ctx      context.Context
	em       map[string]func(j JoinEvent) interface{}
	err      error
	close    bool
}

func newScanner(ctx context.Context, coupler interface{}) (e *Scanner, err error) {
	if ctx == nil {
		ctx = context.Background()
	}

	e = &Scanner{
		ctx: ctx,
		em:  map[string]func(j JoinEvent) interface{}{
			//
		},
	}

	switch c := coupler.(type) {
	case *http.Response:
		e.response = c
	case *websocket.Conn:
		e.conn = c
	default:
		return nil, errors.New("'coupler' must be *http.Response or *websocket.Conn")
	}
	return
}

// 注册事件
func (scan *Scanner) Event(eventId string, funcCall func(j JoinEvent) interface{}) {
	scan.em[eventId] = funcCall
}

// 异常设置，并终止事件
func (scan *Scanner) Failed(err error) {
	scan.err = err
	scan.Cancel()
}

// 终止事件
func (scan *Scanner) Cancel() {
	scan.close = true
}

func (scan *Scanner) Do() error {
	if scan.conn == nil && scan.response == nil {
		panic("'coupler' is nil, please provide a valid 'coupler' value")
	}

	if scan.conn != nil {
		return scan.warpE(scan.doConn())
	}

	return scan.warpE(scan.doResponse())
}

func (scan *Scanner) warpE(err error) error {
	if scan.err != nil {
		return scan.err
	}
	return err
}

func (scan *Scanner) doConn() error {
	for {
		select {
		case <-scan.ctx.Done():
			return scan.ctx.Err()
		default:
			if scan.close {
				return nil
			}

			_, data, err := scan.conn.ReadMessage()
			if err != nil {
				return err
			}

			var j JoinEvent
			err = json.Unmarshal(data, &j)
			if err != nil {
				return err
			}

			var chunk []byte
			j.InitialBytes = data

			if funcCall, ok := scan.em[j.Msg]; ok {
				if r := funcCall(j); r != nil {
					chunk, err = json.Marshal(r)
					if err != nil {
						return err
					}

					err = scan.conn.WriteMessage(websocket.TextMessage, chunk)
					if err != nil {
						return err
					}
				}
			}

			if funcCall, ok := scan.em["*"]; ok {
				if r := funcCall(j); r != nil {
					chunk, err = json.Marshal(r)
					if err != nil {
						return err
					}

					err = scan.conn.WriteMessage(websocket.TextMessage, chunk)
					if err != nil {
						return err
					}
				}
			}

			if j.Success && j.Msg == "process_completed" {
				return nil
			}
		}
	}
}

func (scan *Scanner) doResponse() error {
	defer scan.response.Body.Close()
	scanner := bufio.NewScanner(scan.response.Body)
	scanner.Split(func(data []byte, eof bool) (advance int, token []byte, err error) {
		if eof && len(data) == 0 {
			return
		}

		if i := bytes.IndexByte(data, '\n'); i >= 0 {
			return i + 1, data[0:i], nil
		}

		if eof {
			return len(data), data, nil
		}

		return
	})

	for {
		select {
		case <-scan.ctx.Done():
			return scan.ctx.Err()
		default:
			if scan.close {
				return nil
			}

			if !scanner.Scan() {
				return nil
			}

			data := scanner.Text()
			if len(data) < 6 || data[:6] != "data: " {
				continue
			}
			data = data[6:]

			var j JoinEvent
			j.InitialBytes = []byte(data)

			err := json.Unmarshal(j.InitialBytes, &j)
			if err != nil {
				return err
			}

			if funcCall, ok := scan.em[j.Msg]; ok {
				funcCall(j)
			}

			if funcCall, ok := scan.em["*"]; ok {
				funcCall(j)
			}

			if j.Success && j.Msg == "process_completed" {
				return nil
			}
		}
	}
}

func hash() string {
	bin := "abcdefghijklmnopqrstuvwxyzABCDEFGHIJKLMNOPQRSTUVWXYZ1234567890"
	binL := len(bin)
	var buf bytes.Buffer
	for x := 0; x < 10; x++ {
		ch := bin[rand.Intn(binL-1)]
		buf.WriteByte(ch)
	}

	return buf.String()
}
