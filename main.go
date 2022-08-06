package main

import (
	"bytes"
	"fmt"
	"io/ioutil"

	"github.com/MeloQi/streams/packet"
	"github.com/nareix/joy4/codec/h264parser"
	"github.com/nareix/joy4/format"
)

func init() {
	format.RegisterAll()
}

// func main() {

// 	rtp := packet.NewRRtpTransfer("", packet.LocalCache)

// 	// send ip,port and recv ip,port
// 	rtp.Service("127.0.0.1", "172.20.25.2", 10086, 10087)

// 	f, err := avutil.Open("bukong.mp4")
// 	if err != nil {
// 		fmt.Printf("read file error(%v)", err)
// 		rtp.Exit()
// 		return
// 	}
// 	header := []byte{0x00, 0x00, 0x00, 0x01}
// 	// 获取 视频流ID sps pps
// 	streams, _ := f.Streams()
// 	var sps, pps []byte
// 	var vindex int8
// 	for i, stream := range streams {
// 		if stream.Type() == av.H264 {
// 			vindex = int8(i)
// 			info := stream.(h264parser.CodecData)
// 			sps = append(header, info.SPS()...)
// 			pps = append(header, info.PPS()...)
// 			fmt.Println("-------------- file info --------------")
// 			fmt.Printf("- type: %s, Heigh: %d, Width: %d \n", info.Type(), info.Height(), info.Width())
// 			fmt.Printf("- sps: %x \n", sps)
// 			fmt.Printf("- pps: %x \n", pps)
// 			fmt.Println("---------------------------------------")
// 			break
// 		}
// 	}
// 	// 处理视频流
// 	var pts uint64 = 0
// 	var incremental = 90000 / 30 // 时基 90k， fps=25

// 	for i := 0; i < 10000; i++ {
// 		var pkt av.Packet
// 		var err error
// 		if pkt, err = f.ReadPacket(); err != nil {
// 			fmt.Println("read packet error: ", err)
// 			goto STOP
// 		}
// 		if pkt.Idx != vindex {
// 			continue
// 		}

// 		nalus, _ := h264parser.SplitNALUs(pkt.Data)
// 		// 1个packt 中可能存在多个nalu
// 		var data []byte
// 		for _, nalu := range nalus {
// 			data = append(header, nalu...)
// 		}
// 		fmt.Printf("data: %x\n", data[:8])
// 		// 关键帧加 sps pps
// 		if pkt.IsKeyFrame {
// 			data = append(pps, data...)
// 			data = append(sps, data...)
// 		}
// 		rtp.Send2data(data, pkt.IsKeyFrame, pts)
// 		pts += uint64(incremental)
// 		//time.Sleep(time.Millisecond * 40)
// 	}
// STOP:
// 	f.Close()
// 	rtp.Exit()
// 	return

// }

func main() {
	// ffmpeg  hevc 转 h264  /data/easydarwin/ffmpeg -i bukong.mp4 -map 0 -c:a copy -c:s copy -c:v libx264 output.mp4
	// MP4文件提取annex-b h264裸流  /data/easydarwin/ffmpeg -i 1.mp4 -codec copy -bsf: h264_mp4toannexb -f h264 1.h264
	// https://blog.csdn.net/twoconk/article/details/52217493

	rtp := packet.NewRRtpTransfer("", packet.LocalCache)
	// send ip,port and recv ip,port
	rtp.Service("127.0.0.1", "172.20.25.2", 10086, 10087)

	// rtp.Send2data([]byte{0xff, 0xff, 0xff, 0xff, 0xff, 0xff}, true, 1000)
	buf, err := ioutil.ReadFile("1.h264")
	if err != nil {
		fmt.Println("read file err")
		rtp.Exit()
		return
	}
	nalus, _ := h264parser.SplitNALUs(buf)

	var pts uint64 = 0
	var incremental = 90000 / 30 // 时基 90k， fps=25
	var sps, pps []byte
	for _, nalu := range nalus {
		header := []byte{0x00, 0x00, 0x00, 0x01}
		// 获取sps pps与关键帧 一起发送
		if bytes.Equal(nalu[:1], []byte{0x67}) {
			sps = append(header, nalu...)
			continue
		}
		if bytes.Equal(nalu[:1], []byte{0x68}) {
			pps = append(header, nalu...)
			continue
		}
		if len(sps) == 0 || len(pps) == 0 {
			continue
		}
		// 添加h264 annex-b nalu header
		data := append(header, nalu...)
		// fmt.Printf("sps: %x, pps: %x\n", sps, pps)
		// 判断是否为关键帧
		if bytes.Equal(nalu[:1], []byte{0x65}) {
			iFrameNalu := append(sps, pps...)
			iFrameNalu = append(iFrameNalu, data...)
			fmt.Printf("I: %x\n", iFrameNalu[:52])
			rtp.Send2data(iFrameNalu, true, pts)
		} else {
			// fmt.Printf("noI: %x\n", data[:16])
			rtp.Send2data(data, false, pts)
		}
		pts += uint64(incremental)
	}
	rtp.Exit()
}
