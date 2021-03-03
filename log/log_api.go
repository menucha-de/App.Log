package log

import (
	"bytes"
	"database/sql"
	"encoding/csv"
	"encoding/json"
	"fmt"
	"net/http"
	"strconv"
	"strings"
	"time"

	mqtt "github.com/eclipse/paho.mqtt.golang"

	"github.com/gorilla/mux"
	_ "github.com/mattn/go-sqlite3"
	"github.com/peramic/App.Containerd/go/containers"
	"github.com/peramic/utils"
	"github.com/sirupsen/logrus"
)

var db *sql.DB
var create string = "CREATE TABLE IF NOT EXISTS record ( millis TIMESTAMP,host TEXT, logger TEXT, method TEXT, level INT, message TEXT, thrown TEXT)"
var createTarget string = "CREATE TABLE IF NOT EXISTS targets(host TEXT,target TEXT)"

var insert string = "INSERT INTO record (millis,host,logger,  method, level, message,thrown) VALUES (?, ?, ?, ?, ?,?,?)"
var insertTarget string = "INSERT into targets(host,target) VALUES(?, ?)"
var size string = "SELECT COUNT(*) AS size FROM record WHERE host like ? and logger like ? and level <= ?"
var selectlogasc string = "SELECT rowid, millis,host, logger,  method, level, message,  thrown FROM record  WHERE host like ? AND logger LIKE ? AND level <= ? ORDER BY millis asc LIMIT ? OFFSET ?"
var selectlogdesc string = "SELECT rowid, millis,host, logger,  method, level, message,  thrown FROM record  WHERE host like ? AND logger LIKE ? AND level <= ? ORDER BY millis desc LIMIT ? OFFSET ?"

var clear string = "DELETE FROM record WHERE host like ? and logger like ?"
var trunc string = "DELETE FROM record WHERE rowid < ?"

var targets string = "SELECT DISTINCT host,target  FROM targets order by  host ,target"

//var hosts string = "SELECT DISTINCT host FROM targets"
var max int64 = 500

//TargetStat target
type TargetStat struct {
	Name  string `json:"name"`
	Level string `json:"level"`
}

//HostStats hosts with status
type HostStats struct {
	Targets []TargetStat `json:"targets"`
	Online  bool         `json:"online"`
}

//Logger custom logger
type Logger struct {
	*logrus.Logger
	off bool
}

//Entry log  entry
type Entry struct {
	Time         string `json:"time"`
	App          string `json:"app"`
	TargetName   string `json:"targetName"`
	SourceMethod string `json:"sourceMethod"`
	Level        string `json:"level"`
	Message      string `json:"message"`
	Thrown       string `json:"thrown"`
}

const (
	// error messages
	notAvailable = "Log level \"%s\" is not available"
)

func (e Entry) record() []string {
	return []string{
		e.Time,
		e.App,
		e.TargetName,
		e.SourceMethod,
		e.Level,
		e.Message,
		e.Thrown,
	}
}

var mapLogger map[string]map[string]struct{} = make(map[string]map[string]struct{})
var initErr error
var myClient = &http.Client{Timeout: 3 * time.Second}
var rclient utils.Client

func init() {

	db, initErr = sql.Open("sqlite3", "./log.db")
	if initErr != nil {
		logrus.Error(initErr)
	}
	_, initErr = db.Exec(create)

	if initErr != nil {
		logrus.Error(initErr)
	}
	_, initErr = db.Exec(createTarget)

	if initErr != nil {
		logrus.Error(initErr)
	}
	rows, err := db.Query(targets)
	if err != nil {
		logrus.Error(err)
	}

	defer rows.Close()
	var mhost, mtarget string
	for rows.Next() {
		err := rows.Scan(&mhost, &mtarget)
		if err != nil {
			logrus.Error(err)
		}

		v, ok := mapLogger[mhost]
		if !ok {
			a := make(map[string]struct{})
			a[mtarget] = struct{}{}
			mapLogger[mhost] = a
		} else {
			v[mtarget] = struct{}{}
		}
	}
	rclient.ServerAdress = "art:8080"
}

