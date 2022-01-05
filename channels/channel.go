package channels

import (
	"bytes"
	"fmt"
	"gocv.io/x/gocv"
	"image/jpeg"
	"time"
	"www.seawise.com/client/log"
)

type Channel struct {
	started     bool
	fps         int
	name        int
	init        bool
	capture     *gocv.VideoCapture
	image       gocv.Mat
	StopChannel chan string
	Queue       *chan []byte
	ticker      *time.Ticker
}

type Recording struct {
	isRecording bool
	startTime   time.Time
}

func CreateChannel(channelName int) *Channel {
	q := make(chan []byte)
	channel := &Channel{
		name:        channelName,
		StopChannel: make(chan string),
		Queue:       &q,
	}

	return channel
}

func (c *Channel) Init() error {
	vc, err := gocv.OpenVideoCapture(c.name)
	if err != nil {
		return fmt.Errorf("Init failed to capture video %v: ", err)
	}
	//defer vc.Close()

	//vc.Set(gocv.VideoCaptureFPS, 10)
	vc.Set(gocv.VideoCaptureFrameWidth, 1920)
	vc.Set(gocv.VideoCaptureFrameHeight, 1024)
	vc.Set(gocv.VideoCaptureBufferSize, 10)

	img := gocv.NewMat()
	//defer img.Close()

	ok := vc.Read(&img)
	if !ok {
		return fmt.Errorf("Init failed to read")
	}

	c.capture = vc
	c.image = img
	c.init = true
	return nil
}

func (c *Channel) stop() {
	c.ticker.Stop()
	c.started = false
	log.V5("stopped....")
}

func (c *Channel) Start() {
	if !c.started {
		c.started = true

		c.ticker = time.NewTicker(50 * time.Millisecond)
		for c.init {
			select {
			case <-c.StopChannel:
				c.stop()
			case <-c.ticker.C:
				c.Read()
			}
		}
	}
}

func (c *Channel) Read() {
	err := c.getImage()
	if err != nil {
		log.Warn(fmt.Sprintf("failed to read image: %v", err))
		//return
	}

	c.encodeImage()
}

func (c *Channel) getImage() error {
	ok := c.capture.Read(&c.image)
	if !ok {
		return fmt.Errorf("read encountered channel closed %v\n", c.name)
	}

	if c.image.Empty() {
		return fmt.Errorf("Empty Image")
	}

	return nil
}

func (c *Channel) encodeImage() {
	const jpegQuality = 50

	jpegOption := &jpeg.Options{Quality: jpegQuality}

	image, err := c.image.ToImage()
	if err != nil {
		log.Warn(fmt.Sprintf("Failed to change to image: %v", err))
	}

	buf := new(bytes.Buffer)
	err = jpeg.Encode(buf, image, jpegOption)
	if err != nil {
		log.Warn(fmt.Sprintf("Failed to encode image: %v", err))
	}

	*c.Queue <- buf.Bytes()
}
