package patrol_test

import (
	"reflect"
	"sync"
	"testing"
	"time"

	"net/http"
	"net/http/httptest"
	"net/url"

	"github.com/Fresh-Tracks/bomb-squad/patrol"
	"github.com/Fresh-Tracks/bomb-squad/util"
)

func TestPatrol(t *testing.T) {
	client, err := util.HttpClient()
	Must(t, err)

	wg := sync.WaitGroup{}

	s := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(200)
		w.Write([]byte(`{"status":"success","data":{"resultType":"vector","result":[]}}`))

		AssertEquals(t, "/api/v1/query?query=topk%280%2Cdelta%28card_count%5B1m%5D%29%29", r.RequestURI)

		wg.Done()
	}))
	defer s.Close()

	promurl, err := url.Parse(s.URL)
	Must(t, err)

	p := patrol.Patrol{
		HTTPClient: client,
		PromURL:    promurl,
		Interval:   100 * time.Millisecond,
	}

	wg.Add(1)
	go func() {
		p.Run()
	}()
	wg.Wait()

}

func Must(t *testing.T, err error) {
	if err != nil {
		t.Fatalf("test failed with %s", err)
	}
}

func AssertEquals(t *testing.T, expected, actual interface{}) {
	if !reflect.DeepEqual(expected, actual) {
		t.Errorf("%#v is not as expected: %v", actual, expected)
	}
}
