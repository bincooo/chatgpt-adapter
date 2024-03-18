package common

import "testing"

func Test_UploadFile(t *testing.T) {
	file, err := UploadCatboxFile(
		"http://127.0.0.1:7890",
		"https://krebzonide-sdxl-turbo-with-refiner.hf.space/file=/tmp/gradio/a2fbfa1d1244c324ba394fa2fd0bd9d416ffb033/image.png",
	)
	if err != nil {
		t.Fatal(err)
	}
	t.Log(file)
}
