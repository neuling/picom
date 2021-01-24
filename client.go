package main

import (
	"fmt"
	"net"
	"os"
	"crypto/tls"
	"os/signal"
	"syscall"
	"time"
	"flag"
	_ "log"
	"encoding/binary"
	"github.com/neuling/gumble/gumble"
	"github.com/neuling/gumble/gumbleutil"
	"github.com/neuling/gumble/gumbleopenal"
	"github.com/dchote/gpio"
	"github.com/stianeikeland/go-rpio"
	"github.com/dchote/go-openal/openal"
	"github.com/neuling/volume-go"
	_ "github.com/neuling/gumble/opus"
)

type Intercom struct {
	Stream *gumbleopenal.Stream
	Client *gumble.Client

	Volume int

	GPIOEnabled     bool
	Button  gpio.Pin
	ButtonState     uint
	IsTransmitting bool
}


func main() {
	fmt.Println("Hello, world.")

	// server := flag.String("server", "localhost:64738", "the server to connect to")
	username := flag.String("username", "", "the username of the client")
	password := flag.String("password", "", "the password of the server")

	flag.Parse()

	tlsConfig := &tls.Config{ InsecureSkipVerify: true }

	gumbleConfig := gumble.NewConfig()

	gumbleConfig.Attach(gumbleutil.AutoBitrate)

	gumbleConfig.Username = *username
	gumbleConfig.Password = *password

	intercom := Intercom{}

	v, _ := volume.GetVolume()
	intercom.Volume = v

	gumbleConfig.Attach(&intercom)
	gumbleConfig.AttachAudio(&intercom)

	client, _ := gumble.DialWithDialer(new(net.Dialer), "moritz.pro:64738", gumbleConfig, tlsConfig)

	if stream, err := gumbleopenal.New(intercom.Client); err != nil {
		fmt.Fprintf(os.Stderr, "%s\n", err)
		os.Exit(1)
	} else {
		intercom.Stream = stream
	}

	intercom.initGPIO()

	sigs := make(chan os.Signal, 1)
	signal.Notify(sigs, syscall.SIGINT, syscall.SIGTERM)

	<-sigs

	client.Disconnect()

	fmt.Println("Bye Bye!")
}

func (i *Intercom) OnConnect(e *gumble.ConnectEvent) {
	fmt.Println("Connected …")
	i.Client = e.Client
}

func (i *Intercom) initGPIO() {
	// we need to pull in rpio to pullup our button pin
	if err := rpio.Open(); err != nil {
		fmt.Println(err)
		i.GPIOEnabled = false
		return
	} else {
		i.GPIOEnabled = true
	}

	ButtonPinPullUp := rpio.Pin(17)
	ButtonPinPullUp.PullUp()

	rpio.Close()

	// unfortunately the gpio watcher stuff doesnt work for me in this context, so we have to poll the button instead
	i.Button = gpio.NewInput(17)
	go func() {
		for {
			currentState, err := i.Button.Read()

			if currentState != i.ButtonState && err == nil {
				i.ButtonState = currentState

				if i.Stream != nil {
					if i.ButtonState == 1 {
						fmt.Printf("Button is released\n")
						volume.SetVolume(i.Volume)
						i.Stream.StopSource()
					} else {
						fmt.Printf("Button is pressed\n")
						v, _ := volume.GetVolume()
						i.Volume = v
						volume.SetVolume(60)
						i.Stream.StartSource()
					}
				}

			}

			time.Sleep(100 * time.Millisecond)
		}
	}()
}

func (i *Intercom) OnDisconnect(e *gumble.DisconnectEvent) {
}

func (i *Intercom) OnTextMessage(e *gumble.TextMessageEvent) {
}

func (i *Intercom) OnUserChange(e *gumble.UserChangeEvent) {
}

func (i *Intercom) OnChannelChange(e *gumble.ChannelChangeEvent) {
}

func (i *Intercom) OnPermissionDenied(e *gumble.PermissionDeniedEvent) {
}

func (i *Intercom) OnUserList(e *gumble.UserListEvent) {
}

func (i *Intercom) OnACL(e *gumble.ACLEvent) {
}

func (i *Intercom) OnBanList(e *gumble.BanListEvent) {
}

func (i *Intercom) OnContextActionChange(e *gumble.ContextActionChangeEvent) {
}

func (i *Intercom) OnServerConfig(e *gumble.ServerConfigEvent) {
}

func (i *Intercom) OnAudioStream(e *gumble.AudioStreamEvent) {
	fmt.Println("asads …")

	go func() {
		source := openal.NewSource()
		emptyBufs := openal.NewBuffers(8)
		reclaim := func() {
			if n := source.BuffersProcessed(); n > 0 {
				reclaimedBufs := make(openal.Buffers, n)
				source.UnqueueBuffers(reclaimedBufs)
				emptyBufs = append(emptyBufs, reclaimedBufs...)
			}
		}
		var raw [gumble.AudioMaximumFrameSize * 2]byte
		for packet := range e.C {
			samples := len(packet.AudioBuffer)
			if samples > cap(raw) {
				continue
			}
			for i, value := range packet.AudioBuffer {
				binary.LittleEndian.PutUint16(raw[i*2:], uint16(value))
			}
			reclaim()
			if len(emptyBufs) == 0 {
				continue
			}
			last := len(emptyBufs) - 1
			buffer := emptyBufs[last]
			emptyBufs = emptyBufs[:last]
			buffer.SetData(openal.FormatMono16, raw[:samples*2], gumble.AudioSampleRate)
			source.QueueBuffer(buffer)
			if source.State() != openal.Playing {
				source.Play()
			}
		}
		reclaim()
		emptyBufs.Delete()
		source.Delete()
	}()
}

