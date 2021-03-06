package main

import (
	"encoding/json"
	"fmt"
	"github.com/gorilla/websocket"
	"math/rand"
	"net/http"
	"strconv"
	"strings"
)

const (
	// handshake message
	// message send format
	connectMsg                  = `["{\"msg\":\"connect\",\"version\":\"1\",\"support\":[\"1\",\"pre2\",\"pre1\"]}"]`
	publicTasksCounterMsgFormat = `["{\"msg\":\"method\",\"method\":\"publicTasksCounter\",\"params\":[%s],\"id\":\"%s\"}"]`
	publicTasksMsgFormat        = `["{\"msg\":\"sub\",\"id\":\"%s\",\"name\":\"publicTasks\",\"params\":[%d,%d,%s]}"]`
	processesMsgFormat          = `["{\"msg\":\"sub\",\"id\":\"%s\",\"name\":\"process\",\"params\":[{\"taskID\":{\"$type\":\"oid\",\"$value\":\"%s\"},\"status\":100,\"important\":true}]}"]`
	allIncidentsMsgFormat       = `["{\"msg\":\"sub\",\"id\":\"%s\",\"name\":\"allIncidents\",\"params\":[{\"$type\":\"oid\",\"$value\":\"%s\"}]}"]`
	
	dnsMsgFormat                = `["{\"msg\":\"sub\",\"id\":\"%s\",\"name\":\"dns\",\"params\":[{\"task\":{\"$type\":\"oid\",\"$value\":\"%s\"},\"searchParam\":\"\"},100]}"]`

    ipsMsgFormat                = `["{\"msg\":\"sub\",\"id\":\"%s\",\"name\":\"ips\",\"params\":[{\"taskId\":{\"$type\":\"oid\",\"$value\":\"%s\"},\"searchParam\":null},100]}"]`

    httpRequestsMsgFormat       = `["{\"msg\":\"sub\",\"id\":\"%s\",\"name\":\"reqs\",\"params\":[{\"taskId\":{\"$type\":\"oid\",\"$value\":\"%s\"},\"searchParam\":null},100]}"]`

	threatsMsgFormat 			= `["{\"msg\":\"sub\",\"id\":\"%s\",\"name\":\"threats\",\"params\":[{\"taskId\":{\"$type\":\"oid\",\"$value\":\"%s\"},\"uuid\":\"%s\",\"searchParam\":null}]}"]`

	doneMsgFormat               = `{"msg":"ready","subs":["%s"]}`
	pingMsg                     = `{"msg":"ping"}`
	pongMsg                     = `["{\"msg\":\"pong\"}"]`

	LettersDigits = "abcdefghijklmnopqrstuvwxyzABCDEFGHIJKLMNOPQRSTUVWXYZ0123456789"
)

type (
	AppAnyClient struct {
		conn      *websocket.Conn
		appConfig *AppConfig
	}
	AppAnyClientConfig struct {
		Endpoint  string
		ReqHeader http.Header
		AppConfig *AppConfig
	}
	TaskParams struct {
		IsPublic    bool     `json:"isPublic"`
		Hash        string   `json:"hash"`
		Runtype     []string `json:"runtype"`
		Verdict     []int    `json:"verdict"`
		Ext         []string `json:"ext"`
		IP          string   `json:"ip"`
		Domain      string   `json:"domain"`
		FileHash    string   `json:"fileHash"`
		MitreID     string   `json:"mitreId"`
		Sid         int      `json:"sid"`
		Significant bool     `json:"significant"`
		Tag         string   `json:"tag"`
		Skip        int      `json:"skip"`
	}
)

func randomEndpoint() string {
	return "wss://app.any.run/sockjs/" + strconv.Itoa(rand.Intn(999 - 100) + 100) + "/" + generateRandStr(len("i73_d8dy"), "abcdefghijklmnopqrstuvwxyz0123456789") + "/websocket"
}

func init() {
	rand.Seed(time.Now().UnixNano())
}

