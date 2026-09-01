package main

import (
	"archive/tar"
	"archive/zip"
	"bytes"
	"context"
	"crypto/rand"
	"encoding/hex"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"log"
	"net"
	"net/http"
	"net/http/httputil"
	"net/url"
	"os"
	"path"
	"path/filepath"
	"regexp"
	"strings"
	"sync"
	"time"

	"github.com/websummoner/websummoner/info"

	"dario.cat/mergo"
	"github.com/docker/docker/api/types/container"
	"github.com/docker/docker/pkg/stdcopy"
	"github.com/websummoner/websummoner/event"
	"github.com/websummoner/websummoner/jsonerror"
	"github.com/websummoner/websummoner/service"
	"github.com/websummoner/websummoner/session"
	"golang.org/x/net/websocket"
)

const slash = "/"

var (
	httpClient = &http.Client{
		CheckRedirect: func(req *http.Request, via []*http.Request) error {
			return http.ErrUseLastResponse
		},
	}
	num     uint64
	numLock sync.RWMutex
)

type request struct {
	*http.Request
}

type sess struct {
	addr string
	id   string
}

// TODO There is simpler way to do this
func (r request) localaddr() string {
	addr := r.Context().Value(http.LocalAddrContextKey).(net.Addr).String()
	_, port, _ := net.SplitHostPort(addr)
	return net.JoinHostPort("127.0.0.1", port)
}

func (r request) session(id string) *sess {
	return &sess{r.localaddr(), id}
}

func (s *sess) url() string {
	return fmt.Sprintf("http://%s/wd/hub/session/%s", s.addr, s.id)
}

func (s *sess) Delete(requestId uint64) {
	log.Printf("[%d] [SESSION_TIMED_OUT] [%s]", requestId, s.id)
	metricsSessionsTimedOut.Add(1)
	r, err := http.NewRequest(http.MethodDelete, s.url(), nil)
	if err != nil {
		log.Printf("[%d] [DELETE_FAILED] [%s] [%v]", requestId, s.id, err)
		return
	}
	ctx, cancel := context.WithTimeout(context.Background(), sessionDeleteTimeout)
	defer cancel()
	resp, err := httpClient.Do(r.WithContext(ctx))
	if resp != nil {
		defer resp.Body.Close()
	}
	if err == nil && resp.StatusCode == http.StatusOK {
		return
	}
	if err != nil {
		log.Printf("[%d] [DELETE_FAILED] [%s] [%v]", requestId, s.id, err)
	} else {
		log.Printf("[%d] [DELETE_FAILED] [%s] [%s]", requestId, s.id, resp.Status)
	}
}

func serial() uint64 {
	numLock.Lock()
	defer numLock.Unlock()
	id := num
	num++
	return id
}

func getSerial() uint64 {
	numLock.RLock()
	defer numLock.RUnlock()
	return num
}

