#!/bin/bash
export FFDIR="/C/wh/ffmpeg-master/lib/pkgconfig"
export PKG_CONFIG_PATH=$PKG_CONFIG_PATH:$FFDIR
echo $PKG_CONFIG_PATH