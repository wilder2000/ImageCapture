#!/bin/bash
read -p "输入RTSP流地址：" playurl
echo "try to play: $playurl" 
./ffplay -rtsp_transport     -i $playurl