func getLogLevels(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "application/json; charset=UTF-8")
	var levels [8]string
	var i int
	levels[0] = "OFF"
	i++
	for _, v := range logrus.AllLevels {
		x, _ := v.MarshalText()
		xx := string(bytes.ToUpper(x))
		levels[i] = xx
		i++
	}
	//levels[i] = "ALL"
	enc := json.NewEncoder(w)
	err := enc.Encode(levels)

	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
	}
}

func getLogTargets(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "application/json; charset=UTF-8")
	vars := mux.Vars(r)
	host := strings.TrimSpace(vars["host"])
	targets := getTargets(host)
	enc := json.NewEncoder(w)
	err := enc.Encode(targets)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
	}

}
func getLogHosts(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "application/json; charset=UTF-8")

	hosts := getHosts()
	enc := json.NewEncoder(w)
	err := enc.Encode(hosts)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
	}

}

func getAvailableHosts() map[string]bool {

	var cons []containers.Container
	err := rclient.Call("Service.GetContainers", "default", &cons)

	m := make(map[string]bool)
	if err == nil {
		for _, c := range cons {

			if c.State == "STARTED" {
				m[c.Name] = true
			}
		}

	}
	err = rclient.Call("Service.GetContainers", "system", &cons)
	if err == nil {
		for _, c := range cons {

			if c.State == "STARTED" {
				m[c.Name] = true
			}
		}

	}
	return m

}

func getLogSize(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "application/json; charset=UTF-8")
	vars := mux.Vars(r)
	level := vars["level"]
	host := strings.TrimSpace(vars["host"])
	var lv int
	if level == "ALL" {
		lv = 10
	} else {
		lvv, err := logrus.ParseLevel(strings.ToLower(level))
		if err != nil {
			errMsg := fmt.Sprintf(notAvailable, level)
			http.Error(w, errMsg, http.StatusBadRequest)
			return
		}
		lv = int(lvv)
	}
	target := vars["target"]

	size := getSize(host, target, lv)
	enc := json.NewEncoder(w)
	err := enc.Encode(size)

	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
	}
}

func getLogEntries(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "application/json; charset=UTF-8")
	vars := mux.Vars(r)
	target := vars["target"]
	level := vars["level"]
	host := strings.TrimSpace(vars["host"])
	var lv int
	if level == "ALL" {
		lv = 10
	} else {
		lvv, err := logrus.ParseLevel(strings.ToLower(level))
		if err != nil {
			errMsg := fmt.Sprintf(notAvailable, level)
			http.Error(w, errMsg, http.StatusBadRequest)
			return
		}
		lv = int(lvv)
	}
	limit := vars["limit"]
	offset := vars["offset"]
	order := vars["order"]
	lim, _ := strconv.Atoi(limit)
	offs, _ := strconv.Atoi(offset)

	res := getLogs(host, target, lv, lim, offs, order)
	enc := json.NewEncoder(w)
	err := enc.Encode(res)

	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
	}
}

func deleteLogEntries(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "application/json; charset=UTF-8")
	vars := mux.Vars(r)
	target := vars["target"]
	host := strings.TrimSpace(vars["host"])
	err := deleteLogs(host, target)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
	w.WriteHeader(http.StatusNoContent)
}

func getLogFile(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "application/json; charset=UTF-8")
	vars := mux.Vars(r)
	host := strings.TrimSpace(vars["host"])
	target := vars["target"]
	level := vars["level"]
	order := vars["order"]
	t := time.Now()
	formatted := fmt.Sprintf("%d-%02d-%02dT%02d:%02d:%02d",
		t.Year(), t.Month(), t.Day(),
		t.Hour(), t.Minute(), t.Second())
	fileName := fmt.Sprintf("Log_%s_%s_%s_%s.txt", host, target, level, formatted)
	var lv int
	if level == "ALL" {
		lv = 10
	} else {
		lvv, err := logrus.ParseLevel(strings.ToLower(level))
		if err != nil {
			errMsg := fmt.Sprintf(notAvailable, level)
			http.Error(w, errMsg, http.StatusBadRequest)
			return
		}
		lv = int(lvv)
	}
	res := getLogs(host, target, lv, -1, 0, order)
	b := &bytes.Buffer{}
	wr := csv.NewWriter(b)
	for _, v := range res {
		wr.Write(v.record())
	}
	wr.Flush()

	// data := []byte("")
	// for _, v := range res {
	// 	s, err := json.Marshal(v)
	// 	if err == nil {
	// 		data = append(data, s...)
	// 		data = append(data, []byte{10}...)
	// 	}
	// }

	w.Header().Set("Content-Disposition", "attachment; filename="+fileName)
	w.Header().Set("Content-Type", "text/plain; charset=utf-8")
	w.Write(b.Bytes())
	return
}

