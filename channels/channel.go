package channels

import (
	"bytes"
	"fmt"
	"gocv.io/x/gocv"
	"image/jpeg"
	"www.seawise.com/client/log"
)

type Channel struct {
	fps     int
	name    int
	init    bool
	capture *gocv.VideoCapture
	image   gocv.Mat
	Queue   *chan []byte
}

func CreateChannel(channelName int) *Channel {
	q := make(chan []byte)
	channel := &Channel{
		name:  channelName,
		Queue: &q,
	}

	return channel
}

func (c *Channel) Init() error {
	vc, err := gocv.OpenVideoCapture(c.name)
	if err != nil {
		return fmt.Errorf("Init failed to capture video %v: ", err)
	}

	//vc.Set(gocv.VideoCaptureFPS, 10)
	vc.Set(gocv.VideoCaptureFrameWidth, 1280)
	vc.Set(gocv.VideoCaptureFrameHeight, 720)
	vc.Set(gocv.VideoCaptureBufferSize, 1)

	img := gocv.NewMat()

	ok := vc.Read(&img)
	if !ok {
		return fmt.Errorf("Init failed to read")
	}

	c.capture = vc
	c.image = img
	c.init = true
	return nil
}

func (c *Channel) Read() {
	err := c.getImage()
	if err != nil {
		log.Warn(fmt.Sprintf("failed to read image: %v", err))
		return
	}

	c.EncodeImage()
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

func (c *Channel) EncodeImage() {
	const jpegQuality = 50

	jpegOption := &jpeg.Options{Quality: jpegQuality}

	img, err := c.image.ToImage()
	if err != nil {
		log.Warn(fmt.Sprintf("Failed to change to image: %v", err))
		return
	}

	buf := new(bytes.Buffer)
	err = jpeg.Encode(buf, img, jpegOption)
	if err != nil {
		log.Warn(fmt.Sprintf("Failed to encode image: %v", err))
		return
	}

	*c.Queue <- buf.Bytes()
}

func (c *Channel) close() error {
	err := c.capture.Close()
	if err != nil {
		return fmt.Errorf("failed to close video channel: %v", err)
	}
	err = c.image.Close()
	if err != nil {
		return fmt.Errorf("failed to close image: %v", err)
	}
	close(*c.Queue)
	return nil
}