func create(w http.ResponseWriter, r *http.Request) {
	sessionStartTime := time.Now()
	requestId := serial()
	user, remote := info.RequestInfo(r)
	body, err := io.ReadAll(r.Body)
	_ = r.Body.Close()
	if err != nil {
		log.Printf("[%d] [ERROR_READING_REQUEST] [%v]", requestId, err)
		jsonerror.InvalidArgument(err).Encode(w)
		queue.Drop()
		return
	}
	var browser struct {
		Caps    session.Caps `json:"desiredCapabilities"`
		W3CCaps struct {
			Caps       session.Caps    `json:"alwaysMatch"`
			FirstMatch []*session.Caps `json:"firstMatch"`
		} `json:"capabilities"`
	}
	err = json.Unmarshal(body, &browser)
	if err != nil {
		log.Printf("[%d] [BAD_JSON_FORMAT] [%v]", requestId, err)
		jsonerror.InvalidArgument(err).Encode(w)
		queue.Drop()
		return
	}
	if browser.W3CCaps.Caps.BrowserName() != "" && browser.Caps.BrowserName() == "" {
		browser.Caps = browser.W3CCaps.Caps
	}
	firstMatchCaps := browser.W3CCaps.FirstMatch
	if len(firstMatchCaps) == 0 {
		firstMatchCaps = append(firstMatchCaps, &session.Caps{})
	}
	var caps session.Caps
	var starter service.Starter
	var ok bool
	var sessionTimeout time.Duration
	var finalVideoName, finalLogName string
	for _, fmc := range firstMatchCaps {
		caps = browser.Caps
		_ = mergo.Merge(&caps, *fmc)
		caps.ProcessExtensionCapabilities()
		sessionTimeout, err = getSessionTimeout(caps.SessionTimeout, maxTimeout, timeout)
		if err != nil {
			log.Printf("[%d] [BAD_SESSION_TIMEOUT] [%s]", requestId, caps.SessionTimeout)
			jsonerror.InvalidArgument(err).Encode(w)
			queue.Drop()
			return
		}
		resolution, err := getScreenResolution(caps.ScreenResolution)
		if err != nil {
			log.Printf("[%d] [BAD_SCREEN_RESOLUTION] [%s]", requestId, caps.ScreenResolution)
			jsonerror.InvalidArgument(err).Encode(w)
			queue.Drop()
			return
		}
		caps.ScreenResolution = resolution
		videoScreenSize, err := getVideoScreenSize(caps.VideoScreenSize, resolution)
		if err != nil {
			log.Printf("[%d] [BAD_VIDEO_SCREEN_SIZE] [%s]", requestId, caps.VideoScreenSize)
			jsonerror.InvalidArgument(err).Encode(w)
			queue.Drop()
			return
		}
		caps.VideoScreenSize = videoScreenSize
		finalVideoName = caps.VideoName
		if finalVideoName != "" && !isSafeFileName(finalVideoName) {
			log.Printf("[%d] [BAD_VIDEO_NAME] [%s]", requestId, finalVideoName)
			jsonerror.InvalidArgument(fmt.Errorf("videoName capability must be a plain file name: %s", finalVideoName)).Encode(w)
			queue.Drop()
			return
		}
		if caps.Video && !disableDocker {
			caps.VideoName = getTemporaryFileName(videoOutputDir, videoFileExtension)
		}
		finalLogName = caps.LogName
		if finalLogName != "" && !isSafeFileName(finalLogName) {
			log.Printf("[%d] [BAD_LOG_NAME] [%s]", requestId, finalLogName)
			jsonerror.InvalidArgument(fmt.Errorf("logName capability must be a plain file name: %s", finalLogName)).Encode(w)
			queue.Drop()
			return
		}
		if logOutputDir != "" && (saveAllLogs || caps.Log) {
			caps.LogName = getTemporaryFileName(logOutputDir, logFileExtension)
		}
		starter, ok = manager.Find(caps, requestId)
		if ok {
			break
		}
	}
	if !ok {
		log.Printf("[%d] [ENVIRONMENT_NOT_AVAILABLE] [%s] [%s]", requestId, caps.BrowserName(), caps.Version)
		jsonerror.InvalidArgument(errors.New("requested environment is not available")).Encode(w)
		queue.Drop()
		return
	}
	startedService, err := starter.StartWithCancel()
	if err != nil {
		log.Printf("[%d] [SERVICE_STARTUP_FAILED] [%v]", requestId, err)
		jsonerror.SessionNotCreated(err).Encode(w)
		queue.Drop()
		return
	}
	u := startedService.Url
	cancel := startedService.Cancel
	// on failure also remove the temporary video/log files (renamed on success)
	cancelAndCleanup := func() {
		cancel()
		if caps.Video && !disableDocker {
			tmpVideo := filepath.Join(videoOutputDir, caps.VideoName)
			if err := os.Remove(tmpVideo); err != nil && !os.IsNotExist(err) {
				log.Printf("[%d] [VIDEO_ERROR] [Failed to remove temporary video file %s: %v]", requestId, tmpVideo, err)
			}
		}
		if logOutputDir != "" && (saveAllLogs || caps.Log) {
			tmpLog := filepath.Join(logOutputDir, caps.LogName)
			if err := os.Remove(tmpLog); err != nil && !os.IsNotExist(err) {
				log.Printf("[%d] [LOG_ERROR] [Failed to remove temporary log file %s: %v]", requestId, tmpLog, err)
			}
		}
	}
	host := "localhost"
	if startedService.Origin != "" {
		host = startedService.Origin
	}

	var resp *http.Response
	i := 1
	for ; ; i++ {
		r.URL.Host, r.URL.Path = u.Host, path.Join(u.Path, r.URL.Path)
		newBody := adaptDriverCapabilities(removeVendorOptions(body), caps.BrowserName(), requestId)
		req, _ := http.NewRequest(http.MethodPost, r.URL.String(), bytes.NewReader(newBody))
		contentType := r.Header.Get("Content-Type")
		if len(contentType) > 0 {
			req.Header.Set("Content-Type", contentType)
		}
		req.Host = host
		ctx, done := context.WithTimeout(r.Context(), newSessionAttemptTimeout)
		defer done()
		log.Printf("[%d] [SESSION_ATTEMPTED] [%s] [%d]", requestId, u.String(), i)
		rsp, err := httpClient.Do(req.WithContext(ctx))
		select {
		case <-ctx.Done():
			if rsp != nil {
				_ = rsp.Body.Close()
			}
			switch ctx.Err() {
			case context.DeadlineExceeded:
				log.Printf("[%d] [SESSION_ATTEMPT_TIMED_OUT] [%s]", requestId, newSessionAttemptTimeout)
				if i < retryCount {
					continue
				}
				err := errors.New("new session attempts retry count exceeded")
				log.Printf("[%d] [SESSION_FAILED] [%s] [%s]", requestId, u.String(), err)
				metricsSessionsFailed.Add(1)
				jsonerror.UnknownError(err).Encode(w)
			case context.Canceled:
				log.Printf("[%d] [CLIENT_DISCONNECTED] [%s] [%s] [%.2fs]", requestId, user, remote, info.SecondsSince(sessionStartTime))
			}
			queue.Drop()
			cancelAndCleanup()
			return
		default:
		}
		if err != nil {
			if rsp != nil {
				_ = rsp.Body.Close()
			}
			log.Printf("[%d] [SESSION_FAILED] [%s] [%s]", requestId, u.String(), err)
			metricsSessionsFailed.Add(1)
			jsonerror.SessionNotCreated(err).Encode(w)
			queue.Drop()
			cancelAndCleanup()
			return
		}
		if rsp.StatusCode == http.StatusNotFound && u.Path == "" {
			u.Path = "/wd/hub"
			continue
		}
		resp = rsp
		break
	}
	defer resp.Body.Close()
	var s struct {
		Value struct {
			ID string `json:"sessionId"`
		}
		ID string `json:"sessionId"`
	}
	location := resp.Header.Get("Location")
	if location != "" {
		l, err := url.Parse(location)
		if err == nil {
			fragments := strings.Split(l.Path, slash)
			s.ID = fragments[len(fragments)-1]
			u := &url.URL{
				Scheme: "http",
				Host:   hostname,
				Path:   path.Join("/wd/hub/session", s.ID),
			}
			w.Header().Add("Location", u.String())
			w.WriteHeader(resp.StatusCode)
		}
	} else {
		body, err := io.ReadAll(resp.Body)
		if err != nil {
			log.Printf("[%d] [ERROR_READING_RESPONSE] [%v]", requestId, err)
			queue.Drop()
			cancelAndCleanup()
			w.WriteHeader(resp.StatusCode)
			return
		}
		newBody, sessionId, err := processBody(body, r.Host)
		if err != nil {
			log.Printf("[%d] [ERROR_PROCESSING_RESPONSE] [%v]", requestId, err)
			queue.Drop()
			cancelAndCleanup()
			w.WriteHeader(resp.StatusCode)
			return
		}
		resp.Body = io.NopCloser(bytes.NewReader(newBody))
		resp.ContentLength = int64(len(newBody))
		w.WriteHeader(resp.StatusCode)
		_, _ = w.Write(newBody)
		s.ID = sessionId
	}
	if s.ID == "" {
		log.Printf("[%d] [SESSION_FAILED] [%s] [%s]", requestId, u.String(), resp.Status)
		metricsSessionsFailed.Add(1)
		queue.Drop()
		cancelAndCleanup()
		return
	}
	sess := &session.Session{
		Quota:     user,
		Caps:      caps,
		URL:       u,
		Container: startedService.Container,
		HostPort:  startedService.HostPort,
		Origin:    startedService.Origin,
		Timeout:   sessionTimeout,
		TimeoutCh: onTimeout(sessionTimeout, func() {
			request{r}.session(s.ID).Delete(requestId)
		}),
		Started: time.Now()}
	cancelAndRenameFiles := func() {
		cancel()
		sessionId := preprocessSessionId(s.ID)
		e := event.Event{
			RequestId: requestId,
			SessionId: sessionId,
			Session:   sess,
		}
		if caps.Video && !disableDocker {
			oldVideoName := filepath.Join(videoOutputDir, caps.VideoName)
			if finalVideoName == "" {
				finalVideoName = sessionId + videoFileExtension
				e.Session.Caps.VideoName = finalVideoName
			}
			newVideoName := filepath.Join(videoOutputDir, finalVideoName)
			err := os.Rename(oldVideoName, newVideoName)
			if err != nil {
				log.Printf("[%d] [VIDEO_ERROR] [%s]", requestId, fmt.Sprintf("Failed to rename %s to %s: %v", oldVideoName, newVideoName, err))
			} else {
				createdFile := event.CreatedFile{
					Event: e,
					Name:  newVideoName,
					Type:  "video",
				}
				event.FileCreated(createdFile)
			}
		}
		if logOutputDir != "" && (saveAllLogs || caps.Log) {
			//The following logic will fail if -capture-driver-logs is enabled and a session is requested in driver mode.
			//Specifying both -log-output-dir and -capture-driver-logs in that case is considered a misconfiguration.
			oldLogName := filepath.Join(logOutputDir, caps.LogName)
			if finalLogName == "" {
				finalLogName = sessionId + logFileExtension
				e.Session.Caps.LogName = finalLogName
			}
			newLogName := filepath.Join(logOutputDir, finalLogName)
			err := os.Rename(oldLogName, newLogName)
			if err != nil {
				log.Printf("[%d] [LOG_ERROR] [%s]", requestId, fmt.Sprintf("Failed to rename %s to %s: %v", oldLogName, newLogName, err))
			} else {
				createdFile := event.CreatedFile{
					Event: e,
					Name:  newLogName,
					Type:  "log",
				}
				event.FileCreated(createdFile)
			}
		}
		event.SessionStopped(event.StoppedSession{Event: e})
	}
	sess.Cancel = cancelAndRenameFiles
	sessions.Put(s.ID, sess)
	queue.Create()
	log.Printf("[%d] [SESSION_CREATED] [%s] [%d] [%.2fs]", requestId, s.ID, i, info.SecondsSince(sessionStartTime))
	metricsSessionsCreated.Add(1)
	if caps.VNC {
		metricsVncSessions.Add(1)
	}
	if caps.Video {
		metricsVideoSessions.Add(1)
		// Audio only exists as part of a video recording.
		if caps.AudioEnabled() {
			metricsAudioSessions.Add(1)
		}
	}
}

