package main

import (
	"encoding/base64"
	"encoding/csv"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"io/ioutil"
	"math"
	"math/rand"
	"net/http"
	"net/url"
	"os"
	"os/exec"
	"path"
	"path/filepath"
	"regexp"
	"strings"
	"sync"
	"time"
)

type BomRow struct {
	OGPath     string
	DolbyIn    string
	DolbyOut   string
	DolbyJobId string
	FinalPath  string
	Status     string
	Error      string
}

type DolbyAuthResponse struct {
	AccessToken string `json:"access_token"`
}

type DolbyPresignedUrlResponse struct {
	Url string `json:"url"`
}

type DolbyEnhanceContent struct {
	Type string `json:"type"`
}

type DolbyEnhanceRequestBody struct {
	Input   string              `json:"input"`
	Output  string              `json:"output"`
	Content DolbyEnhanceContent `json:"content"`
}

type DoblyEnhanceResponse struct {
	JobId string `json:"job_id"`
}

type DolbyJobPollResponse struct {
	Path     string `json:"path"`
	Status   string `json:"status"`
	Progress int    `json:"progress"`
}

const (
	// General Config
	BATCH_SIZE       = 10
	BOM_DIR          = "/Users/mattalui/Music/BOM"
	OUT_DIR          = "out"
	DB_FILENAME      = "status.csv"
	POLL_TIMEOUT_MIN = 15
	// Statuses
	STATUS_NEW      = "new"
	STATUS_PENDING  = "pending"
	STATUS_FAILURE  = "failure"
	STATUS_COMPLETE = "complete"
	// Row Indices
	ROW_SIZE              = 7
	OG_PATH_INDEX         = 0
	DOLBY_IN_INDEX        = 1
	DOLBY_OUT_INDEX       = 2
	DOLBY_JOB_INDEX_INDEX = 3
	FINAL_PATH_INDEX      = 4
	STATUS_INDEX          = 5
	ERROR_INDEX           = 6
)

var (
	DB_CREATED         bool
	DB_WRITER          *csv.Writer
	DB_READER          *csv.Reader
	ROW_REGISTRY       map[string]*BomRow
	ENV                map[string]string
	DOLBY_BEARER_TOKEN string
)

func init() {
	fmt.Println("Initializing the cleaner")
	initializeEnv()
	initializeFileDependencies()
	initializeDB()
	initializeDolbyAuth()
}

func main() {
	fmt.Println(fmt.Sprintf("Cleaning the next %d audio files", BATCH_SIZE))
	// filesToProcess := getNextFilesToProcess()
	filesToProcess := getMissedFiles()
	ensureProcessingValidity(filesToProcess)
	rowObjects := initRowObjects(filesToProcess)
	var wg sync.WaitGroup
	wg.Add(len(rowObjects))
	for i := 0; i < len(rowObjects); i++ {
		go processAudioFile(rowObjects, i, &wg)
	}
	wg.Wait()
	writeProcessingUpdates(rowObjects)
}

func initializeEnv() {
	ENV = make(map[string]string)
	// There are packages that will load an env from an env file, but I don't
	// want any dependencies. This is easy enough for what I want to do.
	file, err := os.Open("./.env")
	if err != nil {
		fmt.Println("no env file found")
		return
	}
	defer file.Close()
	bytes, err := ioutil.ReadAll(file)
	if err != nil {
		return
	}
	content := string(bytes)
	lines := strings.Split(content, "\n")
	if len(lines) == 0 {
		return
	}

	for _, line := range strings.Split(content, "\n") {
		data := strings.SplitN(line, "=", 2)
		if len(data) == 2 {
			ENV[data[0]] = data[1]
		}
	}
}

func initializeFileDependencies() {
	dir_path := path.Join(BOM_DIR, OUT_DIR)
	db_path := path.Join(dir_path, DB_FILENAME)
	// make sure there is an output directory
	os.Mkdir(dir_path, os.ModePerm)
	// make sure that there is a CSV DB
	_, err := os.Stat(db_path)
	if errors.Is(err, os.ErrNotExist) {
		DB_CREATED = true
		os.Create(db_path)
	}
}

func initializeDB() {
	db_path := path.Join(BOM_DIR, OUT_DIR, DB_FILENAME)
	file, err := os.OpenFile(db_path, os.O_WRONLY|os.O_CREATE|os.O_APPEND, 0644)
	if err != nil {
		panic("Trying to write to a DB that has not been initialized")
	}

	DB_WRITER = csv.NewWriter(file)

	if DB_CREATED {
		defer DB_WRITER.Flush()
		header := []string{
			"Original Path",
			"Dolby In File",
			"Dolby Out File",
			"Dolby Enhance Job Id",
			"Final Path",
			"Status",
			"Error",
		}
		DB_WRITER.Write(header)
	}
	// Add reader here
}

