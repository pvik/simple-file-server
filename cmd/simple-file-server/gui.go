package main

import (
	"fmt"
	"strconv"
	"strings"

	"fyne.io/fyne/v2"
	"fyne.io/fyne/v2/app"
	"fyne.io/fyne/v2/container"
	"fyne.io/fyne/v2/data/binding"
	"fyne.io/fyne/v2/dialog"
	"fyne.io/fyne/v2/driver/desktop"
	"fyne.io/fyne/v2/layout"
	"fyne.io/fyne/v2/theme"
	"fyne.io/fyne/v2/widget"

	log "github.com/sirupsen/logrus"
)

var uiPortStringBinding = binding.NewString()
var uiRootDirStringBinding = binding.NewString()
var uiPortEntry *widget.Entry
var uiRootDirEntry *widget.Entry
var uiStartStopBtn *widget.Button

func setupGUI(port int) fyne.Window {
	a := app.New()
	win := a.NewWindow("simple-file-server")

	// SHORTCUTS
	// alt + enter
	altEnter := &desktop.CustomShortcut{
		KeyName:  fyne.KeyEnter,
		Modifier: fyne.KeyModifierAlt,
	}
	win.Canvas().AddShortcut(altEnter, func(shortcut fyne.Shortcut) {
		log.Println("start/stop server menu from shortcut")
		startStopServer(win)
	})

	// MENU
	startStopServerMenu := fyne.NewMenuItem("Start/Stop Server",
		func() {
			log.Println("start/stop server menu")
			startStopServer(win)
		})
	startStopServerMenu.Icon = theme.MediaPlayIcon()
	startStopServerMenu.Shortcut = altEnter

	fileMenu := fyne.NewMenu("File", startStopServerMenu)
	mainMenu := fyne.NewMainMenu(fileMenu)

	win.SetMainMenu(mainMenu)

	// ELEMENTS
	uiPortStringBinding.Set(fmt.Sprintf("%d", port))
	uiPortEntry = widget.NewEntryWithData(uiPortStringBinding)
	uiPortEntry.SetPlaceHolder("Port")
	uiPortEntry.Validator = func(v string) error {
		if v == "" {
			return fmt.Errorf("Can't be Empty")
		} else {
			_, err := strconv.Atoi(v)
			if err != nil {
				return fmt.Errorf("Enter a valid number")
			}
			return nil
		}
	}

	uiRootDirStringBinding.Set(rootDir)
	uiRootDirEntry = widget.NewEntryWithData(uiRootDirStringBinding)
	uiRootDirEntry.SetPlaceHolder("Root Directory")
	uiRootDirEntry.Validator = func(v string) error {
		if v == "" {
			return fmt.Errorf("Can't be Empty")
		} else {
			validDir, err := isValidDir(v)
			if err != nil || !validDir {
				return fmt.Errorf("Invalid Root Directory: %s", err)
			}
			return nil
		}
	}

	uiStartStopBtn = widget.NewButtonWithIcon("Start Server", theme.MediaPlayIcon(), func() {
		log.Debugf("start/stop")
		startStopServer(win)
	})

	uiAllowUploadCheck := widget.NewCheck("", func(value bool) {
		allowUpload = value
	})

	uiAllowUploadCheck.SetChecked(allowUpload)

	// LAYOUT
	mainLayout := container.New(layout.NewVBoxLayout(),
		container.New(layout.NewFormLayout(),
			widget.NewLabel("Server IP"),
			widget.NewLabel(systemIP),
			widget.NewLabel("Allow Uploads"),
			uiAllowUploadCheck,
			widget.NewLabel("Port"), uiPortEntry,
			widget.NewLabel("Root Directory"), uiRootDirEntry,
		),
		uiStartStopBtn,
	)

	win.SetContent(mainLayout)

	return win
}

func startStopServer(win fyne.Window) {
	log.Debugf("start/stop server")

	portStr, err := uiPortStringBinding.Get()
	if err != nil {
		log.Errorf("unable to get port value from ui: %s", err)
		dialog.ShowError(fmt.Errorf("unable to get port value from ui: %s", err), win)
		return
	}
	port, err := strconv.Atoi(portStr)
	if err != nil {
		log.Errorf("invalid port value in ui: %s", err)
		dialog.ShowError(fmt.Errorf("Invalid port, please enter a valid number \n %s", err), win)
		return
	}

	rootDir, err := uiRootDirStringBinding.Get()
	if err != nil {
		log.Errorf("unable to get root directory value from ui: \n %s", err)
		dialog.ShowError(fmt.Errorf("unable to get root directory value from ui: \n %s", err), win)
		return
	}

	rootDir = strings.TrimSpace(rootDir)

	if fileServerRunning && fsApp != nil {
		err := stopFileServer()
		if err != nil {
			dialog.ShowError(fmt.Errorf("Unable to stop server: \n\n%s", err), win)
			return
		}
		uiHandler_startServer()
	} else {
		uiHandler_stopServer()
		go func() {
			err := startFileServer(rootDir, port, allowUpload, compress)
			if err != nil {
				dialog.ShowError(fmt.Errorf("Unable to start server: \n\n%s", err), win)
				uiHandler_startServer()
			}
		}()
	}
}

func uiHandler_startServer() {
	uiRootDirEntry.Enable()
	uiPortEntry.Enable()
	uiStartStopBtn.SetText("Start")
	uiStartStopBtn.SetIcon(theme.MediaPlayIcon())
}

func uiHandler_stopServer() {
	uiRootDirEntry.Disable()
	uiPortEntry.Disable()
	uiStartStopBtn.SetText("Stop")
	uiStartStopBtn.SetIcon(theme.MediaStopIcon())
}