// chromiumBinaries maps a browsers.json browser key to the binary its
// in-container driver must be pointed at. All of these ship a chromedriver
// fork (or stock chromedriver) that only accepts browserName "chrome", so the
// hub rewrites what it forwards. Routing still happens on the real browser
// name from browsers.json.
//
// Brave points at the real binary, not /usr/bin/brave-browser: that path is a
// wrapper shell script, and chromedriver launches it and immediately sees the
// process exit ("Chrome instance exited").
var chromiumBinaries = map[string]string{
	"opera":  "/usr/bin/opera",
	"yandex": "/usr/bin/yandex-browser",
	"brave":  "/opt/brave.com/brave/brave-browser",
}

// chromiumExtraArgs are per-browser start-up flags the driver needs on top of
// the common ones. Yandex opens its own start page (https://ya.ru/) shortly
// after launch, which clobbers whatever the driver has just navigated to.
var chromiumExtraArgs = map[string][]string{
	"yandex": {"--homepage=about:blank", "--no-first-run", "--no-default-browser-check"},
}

// removeVendorOptions drops vendor blocks and the legacy `version` routing
// hint before forwarding: modern drivers reject unknown capabilities (#909).
func adaptDriverCapabilities(input []byte, name string, requestId uint64) []byte {
	binary, ok := chromiumBinaries[name]
	if !ok {
		return input
	}
	args := append([]string{"no-sandbox", fmt.Sprintf("--user-data-dir=/tmp/ws-%d", requestId)},
		chromiumExtraArgs[name]...)
	options := map[string]interface{}{
		"binary": binary,
		"args":   args,
	}
	// Opera's own driver still answers in the legacy JSONWP dialect by default,
	// which a W3C-only client such as Selenium 4 cannot decode — it is the
	// reason Selenium dropped Opera in 4.3.0. The driver does support W3C, but
	// only when asked for it explicitly. Harmless on chromedriver, which
	// ignores the option.
	if name == "opera" {
		options["w3c"] = true
	}
	var body map[string]interface{}
	if err := json.Unmarshal(input, &body); err != nil {
		return input
	}
	caps, ok := body["capabilities"].(map[string]interface{})
	if !ok {
		return input
	}
	target, ok := caps["alwaysMatch"].(map[string]interface{})
	if !ok {
		list, ok := caps["firstMatch"].([]interface{})
		if !ok || len(list) == 0 {
			return input
		}
		target, ok = list[0].(map[string]interface{})
		if !ok {
			return input
		}
	}
	target["browserName"] = "chrome"
	target["goog:chromeOptions"] = options
	if out, err := json.Marshal(body); err == nil {
		return out
	}
	return input
}