func setLogLevel(w http.ResponseWriter, r *http.Request) {
	vars := mux.Vars(r)
	target := vars["target"]
	host := strings.TrimSpace(vars["host"])
	var b bytes.Buffer
	n, err := b.ReadFrom(r.Body)
	if err != nil || n == 0 {
		http.Error(w, "Could not read level value", http.StatusBadRequest)

		return
	}
	defer r.Body.Close()
	level := b.String()
	hclient := http.Client{Timeout: 2 * time.Second}
	json, _ := json.Marshal(level)

	url := "http://" + host + "/rest/log/" + target + "/level"
	req, err := http.NewRequest("PUT", url, bytes.NewBuffer(json))
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
	req.Header.Set("Content-Type", "application/json; charset=UTF-8")
	resp, err := hclient.Do(req)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
	defer resp.Body.Close()
	if resp.StatusCode != 200 {
		http.Error(w, http.StatusText(resp.StatusCode), resp.StatusCode)
		return
	}
	w.WriteHeader(http.StatusOK)
}

//--------impl
func getTargets(host string) []string {
	var s []string
	s = make([]string, 0)

	h, ok := mapLogger[host]
	if !ok {
		return s
	}

	for m := range h {
		s = append(s, m)
	}
	return s
}

func getHosts() map[string]HostStats {
	m := getAvailableHosts()
	res := make(map[string]HostStats)

	for host, targets := range mapLogger {
		s := make([]string, 0)
		for target := range targets {
			s = append(s, target)
		}
		s = append(s, "ALL")
		_, ok := m[host]
		if host == "art" {
			ok = true
		}
		p := getLevelsforTargets(host, s, ok)
		hh := HostStats{Targets: p, Online: ok}
		res[host] = hh
	}

	res["ALL"] = HostStats{Targets: []TargetStat{TargetStat{Name: "ALL", Level: ""}}, Online: false}
	return res
}
func getLevelsforTargets(host string, s []string, ok bool) []TargetStat {
	client := http.Client{Timeout: 2 * time.Second}
	res := make([]TargetStat, 0, len(s))
	if ok {
		for _, targetname := range s {
			level := ""
			if targetname != "ALL" {
				url := "http://" + host + "/rest/log/" + targetname + "/level"
				request, err := http.NewRequest("GET", url, nil)

				if err == nil {
					resp, err := client.Do(request)
					if err == nil {
						defer resp.Body.Close()
						if resp.StatusCode == 200 {
							err = json.NewDecoder(resp.Body).Decode(&level)
							if err != nil {
								level = ""
								logrus.Error(err)
							}
							//body, err := ioutil.ReadAll(resp.Body)
							//if err == nil {
							//	level = string(body)
							//}
						}
					}
				}

			}
			if level == "" {
				size := getSize(host, targetname, 10)
				if size > 0 {
					res = append(res, TargetStat{targetname, level})
				}
			} else {
				res = append(res, TargetStat{targetname, level})
			}
		}
	} else {
		for _, targetname := range s {
			size := getSize(host, targetname, 10)
			if size > 0 {
				res = append(res, TargetStat{targetname, ""})
			}
		}

	}
	return res
}
func getSize(host string, name string, level int) int {

	var count int
	if name == "ALL" {
		name = "%"
	}
	if host == "ALL" {
		host = "%"
	}

	err := db.QueryRow(size, host, name, level).Scan(&count)
	switch {
	case err == sql.ErrNoRows:
		return 0
	case err != nil:
		logrus.Error("query error:" + err.Error())
		return 0
	default:
		return count
	}

}

