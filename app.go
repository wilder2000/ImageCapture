package main

import (
	"cv/frame"
	"flag"
	"fmt"
	"github.com/wilder2000/GOSimple/glog"
	// "github.com/disintegration/imaging"
)

func main() {

	glog.Logger.InfoF("")
	rtspUrl := flag.String("rtsp", "url", "rtsp url")

	flag.Parse()
	fmt.Printf("play:%s\n", *rtspUrl)

	// 初始化FFmpeg
	fmt.Println("start to test.")
	if err := frame.Start(*rtspUrl); err != nil {
		fmt.Printf("error:%s", err.Error())
	}
	// reader := frame.ExampleReadFrameAsJpeg(*rtspUrl, 5)
	// img, err := imaging.Decode(reader)
	// if err != nil {
	// 	fmt.Printf("save image failed %s\n", err.Error())
	// }
	// err = imaging.Save(img, "./sample_data/out1.jpeg")
	// if err != nil {
	// 	fmt.Printf("save image failed %s\n", err.Error())
	// }

}
