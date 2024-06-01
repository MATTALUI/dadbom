package main

import (
	"encoding/csv"
	"errors"
	"fmt"
	"os"
	"path"
	"path/filepath"
	"sync"
)

type BomRow struct {
	OGPath    string
	DolbyIn   string
	DolbyOut  string
	FinalPath string
	Status    string
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
	DB_WRITER    csv.Writer
	DB_READER    csv.Reader
	ROW_REGISTRY map[string]*BomRow
)

func init() {
	initializeDependencies()
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

func initializeDependencies() {
	dir_path := path.Join(BOM_DIR, OUT_DIR)
	db_path := path.Join(dir_path, DB_FILENAME)
	// make sure there is an output directory
	os.Mkdir(dir_path, os.ModePerm)
	// make sure that there is a CSV DB
	_, err := os.Stat(db_path)
	if errors.Is(err, os.ErrNotExist) {
		os.Create(db_path)
		writer := getDBWriter()
		defer writer.Flush()
		header := []string{
			"Original Path",
			"Dolby In File",
			"Dolby Out File",
			"Final Path",
			"Status",
		}
		writer.Write(header)
	}
}

func initializeDolbyAuth() {

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

func getDBWriter() *csv.Writer {
	db_path := path.Join(BOM_DIR, OUT_DIR, DB_FILENAME)
	file, err := os.OpenFile(db_path, os.O_WRONLY|os.O_CREATE|os.O_APPEND, 0644)
	if err != nil {
		panic("Trying to write to a DB that has not been initialized")
	}

	return csv.NewWriter(file)
}

func getDBReader() *csv.Reader {
	db_path := path.Join(BOM_DIR, OUT_DIR, DB_FILENAME)
	file, err := os.OpenFile(db_path, os.O_RDONLY, 777)
	if err != nil {
		panic("Trying to read a DB that has not been initialized")
	}

	return csv.NewReader(file)
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

func convertObjectsToRow(rowObj *BomRow) []string {
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
	writer := getDBWriter()
	defer writer.Flush()

	for _, row := range rows {
		writer.Write(convertObjectsToRow(row))
	}
}
