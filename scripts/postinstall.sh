#!/bin/sh
set -e

# 重新加载 systemd 配置
systemctl daemon-reload

# 自动启用并启动服务
systemctl enable lcode-hub.service
systemctl start lcode-hub.service
