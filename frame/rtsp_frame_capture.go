package frame

/*
#cgo pkg-config: libavformat libavcodec libavutil libswscale
#include <libavformat/avformat.h>
#include <libavcodec/avcodec.h>
#include <libavutil/imgutils.h>
#include <libswscale/swscale.h>
#include <libavutil/error.h>
*/
import "C"
import (
	"errors"
	"fmt"
	"image"
	"image/color"
	"log"
	"os"
	"path/filepath"
	"unsafe"
)

// 定义FFmpeg错误码
const (
	// 从libavutil/error.h中获取
	CAVERROR_EAGAIN = C.int(-11)
	CAVERROR_EOF    = C.int(-541478725) // AVERROR_EOF
)

// AppDir 返回应用程序目录
// func AppDir() string {
// 	dir, err := os.UserHomeDir()
// 	if err != nil {
// 		return ""
// 	}
// 	return filepath.Join(dir, "go-rtsp-frame-extractor")
// }

func Start(rtspUrl string) error {
	// 初始化输出目录
	outputDir := filepath.Join(AppDir(), "frames_out")
	if err := os.MkdirAll(outputDir, 0755); err != nil {
		return fmt.Errorf("创建目录失败: %w", err)
	}

	// FFmpeg初始化
	C.avformat_network_init()
	defer C.avformat_network_deinit()

	pFormatCtx := C.avformat_alloc_context()
	if pFormatCtx == nil {
		return errors.New("分配格式上下文失败")
	}
	defer C.avformat_free_context(pFormatCtx)

	// 打开RTSP流
	cUrl := C.CString(rtspUrl)
	defer C.free(unsafe.Pointer(cUrl))
	if ret := C.avformat_open_input(&pFormatCtx, cUrl, nil, nil); ret != 0 {
		return fmt.Errorf("打开RTSP流失败: %d", ret)
	}
	defer C.avformat_close_input(&pFormatCtx)

	// 查找流信息
	if ret := C.avformat_find_stream_info(pFormatCtx, nil); ret < 0 {
		return fmt.Errorf("获取流信息失败: %d", ret)
	}

	// 查找视频流
	var videoStream *C.AVStream
	videoStreamIndex := -1
	for i := 0; i < int(pFormatCtx.nb_streams); i++ {
		// 修复：正确访问流数组
		stream := *(**C.AVStream)(unsafe.Pointer(uintptr(unsafe.Pointer(pFormatCtx.streams)) + uintptr(i)*unsafe.Sizeof(pFormatCtx.streams)))
		if stream.codecpar.codec_type == C.AVMEDIA_TYPE_VIDEO {
			videoStreamIndex = i
			videoStream = stream
			break
		}
	}

	if videoStreamIndex == -1 {
		return errors.New("未找到视频流")
	}

	// 初始化解码器
	codec := C.avcodec_find_decoder(videoStream.codecpar.codec_id)
	if codec == nil {
		return errors.New("找不到解码器")
	}

	codecCtx := C.avcodec_alloc_context3(codec)
	if codecCtx == nil {
		return errors.New("分配解码器上下文失败")
	}
	defer C.avcodec_free_context(&codecCtx)

	if ret := C.avcodec_parameters_to_context(codecCtx, videoStream.codecpar); ret < 0 {
		return fmt.Errorf("复制编解码参数失败: %d", ret)
	}

	if ret := C.avcodec_open2(codecCtx, codec, nil); ret < 0 {
		return fmt.Errorf("打开解码器失败: %d", ret)
	}

	// 初始化帧
	pFrame := C.av_frame_alloc()
	if pFrame == nil {
		return errors.New("分配帧失败")
	}
	defer C.av_frame_free(&pFrame)

	pFrameRGB := C.av_frame_alloc()
	if pFrameRGB == nil {
		return errors.New("分配RGB帧失败")
	}
	defer C.av_frame_free(&pFrameRGB)

	// 设置帧参数
	pFrameRGB.width = codecCtx.width
	pFrameRGB.height = codecCtx.height
	pFrameRGB.format = C.AV_PIX_FMT_RGB24

	// 分配帧缓冲区
	if ret := C.av_frame_get_buffer(pFrameRGB, 32); ret < 0 {
		return fmt.Errorf("分配帧缓冲区失败: %d", ret)
	}

	// 创建转换上下文
	swsCtx := C.sws_getContext(
		codecCtx.width,
		codecCtx.height,
		codecCtx.pix_fmt,
		codecCtx.width,
		codecCtx.height,
		C.AV_PIX_FMT_RGB24,
		C.SWS_BILINEAR,
		nil, nil, nil)
	if swsCtx == nil {
		return errors.New("创建转换上下文失败")
	}
	defer C.sws_freeContext(swsCtx)

	// 主处理循环
	pPacket := C.av_packet_alloc()
	defer C.av_packet_free(&pPacket)

	frameCount := 0
	for C.av_read_frame(pFormatCtx, pPacket) >= 0 {
		if int(pPacket.stream_index) != videoStreamIndex {
			C.av_packet_unref(pPacket)
			continue
		}

		if ret := C.avcodec_send_packet(codecCtx, pPacket); ret < 0 {
			C.av_packet_unref(pPacket)
			continue
		}

		for {
			ret := C.avcodec_receive_frame(codecCtx, pFrame)
			if ret == CAVERROR_EAGAIN || ret == CAVERROR_EOF {
				break
			} else if ret < 0 {
				return fmt.Errorf("解码错误: %d", ret)
			}

			// 转换像素格式
			C.sws_scale(swsCtx,
				(**C.uint8_t)(unsafe.Pointer(&pFrame.data[0])),
				(*C.int)(unsafe.Pointer(&pFrame.linesize[0])),
				0,
				codecCtx.height,
				(**C.uint8_t)(unsafe.Pointer(&pFrameRGB.data[0])),
				(*C.int)(unsafe.Pointer(&pFrameRGB.linesize[0])))

			// 保存帧 - 使用带序列模式的文件名
			filename := filepath.Join(outputDir, fmt.Sprintf("frame%03d.jpg", frameCount))
			log.Printf("尝试保存帧 %d 到: %s", frameCount, filename)
			if err := saveFrameAsJPEG(pFrameRGB, int(codecCtx.width), int(codecCtx.height), filename); err != nil {
				return fmt.Errorf("保存帧失败: %w", err)
			}
			frameCount++
		}
		C.av_packet_unref(pPacket)
	}

	return nil
}