func initializeDolbyAuth() {
	if ENV["DOLBY_APP_SECRET"] == "" || ENV["DOLBY_APP_KEY"] == "" {
		panic("Missing required Dolby API secrets")
	}
	file, err := os.Open("./.bearertoken")
	if errors.Is(err, os.ErrNotExist) {
		rawAuth := fmt.Sprintf("%s:%s", ENV["DOLBY_APP_KEY"], ENV["DOLBY_APP_SECRET"])
		auth := base64.StdEncoding.EncodeToString([]byte(rawAuth))
		body := url.Values{
			"grant_type": {"client_credentials"},
			"expires_in": {"1800"},
		}
		request, err := http.NewRequest("POST", "https://api.dolby.io/v1/auth/token", strings.NewReader(body.Encode()))
		if err != nil {
			panic(err)
		}
		request.Header.Add("Accept", "application/json")
		request.Header.Add("Content-Type", "application/x-www-form-urlencoded")
		request.Header.Add("Authorization", fmt.Sprintf("Basic %s", auth))
		client := &http.Client{}
		response, err := client.Do(request)
		if err != nil {
			panic(err)
		}
		bytes, err := ioutil.ReadAll(response.Body)
		if err != nil {
			panic(err)
		}
		data := DolbyAuthResponse{}
		json.Unmarshal(bytes, &data)
		DOLBY_BEARER_TOKEN = data.AccessToken
		// Write it to the file for later
		// ioutil.WriteFile("./.bearertoken", []byte(DOLBY_BEARER_TOKEN), 777)
		return
	} else if err != nil {
		panic(err)
	}
	defer file.Close()
	bytes, _ := ioutil.ReadAll(file)
	DOLBY_BEARER_TOKEN = string(bytes)
}

func getNextFilesToProcess() []string {
	// We should see if any files need to be reprocessed here

	originalFiles, _ := filepath.Glob(path.Join(BOM_DIR, "*.mp3"))
	outFiles, _ := filepath.Glob(path.Join(BOM_DIR, OUT_DIR, "*.mp3"))
	outCount := len(outFiles)
	nextIndex := int(math.Min(float64(outCount+BATCH_SIZE), float64(len(originalFiles)-1)))

	return originalFiles[outCount:nextIndex]
}

func getMissedFiles() []string {
	missed := make([]string, 0)

	inFiles, _ := filepath.Glob(path.Join(BOM_DIR, "*.mp3"))
	for _, in := range inFiles {
		expectedOut := path.Join(BOM_DIR, OUT_DIR, sanitizeBaseName(in)) + ".mp3"
		_, err := os.Stat(expectedOut)
		if errors.Is(err, os.ErrNotExist) {
			missed = append(missed, in)
		}
	}

	return missed
}

func ensureProcessingValidity(filesToProcess []string) {
	// Panic if any files in the list are already marked as complete
}

func initRowObjects(filesToProcess []string) []*BomRow {
	rowObjects := make([]*BomRow, 0)

	for _, fileName := range filesToProcess {
		rowObj := BomRow{
			OGPath: fileName,
			Status: STATUS_NEW,
		}
		// See if any of the files are already in the db, if so use thaty data

		// else add empty bom row
		rowObjects = append(rowObjects, &rowObj)
	}

	return rowObjects
}

func convertObjectToRow(rowObj *BomRow) []string {
	row := make([]string, ROW_SIZE)

	row[OG_PATH_INDEX] = rowObj.OGPath
	row[DOLBY_IN_INDEX] = rowObj.DolbyIn
	row[DOLBY_OUT_INDEX] = rowObj.DolbyOut
	row[DOLBY_JOB_INDEX_INDEX] = rowObj.DolbyJobId
	row[FINAL_PATH_INDEX] = rowObj.FinalPath
	row[STATUS_INDEX] = rowObj.Status
	row[ERROR_INDEX] = rowObj.Error

	return row
}

func processAudioFile(rows []*BomRow, index int, wg *sync.WaitGroup) {
	defer wg.Done()
	record := rows[index]
	createDolbyInputFile(record)
	makeDolbyEnhancementRequest(record)
	pollforDolbyJobCompletion(record)
	downloadEnhancedFile(record)
}

