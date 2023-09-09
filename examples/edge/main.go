package main

import (
	"fmt"
	"github.com/bincooo/edge-api/util"
)

func main() {
	const token = "xxx"
	if err := util.SolveCaptcha(token); err != nil {
		fmt.Println(err)
	}
}
