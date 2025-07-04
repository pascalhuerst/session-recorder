package main

import (
	"encoding/json"
	"flag"
	"fmt"
	"io/ioutil"
	"log"
	"net/http"
	"os"
	"os/exec"
	"path/filepath"
	"strconv"
	"strings"
	"sync"
	"time"

	"github.com/bogem/id3v2"
	"github.com/fsnotify/fsnotify"
	"github.com/pascalhuerst/session-recorder/storage"
)

const (
	fadeTime = 0.8
)

var (
	version string
)

type fileServer struct {
	chunkDirPath      string
	sessionDirPath    string
	recordingsDirPath string
	lock              sync.Locker
	Recorders         map[string]Recorder
	sessionTTL        time.Duration
	renderRequestCH   chan RenderRequest
	playbackRequestCH chan PlaybackRequest
	watcher           *fsnotify.Watcher
}

// Recorder holds open sessions for a recorder
type Recorder struct {
	OpenSessions []OpenSession `json:"open_sessions,omitempty"`
}

// OpenSession represents an open session which can be closes to a recording with this server
type OpenSession struct {
	ID               string           `json:"id,omitempty"`
	OGGFileName      storage.Filename `json:"ogg_file_name,omitempty"`
	FLACFileName     storage.Filename `json:"flac_file_name,omitempty"`
	WaveformFileName storage.Filename `json:"waveform_file_name,omitempty"`
	Timestamp        time.Time        `json:"timestamp,omitempty"`
	HoursToLive      float64          `json:"hours_to_live,omitempty"`
	Flagged          bool             `json:"flagged,omitempty"`
}

func (os OpenSession) String() string {
	ret := ""
	ret += fmt.Sprintf("ID:               %v\n", os.ID)
	ret += fmt.Sprintf("HoursToLive:      %v\n", os.HoursToLive)
	ret += fmt.Sprintf("Timestamp:        %v\n", os.Timestamp)
	ret += fmt.Sprintf("OGGFileName:      %v\n", os.OGGFileName)
	ret += fmt.Sprintf("FLACFileName:     %v\n", os.FLACFileName)
	ret += fmt.Sprintf("WaveformFileName: %v\n", os.WaveformFileName)
	ret += fmt.Sprintf("Flagged:          %v\n", os.Flagged)
	return ret
}

// A Segment is a cut mark
type Segment struct {
	Name      string   `json:"labelText,omitempty"`
	StartTime float64  `json:"startTime,omitempty"`
	EndTime   float64  `json:"endTime,omitempty"`
	Filetypes []string `json:"filetypes,omitempty"`
}

// RenderRequest is issues by frontend to session -> recording
type RenderRequest struct {
	Segments   map[string]Segment `json:"segments,omitempty"`
	RecorderID string             `json:"recorderID,omitempty"`
	SessionID  string             `json:"sessionID,omitempty"`
}

// DeleteRequest to delete bogus sessions
type DeleteRequest struct {
	RecorderID string `json:"recorderID,omitempty"`
	SessionID  string `json:"sessionID,omitempty"`
}

// FlagRequest to flag sessions that shall not be auto-deleted
type FlagRequest struct {
	RecorderID string `json:"recorderID,omitempty"`
	SessionID  string `json:"sessionID,omitempty"`
	Flagged    bool   `json:"flagged,omitempty"`
}

// PlaybackRequest to playback a session on snapserver
type PlaybackRequest struct {
	RecorderID string  `json:"recorderID,omitempty"`
	SessionID  string  `json:"sessionID,omitempty"`
	Seek       float32 `json:"seek,omitempty"`
}

