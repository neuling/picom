#!/bin/sh
echo "Updating Apt …"

sudo apt-get update -y
sudo apt-get upgrade -y

echo "Installing GIT …"

sudo apt-get install git -y

echo "Installing ReSpeaker driver …"

git clone https://github.com/respeaker/seeed-voicecard.git
cd seeed-voicecard
sudo ./install.sh -y
cd ..

echo "Installing golang, git and required libraries …"

sudo apt-get install golang libopenal-dev libopus-dev dnsmasq hostapd -y

sudo systemctl unmask hostapd.service

echo "Installing PICOM & PICOM-IoT …"

wget https://raw.githubusercontent.com/neuling/picom/master/picom-client.service
wget https://raw.githubusercontent.com/neuling/picom/master/picom-setup-server.service

sudo mv picom-client.service /etc/systemd/system/picom-client.service
sudo mv picom-setup-server.service /etc/systemd/system/picom-setup-server.service

go get github.com/neuling/picom-iot
go get github.com/neuling/picom

go get github.com/gin-gonic/gin
go get github.com/gobuffalo/packr

go build -o /home/pi/bin/picom-client /home/pi/go/src/github.com/neuling/picom/client.go
go build -o /home/pi/bin/picom-setup-server /home/pi/go/src/github.com/neuling/picom-iot/cmd/server/server.go
go build -o /home/pi/bin/picom-reset /home/pi/go/src/github.com/neuling/picom-iot/cmd/wifi-reset/wifi-reset.go

echo "Rebooting in setup mode …"

sudo ./bin/picom-reset
