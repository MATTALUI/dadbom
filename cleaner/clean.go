package main

import (
	"encoding/base64"
	"encoding/csv"
	"encoding/json"
	"errors"
	"fmt"
	"io/ioutil"
	"net/http"
	"net/url"
	"os"
	"path"
	"path/filepath"
	"strings"
	"sync"
)

type BomRow struct {
	OGPath    string
	DolbyIn   string
	DolbyOut  string
	FinalPath string
	Status    string
}

type DolbyAuthResponse struct {
	AccessToken string `json:"access_token"`
}

const (
	// General Config
	BATCH_SIZE  = 10
	BOM_DIR     = "/Users/mattalui/Music/BOM"
	OUT_DIR     = "out"
	DB_FILENAME = "status.csv"
	// Statuses
	STATUS_NEW      = "new"
	STATUS_PENDING  = "pending"
	STATUS_FAILURE  = "failure"
	STATUS_COMPLETE = "complete"
	// Row Indices
	ROW_SIZE         = 5
	OG_PATH_INDEX    = 0
	DOLBY_IN_INDEX   = 1
	DOLBY_OUT_INDEX  = 2
	FINAL_PATH_INDEX = 3
	STATUS_INDEX     = 4
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
	filesToProcess := getNextFilesToProcess()
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
			"Final Path",
			"Status",
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
		ioutil.WriteFile("./.bearertoken", []byte(DOLBY_BEARER_TOKEN), 777)
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

	return originalFiles[outCount : outCount+BATCH_SIZE]
}

func ensureProcessingValidity(filesToProcess []string) {
	// Panic if any files in the list are already marked as complete
}

// func getDBReader() *csv.Reader {
// 	db_path := path.Join(BOM_DIR, OUT_DIR, DB_FILENAME)
// 	file, err := os.OpenFile(db_path, os.O_RDONLY, 777)
// 	if err != nil {
// 		panic("Trying to read a DB that has not been initialized")
// 	}

// 	return csv.NewReader(file)
// }

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
	row[FINAL_PATH_INDEX] = rowObj.FinalPath
	row[STATUS_INDEX] = rowObj.Status

	return row
}

func processAudioFile(rows []*BomRow, index int, wg *sync.WaitGroup) {
	defer wg.Done()
	// create dolby input file
	// make the enhancement request
	// poll for completion
	// download file
}

func writeProcessingUpdates(rows []*BomRow) {
	defer DB_WRITER.Flush()
	for _, row := range rows {
		DB_WRITER.Write(convertObjectToRow(row))
	}
}
