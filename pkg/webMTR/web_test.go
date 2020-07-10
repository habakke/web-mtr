package webMTR

import (
	"encoding/json"
	"github.com/habakke/web-mtr/pkg/hop"
	"net/http"
	"net/http/httptest"
	"os"
	"testing"
)

func isRoot() bool {
	return os.Getuid() == 0
}

func TestTraceHandlerHappyDay(t *testing.T) {
	address := "1.1.1.1"
	url := "/trace"

	if !isRoot() {
		t.Fatalf("This test has to run as root (current UID=%d)", os.Getuid())
	}

	req, err := http.NewRequest("GET", url, nil)
	if err != nil {
		t.Fatal(err)
	}
	q := req.URL.Query()
	q.Add("ip", address)
	req.URL.RawQuery = q.Encode()

	wm := NewWebMTR(":8080", "/opt/web", 3)
	defer wm.Close()

	rr := httptest.NewRecorder()
	handler := http.HandlerFunc(wm.traceHandler)
	handler.ServeHTTP(rr, req)
	if status := rr.Code; status != http.StatusOK {
		t.Fatalf("handler returned wrong status code: got %v want %v", status, http.StatusOK)
	}

	stats := map[int]*hop.HopStatistic{}
	err = json.Unmarshal(rr.Body.Bytes(), &stats)
	if err != nil {
		t.Fatalf("Failedto parse trace output: %v", err)
	}

	if len(stats) == 0 {
		t.Fatalf("Empty trace result returned")
	}
}
