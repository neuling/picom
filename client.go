package main

import (
	"fmt"
	"net"
	"os"
	"crypto/tls"
	"os/signal"
	"syscall"
	"time"
	"github.com/neuling/gumble/gumble"
	"github.com/neuling/gumble/gumbleutil"
	"github.com/neuling/gumble/gumbleopenal"
	"github.com/dchote/gpio"
	"github.com/stianeikeland/go-rpio"
	_ "github.com/neuling/gumble/opus"
)

type Intercom struct {
	Stream *gumbleopenal.Stream
	Client *gumble.Client

	GPIOEnabled     bool
	Button  gpio.Pin
	ButtonState     uint
	IsTransmitting bool
}


func main() {
	fmt.Println("Hello, world.")

	tlsConfig := &tls.Config{ InsecureSkipVerify: true }

	gumbleConfig := gumble.NewConfig()

	gumbleConfig.Attach(gumbleutil.AutoBitrate)

	gumbleConfig.Username = "Client1"
	gumbleConfig.Password = "1nt3rc0m"

	intercom := Intercom{}

	gumbleConfig.Attach(&intercom)

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
						i.Stream.StopSource()
					} else {
						fmt.Printf("Button is pressed\n")
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

	fmt.Println(e.Message)
	if e.Message == "<p>start</p>" { i.Stream.StartSource() }
	if e.Message == "<p>stop</p>" { i.Stream.StopSource() }
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

