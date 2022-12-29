package toolkit

import (
	"bytes"
	"encoding/json"
	"errors"
	"fmt"
	"image"
	"image/png"
	"io"
	"mime/multipart"
	"net/http"
	"net/http/httptest"
	"os"
	"sync"
	"testing"
)

func TestTools_RandomString(t *testing.T) {
	var testTools Tools

	s := testTools.RandomString(10)
	if len(s) != 10 {
		t.Error("Wrong length of random string returned")
	}
}

var uploadTests = []struct {
	name          string
	allowedTypes  []string
	renameFile    bool
	errorExpected bool
}{
	{name: "Allowed no rename", allowedTypes: []string{"image/jpeg", "image/png"}, renameFile: false, errorExpected: false},
	{name: "Allowed rename", allowedTypes: []string{"image/jpeg", "image/png"}, renameFile: true, errorExpected: false},
	{name: "not allowed", allowedTypes: []string{"image/jpeg"}, renameFile: false, errorExpected: true},
	// {name: "Allowed no rename", allowedTypes: []string{"image/jpeg", "image/png"}, renameFile: false, errorExpected: false},
}

func TestTools_UploadFiles(t *testing.T) {
	for _, test := range uploadTests {
		// set up a pipe to avoid buffering
		pr, pw := io.Pipe()
		writer := multipart.NewWriter(pw)
		wg := sync.WaitGroup{}
		wg.Add(1)

		go func() {
			defer writer.Close()
			defer wg.Done()

			// create the form data filed 'file'
			part, err := writer.CreateFormFile("file", "./testdata/img.png")
			if err != nil {
				t.Error(err)
			}

			f, err := os.Open("./testdata/img.png")
			if err != nil {
				t.Error(err)
			}
			defer f.Close()

			img, _, err := image.Decode(f)
			if err != nil {
				t.Error("error decoding image", err)
			}

			err = png.Encode(part, img)
			if err != nil {
				t.Error(err)
			}
		}()

		// read from the pipe which recieves data
		request := httptest.NewRequest("POST", "/", pr)
		request.Header.Add("Content-Type", writer.FormDataContentType())

		var testTools Tools
		testTools.AllowedFileTypes = test.allowedTypes

		uploadedFiles, err := testTools.UploadFiles(request, "./testdata/uploads/", test.renameFile)
		if err != nil && !test.errorExpected {
			t.Error(err)
		}

		if !test.errorExpected {
			if _, err := os.Stat(fmt.Sprintf("./testdata/uploads/%s", uploadedFiles[0].NewFileName)); os.IsNotExist(err) {
				t.Errorf("%s - expected file to exist: %s", test.name, err.Error())
			}

			// clean up
			_ = os.Remove(fmt.Sprintf("./testdata/uploads/%s", uploadedFiles[0].NewFileName))
		}

		if !test.errorExpected && err != nil {
			t.Errorf("%s - error expected but none recieved", test.name)
		}

		wg.Wait()
	}
}

func TestTools_UploadOneFile(t *testing.T) {
	// set up a pipe to avoid buffering
	pr, pw := io.Pipe()
	writer := multipart.NewWriter(pw)

	go func() {
		defer writer.Close()

		// create the form data filed 'file'
		part, err := writer.CreateFormFile("file", "./testdata/img.png")
		if err != nil {
			t.Error(err)
		}

		f, err := os.Open("./testdata/img.png")
		if err != nil {
			t.Error(err)
		}
		defer f.Close()

		img, _, err := image.Decode(f)
		if err != nil {
			t.Error("error decoding image", err)
		}

		err = png.Encode(part, img)
		if err != nil {
			t.Error(err)
		}
	}()

	// read from the pipe which recieves data
	request := httptest.NewRequest("POST", "/", pr)
	request.Header.Add("Content-Type", writer.FormDataContentType())

	var testTools Tools
	// testTools.AllowedFileTypes = test.allowedTypes

	uploadedFiles, err := testTools.UploadOneFile(request, "./testdata/uploads/", true)
	if err != nil {
		t.Error(err)
	}

	if _, err := os.Stat(fmt.Sprintf("./testdata/uploads/%s", uploadedFiles.NewFileName)); os.IsNotExist(err) {
		t.Errorf("expected file to exist: %s", err.Error())
	}

	// clean up
	_ = os.Remove(fmt.Sprintf("./testdata/uploads/%s", uploadedFiles.NewFileName))
}

func TestTools_CreateDirIfNotExist(t *testing.T) {
	var testTool Tools

	err := testTool.CreateDirIfNotExist("./testdata/mydir")
	if err != nil {
		t.Error(err)
	}

	err = testTool.CreateDirIfNotExist("./testdata/mydir")
	if err != nil {
		t.Error(err)
	}

	_ = os.Remove("./testdata/mydir")
}

var slugTests = []struct {
	name           string
	s              string
	expectedResult string
	errorExpected  bool
}{
	{name: "valid string", s: "Now is the Time!", expectedResult: "now-is-the-time", errorExpected: false},
	{name: "empty string", s: "", expectedResult: "", errorExpected: true},
	{name: "complex string", s: "Now is the time for all GOOD men! + lol & lmao 2134$#@!%^", expectedResult: "now-is-the-time-for-all-good-men-lol-lmao-2134", errorExpected: false},
	{name: "japanese string", s: "今こそすべての善良な男性のための時です", expectedResult: "", errorExpected: true},
	{name: "japanese string and roman characters", s: "Hello world, 今こそすべての善良な男性のための時です", expectedResult: "hello-world", errorExpected: false},
}

