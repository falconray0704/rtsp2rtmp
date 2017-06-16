package main

import (
	"flag"
	"log"

	"github.com/falconray0704/rtsp2rtmp"
	//	"github.com/nareix/mp4"
	//	mpegts "github.com/nareix/ts"
)

var (
	syncCount int64
	naluCount int64
)

type GobAllSamples struct {
	TimeScale int
	SPS       []byte
	PPS       []byte
	Samples   []GobSample
}

type GobSample struct {
	Duration int
	Data     []byte
	Sync     bool
}

func handleNALU(nalType byte, payload []byte, ts int64, outFlvChan chan rtsp2rtmp.FlvChunk) {

	var flvChk rtsp2rtmp.FlvChunk
	flvChk.DataSize = uint32(len(payload))
	flvChk.Timestamp = uint32(ts)
	flvChk.NaluData = []byte{}
	flvChk.NaluData = payload

	if nalType == 7 {
		/*
			if len(sps) == 0 {
				sps = payload
			}
		*/
		log.Println("nalType:%x", nalType)
	} else if nalType == 8 {
		/*
			if len(pps) == 0 {
				pps = payload
			}
		*/
		log.Println("nalType:%x", nalType)
	} else if nalType == 5 {
		// keyframe
		syncCount++
		/*
			if syncCount == maxgop {
				quit = true
			}
		*/
		log.Println("=== naluCount:%d get I frame ===", naluCount)
		naluCount++
		//writeNALU(true, int(ts), payload)
		flvChk.TagType = rtsp2rtmp.VIDEO_TAG
		outFlvChan <- flvChk
	} else {
		// non-keyframe
		if syncCount > 0 {
			if syncCount%30 == 0 {
				log.Println("=== naluCount:%d get non-I frame ===", naluCount)
			}
			naluCount++
			//writeNALU(false, int(ts), payload)
			flvChk.TagType = rtsp2rtmp.VIDEO_TAG
			outFlvChan <- flvChk
		}
	}
}

func main() {
	//var saveGob bool
	var rtspUrl string
	var streamName string
	var rtmpUrl string
	//var maxgop int

	var inFlvChan chan rtsp2rtmp.FlvChunk
	var rtmpHandler rtsp2rtmp.RtmpOutboundConnHandler

	// with aac rtsp://admin:123456@80.254.21.110:554/mpeg4cif
	// with aac rtsp://admin:123456@95.31.251.50:5050/mpeg4cif
	// 1808p rtsp://admin:123456@171.25.235.18/mpeg4
	// 640x360 rtsp://admin:123456@94.242.52.34:5543/mpeg4cif

	//flag.BoolVar(&saveGob, "s", false, "save to gob file")
	//flag.IntVar(&maxgop, "g", 10, "max gop recording")
	flag.StringVar(&rtspUrl, "rtspUrl", "rtsp://admin:123456@10.1.51.13/H264?ch=1&subtype=0", "")
	flag.StringVar(&rtmpUrl, "rtmpUrl", "rtmp://10.1.51.20:1935/myapp/", "")
	flag.StringVar(&streamName, "streamName", "cv", "")
	flag.Parse()

	RtspReader := rtsp2rtmp.RtspClientNew()

	inFlvChan = make(chan rtsp2rtmp.FlvChunk, 30*10)
	rtsp2rtmp.InitRtmpHandler(&rtmpHandler, streamName, rtmpUrl, inFlvChan)
	rtsp2rtmp.StartPublish(&rtmpHandler)

	quit := false

	//	sps := []byte{}
	//	pps := []byte{}
	fuBuffer := []byte{}
	syncCount = 0

	// rtp timestamp: 90 kHz clock rate
	// 1 sec = timestamp 90000
	//timeScale := 90000

	type NALU struct {
		ts   int
		data []byte
		sync bool
	}
	//var lastNALU *NALU

	//	var mp4w *mp4.SimpleH264Writer
	//	var tsw *mpegts.SimpleH264Writer

	//var allSamples *GobAllSamples

	if status, message := RtspReader.Client(rtspUrl); status {
		log.Println("connected")
		i := 0
		for {
			i++
			//read 100 frame and exit loop
			if quit {
				break
			}
			select {
			case data := <-RtspReader.OutGoing:

				/*
					if true {
						log.Printf("packet [0]=%x type=%d\n", data[0], data[1])
					}
				*/

				//log.Println("packet recive")
				if data[0] == 36 && data[1] == 0 {
					cc := data[4] & 0xF
					//rtp header
					rtphdr := 12 + cc*4

					//packet time
					ts := (int64(data[8]) << 24) + (int64(data[9]) << 16) + (int64(data[10]) << 8) + (int64(data[11]))

					//packet number
					packno := (int64(data[6]) << 8) + int64(data[7])
					if false {
						log.Println("packet num", packno)
					}

					nalType := data[4+rtphdr] & 0x1F

					if nalType >= 1 && nalType <= 23 {
						handleNALU(nalType, data[4+rtphdr:], ts, inFlvChan)
					} else if nalType == 28 {
						isStart := data[4+rtphdr+1]&0x80 != 0
						isEnd := data[4+rtphdr+1]&0x40 != 0
						nalType := data[4+rtphdr+1] & 0x1F
						//nri := (data[4+rtphdr+1]&0x60)>>5
						nal := data[4+rtphdr]&0xE0 | data[4+rtphdr+1]&0x1F
						if isStart {
							fuBuffer = []byte{0}
						}
						fuBuffer = append(fuBuffer, data[4+rtphdr+2:]...)
						if isEnd {
							fuBuffer[0] = nal
							handleNALU(nalType, fuBuffer, ts, inFlvChan)
						}
					}

				} else if data[0] == 36 && data[1] == 2 {
					// audio

					//cc := data[4] & 0xF
					//rtphdr := 12 + cc*4
					//or not payload := data[4+rtphdr:]
					//payload := data[4+rtphdr+4:]
					//outfileAAC.Write(payload)
					//log.Print("audio payload\n", hex.Dump(payload))
				}

			case <-RtspReader.Signals:
				log.Println("exit signal by class rtsp")
			}
		}
	} else {
		log.Println("error", message)
	}

	RtspReader.Close()
}