func NewAppAnyClient(config *AppAnyClientConfig) (*AppAnyClient, error) {
	conn, _, err := websocket.DefaultDialer.Dial(randomEndpoint(), config.ReqHeader)
	if err != nil {
		return nil, fmt.Errorf("failed to create a new socket client connection: %s", err)
	}
	// RECEIVED: o
	// RECEIVED: a["{\"server_id\":\"0\"}"]
	conn.ReadMessage()
	conn.ReadMessage()
	return &AppAnyClient{
		conn:      conn,
		appConfig: config.AppConfig,
	}, nil
}

func (client *AppAnyClient) getPublicTasksCounterMsg(id string) string {
	return fmt.Sprintf(publicTasksCounterMsgFormat, client.appConfig.ToTaskParamsJsonQuoted(), id)
}

func (client *AppAnyClient) getPublicTasksMsg(id string, taskCount, startIndex int) string {
	return fmt.Sprintf(publicTasksMsgFormat, id, taskCount, startIndex, client.appConfig.ToTaskParamsJsonQuoted())
}

func (client *AppAnyClient) getProcessesMsg(id string, taskId string) string {
	return fmt.Sprintf(processesMsgFormat, id, taskId)
}

func (client *AppAnyClient) getAllIncidentsMsg(id string, taskId string) string {
	return fmt.Sprintf(allIncidentsMsgFormat, id, taskId)
}

func (client *AppAnyClient)  getDNSQueriesMsg(id string, taskId string) string {
	return fmt.Sprintf(dnsMsgFormat, id ,taskId)
}

func (client *AppAnyClient)  getIpsMsg(id string, taskId string) string {
	return fmt.Sprintf(ipsMsgFormat, id, taskId)
}

func (client *AppAnyClient) getAllHttpRequestsMsg(id string, taskId string) string {
	return fmt.Sprintf(httpRequestsMsgFormat, id, taskId)
}

//---------------Threats----------//
func (client *AppAnyClient) getThreatsMsg(id string, taskId string, uuid string) string {
	return fmt.Sprintf(threatsMsgFormat, id, taskId, uuid)
}


func (client *AppAnyClient) getDoneMsg(id string) string {
	return fmt.Sprintf(doneMsgFormat, id)
}

func (client *AppAnyClient) recvMessageAndAssert(expectedMsg string) (bool, error) {
	msg, err := client.recvMessage()
	if err != nil {
		return false, err
	}
	return msg == expectedMsg, nil
}

func generateRandStr(n int, letters string) string {
	randStr := make([]byte, n, n)
	for i := 0; i < n; i++ {
		randStr[i] = letters[rand.Intn(len(letters))]
	}
	return string(randStr)
}

func (client *AppAnyClient) recvMessage() (string, error) {
	_, buffer, err := client.conn.ReadMessage()
	if err != nil {
		return "", fmt.Errorf("in ReadMessage: %s", err)
	}
	if len(buffer) > 2 && buffer[0] == 'a' && buffer[1] == '[' { // message format: a[escaped_json]
		msg, err := strconv.Unquote(string(buffer[2 : len(buffer)-1]))
		if err != nil {
			return "", fmt.Errorf("in strconv.Unquote: %s", err)
		}
		return msg, nil
	}
	if string(buffer) == pingMsg {
		if err := client.sendMessage(pongMsg); err != nil {
			return "", fmt.Errorf("failed to send pong msg: %s", err)
		}
		return client.recvMessage()
	}
	return string(buffer), nil
}

func (client *AppAnyClient) sendMessage(msg string) error {
	if err := client.conn.WriteMessage(websocket.TextMessage, []byte(msg)); err != nil {
		return fmt.Errorf("in WriteMessage: %s", err)
	}
	return nil
}

func (client *AppAnyClient) Connect() error {
	if err := client.sendMessage(connectMsg); err != nil {
		return fmt.Errorf("in sendMessage: %s", err)
	}
	msg, err := client.recvMessage()
	if err != nil {
		return fmt.Errorf("in recvMessage: %s", err)
	}
	if !strings.Contains(msg, "connected") {
		return fmt.Errorf("unexpected received msg: '%s'", msg)
	}
	return nil
}

