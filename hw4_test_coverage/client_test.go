package main

import (
	"encoding/json"
	"encoding/xml"
	"fmt"
	"io"
	"net/http"
	"net/http/httptest"
	"os"
	"sort"
	"strconv"
	"strings"
	"testing"
	"time"
)

func SearchServer(w http.ResponseWriter, r *http.Request) {
	var offset, limit int

	query := r.URL.Query().Get("query")
	limit, err := strconv.Atoi(r.URL.Query().Get("limit"))
	if err != nil || limit < 0 {
		w.Write([]byte(`{"Error":"Incorrect limit param""}`))
		w.WriteHeader(http.StatusBadRequest)
		return
	}
	if limit > 25 {
		limit = 25
	}

	offset, err = strconv.Atoi(r.URL.Query().Get("offset"))
	if err != nil || offset < 0 {
		w.Write([]byte(`{"Error":"Incorrect offset param""}`))
		w.WriteHeader(http.StatusBadRequest)
		return
	}

	orderField := r.URL.Query().Get("order_field")
	if orderField != "" && orderField != "Id" && orderField != "Age" && orderField != "Name" {
		w.Write([]byte(`{"Error":"Incorrect order_field param""}`))
		w.WriteHeader(http.StatusBadRequest)
		return
	}
	if orderField == "" {
		orderField = "Name"
	}

	orderBy := r.URL.Query().Get("order_by")
	if orderBy != "" && orderBy != "1" && orderBy != "-1" && orderBy != "0" {
		w.Write([]byte(`{"Error":"Incorrect order_by param"}`))
		w.WriteHeader(http.StatusBadRequest)
		return
	}
	if orderBy == "" {
		orderBy = "1"
	}

	xmlFile, err := os.Open("dataset.xml")
	if err != nil {
		w.WriteHeader(http.StatusInternalServerError)
		panic(err)
	}
	defer xmlFile.Close()

	users := []User{}
	decoder := xml.NewDecoder(xmlFile)
	for {
		tok, tokenErr := decoder.Token()
		if tokenErr != nil && tokenErr != io.EOF {
			panic(tokenErr)
		} else if tokenErr == io.EOF {
			break
		}
		if tok == nil {

		}
		switch tok := tok.(type) {
		case xml.StartElement:
			if tok.Name.Local == "row" {
				user := User{}
				if err := decoder.DecodeElement(&user, &tok); err != nil {
					fmt.Println(err)
				}
				user.Name = user.FirstName + user.LastName
				if query != "" && (strings.Contains(user.Name, query) ||
					strings.Contains(user.About, query)) {
					users = append(users, user)
				} else if query == "" {
					users = append(users, user)
				}
			}
		}
	}
	sort.Slice(users, func(i, j int) bool {
		if orderBy == "1" {
			switch orderField {
			case "Id":
				return users[i].Id > users[j].Id
			case "Name":
				return users[i].Name > users[j].Name
			case "Age":
				return users[i].Age > users[j].Age
			}
		} else {
			switch orderField {
			case "Id":
				return users[i].Id < users[j].Id
			case "Name":
				return users[i].Name < users[j].Name
			case "Age":
				return users[i].Age < users[j].Age
			}
		}
		return users[i].Name > users[j].Name
	})

	len_res := len(users)
	if offset+limit >= len_res {
		if offset < len_res {
			users = users[offset:]
		} else {
			w.WriteHeader(http.StatusOK)
			w.Write([]byte("[]"))
			return
		}
	} else {
		users = users[offset:(limit + offset)]
	}
	if len(users) == 0 {
		w.WriteHeader(http.StatusOK)
		w.Write([]byte("[]"))
		return
	}

	var jsonData []byte
	jsonData, err = json.Marshal(users)
	if err != nil {
		panic(err)
	}
	w.WriteHeader(http.StatusOK)
	w.Write(jsonData)
}
func TestFindUsersStatusInternalServerError(t *testing.T) {
	handlerFunc := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusInternalServerError)
	})
	ts := httptest.NewServer(http.HandlerFunc(handlerFunc))
	defer ts.Close()

	request := SearchRequest{
		Limit:  1,
		Offset: 1,
	}

	searchClient := &SearchClient{}
	searchClient.URL = ts.URL

	_, err := searchClient.FindUsers(request)
	if err == nil || !strings.Contains(err.Error(), "SearchServer fatal error") {
		t.Errorf("expected error, got nil")
	}
}

