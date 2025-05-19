//go:build cgo

package frame

import (
	"image"
	"image/jpeg"
	"log"
	"os"
	"strconv"
	"time"
)

// AppDir 返回当前应用目录
func AppDir() string {
	path, _ := os.Getwd()

	gogohome := os.Getenv("GOGO_HOME")
	if gogohome == "" {
		//fmt.Printf("app dir:%s\n", path)
		return path
	} else {
		//fmt.Printf("app dir:%s\n", gogohome)
		return gogohome
	}

}

// This example shows how to
// 1. connect to a RTSP server.
// 2. check if there's a H264 stream.
// 3. decode the H264 stream into RGBA frames.
// 4. convert RGBA frames to JPEG images and save them on disk.

// This example requires the FFmpeg libraries, that can be installed with this command:
// apt install -y libavcodec-dev libswscale-dev gcc pkg-config

func saveToFile(img image.Image) error {
	currDir := AppDir()
	// create file
	fname := currDir + "/" + strconv.FormatInt(time.Now().UnixNano()/int64(time.Millisecond), 10) + ".jpg"
	f, err := os.Create(fname)
	if err != nil {
		panic(err)
	}
	defer f.Close()

	log.Println("saving", fname)

	// convert to jpeg
	return jpeg.Encode(f, img, &jpeg.Options{
		Quality: 60,
	})
}
