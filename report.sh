#!/bin/bash
FFDIR="/usr/ffmpeg/ffmpeg-master-latest-linux64-gpl/bin"
export PATH=$PATH:$FFDIR
read -p "输入RTSP流地址：" playurl
echo "try to play: $playurl" 
./ffprobe -i $playurl -t 10 -show_format -show_streams -select_streams v
