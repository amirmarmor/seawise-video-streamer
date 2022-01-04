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
	counter  int
	Array    []*Channel
	attempts int
	Started  bool
	Active   int
	ticker   *time.Ticker
}

func Create(attempts int) (*Channels, error) {
	chs := &Channels{
		attempts: attempts,
		ticker:   time.NewTicker(10 * time.Second),
		Active:   0,
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

func (c *Channels) Start(loop int, channel int) {
	if core.Config.Parallel {
		c.startParallel(channel)
	} else {
		c.startRotating(loop)
	}
}

func (c *Channels) startParallel(channel int) {
	go c.Array[channel].Start()
}

func (c *Channels) startRotating(loop int) {
	if !c.Started {
		go c.Array[c.Active].Start()
		c.Started = true
	}

	if loop == 0 {
		loop = core.DefaultLoop
	}

	c.ticker = time.NewTicker(time.Duration(loop) * time.Second)

	log.V5(fmt.Sprintf("Starting with loop interval of %v", loop))
	for {
		select {
		case <-c.ticker.C:
			c.Switch()
		}
	}
}

func (c *Channels) Switch() {
	if len(c.Array) <= 1 {
		return
	}
	log.V5(fmt.Sprintf("Stopping Channel %v", c.Active))
	c.Array[c.Active].StopChannel <- "stop"
	c.Active += 1
	if c.Active >= len(c.Array) {
		c.Active = 0
	}
	go c.Array[c.Active].Start()
	log.V5(fmt.Sprintf("Starting Channel %v", c.Active))
}

func (c *Channels) Stop(num int) {
	if c.Array[num].started {
		c.Array[num].StopChannel <- "stop"
		log.V5(fmt.Sprintf("Stopped channel - %v", num))
	}
}
