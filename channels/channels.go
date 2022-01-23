package channels

import (
	"encoding/json"
	"fmt"
	"os"
	"regexp"
	"strconv"
	"strings"
	"time"
	"www.seawise.com/client/core"
	"www.seawise.com/client/log"
)

type Channels struct {
	Array       []*Channel
	attempts    int
	Started     bool
	timer       *time.Ticker
	StopChannel chan string
}

func Create(attempts int) (*Channels, error) {
	chs := &Channels{
		attempts:    attempts,
		StopChannel: make(chan string),
	}

	err := chs.DetectCameras()
	if err != nil {
		return nil, fmt.Errorf("failed to detect cameras: %v", err)
	}

	return chs, nil
}

func (c *Channels) getVids() ([]int, error) {
	vids := make([]int, 0)
	if core.Config.VidsString != "" {
		err := json.Unmarshal([]byte(core.Config.VidsString), &vids)
		if err != nil {
			return nil, fmt.Errorf("failed to parse confiuration of vids: %v", err)
		}
		return vids, nil
	}

	devs, err := os.ReadDir("/dev")
	if err != nil {
		return nil, fmt.Errorf("failed to read dir /dev: %v", err)
	}
	re := regexp.MustCompile("[0-9]+")

	for _, vid := range devs {
		if strings.Contains(vid.Name(), "video") {
			log.V5(vid.Name())
			vidNum, err := strconv.Atoi(re.FindAllString(vid.Name(), -1)[0])
			if err != nil {
				return nil, fmt.Errorf("failed to convert video filename to int: %v", err)
			}
			vids = append(vids, vidNum)
		}
	}

	log.V5(fmt.Sprintf("Done checking vid - %v", vids))
	return vids, nil
}

func (c *Channels) DetectCameras() error {
	vids, err := c.getVids()
	if err != nil {
		return fmt.Errorf("failed to get vids: %v", err)
	}

	i := 0
	for i < c.attempts {
		log.V5(fmt.Sprintf("Attempting to start channel - %v / %v", i, c.attempts))
		for _, num := range vids {
			channel := CreateChannel(num)
			err := channel.Init()
			if err != nil {
				continue
			} else {
				c.Array = append(c.Array, channel)
			}
		}

		if len(c.Array) > 0 {
			i = 99
		}
		i++
	}

	log.V5(fmt.Sprintf("Initiated all channels - %v", c.Array))
	return nil
}

func (c *Channels) Start() {
	if !c.Started {
		c.Started = true
		c.timer = time.NewTicker(200 * time.Millisecond)

		for c.Started {
			select {
			case code := <-c.StopChannel:
				c.Stop(code)
			case <-c.timer.C:
				c.Stream()
			}
		}
	}
}

func (c *Channels) Stream() {
	for _, channel := range c.Array {
		channel.Read()
		go channel.EncodeImage()
	}
}

func (c *Channels) Stop(code string) {
	log.V5("STOP - %v", code)
	c.Started = false
}

func (c *Channels) Close() {
	for i, channel := range c.Array {
		err := channel.close()
		if err != nil {
			log.Warn(fmt.Sprintf("failed to close channel %v: %v", i, err))
		}
	}
	c.Array = c.Array[:0]
	c.Started = false
}
