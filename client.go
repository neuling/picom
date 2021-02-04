package main

import (
	"crypto/tls"
	"encoding/binary"
	"flag"
	"fmt"
	"io/ioutil"
	"log"
	"net"
	"os"
	"os/signal"
	"os/user"
	"strings"
	"syscall"
	"time"

	"github.com/dchote/go-openal/openal"
	"github.com/dchote/gpio"
	"github.com/neuling/gumble/gumble"
	"github.com/neuling/gumble/gumbleopenal"
	"github.com/neuling/gumble/gumbleutil"
	_ "github.com/neuling/gumble/opus"
	"github.com/neuling/volume-go"
	"github.com/stianeikeland/go-rpio"
)

type Intercom struct {
	Stream *gumbleopenal.Stream
	Client *gumble.Client

	Volume int

	GPIOEnabled    bool
	Button         gpio.Pin
	Led            rpio.Pin
	ButtonState    uint
	IsTransmitting bool
}

func main() {
	fmt.Println("Starting client …")

	usr, usrErr := user.Current()
	if usrErr != nil {
		log.Fatal(usrErr)
	}

	configPath := fmt.Sprintf("%s/.picom", usr.HomeDir)
	configBuf, configReadError := ioutil.ReadFile(configPath)

	if configReadError != nil {
		log.Fatal(configReadError)
	}

	config := strings.Split(string(configBuf), "\n")

	if len(strings.Split(string(config[0]), ":")) == 1 {
		config[0] = fmt.Sprintf("%s:64738", config[0])
	}

	server := flag.String("server", config[0], "server name and port to connect")
	username := flag.String("username", config[1], "the username of the client")
	password := flag.String("password", config[2], "the password of the server")

	flag.Parse()

	tlsConfig := &tls.Config{InsecureSkipVerify: true}

	gumbleConfig := gumble.NewConfig()

	gumbleConfig.Attach(gumbleutil.AutoBitrate)

	gumbleConfig.Username = *username
	gumbleConfig.Password = *password

	intercom := Intercom{}

	v, _ := volume.GetVolume()
	intercom.Volume = v

	gumbleConfig.Attach(&intercom)
	gumbleConfig.AttachAudio(&intercom)

	client, _ := gumble.DialWithDialer(new(net.Dialer), *server, gumbleConfig, tlsConfig)

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

	LedPin := rpio.Pin(26)
	LedPin.Mode(rpio.Output)
	LedPin.Low()

	ButtonPinPullUp := rpio.Pin(16)
	ButtonPinPullUp.PullUp()

	// rpio.Close()

	// unfortunately the gpio watcher stuff doesnt work for me in this context, so we have to poll the button instead
	i.Button = gpio.NewInput(16)
	i.Led = LedPin
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
						i.Led.Low()
					} else {
						i.Led.High()
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