func main() {

	log.Printf("Starting Observer Server: Version %s", version)

	chunkDirPath := flag.String("chunk", "chunks", "Directory to look for chunks")
	sessionDirPath := flag.String("session", "sessions", "Directory to look for sessions")
	recordingDirPath := flag.String("recording", "recordings", "Directory to store recordings")
	sessionTTL := flag.Duration("age", time.Duration(3*24*time.Hour), "Duration to keep sessions, before they are deleted")
	flag.Parse()

	chunkDirAbs, err := filepath.Abs(*chunkDirPath)
	if err != nil {
		fmt.Printf("Cannot get absolute path for %s: %v\n", *sessionDirPath, err)
	}
	sessionDirAbs, err := filepath.Abs(*sessionDirPath)
	if err != nil {
		fmt.Printf("Cannot get absolute path for %s: %v\n", *sessionDirPath, err)
	}
	recordingsDirAbs, err := filepath.Abs(*recordingDirPath)
	if err != nil {
		fmt.Printf("Cannot get absolute path for %s: %v\n", *sessionDirPath, err)
	}

	fs := fileServer{
		chunkDirPath:      chunkDirAbs,
		sessionDirPath:    sessionDirAbs,
		recordingsDirPath: recordingsDirAbs,
		lock:              &sync.Mutex{},
		sessionTTL:        *sessionTTL,
		renderRequestCH:   make(chan RenderRequest),
		playbackRequestCH: make(chan PlaybackRequest),
		Recorders:         make(map[string]Recorder),
		watcher:           nil,
	}

	fs.watcher, err = fsnotify.NewWatcher()
	if err != nil {
		log.Fatal(err)
	}
	defer fs.watcher.Close()

	// Scan sessions dir for all recorders and sessions
	// Also setup watchers for the recorder dirs
	err = fs.parseOpenSessions()
	if err != nil {
		fmt.Printf("Cannot parse sessions: %v\n", err)
	}

	server := http.FileServer(http.Dir(fs.sessionDirPath))
	http.HandleFunc("/", func(rw http.ResponseWriter, r *http.Request) {
		rw.Header().Set("Access-Control-Allow-Origin", "*")
		server.ServeHTTP(rw, r)
	})

	http.HandleFunc("/introspect", fs.introspect)
	http.HandleFunc("/render", fs.render)
	http.HandleFunc("/delete", fs.delete)
	http.HandleFunc("/play", fs.play)
	http.HandleFunc("/flag", fs.flag)

	go func() {
		for {
			select {
			case <-fs.watcher.Events:

				//fmt.Printf("######## event!!! %v\n", event.Op.String())

				//if event.Op&fsnotify.Write == fsnotify.Write {
				err = fs.parseOpenSessions()
				if err != nil {
					fmt.Printf("Cannot parse sessions: %v\n", err)
				}
				//	fmt.Printf("#### Watcher event on: %s\n", event.Name)
				//}
			case err := <-fs.watcher.Errors:

				fmt.Println("################ File Watcher Error:", err)
			case <-time.After(time.Minute * 10):
				fmt.Println("Checking sessions directory")
				err = fs.parseOpenSessions()
				if err != nil {
					fmt.Printf("Cannot parse sessions: %v\n", err)
				}
				fmt.Println("Checking sessions directory - Done.")
			case request := <-fs.renderRequestCH:
				fmt.Printf("RenderRequest: %v\n", request)
				fs.renderRequest(request)
			}
		}
	}()

	//go fs.player()

	http.ListenAndServe(":8234", nil)

	select {}
}

func (f *fileServer) player() {

	fmt.Printf("Starting Snapcast player\n")

	for {
		select {
		case request := <-f.playbackRequestCH:
			fmt.Printf("Playback Request: %vz\n", request)

		default:
			fmt.Printf("Writing to snapserver fifo\n")
			time.Sleep(time.Millisecond * 1000)

		}

	}

}

func (f *fileServer) renderRequest(r RenderRequest) error {

	sourceFilePathRel := filepath.Join(f.sessionDirPath, r.RecorderID, r.SessionID, "data.flac")
	sourceFilePath, err := filepath.Abs(sourceFilePathRel)
	if err != nil {
		fmt.Printf("Cannot get absolute path: %v\n", err)
		return fmt.Errorf("Cannot get absolute path: %v", err)
	}

	targetDir := filepath.Join(f.recordingsDirPath, r.SessionID)
	err = os.MkdirAll(targetDir, 0775)
	if err != nil {
		fmt.Printf("Cannot create directory: %s\n", targetDir)
		return fmt.Errorf("Cannot create directory: %s", targetDir)
	}

	createAudioFile := func(name, fileExtension string, startTime, endTime float64) {
		targetAudioFilePathRel := filepath.Join(targetDir, fmt.Sprintf("domestic_affairs_%s.%s", name, fileExtension))
		targetAudioFilePath, err := filepath.Abs(targetAudioFilePathRel)
		if err != nil {
			fmt.Printf("Cannot get absolute path: %v\n", err)
			return
		}

		strFadeTime := fmt.Sprintf("%.1f", fadeTime)

		soxCmd := exec.Command("/usr/bin/sox", sourceFilePath, targetAudioFilePath, "trim", fmt.Sprintf("%v", startTime), fmt.Sprintf("=%v", endTime), "fade", strFadeTime, "-0", strFadeTime, "norm", "-0.1")
		fmt.Printf("Create: %s\n", targetAudioFilePathRel)

		err = soxCmd.Start()
		if err != nil {
			fmt.Printf("Cannot create %s file: %v\n", fileExtension, err)
			return
		}

		err = soxCmd.Wait()
		if err != nil {
			fmt.Printf("Cannot create %s file: %v\n", fileExtension, err)
			return
		}
		fmt.Printf("Create: %s - Done.\n", targetAudioFilePathRel)
		fmt.Printf("Write ID3 Tag: %s\n", targetAudioFilePathRel)

		tag, err := id3v2.Open(targetAudioFilePath, id3v2.Options{Parse: true})
		if err != nil {
			fmt.Printf("Cannot write ID3 Tag: %v\n", err)
			return
		}
		defer tag.Close()

		tag.SetArtist("Domestic Affairs")
		tag.SetTitle("Pandemia")
		tag.SetYear(fmt.Sprintf("%d", time.Now().Year()))
		tag.SetAlbum("Domestic Affairs Recordings")

		artwork, err := ioutil.ReadFile("logo_black.png")
		if err != nil {
			fmt.Printf("Cannot read artwork: %v\n", err)
		}

		pic := id3v2.PictureFrame{
			Encoding:    id3v2.EncodingUTF8,
			MimeType:    "image/png",
			PictureType: id3v2.PTFrontCover,
			Description: "Front cover",
			Picture:     artwork,
		}
		tag.AddAttachedPicture(pic)
		// Write tag to file.
		if err = tag.Save(); err != nil {
			fmt.Printf("Cannot write ID3 Tag: %v\n", err)
			return
		}

		fmt.Printf("Write ID3 Tag: %s - Done.\n", targetAudioFilePathRel)
	}

	for _, value := range r.Segments {
		for _, filetype := range value.Filetypes {
			fixedName := strings.ReplaceAll(value.Name, " ", "_")

			go createAudioFile(fixedName, filetype, value.StartTime, value.EndTime)
		}
	}

	return nil
}

