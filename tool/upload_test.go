package tool

import (
	"net/http"
	"os"
	"testing"

	"github.com/bincooo/ja3"
	xtls "github.com/refraction-networking/utls"
)

func TestUpload(t *testing.T) {
	ja3.NewTransport(
		ja3.WithClientHelloID(xtls.HelloChrome_133),
		ja3.WithOriginalTransport(http.DefaultTransport.(*http.Transport)),
		//ja3.WithProxy("http://127.0.0.1:7890"),
	)

	chunk, err := os.ReadFile("/home/bincooo/Pictures/head.png")
	if err != nil {
		panic(err)
	}

	imageUrl, mimetype, err := Upload(chunk, "png")
	if err != nil {
		panic(err)
	}

	println("mime: ", mimetype)
	println("image: ", imageUrl)
}
