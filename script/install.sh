#!/bin/bash
useradd proxyscript
mkdir -P /home/proxyscript/config
mkdir /var/log/proxyscript
chown -R proxyscript: /home/proxyscript /var/log/proxyscript
cp proxyscript /home/proxyscript/
cp config/proxyscript.toml /home/proxyscript/config/
cp systemd/proxyscript.service /lib/systemd/system/
systemctl enable proxyscript
#systemctl start proxyscript
