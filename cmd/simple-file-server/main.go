package main

import (
	"embed"
	"fmt"
	"io/fs"
	"net/http"
	"time"

	"github.com/pvik/simple-file-server/internal/service"

	"github.com/gofiber/fiber/v2"
	"github.com/gofiber/fiber/v2/middleware/filesystem"
	jettmplcore "github.com/gofiber/template"
	jet "github.com/gofiber/template/jet/v2"

	bytesize "github.com/inhies/go-bytesize"
	log "github.com/sirupsen/logrus"
)

var port int
var isGUI bool
var allowUpload bool
var compress bool
var rootDir string

var fileServerRunning bool
var fsApp *fiber.App

var systemIP string

var (
	//go:embed resources
	res embed.FS
)

func init() {
	log.Info("Simple File Server v0.3")
	commit := getCommit()
	if commit != "" {
		log.Infof("Git Commit: %s", commit)
	}
	log.Debug("=======================")

	// Initialize config file
	// Setup Logging
	rootDir, port, allowUpload, compress, isGUI = service.InitService()

	log.Infof("Serving files from %s", rootDir)
	log.Infof("Allow Upload:  %t", allowUpload)

	serverIP, err := getServerIP()
	if err != nil {
		log.Debugf("unable to interospect system ip, defaulting to 0.0.0.0")
		systemIP = "0.0.0.0"
	} else {
		systemIP = serverIP.String()
	}

	log.Infof("System IP: %s", systemIP)
}

func main() {
	defer service.Shutdown()
	defer func() {
		err := stopFileServer()
		if err != nil {
			log.Errorf("unable to shutdown file server: %s", err)
		}
	}()

	if isGUI {
		w := setupGUI(port)
		w.ShowAndRun()
	} else {
		// handle CLI
		err := startFileServer(rootDir, port, allowUpload, compress)
		if err != nil {
			log.Errorf("unable to start file server: %s", err)
		}
	}
}

func stopFileServer() error {
	if fileServerRunning && fsApp != nil {
		log.Info("shutdown file-server")
		err := fsApp.Shutdown()
		if err != nil {
			log.Errorf("unable to shutdown file server: %s", err)
			return err
		}
		fileServerRunning = false
		fsApp = nil
	}
	return nil
}

func startFileServer(rootDir string, port int, allowUpload, compress bool) error {
	fileServErr := make(chan error, 2)
	fileServApp := make(chan *fiber.App, 1)

	go setupFileServer(fileServApp, fileServErr, rootDir, port, allowUpload, compress)

	fsErr := <-fileServErr
	if fsErr != nil {
		log.Errorf("%s", fsErr)
		return fsErr
	}

	fsApp = <-fileServApp
	fileServerRunning = true

	// block here
	fsErr = <-fileServErr
	if fsErr != nil {
		log.Errorf("%s", fsErr)
		return fsErr
	}

	return nil
}

func setupFileServer(fileServApp chan<- *fiber.App, fileServErr chan<- error, rootDir string, port int, allowUpload, compress bool) {

	validDir, err := isValidDir(rootDir)
	if err != nil {
		fileServErr <- fmt.Errorf("Invalid Root Directory: %s", err)
		return
	}
	if !validDir {
		fileServErr <- fmt.Errorf("Invalid Root Directory")
		return
	}

	viewsfSys, err := fs.Sub(res, "resources/views")
	if err != nil {
		log.Errorf("unable to open views: %s", err)
		fileServErr <- fmt.Errorf("unable to open views: %s", err)
		return
	}

	staticfSys, err := fs.Sub(res, "resources/static")
	if err != nil {
		log.Errorf("unable to open static: %s", err)
		fileServErr <- fmt.Errorf("unable to open static: %s", err)
		return
	}

	engine := &jet.Engine{
		Engine: jettmplcore.Engine{
			Directory:  "/",
			FileSystem: http.FS(viewsfSys),
			Extension:  ".jet",
			LayoutName: "embed",
			Funcmap: map[string]interface{}{
				"filesizeString": func(size int64) string {
					bs := bytesize.ByteSize(size)
					return bs.String()
				},
				"formatTime": func(t time.Time) string {
					return t.Format(time.ANSIC)
				},
			},
		},
	}

	app := fiber.New(fiber.Config{
		// EnableTrustedProxyCheck: true,
		// TrustedProxies:          []string{"0.0.0.0", "1.1.1.1/30"}, // IP address or IP address range
		// ProxyHeader:             fiber.HeaderXForwardedFor,
		//Prefork:                  true,
		AppName:           "simple-file-server",
		BodyLimit:         5 * 1024 * 1024, // bytes = 5 MB
		ReadTimeout:       60 * time.Second,
		EnablePrintRoutes: true,
		Views:             engine,
	})

	fileServErr <- nil
	fileServApp <- app
	defer app.Shutdown()

	app.Use("/static", filesystem.New(filesystem.Config{
		Root: http.FS(staticfSys),
	}))

	app.Post("/upload", func(c *fiber.Ctx) error {
		file, err := c.FormFile("file")
		if err != nil {
			log.Errorf("unable to get upload file: %s", err)
			return c.Render("error", fiber.Map{
				"err": err,
				"dir": rootDir,
			})
		}

		dir := rootDir
		subPath := c.FormValue("subdir", "")
		if subPath != "" {
			dir = fmt.Sprintf("%s/%s", rootDir, subPath)
		}
		c.SaveFile(file, fmt.Sprintf("%s/%s", dir, file.Filename))

		return c.Redirect(fmt.Sprintf("/%s", subPath))
	})

	app.Get("/*", func(c *fiber.Ctx) error {
		subPath := c.Params("*")

		log.Infof("GET: /%s", subPath)

		fiberMap, dir, isFile, err := handleIndex(subPath, allowUpload)
		if err != nil {
			log.Errorf("unable to handle index: %s", err)
			return c.Render("error", fiberMap)
		}

		if isFile {
			// this is a file, to be serverd for download
			log.Debugf("serving file: %s", dir)
			return c.SendFile(dir, compress)
		}

		return c.Render("index", fiberMap)
	}).Name("dir handler")

	err = app.Listen(fmt.Sprintf(":%d", port))
	if err != nil {
		fileServErr <- fmt.Errorf("Unable to listen on %d, check the port is not in use by another process", port)
		return
	}

	fileServErr <- nil
}

func handleIndex(subPath string, allowUpload bool) (fiber.Map, string, bool, error) {
	dir := rootDir
	if subPath != "" {
		dir = dir + "/" + subPath

		validDir, err := isValidDir(dir)
		if err != nil {
			return fiber.Map{
				"err": err,
				"dir": dir,
			}, dir, false, err
		}
		if !validDir {
			// this is a file, to be serverd for download
			return fiber.Map{}, dir, true, nil
		}
	}

	files, err := getDirContent(dir)
	if err != nil {
		return fiber.Map{
			"err": err,
			"dir": dir,
		}, dir, false, fmt.Errorf("unable to get files in directory(%s): %s", dir, err)
	}
	return fiber.Map{
		"WorkingDirectory": dir,
		"SubDir":           subPath,
		"AllowUpload":      allowUpload,
		"Files":            files,
	}, dir, false, nil
}
