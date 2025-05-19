#!/bin/bash
FFPK=/usr/ffmpeg/ffmpeg-master-latest-linux64-gpl-shared/lib/pkgconfig/
export PKG_CONFIG_PATH=$FFPK:$PKG_CONFIG_PATH
echo $PKG_CONFIG_PATH
export LD_LIBRARY_PATH=/usr/ffmpeg/ffmpeg-master-latest-linux64-gpl-shared/lib:$LD_LIBRARY_PATH
pkg-config --cflags --libs libavcodec
pkg-config --list-all | grep ffmpeg
go mod tidy
go build
