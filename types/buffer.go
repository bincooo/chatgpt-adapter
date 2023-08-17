package types

import (
	"github.com/bincooo/MiaoX/vars"
	"time"
)

var (
	CacheMessageL    = 200
	CacheWaitTimeout = 10 * time.Second
)

type CacheBuffer struct {
	timer    time.Time
	Cache    string
	Complete string

	H func(self *CacheBuffer) error

	Closed bool
}

func (r *CacheBuffer) condition() bool {
	if r.Closed {
		return true
	}
	// n秒内缓存消息
	if r.timer.After(time.Now()) {
		return false
	}

	// n秒后消耗消息
	// 字数太少续n秒
	if len(r.Cache) < CacheMessageL {
		if CacheWaitTimeout > 0 {
			r.timer = time.Now().Add(CacheWaitTimeout)
		}
		return false
	}

	return true
}

func (r *CacheBuffer) Read() PartialResponse {
	if r.H == nil {
		panic("Please define handle first.")
	}

	var partialResponse PartialResponse
	if err := r.H(r); err != nil {
		partialResponse.Error = err
		return partialResponse
	}

	if !r.condition() {
		return partialResponse
	}

	if len(r.Cache) > 0 {
		r.Complete += r.Cache
		partialResponse.Message = r.Cache
		r.Cache = ""
	}

	if r.Closed {
		partialResponse.Status = vars.Closed
	} else {
		partialResponse.Status = vars.Trying
	}
	return partialResponse
}