func (f *fileServer) parseOpenSessions() error {

	ret := make(map[string]Recorder, 1)

	recorders, err := ioutil.ReadDir(f.sessionDirPath)
	if err != nil {
		return fmt.Errorf("Cannot read recorders in: %v", f.sessionDirPath)
	}

	for _, recorder := range recorders {

		newSessions := []OpenSession{}
		fmt.Printf("Recordername from fs: %s\n", recorder.Name())

		sessionsPath := filepath.Join(f.sessionDirPath, recorder.Name())
		sessions, err := ioutil.ReadDir(sessionsPath)
		if err != nil {
			return fmt.Errorf("Cannot read sessions in: %v", sessionsPath)
		}

		for _, session := range sessions {

			fmt.Printf("Session from fs: %s\n", session.Name())

			epoche, err := strconv.ParseInt(session.Name(), 10, 64)
			if err != nil {
				return fmt.Errorf("Cannot parse epoche: %s", session.Name())
			}

			toLive := f.sessionTTL - time.Duration(time.Now().Sub(time.Unix(0, epoche)))
			fmt.Printf("Session [%s] from %s has %f hours left, before it gets deleted\n", session.Name(), recorder.Name(), toLive.Hours())

			isFlagged := true
			flaggedFilePath := filepath.Join(f.sessionDirPath, recorder.Name(), session.Name(), "keep_session")
			fmt.Printf("flaggedFilePath: %s\n", flaggedFilePath)
			if _, err := os.Stat(flaggedFilePath); os.IsNotExist(err) {
				isFlagged = false
			}

			if toLive.Hours() < 0 {
				toDelete := filepath.Join(f.sessionDirPath, recorder.Name(), session.Name())
				if !isFlagged {
					fmt.Printf("Attempting to delete: %s\n", toDelete)
					err = os.RemoveAll(toDelete)
					if err != nil {
						fmt.Printf("Cannot remove folder: %v\n", err)
					}
					continue
				}
				fmt.Printf("Not deleting: %s - Session is Flagged!", toDelete)
			}

			s := OpenSession{
				ID:               session.Name(),
				OGGFileName:      storage.FILENAME_OGG,
				FLACFileName:     storage.FILENAME_FLAC,
				WaveformFileName: storage.FILENAME_WAVEFORM,
				Timestamp:        time.Unix(0, epoche),
				HoursToLive:      toLive.Hours(),
				Flagged:          isFlagged,
			}
			newSessions = append(newSessions, s)

			fmt.Printf("Temp Sessions has: %v elements\n", len(newSessions))

		}

		fmt.Printf("Adding %d sessions to recorder: %s\n", len(newSessions), recorder.Name())

		ret[recorder.Name()] = Recorder{
			OpenSessions: newSessions,
		}
	}

	f.lock.Lock()
	watchDirs := []string{}
	// remove recorders we have been watching before this scan in the sessions dir
	for recorderName := range f.Recorders {
		wd := filepath.Join(f.sessionDirPath, recorderName)
		err = f.watcher.Remove(wd)
		if err != nil {
			fmt.Printf("Cannot remove watcher on %s: %v\n", wd, err)
		}
	}
	// now add all new recorders we found by scanning the sessions dir
	for recorderName := range ret {
		wd := filepath.Join(f.sessionDirPath, recorderName)
		watchDirs = append(watchDirs, wd)
		err = f.watcher.Add(wd)
		if err != nil {
			fmt.Printf("Cannot install watcher on %s: %v\n", wd, err)
		}
	}
	f.Recorders = ret
	f.lock.Unlock()

	fmt.Printf("Watching for changes in:\n")
	for _, watchDir := range watchDirs {
		fmt.Printf("  %s\n", watchDir)
	}

	return nil
}

