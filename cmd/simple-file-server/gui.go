package main

import (
	"fyne.io/fyne/v2"
	"fyne.io/fyne/v2/app"
	"fyne.io/fyne/v2/driver/desktop"
	"fyne.io/fyne/v2/theme"
	"fyne.io/fyne/v2/widget"

	log "github.com/sirupsen/logrus"
)

func setupGUI(port int) fyne.Window {
	a := app.New()
	win := a.NewWindow("simple-file-server")

	// Shortcuts

	// alt + enter
	altEnter := &desktop.CustomShortcut{
		KeyName:  fyne.KeyEnter,
		Modifier: fyne.KeyModifierAlt,
	}
	win.Canvas().AddShortcut(altEnter, func(shortcut fyne.Shortcut) {
		startStopServer()
	})

	// MENU
	newCollectionMenu := fyne.NewMenuItem("Start/Stop Server",
		func() {
			log.Println("start/stop server menu")
			startStopServer()
		})
	newCollectionMenu.Icon = theme.FolderNewIcon()
	newCollectionMenu.Shortcut = altEnter

	fileMenu := fyne.NewMenu("File", newCollectionMenu)
	mainMenu := fyne.NewMainMenu(fileMenu)

	win.SetMainMenu(mainMenu)

	// Content
	win.SetContent(widget.NewLabel("simple-file-server"))

	return win
}

func startStopServer() {
	//TODO
	log.Debugf("start/stop server")

	if fileServerRunning && fsApp != nil {
		stopFileServer()
	} else {
		go startFileServer(rootDir, port)
	}
}
