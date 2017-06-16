package rtsp2rtmp

import (
	//	"flag"
	"fmt"
	"os"
	"time"

	//	"github.com/zhangpeihao/goflv"
	rtmp "github.com/falconray0704/gortmp"
	"github.com/zhangpeihao/log"
)

/*
const (
	programName = "RtmpPublisher"
	version     = "0.0.1"
)

var (
	url         *string = flag.String("URL", "rtmp://video-center.alivecdn.com/AppName/StreamName?vhost=live.gz-app.com", "The rtmp url to connect.")
	streamName  *string = flag.String("Stream", "camstream", "Stream name to play.")
	flvFileName *string = flag.String("FLV", "./v_4097.flv", "FLV file to publishs.")
)
*/

const (
	AUDIO_TAG       = byte(0x08)
	VIDEO_TAG       = byte(0x09)
	SCRIPT_DATA_TAG = byte(0x12)
	DURATION_OFFSET = 53
	HEADER_LEN      = 13
)

type FlvChunk struct {
	TagType   byte
	DataSize  uint32
	Timestamp uint32

	NaluData []byte
}

type RtmpPublisherContext struct {
	rtmpUrl          string
	streamName       string
	obConn           rtmp.OutboundConn
	createStreamChan chan rtmp.OutboundStream
	logger           log.Logger
	stopStreamChan   chan uint32

	flvChan chan FlvChunk
}

type RtmpOutboundConnHandler struct {
	RtmpPublisherCtx RtmpPublisherContext

	videoDataSize int64
	audioDataSize int64
}

//var flvFile *flv.File

var status uint

func (handler *RtmpOutboundConnHandler) OnStatus(conn rtmp.OutboundConn) {
	var err error
	if handler.RtmpPublisherCtx.obConn == nil {
		return
	}
	status, err = handler.RtmpPublisherCtx.obConn.Status()
	fmt.Printf("@@@@@@@@@@@@@status: %d, err: %v\n", status, err)
}

func (handler *RtmpOutboundConnHandler) OnClosed(conn rtmp.Conn) {
	fmt.Printf("@@@@@@@@@@@@@Closed\n")
}

func (handler *RtmpOutboundConnHandler) OnReceived(conn rtmp.Conn, message *rtmp.Message) {
}

func (handler *RtmpOutboundConnHandler) OnReceivedRtmpCommand(conn rtmp.Conn, command *rtmp.Command) {
	fmt.Printf("ReceviedRtmpCommand: %+v\n", command)
}

func (handler *RtmpOutboundConnHandler) OnStreamCreated(conn rtmp.OutboundConn, stream rtmp.OutboundStream) {
	fmt.Printf("Stream created: %d\n", stream.ID())
	handler.RtmpPublisherCtx.createStreamChan <- stream
}

func (handler *RtmpOutboundConnHandler) OnPlayStart(stream rtmp.OutboundStream) {

}

func (handler *RtmpOutboundConnHandler) OnPublishStart(stream rtmp.OutboundStream) {
	// Set chunk buffer size
	go publish(stream, &handler.RtmpPublisherCtx)
}