func writeProcessingUpdates(rows []*BomRow) {
	failures := 0
	defer DB_WRITER.Flush()
	for _, row := range rows {
		if row.Status != STATUS_COMPLETE {
			failures++
		}
		rowData := convertObjectToRow(row)
		fmt.Println("WRITING:", rowData)
		DB_WRITER.Write(rowData)
	}
	fmt.Println()
	fmt.Println(fmt.Sprintf("Non-completions: %d/%d", failures, BATCH_SIZE))
}

func createDolbyInputFile(record *BomRow) {
	fname := sanitizeBaseName(record.OGPath)
	client := &http.Client{}
	// Send the first request to register a private media input url
	record.DolbyIn = fmt.Sprintf("dlb://in/%s.mp3", fname) // can be whatever we want
	fmt.Println(fmt.Sprintf("CREATE: input file %s", record.DolbyIn))
	rawBody := map[string]string{"url": record.DolbyIn}
	bodyBytes, _ := json.Marshal(rawBody)
	request, err := http.NewRequest("POST", "https://api.dolby.com/media/input", strings.NewReader(string(bodyBytes)))
	if err != nil {
		record.Status = STATUS_FAILURE
		record.Error = "Unable to build media input request"
		return
	}
	request.Header.Add("Accept", "application/json")
	request.Header.Add("Content-Type", "application/json")
	request.Header.Add("Authorization", fmt.Sprintf("Bearer %s", DOLBY_BEARER_TOKEN))
	response, err := client.Do(request)
	if err != nil {
		record.Status = STATUS_FAILURE
		record.Error = "Unknown media input response error"
		return
	}
	if response.StatusCode != 200 {
		record.Status = STATUS_FAILURE
		record.Error = "Non-200 response for media input creation request"
		return
	}

	// Get the presigned upload endpoint from the response
	bytes, err := ioutil.ReadAll(response.Body)
	if err != nil {
		record.Status = STATUS_FAILURE
		record.Error = "Unable to read presigned response body"
		return
	}
	presignedResponse := DolbyPresignedUrlResponse{}
	json.Unmarshal(bytes, &presignedResponse)

	// Upload the actual file to the presigned endpoint

	// For some reason I'm getting different results from a direct request and
	// using CURL even though headers are the same. Instead of debugging I'm
	// just doing this for now
	cmd := exec.Command("curl", "-X", "PUT", presignedResponse.Url, "-T", record.OGPath)
	err = cmd.Run()
	if err != nil {
		record.Status = STATUS_FAILURE
		record.Error = "Error executing PUT curl"
		return
	}

	// The real way fo doing things

	// stats, err := os.Stat(record.OGPath)
	// if err != nil {
	// 	record.Status = STATUS_FAILURE
	// 	return
	// }
	// uploadFile, err := os.Open(record.OGPath)
	// fileUploadRequest, err := http.NewRequest("PUT", presignedResponse.Url, uploadFile)
	// if err != nil {
	// 	record.Status = STATUS_FAILURE
	// 	return
	// }
	// fileUploadRequest.Header.Add("Content-Type", "application/octet-stream")
	// fileUploadRequest.Header.Add("Content-Length", strconv.Itoa(int(stats.Size())))
	// fileUploadRequest.Header.Add("Accept", "*/*")
	// uploadResponse, err := client.Do(fileUploadRequest)
	// if err != nil {
	// 	record.Status = STATUS_FAILURE
	// 	return
	// }
	// b, _ := ioutil.ReadAll(uploadResponse.Body)
}

func makeDolbyEnhancementRequest(record *BomRow) {
	if record.Status == STATUS_FAILURE {
		return
	}
	fname := sanitizeBaseName(record.OGPath)
	client := &http.Client{}

	record.DolbyOut = fmt.Sprintf("dlb://out/%s.mp3", fname) // can be whatever we want
	fmt.Println(fmt.Sprintf("ENHANCE: output file %s", record.DolbyOut))
	rawBody := DolbyEnhanceRequestBody{}
	rawBody.Input = record.DolbyIn
	rawBody.Output = record.DolbyOut
	rawBody.Content.Type = "studio"
	bodyBytes, err := json.Marshal(rawBody)
	if err != nil {
		record.Status = STATUS_FAILURE
		record.Error = "Error converting enhance request body to bytes"
		return
	}
	request, err := http.NewRequest("POST", "https://api.dolby.com/media/enhance", strings.NewReader(string(bodyBytes)))
	if err != nil {
		record.Status = STATUS_FAILURE
		record.Error = "Error creating enhance request"
		return
	}
	request.Header.Add("Accept", "application/json")
	request.Header.Add("Content-Type", "application/json")
	request.Header.Add("Authorization", fmt.Sprintf("Bearer %s", DOLBY_BEARER_TOKEN))
	response, err := client.Do(request)
	if err != nil {
		record.Status = STATUS_FAILURE
		record.Error = "Unknown error in enhance request"
		return
	}
	if response.StatusCode != 200 {
		record.Status = STATUS_FAILURE
		record.Error = "Non-200 response code on enhancement request"
		return
	}
	responseBytes, err := ioutil.ReadAll(response.Body)
	if err != nil {
		record.Status = STATUS_FAILURE
		record.Error = "Error reading bytes from enhancement response body"
		return
	}
	dolbyResponse := DoblyEnhanceResponse{}
	json.Unmarshal(responseBytes, &dolbyResponse)
	if dolbyResponse.JobId == "" {
		record.Status = STATUS_FAILURE
		record.Error = "Empty job Id in enhance response"
		return
	}
	record.DolbyJobId = dolbyResponse.JobId
	record.Status = STATUS_PENDING
}