func saveFrameAsJPEG(frame *C.AVFrame, width, height int, filename string) error {
	// 初始化输出上下文
	var fmtCtx *C.AVFormatContext
	cFilename := C.CString(filename)
	defer C.free(unsafe.Pointer(cFilename))

	// 强制设置输出格式为image2
	var fmt *C.struct_AVOutputFormat
	fmt = C.av_guess_format(C.CString("image2"), nil, nil)
	if fmt == nil {
		return errors.New("无法猜测image2格式")
	}

	// 正确处理返回值和错误
	if ret := C.avformat_alloc_output_context2(&fmtCtx, fmt, nil, cFilename); ret < 0 {
		errBuf := make([]C.char, 256)
		C.av_strerror(ret, &errBuf[0], C.size_t(len(errBuf)))
		msg := C.GoString(&errBuf[0])
		return errors.New("分配输出上下文失败: " + msg)
	}
	defer C.avformat_free_context(fmtCtx)

	// 查找编码器
	codec := C.avcodec_find_encoder(C.AV_CODEC_ID_MJPEG)
	if codec == nil {
		return errors.New("找不到MJPEG编码器")
	}

	// 创建编码器上下文
	codecCtx := C.avcodec_alloc_context3(codec)
	if codecCtx == nil {
		return errors.New("分配编码器上下文失败")
	}
	defer C.avcodec_free_context(&codecCtx)

	// 配置编码参数 - 使用非弃用的像素格式
	codecCtx.width = C.int(width)
	codecCtx.height = C.int(height)
	codecCtx.pix_fmt = C.AV_PIX_FMT_YUV420P   // 使用非弃用的格式
	codecCtx.color_range = C.AVCOL_RANGE_JPEG // 设置颜色范围为JPEG（全范围）
	codecCtx.time_base.num = 1
	codecCtx.time_base.den = 25

	// 打开编码器
	if ret := C.avcodec_open2(codecCtx, codec, nil); ret < 0 {
		errBuf := make([]C.char, 256)
		C.av_strerror(ret, &errBuf[0], C.size_t(len(errBuf)))

		return errors.New("打开编码器失败:" + C.GoString(&errBuf[0]))
	}

	// 创建输出流
	stream := C.avformat_new_stream(fmtCtx, nil)
	if stream == nil {
		return errors.New("创建输出流失败")
	}
	stream.codecpar.codec_type = C.AVMEDIA_TYPE_VIDEO
	stream.codecpar.codec_id = C.AV_CODEC_ID_MJPEG
	stream.codecpar.width = C.int(width)
	stream.codecpar.height = C.int(height)
	stream.codecpar.format = C.AV_PIX_FMT_YUV420P    // 使用非弃用的格式
	stream.codecpar.color_range = C.AVCOL_RANGE_JPEG // 设置颜色范围

	// 复制编码器参数到流
	if ret := C.avcodec_parameters_from_context(stream.codecpar, codecCtx); ret < 0 {
		errBuf := make([]C.char, 256)
		C.av_strerror(ret, &errBuf[0], C.size_t(len(errBuf)))
		return errors.New("复制编码器参数失败: " + C.GoString(&errBuf[0]))
	}

	// 打开输出文件
	if ret := C.avio_open(&fmtCtx.pb, cFilename, C.AVIO_FLAG_WRITE); ret < 0 {
		errBuf := make([]C.char, 256)
		C.av_strerror(ret, &errBuf[0], C.size_t(len(errBuf)))
		return errors.New("打开输出文件失败: " + C.GoString(&errBuf[0]))
	}
	defer C.avio_closep(&fmtCtx.pb)

	// 写入文件头
	if ret := C.avformat_write_header(fmtCtx, nil); ret < 0 {
		errBuf := make([]C.char, 256)
		C.av_strerror(ret, &errBuf[0], C.size_t(len(errBuf)))
		return errors.New("写入文件头失败: " + C.GoString(&errBuf[0]))
	}

	// 使用新版AVPacket API
	pkt := C.av_packet_alloc()
	defer C.av_packet_free(&pkt)

	// 将Go的int类型转换为C的int类型
	cWidth := C.int(width)
	cHeight := C.int(height)

	// 创建RGB到YUV420P的转换上下文，明确设置颜色范围
	swsCtxJPEG := C.sws_getContext(
		cWidth, cHeight, C.AV_PIX_FMT_RGB24,
		cWidth, cHeight, C.AV_PIX_FMT_YUV420P,
		C.SWS_BILINEAR, nil, nil, nil)
	if swsCtxJPEG == nil {
		return errors.New("无法创建JPEG转换上下文")
	}
	defer C.sws_freeContext(swsCtxJPEG)

	yuvFrame := C.av_frame_alloc()
	if yuvFrame == nil {
		return errors.New("无法分配YUV帧")
	}
	defer C.av_frame_free(&yuvFrame)

	yuvFrame.format = C.AV_PIX_FMT_YUV420P
	yuvFrame.width = cWidth
	yuvFrame.height = cHeight
	yuvFrame.color_range = C.AVCOL_RANGE_JPEG // 设置颜色范围

	if ret := C.av_frame_get_buffer(yuvFrame, 0); ret < 0 {
		errBuf := make([]C.char, 256)
		C.av_strerror(ret, &errBuf[0], C.size_t(len(errBuf)))
		return errors.New("分配YUV帧缓冲区失败: " + C.GoString(&errBuf[0]))
	}

	C.sws_scale(swsCtxJPEG,
		(**C.uint8_t)(unsafe.Pointer(&frame.data[0])),
		(*C.int)(unsafe.Pointer(&frame.linesize[0])),
		0, cHeight,
		(**C.uint8_t)(unsafe.Pointer(&yuvFrame.data[0])),
		(*C.int)(unsafe.Pointer(&yuvFrame.linesize[0])))

	if ret := C.avcodec_send_frame(codecCtx, yuvFrame); ret < 0 {
		errBuf := make([]C.char, 256)
		C.av_strerror(ret, &errBuf[0], C.size_t(len(errBuf)))
		return errors.New("发送帧到编码器失败: " + C.GoString(&errBuf[0]))
	}

	if ret := C.avcodec_receive_packet(codecCtx, pkt); ret < 0 {
		errBuf := make([]C.char, 256)
		C.av_strerror(ret, &errBuf[0], C.size_t(len(errBuf)))
		return errors.New("从编码器接收包失败: " + C.GoString(&errBuf[0]))
	}

	// 设置包的时间戳
	pkt.pts = C.int64_t(0)
	pkt.dts = C.int64_t(0)

	// 写入数据
	if ret := C.av_interleaved_write_frame(fmtCtx, pkt); ret < 0 {
		errBuf := make([]C.char, 256)
		C.av_strerror(ret, &errBuf[0], C.size_t(len(errBuf)))
		return errors.New("写入帧数据失败: " + C.GoString(&errBuf[0]))
	}

	// 写入文件尾
	if ret := C.av_write_trailer(fmtCtx); ret < 0 {
		errBuf := make([]C.char, 256)
		C.av_strerror(ret, &errBuf[0], C.size_t(len(errBuf)))
		return errors.New("写入文件尾失败: " + C.GoString(&errBuf[0]))
	}

	return nil
}

// 将AVFrame转换为Go的image.Image对象
func avFrameToImage(frame *C.AVFrame, width, height int) (image.Image, error) {
	img := image.NewRGBA(image.Rect(0, 0, width, height))
	dataPtr := unsafe.Pointer(frame.data[0])

	for y := 0; y < height; y++ {
		// 计算行偏移
		row := (*[1 << 30]C.uint8_t)(unsafe.Pointer(uintptr(dataPtr) + uintptr(y)*uintptr(frame.linesize[0])))

		for x := 0; x < width; x++ {
			offset := x * 3
			r := row[offset]
			g := row[offset+1]
			b := row[offset+2]
			img.Set(x, y, color.RGBA{uint8(r), uint8(g), uint8(b), 255})
		}
	}
	return img, nil
}