func removeVendorOptions(input []byte) []byte {
	body := make(map[string]interface{})
	_ = json.Unmarshal(input, &body)
	staleCapabilities := []string{"websummoner:options", "selenoid:options", "version"}
	deleteStaleCapabilities := func(caps map[string]interface{}) {
		for _, prefix := range staleCapabilities {
			delete(caps, prefix)
		}
	}
	if raw, ok := body["desiredCapabilities"]; ok {
		if dc, ok := raw.(map[string]interface{}); ok {
			deleteStaleCapabilities(dc)
		}
	}
	if raw, ok := body["capabilities"]; ok {
		if c, ok := raw.(map[string]interface{}); ok {
			if raw, ok := c["alwaysMatch"]; ok {
				if am, ok := raw.(map[string]interface{}); ok {
					deleteStaleCapabilities(am)
				}
			}
			if raw, ok := c["firstMatch"]; ok {
				if fm, ok := raw.([]interface{}); ok {
					for _, raw := range fm {
						if c, ok := raw.(map[string]interface{}); ok {
							deleteStaleCapabilities(c)
						}
					}
				}
			}
		}
	}
	ret, _ := json.Marshal(body)
	return ret
}

func processBody(input []byte, host string) ([]byte, string, error) {
	body := make(map[string]interface{})
	sessionId := ""
	err := json.Unmarshal(input, &body)
	if err != nil {
		return nil, sessionId, fmt.Errorf("parse body response: %v", err)
	}
	// handle jsonwp response from older browsers (chrome < 75)
	if rawId, ok := body["sessionId"]; ok {
		// JSONWP drivers reply with HTTP 200 even on failure
		jsonwpFailed := false
		if rawStatus, ok := body["status"]; ok {
			if status, ok := rawStatus.(float64); ok && status != 0 {
				jsonwpFailed = true
			}
		}
		if si, ok := rawId.(string); ok && !jsonwpFailed {
			sessionId = si
			// wrap in the W3C envelope modern clients expect
			w3c := map[string]interface{}{"value": map[string]interface{}{
				"sessionId":    si,
				"capabilities": body["value"],
			}}
			if out, err := json.Marshal(w3c); err == nil {
				return out, sessionId, nil
			}
		}
	} else {
		if raw, ok := body["value"]; ok {
			if v, ok := raw.(map[string]interface{}); ok {
				if raw, ok := v["capabilities"]; ok {
					if c, ok := raw.(map[string]interface{}); ok {
						si, ok := v["sessionId"].(string)
						if !ok {
							return nil, "", fmt.Errorf("missing or invalid sessionId in driver response")
						}
						sessionId = si
						// WebKit and Gecko have no CDP endpoint; advertising one
						// makes clients fail their first WebSocket handshake.
						if bn, _ := c["browserName"].(string); bn != "safari" && bn != "firefox" {
							c["se:cdp"] = fmt.Sprintf("ws://%s/devtools/%s/", host, sessionId)
							if rbv, ok := c["browserVersion"]; ok {
								if bv, ok := rbv.(string); ok {
									c["se:cdpVersion"] = bv
								}
							}
						}
					}
				}
			}
		}
	}
	ret, err := json.Marshal(body)
	if err != nil {
		return nil, sessionId, fmt.Errorf("marshal response: %v", err)
	}
	return ret, sessionId, nil
}

func preprocessSessionId(sid string) string {
	if ggrHost != nil {
		return ggrHost.Sum() + sid
	}
	return sid
}

const (
	videoFileExtension = ".mp4"
	logFileExtension   = ".log"
)

var (
	fullFormat  = regexp.MustCompile(`^([0-9]+x[0-9]+)x(8|16|24)$`)
	shortFormat = regexp.MustCompile(`^[0-9]+x[0-9]+$`)
)

func getScreenResolution(input string) (string, error) {
	if input == "" {
		return "1920x1080x24", nil
	}
	if fullFormat.MatchString(input) {
		return input, nil
	}
	if shortFormat.MatchString(input) {
		return fmt.Sprintf("%sx24", input), nil
	}
	return "", fmt.Errorf(
		"malformed screenResolution capability: %s, correct format is WxH (1920x1080) or WxHxD (1920x1080x24)",
		input,
	)
}

func shortenScreenResolution(screenResolution string) string {
	return fullFormat.FindStringSubmatch(screenResolution)[1]
}

func getVideoScreenSize(videoScreenSize string, screenResolution string) (string, error) {
	if videoScreenSize != "" {
		if shortFormat.MatchString(videoScreenSize) {
			return videoScreenSize, nil
		}
		return "", fmt.Errorf(
			"malformed videoScreenSize capability: %s, correct format is WxH (1920x1080)",
			videoScreenSize,
		)
	}
	return shortenScreenResolution(screenResolution), nil
}