func pollforDolbyJobCompletion(record *BomRow) {
	if record.Status != STATUS_PENDING {
		record.Status = STATUS_FAILURE
		if record.Error == "" {
			record.Error = "Trying to poll for Dolby enhancement job without a pending status"
		}

	}
	if record.Status == STATUS_FAILURE {
		return
	}
	client := &http.Client{}
	startTime := time.Now()
	for {
		if time.Since(startTime) >= POLL_TIMEOUT_MIN*time.Minute {
			record.Status = STATUS_FAILURE
			record.Error = "Timout while polling for dolby job completion"
			return
		}
		request, err := http.NewRequest(
			"GET",
			fmt.Sprintf("https://api.dolby.com/media/enhance?job_id=%s", record.DolbyJobId),
			nil,
		)
		if err != nil {
			record.Status = STATUS_FAILURE
			record.Error = "Unable to create polling request"
			return
		}
		request.Header.Add("Authorization", fmt.Sprintf("Bearer %s", DOLBY_BEARER_TOKEN))
		response, err := client.Do(request)
		if err != nil {
			record.Status = STATUS_FAILURE
			record.Error = "Unknown error sending polling request"
			return
		}
		if response.StatusCode != 200 {
			record.Status = STATUS_FAILURE
			record.Error = "Non-200 status for polling request"
			return
		}
		responseBytes, err := ioutil.ReadAll(response.Body)
		if err != nil {
			record.Status = STATUS_FAILURE
			record.Error = "Error reading response body of polling request"
			return
		}
		job := DolbyJobPollResponse{}
		json.Unmarshal(responseBytes, &job)
		fmt.Println(fmt.Sprintf("POLLING: %s %d", record.DolbyJobId, job.Progress))

		jobIsComplete := job.Progress >= 100
		if jobIsComplete {
			fmt.Println("FINISHED: Dolby enhance job has completed")
			return
		} else {
			sleepTime := rand.Intn(10) + 1
			time.Sleep(time.Duration(sleepTime) * time.Second)
		}
	}
}

func downloadEnhancedFile(record *BomRow) {
	if record.Status == STATUS_FAILURE {
		return
	}
	fmt.Println(fmt.Sprintf("DOWNLOAD: target file %s", record.DolbyOut))
	request, err := http.NewRequest(
		"GET",
		fmt.Sprintf("https://api.dolby.com/media/output?url=%s", record.DolbyOut),
		nil,
	)
	if err != nil {
		record.Status = STATUS_FAILURE
		record.Error = "Unable to create polling request"
		return
	}
	request.Header.Add("Authorization", fmt.Sprintf("Bearer %s", DOLBY_BEARER_TOKEN))
	client := &http.Client{}
	response, err := client.Do(request)
	if err != nil {
		record.Error = "Error sending file download request"
		record.Status = STATUS_FAILURE
		return
	}
	if response.StatusCode != 200 {
		fmt.Println(response.Status)
		record.Error = "non-200 response code for download request"
		record.Status = STATUS_FAILURE
		return
	}
	outFilePath := path.Join(BOM_DIR, OUT_DIR, sanitizeBaseName(record.OGPath)) + ".mp3"
	outputFile, err := os.Create(outFilePath)
	if err != nil {
		record.Error = "Error creating output file"
		record.Status = STATUS_FAILURE
		return
	}
	_, err = io.Copy(outputFile, response.Body)
	if err != nil {
		record.Error = "Error Error copying file to destination"
		record.Status = STATUS_FAILURE
		return
	}
	record.Status = STATUS_COMPLETE
	fmt.Println("SAVED: ", outFilePath)
}

func sanitizeBaseName(fpath string) string {
	re := regexp.MustCompile("[^\\w.-]|.mp3")
	return re.ReplaceAllString(path.Base(fpath), "")
}