func TestFindUsersTimeOutError(t *testing.T) {
	handlerFunc := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		time.Sleep(time.Second + 10*time.Millisecond)
	})
	ts := httptest.NewServer(http.HandlerFunc(handlerFunc))
	defer ts.Close()

	request := SearchRequest{
		Limit:  1,
		Offset: 1,
	}

	searchClient := &SearchClient{}
	searchClient.URL = ts.URL

	_, err := searchClient.FindUsers(request)
	if err == nil || !strings.Contains(err.Error(), "timeout for") {
		t.Errorf("expected error, got nil")
	}
}

func TestFindUsersUnknownError(t *testing.T) {
	request := SearchRequest{
		Limit:  1,
		Offset: 1,
	}

	searchClient := &SearchClient{}
	_, err := searchClient.FindUsers(request)
	if err == nil || !strings.Contains(err.Error(), "unknown error") {
		t.Errorf("expected error, got nil")
	}
}

func TestFindUsersIncorrectLimitAndOffset(t *testing.T) {
	type TestCase struct {
		Name    string
		Request *SearchRequest
		Error   string
	}

	cases := []TestCase{
		TestCase{
			Name: "Incorrect Limit param",
			Request: &SearchRequest{
				Limit: -1,
			},
			Error: "limit must be > 0",
		},
		TestCase{
			Name: "Incorrect Offset param",
			Request: &SearchRequest{
				Limit:  10,
				Offset: -1,
			},
			Error: "offset must be > 0",
		},
	}

	ts := httptest.NewServer(nil)
	defer ts.Close()

	for caseNum, item := range cases {

		searchClient := &SearchClient{}
		searchClient.URL = ts.URL

		_, err := searchClient.FindUsers(*item.Request)
		if err == nil || !strings.Contains(err.Error(), item.Error) {
			t.Errorf("[%d] unexpected error: %#v", caseNum, err)
		}
	}
}

func TestFindUsersStatusUnauthorized(t *testing.T) {
	ts := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		accessToken := r.URL.Query().Get("AccessToken")
		if accessToken == "" {
			w.WriteHeader(http.StatusUnauthorized)
		}
	}))
	defer ts.Close()

	searchClient := &SearchClient{}
	searchClient.URL = ts.URL

	request := &SearchRequest{
		Limit:  1,
		Offset: 1,
	}

	_, err := searchClient.FindUsers(*request)
	if err == nil || !strings.Contains(err.Error(), "Bad AccessToken") {
		t.Errorf("expected error, got nil")
	}
}

func TestFindUsersStatusBadRequest(t *testing.T) {
	type TestCase struct {
		Body    string
		Request *SearchRequest
		Error   string
	}
	cases := []TestCase{
		TestCase{
			Body: "{",
			Request: &SearchRequest{
				Limit:  10,
				Offset: 10,
			},
			Error: "cant unpack error json:",
		},
		TestCase{
			Body: `{"Error":"ErrorBadOrderField"}`,
			Request: &SearchRequest{
				Limit:  10,
				Offset: 10,
			},
			Error: "OrderFeld",
		},
		TestCase{
			Body: `{"Error":"Unknown Error"}`,
			Request: &SearchRequest{
				Limit:  10,
				Offset: 10,
			},
			Error: "unknown bad request error",
		},
	}

	for _, item := range cases {
		ts := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			w.WriteHeader(http.StatusBadRequest)
			w.Write([]byte(item.Body))
		}))

		searchClient := &SearchClient{}
		searchClient.URL = ts.URL

		request := &SearchRequest{
			Limit:  1,
			Offset: 1,
		}

		_, err := searchClient.FindUsers(*request)
		if err == nil || !strings.Contains(err.Error(), item.Error) {
			t.Errorf("expected error, got nil")
		}
		defer ts.Close()
	}

}

