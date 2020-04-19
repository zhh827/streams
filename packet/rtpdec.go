package packet

import (
	"bytes"
	"github.com/MeloQi/rtp"
	"github.com/gansidui/priority_queue"
	"os"

	"github.com/32bitkid/bitreader"
)

type RtpParsePacket struct {
	psDenc            *DecPSPackage
	psFrames          map[uint32]*priority_queue.PriorityQueue //一个时间戳一个队列
	psPkg             []byte
	psPkgLen          int
	TimestampRTPCur   uint32
	DebugSavePsFile   *os.File
	DebugSaveH264File *os.File
}

func NewRtpParsePacket() *RtpParsePacket {
	return &RtpParsePacket{
		psDenc: &DecPSPackage{
			rawData: make([]byte, MAXFrameLen),
			rawLen:  0,
		},
		psFrames: make(map[uint32]*priority_queue.PriorityQueue),
		psPkg:    make([]byte, MAXFrameLen),
		psPkgLen: 0,
	}
}

func (r *RtpParsePacket) Close() {
	if r.DebugSavePsFile != nil {
		r.DebugSavePsFile.Close()
	}
	if r.DebugSaveH264File != nil {
		r.DebugSaveH264File.Close()
	}
}

// data包含 接受到完整一帧Ps数据后，所有的payload, 解析出去后是一阵完整的raw数据
func (r *RtpParsePacket) ReadPsFrame(data []byte) ([]byte, error) {

	// add the MPEG Program end code
	data = append(data, 0x00, 0x00, 0x01, 0xb9)
	br := bitreader.NewReader(bytes.NewReader(data))
	r.psDenc.reSet()

	if r.psDenc != nil {
		return r.psDenc.decPackHeader(br)
	}

	return nil, nil
}

type Node struct {
	priority int
	value    *rtp.RTPHeaderInfo
}

func (this *Node) Less(other interface{}) bool {
	return this.priority < other.(*Node).priority
}

// data包含 接受RTP包， 解析出去后是一阵完整的raw数据
func (r *RtpParsePacket) ReadRtp(data []byte) ([]byte, error) {
	rtpPkg := rtp.ParseRTPHeader(data)
	if rtpPkg == nil || rtpPkg.Payload == nil || len(rtpPkg.Payload) == 0 {
		return nil, ErrParseRtp
	}

	//rtp包暂存到队列
	if _, ok := r.psFrames[rtpPkg.Timestamp]; !ok {
		r.psFrames[rtpPkg.Timestamp] = priority_queue.New()
	}
	r.psFrames[rtpPkg.Timestamp].Push(&Node{priority: int(rtpPkg.Cseq), value: rtpPkg})

	//rtp,组合一个PS帧
	if len(r.psFrames) < 3 {
		return nil, nil
	}
	var q *priority_queue.PriorityQueue
	var timestamp int64 = -1
	for k, v := range r.psFrames {
		if timestamp < 0 || int64(k) < timestamp {
			timestamp = int64(k)
			q = v
		}
	}

	isVideo := true
	for q.Len() > 0 {
		rtpPkg = q.Top().(*Node).value

		//不要音频，过滤掉只有pes没有psh的音频包
		if len(rtpPkg.Payload) > 4 && rtpPkg.Payload[0] == 0 && rtpPkg.Payload[1] == 0 && rtpPkg.Payload[2] == 1 && rtpPkg.Payload[3] == 0xe0 { //视频
			if rtpPkg.Payload[3] == 0xe0 || rtpPkg.Payload[3] == 0xba {
				isVideo = true
			} else if rtpPkg.Payload[3] == 0xc0 {
				isVideo = false
			}

		}
		if !isVideo {
			q.Pop()
			continue
		}

		//帧开头和结尾
		if r.psPkgLen == 0 && len(rtpPkg.Payload) > 4 { //起始非ps header，跳过
			if !(rtpPkg.Payload[0] == 0 && rtpPkg.Payload[1] == 0 && rtpPkg.Payload[2] == 1 && rtpPkg.Payload[3] == 0xba) {
				q.Pop()
				continue
			}
		}
		if r.psPkgLen != 0 && len(rtpPkg.Payload) > 4 { //遇到另外ps header，本帧结束
			if rtpPkg.Payload[0] == 0 && rtpPkg.Payload[1] == 0 && rtpPkg.Payload[2] == 1 && rtpPkg.Payload[3] == 0xba {
				break
			}
		}

		//组装一帧
		if r.psPkgLen+len(rtpPkg.Payload) > len(r.psPkg) {
			r.psPkgLen = 0
			continue
		}
		copy(r.psPkg[r.psPkgLen:], rtpPkg.Payload)
		r.psPkgLen += len(rtpPkg.Payload)
		q.Pop()
	}
	if q.Len() == 0 {
		delete(r.psFrames, uint32(timestamp))
	}

	//从ps解析出原始帧
	frame := r.psPkg[:r.psPkgLen]
	if r.DebugSavePsFile != nil {
		r.DebugSavePsFile.Write(frame)
	}

	r.psPkgLen = 0
	r.TimestampRTPCur = rtpPkg.Timestamp

	raw, err := r.ReadPsFrame(frame)
	if r.DebugSaveH264File != nil && err == nil && raw != nil {
		r.DebugSaveH264File.Write(raw)
	}

	return raw, err
}