func (f *fileServer) introspect(w http.ResponseWriter, r *http.Request) {

	f.lock.Lock()
	defer f.lock.Unlock()

	fmt.Printf("Introspect:\n")
	for recorderName, data := range f.Recorders {
		fmt.Printf("  %s:\n", recorderName)
		for _, openSession := range data.OpenSessions {
			fmt.Printf("%s\n", openSession.String())
		}
	}

	js, err := json.Marshal(f.Recorders)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
	w.Header().Set("Access-Control-Allow-Origin", "*")
	w.Header().Set("Content-Type", "application/json")
	w.Write(js)
}

func (f *fileServer) render(w http.ResponseWriter, r *http.Request) {

	fmt.Printf("Render...\n")

	w.Header().Set("Access-Control-Allow-Headers", "*")
	w.Header().Set("Access-Control-Allow-Origin", "*")
	w.Header().Set("Content-Type", "application/json")

	renderRequest := RenderRequest{}

	if r.Method == "POST" {

		err := json.NewDecoder(r.Body).Decode(&renderRequest)
		if err != nil {
			http.Error(w, err.Error(), http.StatusBadRequest)
			return
		}
		f.renderRequestCH <- renderRequest
	}

	w.Write([]byte("Success"))

}

func (f *fileServer) delete(w http.ResponseWriter, r *http.Request) {

	w.Header().Set("Access-Control-Allow-Headers", "*")
	w.Header().Set("Access-Control-Allow-Origin", "*")
	w.Header().Set("Content-Type", "application/json")

	deleteRequest := DeleteRequest{}

	if r.Method == "POST" {

		err := json.NewDecoder(r.Body).Decode(&deleteRequest)
		if err != nil {
			http.Error(w, err.Error(), http.StatusBadRequest)
			fmt.Printf("Cannot decode delete request: %v\n", err)
			return
		}

		toDelete := filepath.Join(f.sessionDirPath, deleteRequest.RecorderID, deleteRequest.SessionID)
		fmt.Printf("Delete request: %s\n", toDelete)
		err = os.RemoveAll(toDelete)
		if err != nil {
			http.Error(w, err.Error(), http.StatusBadRequest)
			fmt.Printf("Cannot delete session %s: %v\n", toDelete, err)
			return
		}

	}

	w.Write([]byte("Success"))

}

func (f *fileServer) flag(w http.ResponseWriter, r *http.Request) {

	w.Header().Set("Access-Control-Allow-Headers", "*")
	w.Header().Set("Access-Control-Allow-Origin", "*")
	w.Header().Set("Content-Type", "application/json")

	flagRequest := FlagRequest{}

	if r.Method == "POST" {

		err := json.NewDecoder(r.Body).Decode(&flagRequest)
		if err != nil {
			http.Error(w, err.Error(), http.StatusBadRequest)
			fmt.Printf("Cannot decode flag request: %v\n", err)
			return
		}

		fmt.Printf("%v\n", flagRequest)

		toFlag := filepath.Join(f.sessionDirPath, flagRequest.RecorderID, flagRequest.SessionID)
		fmt.Printf("Flag request: %s\n", toFlag)

		toFlagFile := filepath.Join(toFlag, "keep_session")

		// flagged means don't auto delete ATM
		if flagRequest.Flagged {
			f, err := os.Create(toFlagFile)
			if err != nil {
				http.Error(w, err.Error(), http.StatusBadRequest)
				fmt.Printf("Cannot set flag: %v\n", err)
				return
			}
			f.Close()
		} else {
			err = os.Remove(toFlagFile)
			if err != nil {
				http.Error(w, err.Error(), http.StatusBadRequest)
				fmt.Printf("Cannot delete flag %s: %v\n", toFlagFile, err)
				return
			}
		}
	}

	w.Write([]byte("Success"))

}

func (f *fileServer) play(w http.ResponseWriter, r *http.Request) {

	w.Header().Set("Access-Control-Allow-Headers", "*")
	w.Header().Set("Access-Control-Allow-Origin", "*")
	w.Header().Set("Content-Type", "application/json")

	playbackRequest := PlaybackRequest{}

	if r.Method == "POST" {

		err := json.NewDecoder(r.Body).Decode(&playbackRequest)
		if err != nil {
			http.Error(w, err.Error(), http.StatusBadRequest)
			fmt.Printf("Cannot decode playback request: %v\n", err)
			return
		}

		f.playbackRequestCH <- playbackRequest
	}

	w.Write([]byte("Success"))
}
