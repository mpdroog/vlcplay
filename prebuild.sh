#!/bin/bash
sudo ln -s /Applications/VLC.app/Contents/MacOS/include/vlc /usr/local/include/
sudo ln -s /Applications/VLC.app/Contents/MacOS/lib/* /usr/local/lib/
export VLC_PLUGIN_PATH="/Applications/VLC.app/Contents/MacOS"
