// Go support for leveled logs, analogous to https://code.google.com/p/google-glog/
//
// Copyright 2013 Google Inc. All Rights Reserved.
//
// Licensed under the Apache License, Version 2.0 (the "License");
// you may not use this file except in compliance with the License.
// You may obtain a copy of the License at
//
//     http://www.apache.org/licenses/LICENSE-2.0
//
// Unless required by applicable law or agreed to in writing, software
// distributed under the License is distributed on an "AS IS" BASIS,
// WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
// See the License for the specific language governing permissions and
// limitations under the License.

// File I/O for logs.

package glog

import (
	// "encoding/json"
	"errors"
	"flag"
	"fmt"
	// "io/ioutil"
	"net"
	// "net/http"
	"os"
	"os/user"
	"path/filepath"
	"strings"
	"sync"
	"time"
)

// MaxSize is the maximum size of a log file in bytes.
var MaxSize uint64 = 1024 * 1024 * 1800

// logDirs lists the candidate directories for new log files.
var logDirs []string

var currentLogFile string

// If non-empty, overrides the choice of directory in which to write logs.
// See createLogDirs for the full list of possible destinations.
var logDir = flag.String("log_dir_deepglint", "/tmp/", "If non-empty, write log files in this directory")

func createLogDirs() {
	if *logDir != "" {
		logDirs = append(logDirs, *logDir)
	}
	logDirs = append(logDirs, "/tmp/")
}

var (
	pid      = os.Getpid()
	program  = filepath.Base(os.Args[0])
	host     = "unknownhost"
	userName = "unknownuser"
	ip       = "unknownip"
)

type SensorId struct {
	Key           string
	Value         string
	ModifiedIndex int
	CreatedIndex  int
}

type Sensor struct {
	Action string
	Node   SensorId
}

func init() {
	// resp, err := http.Get("http://localhost:4001/v2/keys/config/global/sensor_uid")
	// if err != nil || resp.StatusCode != 200 {
	// 	// fmt.Printf("%v", err)
	// 	// fmt.Println(os.Hostname())
	// 	h, err := os.Hostname()
	// 	if err == nil {
	// 		host = shortHostname(h)
	// 	}
	// } else {
	// 	defer resp.Body.Close()
	// 	body, err := ioutil.ReadAll(resp.Body)
	// 	fmt.Printf("%s", string(body))
	// 	var sen Sensor
	// 	err = json.Unmarshal(body, &sen)
	// 	if err == nil {
	// 		host = sen.Node.Value
	// 	}

	// }
	current, err := user.Current()
	if err == nil {
		userName = current.Username
	}
	// Sanitize userName since it may contain filepath separators on Windows.
	userName = strings.Replace(userName, `\`, "_", -1)

	// externalip, err := getIP()
	// if err == nil {
	// 	ip = externalip
	// }
}

func getIP() (string, error) {
	ifaces, err := net.Interfaces()
	if err != nil {
		return "", err
	}
	for _, iface := range ifaces {
		if iface.Flags&net.FlagUp == 0 {
			continue // interface down
		}
		if iface.Flags&net.FlagLoopback != 0 {
			continue // loopback interface
		}
		addrs, err := iface.Addrs()
		if err != nil {
			return "", err
		}
		for _, addr := range addrs {
			var ip net.IP
			switch v := addr.(type) {
			case *net.IPNet:
				ip = v.IP
			case *net.IPAddr:
				ip = v.IP
			}
			if ip == nil || ip.IsLoopback() {
				continue
			}
			ip = ip.To4()
			if ip == nil {
				continue // not an ipv4 address
			}
			return ip.String(), nil
		}
	}
	return "", errors.New("Error connecting to network")
}

// shortHostname returns its argument, truncating at the first period.
// For instance, given "www.google.com" it returns "www".
func shortHostname(hostname string) string {
	if i := strings.Index(hostname, "."); i >= 0 {
		return hostname[:i]
	}
	return hostname
}

// logName returns a new log file name containing tag, with start time t, and
// the name for the symlink for tag.
func logName(tag string, t time.Time) (name, link string) {
	// name = fmt.Sprintf("%s.%s.%s.log.%s.%04d%02d%02d-%02d%02d%02d.%d",
	// 	program,
	// 	host,
	// 	userName,
	// 	tag,
	// 	t.Year(),
	// 	t.Month(),
	// 	t.Day(),
	// 	t.Hour(),
	// 	t.Minute(),
	// 	t.Second(),
	// 	pid)
	//name = fmt.Sprintf(".%s", tag)
	if !logging.debug {
		return fmt.Sprintf("LOG.%4d-%02d-%02dT00:00:00Z", time.Now().Year(), time.Now().Month(), time.Now().Day()), "LOG.IWEF." + time.Now().Round(time.Hour*24).Format("2006-01-02T15:04:05Z")
	} else {
		return "DEBUGLOGS." + tag, "DEBUG" + name + "." + program
	}
}

var onceLogDirs sync.Once

// create creates a new log file and returns the file and its filename, which
// contains tag ("INFO", "FATAL", etc.) and t.  If the file is created
// successfully, create also attempts to update the symlink for that tag, ignoring
// errors.
func create(tag string, t time.Time) (f *os.File, filename string, err error) {
	// onceLogDirs.Do(createLogDirs)
	// if len(logDirs) == 0 {
	// 	return nil, "", errors.New("log: no log dirs")
	// }
	name, link := logName(tag, t)
	// name, _ := logName(tag, t)
	// var lastErr error
	// for _, dir := range logDirs {
	fname := filepath.Join(*logDir, name)
	currentLogFile = fname
	//f, err := os.Create(fname)
	fd, err := os.OpenFile(fname, os.O_RDWR|os.O_CREATE|os.O_APPEND, 0666)
	if err == nil {
		if logging.debug {
			symlink := filepath.Join(*logDir, link)
			os.Remove(symlink)        // ignore err
			os.Symlink(name, symlink) // ignore err
		}
		return fd, fname, nil
	}
	// lastErr = err
	// }
	return nil, "", fmt.Errorf("log: cannot create log: %v", err)
}
