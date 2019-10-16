package cmd

import (
	"testing"
)

func Test_contentRange(t *testing.T) {
	file := "http://iij.dl.osdn.jp/storage/g/m/ma/manjaro/gnome/18.1.0/manjaro-gnome-18.1.0-stable-x86_64.iso"
	_, _, _, total, err := contentRange(file)
	t.Logf("total=%v, err=%v", total, err)
}

func Test_rootCmdRun(t *testing.T) {
	downloadFile = "https://vlc.letterboxdelivery.org/vlc/3.0.8/win32/vlc-3.0.8-win32.exe"
	// downloadFile = "https://forum.manjaro.org/uploads/default/original/3X/e/9/e96048fcca8e097ade7d260c8e71381d9a5ae27a.png"
	// downloadFile = "http://iij.dl.osdn.jp/storage/g/m/ma/manjaro/gnome/18.1.0/manjaro-gnome-18.1.0-stable-x86_64.iso"
	gn = 100

	rootCmdRun(RootCmd, nil)
}