func TestTools_Slugify(t *testing.T) {
	var testTool Tools

	for _, test := range slugTests {
		slug, err := testTool.Slugify(test.s)
		if err != nil && !test.errorExpected {
			t.Errorf("%s - error recieved when none expected: %s", test.name, err.Error())
		}

		if !test.errorExpected && slug != test.expectedResult {
			t.Errorf("%s - wrong slug returned. expected %s but got %s", test.name, test.expectedResult, slug)
		}
	}

}

func TestTools_DownloadStaticFile(t *testing.T) {
	rr := httptest.NewRecorder()
	req, _ := http.NewRequest("GET", "/", nil)

	var testTool Tools

	testTool.DownloadStaticFile(rr, req, "./testdata/pic.jpg", "puppy.jpg")

	res := rr.Result()
	defer res.Body.Close()

	if res.Header["Content-Length"][0] != "98827" {
		t.Errorf("found wrong content length of %s; expecting 98827", res.Header["Content-Length"][0])
	}

	if res.Header["Content-Disposition"][0] != "attachment; filename=\"puppy.jpg\"" {
		t.Error("wrong content disposition")
	}

	_, err := io.ReadAll(res.Body)
	if err != nil {
		t.Error(err)
	}
}

var jsonTests = []struct {
	name               string
	json               string
	maxSize            int
	allowUnknownFields bool
	errorExpected      bool
}{
	{name: "good json", json: `{"foo": "bar"}`, maxSize: 1024, allowUnknownFields: false, errorExpected: false},
	{name: "badly formed json", json: `{"foo": }`, maxSize: 1024, allowUnknownFields: false, errorExpected: true},
	{name: "incorrect type", json: `{"foo": 1}`, maxSize: 1024, allowUnknownFields: false, errorExpected: true},
	{name: "two json files", json: `{"foo": "1"}{"foo": "bar"}`, maxSize: 1024, allowUnknownFields: false, errorExpected: true},
	{name: "empty body", json: ``, maxSize: 1024, allowUnknownFields: false, errorExpected: true},
	{name: "syntax error in json", json: `{"foo": 1"`, maxSize: 1024, allowUnknownFields: false, errorExpected: true},
	{name: "unknown field in json", json: `{"alpha": "bar"}`, maxSize: 1024, allowUnknownFields: false, errorExpected: true},
	{name: "good json", json: `{"alpha": "bar"}`, maxSize: 1024, allowUnknownFields: true, errorExpected: false},
	{name: "missing field name", json: `{alpha: "bar"}`, maxSize: 1024, allowUnknownFields: true, errorExpected: true},
	{name: "file too large", json: `{"foo": "bar"}`, maxSize: 5, allowUnknownFields: false, errorExpected: true},
	{name: "not json", json: `hello world`, maxSize: 1024, allowUnknownFields: false, errorExpected: true},
}

func TestTools_ReadJSON(t *testing.T) {
	var testTool Tools

	for _, test := range jsonTests {
		// set the max file size
		testTool.MaxJSONSize = test.maxSize

		// allow/disallow unknown fields
		testTool.AllowUnknownFields = test.allowUnknownFields

		var decodedJSON struct {
			Foo string `json:"foo"`
		}

		// create a request with the body
		req, err := http.NewRequest("POST", "/", bytes.NewReader([]byte(test.json)))
		if err != nil {
			t.Error(err)
		}

		// create a recorder
		rr := httptest.NewRecorder()

		err = testTool.ReadJSON(rr, req, &decodedJSON)

		if test.errorExpected && err == nil {
			t.Errorf("%s - error expected but none recieved", test.name)
		}

		if !test.errorExpected && err != nil {
			t.Errorf("%s - error not expected, but recieved: %s", test.name, err.Error())
		}

		req.Body.Close()
	}
}

func TestTools_WriteJSON(t *testing.T) {
	var testTools Tools

	rr := httptest.NewRecorder()
	payload := JSONResponse{
		Error:   false,
		Message: "foo",
	}

	headers := make(http.Header)
	headers.Add("FOO", "BAR")

	err := testTools.WriteJSON(rr, http.StatusOK, payload, headers)
	if err != nil {
		t.Errorf("failed to write json: %s", err)
	}
}

func TestTools_ErrorJSON(t *testing.T) {
	var testTools Tools

	rr := httptest.NewRecorder()
	err := testTools.ErrorJSON(rr, errors.New("some error"), http.StatusServiceUnavailable)
	if err != nil {
		t.Error(err)
	}

	var payload JSONResponse
	decoder := json.NewDecoder(rr.Body)
	err = decoder.Decode(&payload)
	if err != nil {
		t.Error("recieved error when decoding json", err)
	}

	if !payload.Error {
		t.Error("error set to false in JSON, should be true")
	}

	if rr.Code != http.StatusServiceUnavailable {
		t.Errorf("wrong status code returned; expected 503, but got %d", rr.Code)
	}
}

type RoundTripFunc func(req *http.Request) *http.Response

func (f RoundTripFunc) RoundTrip(req *http.Request) (*http.Response, error) {
	return f(req), nil
}

func NewTestClient(fn RoundTripFunc) *http.Client {
	return &http.Client{
		Transport: fn,
	}
}

func TestTools_PushJSONToRemote(t *testing.T) {
	client := NewTestClient(func(req *http.Request) *http.Response {
		// Test Request Parameters
		return &http.Response{
			StatusCode: http.StatusOK,
			Body:       io.NopCloser(bytes.NewBufferString("ok")),
			Header:     make(http.Header),
		}
	})

	var testTools Tools
	var foo struct {
		Bar string `json:"bar"`
	}
	foo.Bar = "bar"

	_, _, err := testTools.PushJSONToRemote("http://example.com/some/path", foo, client)
	if err != nil {
		t.Error("failed to call remote url:", err)
	}
}