func publish(stream rtmp.OutboundStream, ctx *RtmpPublisherContext) {
	fmt.Println("1")
	var err error
	fmt.Println("2")
	//defer flvFile.Close()
	startTs := uint32(0)
	startAt := time.Now().UnixNano()
	preTs := uint32(0)
	fmt.Println("3")
	for status == rtmp.OUTBOUND_CONN_STATUS_CREATE_STREAM_OK {
		/*
			if flvFile.IsFinished() {
				fmt.Println("@@@@@@@@@@@@@@File finished")
				flvFile.LoopBack()
				startAt = time.Now().UnixNano()
				startTs = uint32(0)
				preTs = uint32(0)
			}
			header, data, err := flvFile.ReadTag()
			if err != nil {
				fmt.Println("flvFile.ReadTag() error:", err)
				break
			}
			switch header.TagType {
			case flv.VIDEO_TAG:
				videoDataSize += int64(len(data))
			case flv.AUDIO_TAG:
				audioDataSize += int64(len(data))
			}
		*/
		var flvChunk FlvChunk
		select {
		case flvChunk = <-ctx.flvChan:
		}

		if startTs == uint32(0) {
			startTs = flvChunk.Timestamp
		}
		diff1 := uint32(0)
		//		deltaTs := uint32(0)
		if flvChunk.Timestamp > startTs {
			diff1 = flvChunk.Timestamp - startTs
		} else {
			//fmt.Printf("@@@@@@@@@@@@@@diff1 header(%+v), startTs: %d\n", flvChunk, startTs)
			fmt.Printf("@@@@@@@@@@@@@@diff1 startTs: %d\n", startTs)
		}
		if diff1 > preTs {
			//			deltaTs = diff1 - preTs
			preTs = diff1
		}
		//fmt.Printf("@@@@@@@@@@@@@@diff1 header(%+v), startTs: %d\n", flvChunk, startTs)
		fmt.Printf("@@@@@@@@@@@@@@diff1 startTs: %d\n", startTs)
		if err = stream.PublishData(flvChunk.TagType, flvChunk.NaluData, diff1); err != nil {
			fmt.Println("PublishData() error:", err)
			break
		}
		diff2 := uint32((time.Now().UnixNano() - startAt) / 1000000)
		//		fmt.Printf("diff1: %d, diff2: %d\n", diff1, diff2)
		if diff1 > diff2+100 {
			//			fmt.Printf("header.Timestamp: %d, now: %d\n", header.Timestamp, time.Now().UnixNano())
			time.Sleep(time.Millisecond * time.Duration(diff1-diff2))
		}
	}
}

func InitRtmpHandler(handler *RtmpOutboundConnHandler, streamName string, rtmpUrl string, inFlvChan chan FlvChunk) {

	handler.RtmpPublisherCtx.streamName = streamName
	handler.RtmpPublisherCtx.rtmpUrl = rtmpUrl
	handler.RtmpPublisherCtx.flvChan = inFlvChan

	handler.RtmpPublisherCtx.logger = *(log.NewLogger(".", "publisher", nil, 60, 3600*24, true))
	rtmp.InitLogger(&handler.RtmpPublisherCtx.logger)
	//defer ctx.logger.Close()
	handler.RtmpPublisherCtx.createStreamChan = make(chan rtmp.OutboundStream)
	handler.RtmpPublisherCtx.stopStreamChan = make(chan uint32)
}

func runPublish(handler *RtmpOutboundConnHandler) {

	for {
		select {
		case stream := <-handler.RtmpPublisherCtx.createStreamChan:
			// Publish
			stream.Attach(handler)
			err := stream.Publish(handler.RtmpPublisherCtx.streamName, "live")
			if err != nil {
				fmt.Printf("Publish error: %s", err.Error())
				os.Exit(-1)
			}
		case stop := <-handler.RtmpPublisherCtx.stopStreamChan:
			if stop == uint32(0x01) {
				break
			}

		case <-time.After(1 * time.Second):
			fmt.Printf("Audio size: %d bytes; Vedio size: %d bytes\n", handler.audioDataSize, handler.videoDataSize)
		}
	}

}

func StartPublish(handler *RtmpOutboundConnHandler) {

	//rtmpHandler := &RtmpOutboundConnHandler{ RtmpPublisherCtx : ctx}

	fmt.Println("to dial")
	fmt.Println("a")
	var err error
	handler.RtmpPublisherCtx.obConn, err = rtmp.Dial(handler.RtmpPublisherCtx.rtmpUrl, handler, 100)
	if err != nil {
		fmt.Println("Dial error", err)
		os.Exit(-1)
	}
	fmt.Println("b")
	//defer obConn.Close()
	fmt.Println("to connect")
	err = handler.RtmpPublisherCtx.obConn.Connect()
	if err != nil {
		fmt.Printf("Connect error: %s", err.Error())
		os.Exit(-1)
	}
	fmt.Println("c")

	go runPublish(handler)
}