func getSessionTimeout(sessionTimeout string, maxTimeout time.Duration, defaultTimeout time.Duration) (time.Duration, error) {
	if sessionTimeout != "" {
		st, err := time.ParseDuration(sessionTimeout)
		if err != nil {
			return 0, fmt.Errorf("invalid sessionTimeout capability: %v", err)
		}
		if st <= maxTimeout {
			return st, nil
		}
		return maxTimeout, nil
	}
	return defaultTimeout, nil
}

func getTemporaryFileName(dir string, extension string) string {
	filename := ""
	for {
		filename = generateRandomFileName(extension)
		_, err := os.Stat(filepath.Join(dir, filename))
		if err != nil {
			break
		}
	}
	return filename
}

func generateRandomFileName(extension string) string {
	randBytes := make([]byte, 16)
	_, _ = rand.Read(randBytes)
	return "websummoner" + hex.EncodeToString(randBytes) + extension
}

// isSafeFileName reports whether name is a plain file name with no path elements.
func isSafeFileName(name string) bool {
	if name == "" || name == "." || name == ".." {
		return false
	}
	if strings.ContainsAny(name, `/\`) {
		return false
	}
	return name == filepath.Base(name)
}

const vendorPrefix = "websummoner"

// legacyVendorPrefix keeps upstream Selenoid clients working.
const legacyVendorPrefix = "aerokube"

func isVendorPrefix(fragment string) bool {
	return fragment == vendorPrefix || fragment == legacyVendorPrefix
}

func proxy(w http.ResponseWriter, r *http.Request) {
	if uploadToContainer(w, r) {
		return
	}
	done := make(chan func())
	go func() {
		(<-done)()
	}()
	cancel := func() {}
	defer func() {
		done <- cancel
	}()
	requestId := serial()
	(&httputil.ReverseProxy{
		Rewrite: func(pr *httputil.ProxyRequest) {
			pr.SetXForwarded()
			r := pr.Out
			fragments := strings.Split(r.URL.Path, slash)
			id := fragments[2]
			sess, ok := sessions.Get(id)
			if ok {
				if len(fragments) >= 5 && isVendorPrefix(fragments[3]) {
					newFragments := append([]string{"", fragments[4], id}, fragments[5:]...)
					r.URL.Host = (&request{r}).localaddr()
					r.URL.Path = path.Clean(strings.Join(newFragments, slash))
					return
				}
				sess.Lock.Lock()
				defer sess.Lock.Unlock()
				select {
				case <-sess.TimeoutCh:
				default:
					close(sess.TimeoutCh)
				}
				if r.Method == http.MethodDelete && len(fragments) == 3 {
					if enableFileUpload {
						_ = os.RemoveAll(filepath.Join(os.TempDir(), id))
					}
					cancel = sess.Cancel
					sessions.Remove(id)
					queue.Release()
					log.Printf("[%d] [SESSION_DELETED] [%s]", requestId, id)
					metricsSessionsDeleted.Add(1)
				} else {
					sess.TimeoutCh = onTimeout(sess.Timeout, func() {
						request{r}.session(id).Delete(requestId)
					})
					if len(fragments) == 4 && fragments[len(fragments)-1] == "file" && enableFileUpload {
						r.Header.Set("X-WebSummoner-File", filepath.Join(os.TempDir(), id))
						r.URL.Path = "/file"
						return
					}
				}
				seUploadPath, uploadPath := "/se/file", "/file"
				if strings.HasSuffix(r.URL.Path, seUploadPath) {
					r.URL.Path = strings.TrimSuffix(r.URL.Path, seUploadPath) + uploadPath
				}
				r.URL.Host, r.URL.Path = sess.URL.Host, path.Clean(sess.URL.Path+r.URL.Path)
				r.Host = "localhost"
				if sess.Origin != "" {
					r.Host = sess.Origin
				}
				return
			}
			r.URL.Path = paths.Error
		},
		ErrorHandler: defaultErrorHandler(requestId),
	}).ServeHTTP(w, r)
}

func defaultErrorHandler(requestId uint64) func(http.ResponseWriter, *http.Request, error) {
	return func(w http.ResponseWriter, r *http.Request, err error) {
		user, remote := info.RequestInfo(r)
		log.Printf("[%d] [CLIENT_DISCONNECTED] [%s] [%s] [Error: %v]", requestId, user, remote, err)
		w.WriteHeader(http.StatusBadGateway)
	}
}

func reverseProxy(hostFn func(sess *session.Session) string, status string) func(http.ResponseWriter, *http.Request) {
	return func(w http.ResponseWriter, r *http.Request) {
		requestId := serial()
		sid, remainingPath := splitRequestPath(r.URL.Path)
		sess, ok := sessions.Get(sid)
		if ok {
			select {
			case <-sess.TimeoutCh:
			default:
				close(sess.TimeoutCh)
			}
			sess.TimeoutCh = onTimeout(sess.Timeout, func() {
				request{r}.session(sid).Delete(requestId)
			})
			(&httputil.ReverseProxy{
				Rewrite: func(pr *httputil.ProxyRequest) {
					pr.SetXForwarded()
					r := pr.Out
					r.URL.Scheme = "http"
					r.URL.Host = hostFn(sess)
					r.URL.Path = remainingPath
					log.Printf("[%d] [%s] [%s] [%s]", requestId, status, sid, remainingPath)
				},
				ErrorHandler: defaultErrorHandler(requestId),
			}).ServeHTTP(w, r)
		} else {
			jsonerror.InvalidSessionID(fmt.Errorf("unknown session %s", sid)).Encode(w)
			log.Printf("[%d] [SESSION_NOT_FOUND] [%s]", requestId, sid)
		}
	}
}

func splitRequestPath(p string) (string, string) {
	fragments := strings.Split(p, slash)
	return fragments[2], slash + strings.Join(fragments[3:], slash)
}

// uploadToContainer serves file upload for browsers running in Docker.
//
// The hub's own /file handler unpacks onto the hub's filesystem, which a
// containerised browser cannot see, so historically only drivers that
// implement the upload endpoint themselves (chromedriver) or images that run a
// hub next to the driver (Firefox) could accept files. Everything else —
// WebKitWebDriver in particular — answered "Unknown command:
// /session/<id>/file". Copying the file straight into the browser container
// makes upload work for every image and needs no shared volume, which matters
// because the hub itself usually runs in a container.
//
// It also accepts the W3C "/se/file" spelling that Selenium 4 actually sends;
// the older path only matched the legacy "/file" form.
//
// Returns true when it has handled (or failed) the request.
func uploadToContainer(w http.ResponseWriter, r *http.Request) bool {
	if !enableFileUpload || r.Method != http.MethodPost || cli == nil {
		return false
	}
	fragments := strings.Split(r.URL.Path, slash)
	// /session/<id>/file  or  /session/<id>/se/file
	isUpload := (len(fragments) == 4 && fragments[3] == "file") ||
		(len(fragments) == 5 && fragments[3] == "se" && fragments[4] == "file")
	if len(fragments) < 4 || fragments[1] != "session" || !isUpload {
		return false
	}
	id := fragments[2]
	sess, ok := sessions.Get(id)
	if !ok || sess.Container == nil || sess.Container.ID == "" {
		// Driver mode, or an unknown session: leave it to the existing path.
		return false
	}

	name, content, err := singleFileFromZip(r.Body)
	if err != nil {
		jsonerror.InvalidArgument(err).Encode(w)
		return true
	}

	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()
	if err := writeIntoContainer(ctx, sess.Container.ID, name, content); err != nil {
		jsonerror.UnknownError(fmt.Errorf("placing uploaded file in the browser container: %v", err)).Encode(w)
		return true
	}
	uploaded := path.Join(uploadedFilesDir, name)

	w.Header().Set("Content-Type", "application/json")
	_ = json.NewEncoder(w).Encode(map[string]interface{}{"value": uploaded})
	return true
}

// uploadedFilesDir is where uploads land inside the browser container.
//
// A directory of our own rather than a shared system one: it cannot collide
// with the browser's own temporary files, and it is obvious what it is when
// someone looks inside a container.
//
// Deliberately NOT under /tmp: browsers.json mounts a tmpfs there, and copying
// into a path covered by a tmpfs mount silently writes to the image layer
// underneath — the running container never sees the file.
//
// The directory does not have to exist in the image. The archive carries its
// parent directories and is unpacked at /, so the tree is created on first
// upload; that keeps this working for custom images too.
const uploadedFilesDir = "/opt/websummoner/uploads"

// writeIntoContainer places one uploaded file inside the browser container.
//
// It prefers extracting from a process running in the container, because that
// honours the container's mounts: the Docker copy API does not, and writes
// underneath any volume or tmpfs covering the destination while still
// reporting success. Extracting in-container also means the file is owned by
// the browser's own user, the way chromedriver's built-in upload does it.
//
// Images without tar fall back to the copy API, which keeps this working for
// custom images — at the cost of not supporting a mounted destination.
func writeIntoContainer(ctx context.Context, containerID, name string, content []byte) error {
	execErr := extractInContainer(ctx, containerID, name, content)
	if execErr == nil {
		return nil
	}
	log.Printf("[-] [UPLOAD_EXEC_FALLBACK] [%s] [%v]", containerID, execErr)

	archive, err := uploadArchive(name, content)
	if err != nil {
		return err
	}
	if err := cli.CopyToContainer(ctx, containerID, slash, archive, container.CopyToContainerOptions{}); err != nil {
		return err
	}
	// The copy API cannot write through a mount, so make sure the browser can
	// actually see what we just wrote rather than hand it a phantom path.
	visible, err := fileVisibleInContainer(ctx, containerID, path.Join(uploadedFilesDir, name))
	if err != nil {
		return nil // cannot verify; the copy itself reported success
	}
	if !visible {
		return fmt.Errorf(
			"file is not visible at %s inside the container — is a volume or tmpfs mounted over %s?",
			path.Join(uploadedFilesDir, name), uploadedFilesDir)
	}
	return nil
}

// extractInContainer streams a tar to `tar -x` running inside the container.
func extractInContainer(ctx context.Context, containerID, name string, content []byte) error {
	// uploadedFilesDir is a compile-time constant, so nothing user-supplied
	// reaches this shell. Sticky-writable like /tmp, so the unprivileged
	// browser user can write there.
	if err := runInContainer(ctx, containerID, "root", nil,
		"sh", "-c", "mkdir -p "+uploadedFilesDir+" && chmod 1777 "+uploadedFilesDir); err != nil {
		return err
	}
	archive, err := flatArchive(name, content)
	if err != nil {
		return err
	}
	// No shell here: the file name is user-supplied and is only ever a tar
	// entry, never an argument.
	return runInContainer(ctx, containerID, "", archive.Bytes(), "tar", "-x", "-f", "-", "-C", uploadedFilesDir)
}

// runInContainer executes argv in the container, optionally feeding stdin, and
// fails unless it exits zero. An empty user means the image's own user.
func runInContainer(ctx context.Context, containerID, user string, stdin []byte, argv ...string) error {
	exec, err := cli.ContainerExecCreate(ctx, containerID, container.ExecOptions{
		Cmd:          argv,
		User:         user,
		AttachStdin:  stdin != nil,
		AttachStderr: true,
	})
	if err != nil {
		return err
	}
	resp, err := cli.ContainerExecAttach(ctx, exec.ID, container.ExecAttachOptions{})
	if err != nil {
		return err
	}
	defer resp.Close()
	if stdin != nil {
		if _, err := resp.Conn.Write(stdin); err != nil {
			return err
		}
		if cw, ok := resp.Conn.(interface{ CloseWrite() error }); ok {
			_ = cw.CloseWrite()
		}
	}
	_, _ = io.Copy(io.Discard, resp.Reader)
	for i := 0; i < 100; i++ {
		info, err := cli.ContainerExecInspect(ctx, exec.ID)
		if err != nil {
			return err
		}
		if !info.Running {
			if info.ExitCode != 0 {
				return fmt.Errorf("%v exited %d", argv, info.ExitCode)
			}
			return nil
		}
		time.Sleep(20 * time.Millisecond)
	}
	return fmt.Errorf("%v timed out", argv)
}

// flatArchive is a tar holding just the file, for extraction into a directory
// that already exists.
func flatArchive(name string, content []byte) (*bytes.Buffer, error) {
	buf := &bytes.Buffer{}
	tw := tar.NewWriter(buf)
	if err := tw.WriteHeader(&tar.Header{
		Name: name, Typeflag: tar.TypeReg, Mode: 0600,
		Size: int64(len(content)), ModTime: time.Now(),
	}); err != nil {
		return nil, err
	}
	if _, err := tw.Write(content); err != nil {
		return nil, err
	}
	return buf, tw.Close()
}

// fileVisibleInContainer runs a check in the container's own mount namespace,
// which is the only way to see what the browser will see.
func fileVisibleInContainer(ctx context.Context, containerID, file string) (bool, error) {
	exec, err := cli.ContainerExecCreate(ctx, containerID, container.ExecOptions{
		Cmd: []string{"test", "-f", file},
	})
	if err != nil {
		return false, err
	}
	if err := cli.ContainerExecStart(ctx, exec.ID, container.ExecStartOptions{}); err != nil {
		return false, err
	}
	for i := 0; i < 50; i++ {
		info, err := cli.ContainerExecInspect(ctx, exec.ID)
		if err != nil {
			return false, err
		}
		if !info.Running {
			return info.ExitCode == 0, nil
		}
		time.Sleep(20 * time.Millisecond)
	}
	return false, fmt.Errorf("timed out checking %s", file)
}

// uploadArchive builds a tar rooted at / that carries uploadedFilesDir and the
// uploaded file, so unpacking it creates the directory tree when the image does
// not already have it. World-readable, because the browser runs as a different
// user than the one the copy is performed as.
func uploadArchive(name string, content []byte) (*bytes.Buffer, error) {
	buf := &bytes.Buffer{}
	tw := tar.NewWriter(buf)
	now := time.Now()
	var dir string
	for _, part := range strings.Split(strings.Trim(uploadedFilesDir, slash), slash) {
		dir = path.Join(dir, part)
		if err := tw.WriteHeader(&tar.Header{
			Name: dir + slash, Typeflag: tar.TypeDir, Mode: 0755, ModTime: now,
		}); err != nil {
			return nil, err
		}
	}
	if err := tw.WriteHeader(&tar.Header{
		Name: path.Join(dir, name), Typeflag: tar.TypeReg, Mode: 0644,
		Size: int64(len(content)), ModTime: now,
	}); err != nil {
		return nil, err
	}
	if _, err := tw.Write(content); err != nil {
		return nil, err
	}
	if err := tw.Close(); err != nil {
		return nil, err
	}
	return buf, nil
}

func singleFileFromZip(body io.Reader) (string, []byte, error) {
	var jsonRequest struct {
		File []byte `json:"file"`
	}
	if err := json.NewDecoder(body).Decode(&jsonRequest); err != nil {
		return "", nil, err
	}
	z, err := zip.NewReader(bytes.NewReader(jsonRequest.File), int64(len(jsonRequest.File)))
	if err != nil {
		return "", nil, err
	}
	if len(z.File) != 1 {
		return "", nil, fmt.Errorf("expected there to be only 1 file. There were: %d", len(z.File))
	}
	f := z.File[0]
	if !isSafeFileName(f.Name) {
		return "", nil, fmt.Errorf("invalid file name in archive: %s", f.Name)
	}
	src, err := f.Open()
	if err != nil {
		return "", nil, err
	}
	defer src.Close()
	data, err := io.ReadAll(src)
	if err != nil {
		return "", nil, err
	}
	return f.Name, data, nil
}

func fileUpload(w http.ResponseWriter, r *http.Request) {
	var jsonRequest struct {
		File []byte `json:"file"`
	}
	err := json.NewDecoder(r.Body).Decode(&jsonRequest)
	if err != nil {
		jsonerror.InvalidArgument(err).Encode(w)
		return
	}
	z, err := zip.NewReader(bytes.NewReader(jsonRequest.File), int64(len(jsonRequest.File)))
	if err != nil {
		jsonerror.InvalidArgument(err).Encode(w)
		return
	}
	if len(z.File) != 1 {
		err := fmt.Errorf("expected there to be only 1 file. There were: %d", len(z.File))
		jsonerror.InvalidArgument(err).Encode(w)
		return
	}
	file := z.File[0]
	// zip entry names are untrusted
	if !isSafeFileName(file.Name) {
		jsonerror.InvalidArgument(fmt.Errorf("invalid file name in archive: %s", file.Name)).Encode(w)
		return
	}
	src, err := file.Open()
	if err != nil {
		jsonerror.InvalidArgument(err).Encode(w)
		return
	}
	defer src.Close()
	dir := r.Header.Get("X-WebSummoner-File")
	if dir == "" {
		dir = r.Header.Get("X-Selenoid-File")
	}
	err = os.MkdirAll(dir, 0755)
	if err != nil {
		jsonerror.UnknownError(err).Encode(w)
		return
	}
	fileName := filepath.Join(dir, file.Name)
	dst, err := os.OpenFile(fileName, os.O_WRONLY|os.O_CREATE, 0644)
	if err != nil {
		jsonerror.UnknownError(err).Encode(w)
		return
	}
	defer dst.Close()
	_, err = io.Copy(dst, src)
	if err != nil {
		jsonerror.UnknownError(err).Encode(w)
		return
	}

	reply := struct {
		V string `json:"value"`
	}{
		V: fileName,
	}
	_ = json.NewEncoder(w).Encode(reply)
}

func vnc(wsconn *websocket.Conn) {
	defer wsconn.Close()
	requestId := serial()
	sid, _ := splitRequestPath(wsconn.Request().URL.Path)
	sess, ok := sessions.Get(sid)
	if ok {
		vncHostPort := sess.HostPort.VNC
		if vncHostPort != "" {
			log.Printf("[%d] [VNC_ENABLED] [%s]", requestId, sid)
			var d net.Dialer
			conn, err := d.DialContext(wsconn.Request().Context(), "tcp", vncHostPort)
			if err != nil {
				log.Printf("[%d] [VNC_ERROR] [%v]", requestId, err)
				return
			}
			defer conn.Close()
			wsconn.PayloadType = websocket.BinaryFrame
			go func() {
				_, _ = io.Copy(wsconn, conn)
				_ = wsconn.Close()
				log.Printf("[%d] [VNC_SESSION_CLOSED] [%s]", requestId, sid)
			}()
			_, _ = io.Copy(conn, wsconn)
			log.Printf("[%d] [VNC_CLIENT_DISCONNECTED] [%s]", requestId, sid)
		} else {
			log.Printf("[%d] [VNC_NOT_ENABLED] [%s]", requestId, sid)
		}
	} else {
		log.Printf("[%d] [SESSION_NOT_FOUND] [%s]", requestId, sid)
	}
}

const (
	jsonParam = "json"
)

func logs(w http.ResponseWriter, r *http.Request) {
	requestId := serial()
	fileNameOrSessionID := strings.TrimPrefix(r.URL.Path, paths.Logs)
	if logOutputDir != "" && (fileNameOrSessionID == "" || strings.HasSuffix(fileNameOrSessionID, logFileExtension)) {
		if r.Method == http.MethodDelete {
			deleteFileIfExists(requestId, w, r, logOutputDir, paths.Logs, "DELETED_LOG_FILE")
			return
		}
		user, remote := info.RequestInfo(r)
		if _, ok := r.URL.Query()[jsonParam]; ok {
			listFilesAsJson(requestId, w, logOutputDir, "LOG_ERROR")
			return
		}
		log.Printf("[%d] [LOG_LISTING] [%s] [%s]", requestId, user, remote)
		fileServer := http.StripPrefix(paths.Logs, http.FileServer(http.Dir(logOutputDir)))
		fileServer.ServeHTTP(w, r)
		return
	}
	websocket.Handler(streamLogs).ServeHTTP(w, r)
}

func listFilesAsJson(requestId uint64, w http.ResponseWriter, dir string, errStatus string) {
	files, err := os.ReadDir(dir)
	if err != nil {
		log.Printf("[%d] [%s] [%s]", requestId, errStatus, fmt.Sprintf("Failed to list directory %s: %v", logOutputDir, err))
		w.WriteHeader(http.StatusInternalServerError)
		return
	}
	var ret []string
	for _, f := range files {
		ret = append(ret, f.Name())
	}
	w.Header().Add("Content-Type", "application/json")
	_ = json.NewEncoder(w).Encode(ret)
}

func streamLogs(wsconn *websocket.Conn) {
	defer wsconn.Close()
	requestId := serial()
	sid, _ := splitRequestPath(wsconn.Request().URL.Path)
	sess, ok := sessions.Get(sid)
	if ok && sess.Container != nil {
		log.Printf("[%d] [CONTAINER_LOGS] [%s]", requestId, sess.Container.ID)
		r, err := cli.ContainerLogs(wsconn.Request().Context(), sess.Container.ID, container.LogsOptions{
			ShowStdout: true,
			ShowStderr: true,
			Follow:     true,
		})
		if err != nil {
			log.Printf("[%d] [CONTAINER_LOGS_ERROR] [%v]", requestId, err)
			return
		}
		defer r.Close()
		wsconn.PayloadType = websocket.BinaryFrame
		_, _ = stdcopy.StdCopy(wsconn, wsconn, r)
		log.Printf("[%d] [CONTAINER_LOGS_DISCONNECTED] [%s]", requestId, sid)
	} else {
		log.Printf("[%d] [SESSION_NOT_FOUND] [%s]", requestId, sid)
	}
}

func status(w http.ResponseWriter, _ *http.Request) {
	w.Header().Set("Content-Type", "application/json")
	ready := limit > sessions.Len()
	_ = json.NewEncoder(w).Encode(
		map[string]interface{}{
			"value": map[string]interface{}{
				"message": fmt.Sprintf("WebSummoner %s built at %s", gitRevision, buildStamp),
				"ready":   ready,
			},
		})
}

func welcome(w http.ResponseWriter, _ *http.Request) {
	w.WriteHeader(http.StatusOK)
	_, _ = fmt.Fprintf(w, "You are using WebSummoner %s!", gitRevision)
}

func onTimeout(t time.Duration, f func()) chan struct{} {
	cancel := make(chan struct{})
	go func(cancel chan struct{}) {
		select {
		case <-time.After(t):
			f()
		case <-cancel:
		}
	}(cancel)
	return cancel
}