func TestFindUsersCantUnpackResult(t *testing.T) {
	handlerFunc := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Write([]byte("INCORRECT JSON{"))
	})
	ts := httptest.NewServer(http.HandlerFunc(handlerFunc))
	defer ts.Close()

	request := SearchRequest{
		Limit:  1,
		Offset: 1,
	}

	searchClient := &SearchClient{}
	searchClient.URL = ts.URL

	_, err := searchClient.FindUsers(request)
	if err == nil || !strings.Contains(err.Error(), "cant unpack result json") {
		t.Errorf("expected error, got nil")
	}
}

func TestFindUsers(t *testing.T) {
	handlerFunc := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Write([]byte("INCORRECT JSON{"))
	})
	ts := httptest.NewServer(http.HandlerFunc(handlerFunc))
	defer ts.Close()

	request := SearchRequest{
		Limit:  1,
		Offset: 1,
	}

	searchClient := &SearchClient{}
	searchClient.URL = ts.URL

	_, err := searchClient.FindUsers(request)
	if err == nil || !strings.Contains(err.Error(), "cant unpack result json") {
		t.Errorf("expected error, got nil")
	}
}

func TestFindUsersSuccess(t *testing.T) {
	ts := httptest.NewServer(http.HandlerFunc(SearchServer))
	defer ts.Close()

	type TestCase struct {
		Request *SearchRequest
		Count   int
		Ids     []int
	}

	compareSlices := func(a, b []int) bool {
		if len(a) != len(b) {
			return false
		}
		for i := range a {
			if a[i] != b[i] {
				return false
			}
		}
		return true
	}

	cases := []TestCase{
		TestCase{
			Request: &SearchRequest{
				Limit:      4,
				Offset:     2,
				OrderBy:    -1,
				OrderField: "Id",
			},
			Count: 4,
			Ids:   []int{2, 3, 4, 5},
		},
		TestCase{
			Request: &SearchRequest{
				Limit:      4,
				Offset:     2,
				OrderBy:    1,
				OrderField: "Id",
			},
			Count: 4,
			Ids:   []int{32, 31, 30, 29},
		},
		// With  correct query param
		TestCase{
			Request: &SearchRequest{
				Limit:      4,
				Offset:     0,
				OrderBy:    1,
				OrderField: "Id",
				Query:      "Christy",
			},
			Count: 1,
			Ids:   []int{32},
		},
		// With empty result of searching
		TestCase{
			Request: &SearchRequest{
				Limit:      4,
				Offset:     0,
				OrderBy:    1,
				OrderField: "Id",
				Query:      "blablablablablabla",
			},
			Count: 0,
			Ids:   []int{},
		},
		TestCase{
			Request: &SearchRequest{
				Limit:      4,
				Offset:     32,
				OrderBy:    -1,
				OrderField: "Id",
			},
			Count: 3,
			Ids:   []int{32, 33, 34},
		},
		TestCase{
			Request: &SearchRequest{
				Limit:      30,
				Offset:     32,
				OrderBy:    -1,
				OrderField: "Id",
			},
			Count: 3,
			Ids:   []int{32, 33, 34},
		},
	}

	searchClient := &SearchClient{}
	searchClient.URL = ts.URL
	for itemNum, item := range cases {
		res, err := searchClient.FindUsers(*item.Request)
		if err != nil {
			fmt.Println(err)
		}
		countUsers := len(res.Users)
		if item.Count == 0 && item.Count == countUsers {
			continue
		}
		var ids []int
		for _, u := range res.Users {
			ids = append(ids, u.Id)
		}
		if countUsers != item.Count {
			t.Errorf("[%d] Expected count of result: %d; equal: %d", itemNum, item.Count, countUsers)
		}
		if !compareSlices(ids, item.Ids) {
			t.Errorf("[%d] Expected %v instance of %v", itemNum, item.Ids, ids)
		}
	}
}
