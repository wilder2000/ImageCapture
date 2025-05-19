#!/bin/bash
export LD_LIBRARY_PATH=/usr/ffmpeg/ffmpeg-master-latest-linux64-gpl-shared/lib:$LD_LIBRARY_PATH

read -p "输入RTSP流地址：" playurl
echo "try to play: $playurl" 
./cv -rtsp=$playurl
