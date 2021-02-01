#!/bin/sh -e
echo "Updating Apt …"

apt-get update
apt-get upgrade

echo "Installing GIT …"

apt-get install git -y

echo "Installing ReSpeaker driver …"

git clone https://github.com/respeaker/seeed-voicecard.git
./seeed-voicecard/install.sh

echo "Installing golang, git and required libraries …"

apt-get install golang libopenal-dev libopus-dev dnsmasq hostapd -y

echo "Installing PICOM & PICOM-IoT …"

go get github.com/neuling/picom-iot
go get github.com/neuling/picom

go build -o /home/pi/bin/picom-client /home/pi/go/src/github.com/neuling/picom/client.go
go build -o /home/pi/bin/picom-setup-server /home/pi/go/src/github.com/neuling/picom-iot/cmd/server/main.go
go build -o /home/pi/bin/picom-reset /home/pi/go/src/github.com/neuling/picom-iot/cmd/wifi-reset/main.go
