#!/bin/sh
set -e

systemctl disable lcode-hub.service
systemctl stop lcode-hub.service
