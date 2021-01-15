package multiform

import (
	"bytes"
	"io"
	"io/ioutil"
	"mime/multipart"
	"net/http"
	"net/http/httptest"
	"os"
	"strings"
	"testing"
)

type FilePart struct {
	name string
	path string
}

type EncodeTest struct {
	fields map[string]string
	files  map[string]FilePart
}

func TestEncode(t *testing.T) {
	tests := []EncodeTest{
		{
			map[string]string{"fileName": "test"},
			map[string]FilePart{},
		},
		{
			map[string]string{"two": "test2"},
			map[string]FilePart{"file": {"test-file.txt", "test_resources/test.txt"}},
		},
		{
			map[string]string{"two": "test3", "three": "test4"},
			map[string]FilePart{"file": {"test-file.jpeg", "test_resources/baby-yoda.jpeg"}},
		},
		{
			map[string]string{"two": "test3", "three": "test4"},
			map[string]FilePart{
				"file":  {"test-file.jpeg", "test_resources/baby-yoda.jpeg"},
				"file2": {"test-file2.txt", "test_resources/test.txt"},
			},
		},
	}
	boundary := "test-boundary"
	for _, test := range tests {

		builder := NewBuilder()
		builder.SetBoundary(boundary)

		var b bytes.Buffer
		form := multipart.NewWriter(&b)
		form.SetBoundary(boundary)

		for key, val := range test.fields {
			builder.AddField(key, val)
			fw, err := form.CreateFormField(key)
			if err != nil {
				return
			}
			io.Copy(fw, strings.NewReader(val))
		}
		for key, part := range test.files {
			file1, err := os.Open(part.path)
			if err != nil {
				t.Error(err)
			}
			builder.AddFormFile(key, part.name, file1)
			file2, err := os.Open(part.path)
			fw, err := form.CreateFormFile(key, part.name)
			if err != nil {
				t.Error(err)
			}
			_, err = io.Copy(fw, file2)
			if err != nil {
				t.Error(err)
			}
		}

		form.Close()
		builder.Done()
		result, err := ioutil.ReadAll(builder)

		if err != nil {
			t.Error(err)
			return
		}

		expected, err := ioutil.ReadAll(&b)
		if !bytes.Equal(expected, result) {
			t.Error("Output did not match expected")
			t.Errorf("Expected %s\r\n Got %s\r\n", expected, result)
		}
	}

}

func TestPost(t *testing.T) {
	testFile := "test_resources/test.txt"
	didRespond := false
	ts := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		form := r.FormValue("name")
		if form != "test name" {
			t.Error("Expected name to be in the form")
		}
		file, _, err := r.FormFile("file")
		if err != nil {
			t.Error(err)
		}
		expectedFile, err := ioutil.ReadFile(testFile)
		if err != nil {
			t.Error(err)
		}
		resultBytes, err := ioutil.ReadAll(file)
		if err != nil {
			t.Error(err)
		}
		if !bytes.Equal(resultBytes, expectedFile) {
			t.Error("sent and parsed files are not identical")
		}
		didRespond = true
		w.WriteHeader(200)
	}))
	defer ts.Close()
	file, err := os.Open(testFile)
	if err != nil {
		t.Error(err)
	}
	fb := NewBuilder()
	fb.AddField("name", "test name")
	fb.AddFormFile("file", "baby-yoda.jpeg", file)
	fb.Done()
	_, err = http.Post(ts.URL, fb.FormDataContentType(), fb)
	if err != nil {
		t.Error(err)
	}
	if !didRespond {
		t.Error("Expected a server response")
	}
}