func (client *AppAnyClient) GetNumOfTasks() (uint, error) {
	id := strconv.FormatInt(rand.Int63n(64), 10)
	msg := client.getPublicTasksCounterMsg(id)
	if err := client.sendMessage(msg); err != nil {
		return 0, fmt.Errorf("in sendMessage: %s", err)
	}
	// RECEIVED: a["{\"msg\":\"updated\",\"methods\":[\"5\"]}"]
	if _, err := client.recvMessage(); err != nil {
		return 0, fmt.Errorf("in recvMessage: %s", err)
	}
	// RECEIVED: a["{\"msg\":\"result\",\"id\":\"5\",\"result\":{\"count\":2089989}}"]
	buffer, err := client.recvMessage()
	if err != nil {
		return 0, fmt.Errorf("in recvMessage: %s", err)
	}
	result := new(PublicTasksCounterResult)
	if err := json.Unmarshal([]byte(buffer), &result); err != nil {
		return 0, fmt.Errorf("in Unmarshal: %s", err)
	}
	if id != result.ID {
		return 0, fmt.Errorf("corrupted recieved data: mismatched id")
	}
	return result.Result.Count, nil
}

// GetProcesses returns a list of task information as "public tasks" tab
func (client *AppAnyClient) GetTasks(numOfTasks, startIndex int) ([]*RawTask, error) {
	tasks := make([]*RawTask, 0)
	for numOfTasks > 0 {
		id := generateRandStr(len("DrDA7Qycqa8w9aLF9"), LettersDigits)
		var taskCount int
		if numOfTasks >= 50 {
			taskCount = 50
		} else {
			taskCount = numOfTasks
		}
		msg := client.getPublicTasksMsg(id, taskCount, startIndex)
		doneMsg := client.getDoneMsg(id)

		if err := client.sendMessage(msg); err != nil {
			return nil, fmt.Errorf("in sendMessage: %s", err)
		}
		for { // receive tasks
			var task *RawTask
			buffer, err := client.recvMessage()
			if err != nil {
				return nil, fmt.Errorf("in recvMessage: %s", err)
			}
			if buffer == doneMsg {
				break
			}
			if err := json.Unmarshal([]byte(buffer), &task); err != nil {
				return nil, fmt.Errorf("in Unmarshal: %s", err)
			}
			tasks = append(tasks, task)
		}
		numOfTasks -= taskCount
		startIndex += taskCount
	}
	return tasks, nil
}

// GetProcesses returns a list of processes as "processes" tab
func (client *AppAnyClient) GetProcesses(task *RawTask) ([]*Process, error) {
	processes := make([]*Process, 0)
	id := generateRandStr(len("E8ZWdmyNwRD3XBvcc"), LettersDigits)
	msg := client.getProcessesMsg(id, task.ID)
	doneMsg := client.getDoneMsg(id)

	if err := client.sendMessage(msg); err != nil {
		return nil, fmt.Errorf("in sendMessage: %s", err)
	}
	for { // receive processes
		var process *RawProcess
		buffer, err := client.recvMessage()
		if err != nil {
			return nil, fmt.Errorf("in recvMessage: %s", err)
		}
		if buffer == doneMsg {
			break
		}
		if err := json.Unmarshal([]byte(buffer), &process); err != nil {
			return nil, fmt.Errorf("in Unmarshal: %s", err)
		}
		processes = append(processes, NewProcess(process))
	}
	return processes, nil
}

// GetIncidents returns a list of MITRE ATT&CK  as "ATT&CK" tab
func (client *AppAnyClient) GetIncidents(task *RawTask) ([]*Incident, error) {
	incidents := make([]*Incident, 0)
	id := generateRandStr(len("4aYatF54JSoCNG94C"), LettersDigits)
	msg := client.getAllIncidentsMsg(id, task.ID)
	doneMsg := client.getDoneMsg(id)

	if err := client.sendMessage(msg); err != nil {
		return nil, fmt.Errorf("in sendMessage: %s", err)
	}
	for { // receive incidents
		var incident *RawIncident
		buffer, err := client.recvMessage()
		if err != nil {
			return nil, fmt.Errorf("in recvMessage: %s", err)
		}
		if buffer == doneMsg {
			break
		}
		if err := json.Unmarshal([]byte(buffer), &incident); err != nil {
			return nil, fmt.Errorf("in Unmarshal: %s", err)
		}
		incidents = append(incidents, NewIncident(incident))
	}
	return incidents, nil
}