func getLogs(host string, logger string, level int, limit int, offset int, order string) []Entry {

	if logger == "ALL" {
		logger = "%"
	}
	if host == "ALL" {
		host = "%"
	}
	var rows *sql.Rows
	var err error
	if order == "asc" {
		rows, err = db.Query(selectlogasc, host, logger, level, limit, offset)
	} else {
		rows, err = db.Query(selectlogdesc, host, logger, level, limit, offset)

	}

	if err != nil {
		logrus.Error(err)
		return nil
	}

	defer rows.Close()
	var s []Entry
	s = make([]Entry, 0)
	var (
		mlogger, mmethod, mhost, message, thrown sql.NullString
		mlevel, id                               int
		millis                                   time.Time
	)
	for rows.Next() {

		err := rows.Scan(&id, &millis, &mhost, &mlogger, &mmethod, &mlevel, &message, &thrown)
		if err != nil {
			logrus.Error(err)
		}
		i := logrus.Level(mlevel)
		var lv string
		if b, err := i.MarshalText(); err == nil {
			lv = strings.ToUpper(string(b))
		} else {
			lv = "unknown"
		}
		b := Entry{
			Time:         time.Unix(0, millis.UnixNano()).Format("2006-01-02 15:04:05.000"),
			App:          mhost.String,
			TargetName:   mlogger.String,
			SourceMethod: mmethod.String,
			Level:        lv,
			Message:      message.String,
			Thrown:       thrown.String,
		}

		s = append(s, b)
	}

	return s
}

func deleteLogs(host string, target string) error {

	if target == "ALL" {
		target = "%"
	}
	if host == "ALL" {
		host = "%"
	}
	_, err := db.Exec(clear, host, target)
	return err

}

func savetoDatabase(e map[string]interface{}, topic string) {
	if initErr != nil {
		return
	}
	l, err := logrus.ParseLevel(strings.ToUpper(fmt.Sprintf("%v", e["level"])))
	if err != nil {
		l = 6
	}
	res, err := db.Exec(insert,

		e["time"],
		e["host"],
		topic[4:len(topic)],
		e["file"],
		l,
		e["msg"],
		e["error"],
	)
	if err != nil {
		logrus.Error(err)
	} else {
		id, err := res.LastInsertId()
		if err != nil {
			logrus.Error(err)
		} else {
			if id > max {
				db.Exec(trunc, id-max+1)
			}
		}
	}
	host := e["host"].(string)
	target := topic[4:len(topic)]
	v, ok := mapLogger[host]
	if !ok {
		a := make(map[string]struct{})
		a[target] = struct{}{}
		mapLogger[host] = a
		saveTarget(host, target)
	} else {
		_, ok := v[target]
		if !ok {
			v[target] = struct{}{}
			saveTarget(host, target)
		}
	}
}

//MsgRcvdLog handler for log messages
var MsgRcvdLog mqtt.MessageHandler = func(client mqtt.Client, msg mqtt.Message) {
	//save to database
	var e map[string]interface{}

	if err := json.Unmarshal(msg.Payload(), &e); err != nil {
		logrus.Error(err)
	}
	savetoDatabase(e, msg.Topic())

}

//MsgRcvdTarget handler for topic messages
var MsgRcvdTarget mqtt.MessageHandler = func(client mqtt.Client, msg mqtt.Message) {
	var e map[string]interface{}

	if err := json.Unmarshal(msg.Payload(), &e); err != nil {
		logrus.Error(err)
	}
	if e["host"] == nil || e["target"] == nil {
		return
	}
	var host, target string
	var ok bool
	if host, ok = e["host"].(string); !ok {

		return
	}
	if target, ok = e["target"].(string); !ok {
		return
	}
	v, ok := mapLogger[host]
	if !ok {
		a := make(map[string]struct{})
		a[target] = struct{}{}
		mapLogger[host] = a
		saveTarget(host, target)
	} else {
		_, ok := v[target]
		if !ok {
			v[target] = struct{}{}
			saveTarget(host, target)
		}
	}
}

func saveTarget(host string, target string) {
	if initErr != nil {
		return
	}

	_, err := db.Exec(insertTarget,
		host,
		target,
	)
	if err != nil {
		logrus.Error(err)
	}
}
