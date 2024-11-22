package services

import (
	"Executor/constant"
	model "Executor/models"
	"bytes"
	"compress/gzip"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"strings"
	"time"
)

type ExecutorRepository interface {
	ExecutorRequest(ExecutorRQ string) string
}

type executorRepository struct {
	// blobStorage BlobHandler
}

func NewExecutorRequestRepository() ExecutorRepository {
	return &executorRepository{
		// blobStorage: NewBlobStorageHandler(),
	}
}

// ExecutorRequest sends a request to the API and returns the response as a string.
func (r *executorRepository) ExecutorRequest(ExecutorRQ string) string {
	// Declare response structure and variables to hold room information
	var RS = model.Resp{}
	var roomInfo []model.RoomInfo
	var data map[string]interface{}

	// Attempt to unmarshal ExecutorRQ JSON string into a map
	if err := json.Unmarshal([]byte(ExecutorRQ), &data); err != nil {
		// Log and handle unmarshalling error
		fmt.Println("Error unmarshaling:", err)
	}

	// Extract supplier_request from the parsed JSON data
	supplierRequestRaw := data["supplier_request"].(string)

	// Create HTTP client with a specified timeout constant
	client := &http.Client{
		Timeout: constant.Timeout,
	}

	// Store room_info from data into RS and attempt to unmarshal it into roomInfo slice
	RS.RoomInfo = data["room_info"].(string)
	if err := json.Unmarshal([]byte(RS.RoomInfo), &roomInfo); err != nil {
		// Log and handle unmarshalling error
		fmt.Println("Error unmarshaling:", err)
	}

	// Generate a new unique ID for the log file
	// newId := uuid.New().String()

	// Format the log file name using the newId and TrackerID from the roomInfo slice
	// logfile := fmt.Sprintf("request_%s_%s", newId, roomInfo[0].TrackerID)

	// Write the request data asynchronously to blob storage
	// This allows the function to continue without waiting for the write to complete
	// go r.blobStorage.WriteFile(roomInfo[0].TrackerID, constant.SearchMethod, logfile, supplierRequestRaw)

	// Create HTTP request message using the supplierRequestRaw data
	httpRequestMessage, err := CreateHttpRequestMessage(constant.Url, supplierRequestRaw)
	if err != nil {
		// Return error message if request creation fails
		return err.Error()
	}

	// Send the HTTP request and get the response
	resp, err := client.Do(httpRequestMessage)
	if err != nil {
		// Return error message if HTTP request fails
		return err.Error()
	}
	defer resp.Body.Close() // Ensure the response body is closed after function returns

	// Check if the response is compressed (gzip) and handle decompression if needed
	var reader io.ReadCloser
	switch resp.Header.Get("Content-Encoding") {
	case "gzip":
		// If gzip compression is used, create a gzip reader for the response body
		reader, err = gzip.NewReader(resp.Body)
		if err != nil {
			// Return error if gzip decompression fails
			return err.Error()
		}
		defer reader.Close()
	default:
		// If no compression, use the response body as is
		reader = resp.Body
	}

	// Check if the response status is OK (200)
	if resp.StatusCode != http.StatusOK {
		// Return error if the status code is not OK
		return err.Error()
	}

	// Read the response data from the body (decompressed if necessary)
	responseData, err := io.ReadAll(reader)
	if err != nil {
		// Return error if reading response fails
		return err.Error()
	}

	// Write the response data asynchronously to blob storage (log response)
	// go r.blobStorage.WriteFile(roomInfo[0].TrackerID, constant.SearchMethod, "response_"+newId+"_"+roomInfo[0].TrackerID, string(responseData))

	// Clean up the response string (remove unnecessary "&" characters)
	hotelresp := string(responseData)
	RS.SupplierResp = strings.Replace(hotelresp, "&", "", -1)

	// Marshal the final response structure and return it as a string
	executor_resp, _ := json.Marshal(RS)
	return string(executor_resp)
}

// CreateHttpRequestMessage creates and configures a new HTTP POST request.
func CreateHttpRequestMessage(url, request string) (*http.Request, error) {
	// Create a new POST request
	req, err := http.NewRequest(http.MethodPost, url, bytes.NewBuffer([]byte(request)))
	if err != nil {
		return nil, fmt.Errorf("error creating HTTP request: %w", err)
	}

	// Add headers to the request
	req.Header.Set("Content-Type", constant.MediaType)
	req.Header.Set("Accept", constant.MediaType)
	req.Header.Set("Accept-Encoding", constant.Gip)
	req.Header.Set(constant.APIKeyHeaderName, constant.UserName)
	req.Header.Set(constant.SignatureHeaderName, CreateSignature())

	return req, nil
}

// CreateSignature generates a signature using the API key, shared secret, and current timestamp.
func CreateSignature() string {
	// Combine API key, shared secret, and current timestamp
	ts := time.Now().Unix()
	hashString := fmt.Sprintf("%s%s%d", constant.UserName, constant.Password, ts)

	// Compute SHA256 hash
	hash := sha256.New()
	hash.Write([]byte(hashString))

	// Return the hash as a hexadecimal string
	return hex.EncodeToString(hash.Sum(nil))
}
