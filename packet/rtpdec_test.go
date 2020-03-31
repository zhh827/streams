package packet

import (
	"github.com/MeloQi/rtp"
	"github.com/max-min/streams/packet"
	"io"
	"os"
	"testing"
)

var dec *RtpParsePacket = NewRtpParsePacket()
var outfile *os.File
var rtpTransfer *rtp.RtpTransfer = rtp.NewRRtpTransfer()

func TestRtpPsDec(t *testing.T) {
	infile, err := os.Open("D:/workspace/video/test.ps")
	if err != nil {
		t.Error(err)
		t.Fail()
		return
	}
	defer infile.Close()
	outfile, err = os.Create("D:/workspace/video/test.h264")
	if err != nil {
		t.Error(err)
		t.Fail()
		return
	}
	defer outfile.Close()

	buf := make([]byte, packet.MAXFrameLen)
	bufLen := 0
	timestamp := 0
	for {
		num, err := infile.Read(buf[bufLen:])
		if err != nil {
			if err != io.EOF {
				t.Error(err)
				t.Fail()
			}
			break
		}
		bufLen += num

		frameEnd := 0
		for frameEnd < bufLen-4 {
			if !(buf[0] == 0 && buf[1] == 0 && buf[2] == 1 && buf[3] == 0xba) {
				t.Fail()
				return
			}
			frameEnd++
			if buf[frameEnd] == 0 && buf[frameEnd+1] == 0 && buf[frameEnd+2] == 1 {
				if buf[frameEnd+3] == 0xba {
					break
				}
			}
		}

		frame := make([]byte, frameEnd)
		copy(frame, buf[:frameEnd])
		rtpTransfer.PkgRtpOut(frame, rtp.RTP_TYPE_VIDEO, false, 96, false, uint32(timestamp), 000, rtpDataCallback)
		timestamp += 30
		copy(buf[:bufLen-frameEnd], buf[frameEnd:bufLen])
		bufLen -= frameEnd
	}
}


func rtpDataCallback(pack *rtp.RTPPack) {
	rtpPkg := make([]byte, pack.Buffer.Len())
	copy(rtpPkg, pack.Buffer.Bytes())
	raw, _ := dec.ReadRtp(rtpPkg)
	outfile.Write(raw)

}
