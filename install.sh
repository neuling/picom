#!/bin/sh -e
echo "Updating Apt …"

apt-get update -y
apt-get upgrade -y

echo "Installing GIT …"

apt-get install git -y

echo "Installing ReSpeaker driver …"

git clone https://github.com/respeaker/seeed-voicecard.git
cd seeed-voicecard
./install.sh -y
cd ..

echo "Installing golang, git and required libraries …"

apt-get install golang libopenal-dev libopus-dev dnsmasq hostapd -y

systemctl unmask hostapd.service

echo "Installing PICOM & PICOM-IoT …"

go get github.com/neuling/picom-iot
go get github.com/neuling/picom

go get github.com/gin-gonic/gin
go get github.com/gobuffalo/packr

go build -o /home/pi/bin/picom-client /home/pi/go/src/github.com/neuling/picom/client.go
go build -o /home/pi/bin/picom-setup-server /home/pi/go/src/github.com/neuling/picom-iot/cmd/server/server.go
go build -o /home/pi/bin/picom-reset /home/pi/go/src/github.com/neuling/picom-iot/cmd/wifi-reset/wifi-reset.go

echo "Rebooting in setup mode …"

reboot