// GetDNSQueries returns a list of DSN queries as "DNS Queries" tab
func (client *AppAnyClient) GetDNSQueries(task *RawTask) ([]*DNSQueries, error) {
	dnsQueries := make([]*DNSQueries, 0)
	id := generateRandStr(len("4aYatF54JSoCNG94C"), LettersDigits)
	msg := client.getDNSQueriesMsg(id, task.ID)
	doneMsg := client.getDoneMsg(id)

	if err := client.sendMessage(msg); err != nil {
		return nil, fmt.Errorf("in sendMessage: %s", err)
	}
	for { // receive dns
		var dns *RawDNSQueries
		buffer, err := client.recvMessage()
		if err != nil {
			return nil, fmt.Errorf("in recvMessage: %s", err)
		}
		if buffer == doneMsg {
			break
		}
		if err := json.Unmarshal([]byte(buffer), &dns); err != nil {
			return nil, fmt.Errorf("in Unmarshal: %s", err)
		}
		dnsQueries = append(dnsQueries, NewDNSQueries(dns))
	}
	return dnsQueries, nil
}

// GetIps returns a list of ips connections as "ips" tab
func (client *AppAnyClient) GetIps(task *RawTask) ([]*Ips, error) {
	ipsQuer := make([]*Ips, 0)
	id := generateRandStr(len("4aYatF54JSoCNG94C"), LettersDigits)
	msg := client.getIpsMsg(id, task.ID)
	doneMsg := client.getDoneMsg(id)

	if err := client.sendMessage(msg); err != nil {
		return nil, fmt.Errorf("in sendMessage: %s", err)
	}
	for { // receive dns
		var ips *RawIps
		buffer, err := client.recvMessage()
		if err != nil {
			return nil, fmt.Errorf("in recvMessage: %s", err)
		}
		if buffer == doneMsg {
			break
		}
		if err := json.Unmarshal([]byte(buffer), &ips); err != nil {
			return nil, fmt.Errorf("in Unmarshal: %s", err)
		}
		ipsQuer = append(ipsQuer, NewIps(ips))
	}
	return ipsQuer, nil
}

// GetHttpRequests returns a list of HTTP requests as "HTTP Requests" tab
func (client *AppAnyClient) GetHttpRequests(task *RawTask) ([]*HttpRequests, error) {
	httpRequests := make([]*HttpRequests, 0)
	id := generateRandStr(len("6ehw2pycH63vBTmKe"), LettersDigits)
	msg := client.getAllHttpRequestsMsg(id, task.ID)
	doneMsg := client.getDoneMsg(id)

	if err := client.sendMessage(msg); err != nil {
		return nil, fmt.Errorf("in sendMessage: %s", err)
	}
	for { // receive http requests
		var http *RawHttpRequests
		buffer, err := client.recvMessage()
		if err != nil {
			return nil, fmt.Errorf("in recvMessage: %s", err)
		}
		if buffer == doneMsg {
			break
		}
		if err := json.Unmarshal([]byte(buffer), &http); err != nil {
			return nil, fmt.Errorf("in Unmarshal: %s", err)
		}
		httpRequests = append(httpRequests, NewHttpRequests(http))
	}
	return httpRequests, nil
}

// GetGetThreats returns a list of Threats as "Threats" tab
func (client *AppAnyClient) GetThreats(task *RawTask) ([]*Threats, error) {
	threats := make([]*Threats, 0)
	id := generateRandStr(len("4aYatF54JSoCNG94C"), LettersDigits)
	msg := client.getThreatsMsg(id, task.ID, task.Fields.UUID)
	doneMsg := client.getDoneMsg(id)

	if err := client.sendMessage(msg); err != nil {
		return nil, fmt.Errorf("in sendMessage: %s", err)
	}
	for { // receive threats
		var threat *RawThreats
		buffer, err := client.recvMessage()
		if err != nil {
			return nil, fmt.Errorf("in recvMessage: %s", err)
		}
		if buffer == doneMsg {
			break
		}
		if err := json.Unmarshal([]byte(buffer), &threat); err != nil {
			return nil, fmt.Errorf("in Unmarshal: %s", err)
		}
		threats = append(threats, NewThreats(threat))
	}
	return threats, nil
}
