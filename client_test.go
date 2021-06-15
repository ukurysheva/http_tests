package main

import (
	"encoding/json"
	"encoding/xml"
	"fmt"
	"io/ioutil"
	"net/http"
	"net/http/httptest"
	"os"
	"reflect"
	"sort"
	"strconv"
	"strings"
	"testing"
	"time"
)

type Client struct {
	ID        int    `xml:"id"`
	Age       int    `xml:"age"`
	Login     string `xml:"login"`
	FirstName string `xml:"first_name"`
	LastName  string `xml:"last_name"`
	Name      string
	About     string `xml:"about"`
	Gender    string `xml:"gender"`
}

type Clients struct {
	List []Client `xml:"row"`
}

type TestCase struct {
	Request *SearchRequest
	Result  *SearchResponse
	IsError bool
}
type FuncResult struct {
	UsersInfo *ReqResult
	Err       error
}
type ReqResult struct {
	Users    []User
	NextPage bool
}

func SearchServer(w http.ResponseWriter, r *http.Request) {
    var sleep bool = false

	AccessToken := r.Header.Get("AccessToken")
	switch AccessToken {
	case "root":
		w.WriteHeader(http.StatusBadRequest)
		ErrorResponse := &SearchErrorResponse {
			Error: "ErrorRootAnable",
		}
		result, err := json.Marshal(ErrorResponse)
		if err != nil {
			panic(err)
		}
		fmt.Fprintf(w, string(result))
		return
	case "sleep":
		w.Header().Set("Content-Type", "application/json; charset=utf-8")
		fmt.Fprintf(w, "")
		w.Write([]byte("[]"))
		timer := time.NewTimer(2 * time.Second)
		defer timer.Stop()
		select {
		case <-timer.C:
			return
		}
	case "":
		w.WriteHeader(http.StatusUnauthorized)
		fmt.Fprintln(w, "")
		return
	}

	xmlFile, err := os.Open("dataset.xml")
	if err != nil {
		fmt.Println(err)
	}

	defer xmlFile.Close()
	byteValue, _ := ioutil.ReadAll(xmlFile)
	XmlClients := ReadXml(byteValue)

	query := r.URL.Query().Get("query")
	orderField := r.URL.Query().Get("order_field")
	orderBy := r.URL.Query().Get("order_by")
	limit := r.URL.Query().Get("limit")
	offset := r.URL.Query().Get("offset")
	// Фильтруем
	if query != "" && query != "xml" {
		XmlClients = Filter(XmlClients, query)
	}

	// Сортируем
	if orderField == "" {
		orderField = "Name"
	}
    
	if (orderField != "Name" && orderField != "ID" && orderField != "Age") {
		w.WriteHeader(http.StatusBadRequest)
		ErrorResponse := &SearchErrorResponse {
			Error: "ErrorBadOrderField",
		}
		result, err := json.Marshal(ErrorResponse)
		if err != nil {
			panic(err)
		}
		fmt.Fprintf(w, string(result))
		return
	}

	var sortTypes map[string]sort.Interface = map[string]sort.Interface{
		"ID":   ByID(XmlClients),
		"Name": ByName(XmlClients),
		"Age":  ByAge(XmlClients),
	}

	if orderBy == "-1" {
		sort.Sort(sortTypes[orderField])
	} else if orderBy == "1" {
		sort.Sort(sort.Reverse(sortTypes[orderField]))
	} else {
		w.WriteHeader(http.StatusBadRequest)
		fmt.Fprintf(w, "error: bad orderby field")
		return
	}

	// Вносим ограничения
	if offset != "" && offset != "0" {
		i, err := strconv.Atoi(offset)
		if err != nil {
			panic(err)
		}
		if i > len(XmlClients) {
			XmlClients = []Client{}
		} else {
			XmlClients = XmlClients[i:]
		}
	}

	
	if limit != "" && len(XmlClients) > 0 {
		lim, err := strconv.Atoi(limit)
		lim--
		if err != nil {
			panic(err)
			return
		}
		if lim > len(XmlClients) {
			XmlClients = []Client{}
		} else if lim ==23 {
			lim+=1
			XmlClients = XmlClients[:lim]
		} else if lim>0 {
			XmlClients = XmlClients[:lim]
		} else if lim ==0 {
			w.WriteHeader(http.StatusInternalServerError)
			fmt.Fprintf(w, "")
			return
		}
	} else {
		w.WriteHeader(http.StatusInternalServerError)
		fmt.Fprintf(w, "")
		return
	}

	result, err := json.Marshal(XmlClients)
    if query == "xml" {
		result = []byte("xml")
	}
	

    // fmt.Println( string(result))
	w.Header().Set("Content-Type", "application/json; charset=utf-8")
	if sleep  {
		fmt.Fprintf(w, "")
		w.Write([]byte("[]"))
		timer := time.NewTimer(2 * time.Second)
		defer timer.Stop()

		select {
		case <-timer.C:
			return
		}
	} else {
		w.WriteHeader(http.StatusOK)
		fmt.Fprintf(w, string(result))
	}

	return
}

func Filter(rawClients []Client, query string) []Client {
	filtered := make([]Client, 0)
	for _, v := range rawClients {
		if strings.Contains(v.Name, query) || strings.Contains(v.About, query) {
			filtered = append(filtered, v)
		}
	}
	return filtered
}
func ReadXml(xmlData []byte) []Client {
	var clients Clients

	err := xml.Unmarshal(xmlData, &clients)
	if err != nil {
		fmt.Printf("error: %v", err)
		panic(err)
	}
	for i, val := range clients.List {
		clients.List[i].Name = val.FirstName + " " + val.LastName
	}
	return clients.List
}

func ReadXmlBase(xmlData []byte) []User {
	var clients Clients
	var users []User

	err := xml.Unmarshal(xmlData, &clients)
	if err != nil {
		fmt.Printf("error: %v", err)
		panic(err)
	}
	for _, val := range clients.List {
		val.Name = val.FirstName + " " + val.LastName
		var MyUser User = User{Id: val.ID, Name: val.Name, Age: val.Age, About: val.About, Gender: val.Gender}
		users = append(users, MyUser)
	}
	return users
}

type ByName []Client

func (a ByName) Len() int           { return len(a) }
func (a ByName) Less(i, j int) bool { return a[i].Name < a[j].Name }
func (a ByName) Swap(i, j int)      { a[i], a[j] = a[j], a[i] }

type ByAge []Client

func (a ByAge) Len() int           { return len(a) }
func (a ByAge) Less(i, j int) bool { return a[i].Age < a[j].Age }
func (a ByAge) Swap(i, j int)      { a[i], a[j] = a[j], a[i] }

type ByID []Client

func (a ByID) Len() int           { return len(a) }
func (a ByID) Less(i, j int) bool { return a[i].ID < a[j].ID }
func (a ByID) Swap(i, j int)      { a[i], a[j] = a[j], a[i] }

func TestLimitOffset(t *testing.T) {

	xmlFile, err := os.Open("dataset.xml")
	if err != nil {
		fmt.Println(err)
	}
	defer xmlFile.Close()
	byteValue, _ := ioutil.ReadAll(xmlFile)
	ClientsBase := ReadXmlBase(byteValue)
	cases := []TestCase{
		TestCase{
			Request: &SearchRequest{
				Limit:      1,
				Offset:     3,
				Query:      "",
				OrderField: "ID",
				OrderBy:    -1,
			},
			Result: &SearchResponse{
				Users: ClientsBase[3:4],
				NextPage: false,
			},
			IsError: false,
		},
		TestCase{
			Request: &SearchRequest{
				Limit:      -5,
				Offset:     0,
				Query:      "",
				OrderField: "ID",
				OrderBy:    -1,
			},
			Result:  nil,
			IsError: true,
		},
		TestCase{
			Request: &SearchRequest{
				Limit:      30,
				Offset:     -1,
				Query:      "",
				OrderField: "ID",
				OrderBy:    -1,
			},
			Result: nil,
			IsError: true,
		},
		TestCase{
			Request: &SearchRequest{
				Limit:      25,
				Offset:     0,
				Query:      "",
				OrderField: "ID",
				OrderBy:    -1,
			},
			Result: &SearchResponse{
				Users:    ClientsBase[:25],
				NextPage: false,
			},
			IsError: false,
		},
		TestCase{
			Request: &SearchRequest{
				Limit:      0,
				Offset:     3,
				Query:      "",
				OrderField: "ID",
				OrderBy:    -1,
			},
			Result: nil,
			IsError: true,
		},
	}

	ts := httptest.NewServer(http.HandlerFunc(SearchServer))

	for caseNum, item := range cases {
		c := &SearchClient{
			AccessToken: ts.URL,
			URL:         ts.URL,
		}
		result, err := c.FindUsers(*item.Request)

		if err != nil && !item.IsError {
			t.Errorf("[%d] unexpected error: %#v", caseNum, err)
		}
		if err == nil && item.IsError {
			t.Errorf("[%d] expected error, got nil", caseNum)
		}
		if !reflect.DeepEqual(item.Result, result) {
			t.Errorf("[%d] wrong result, expected %#v\n\n, got %#v", caseNum, item.Result, result)
		}
	}
	ts.Close()
}
func TestBadParams(t *testing.T) {

	cases := []TestCase{
		TestCase{
			Request: &SearchRequest{
				Limit:      1,
				Offset:     3,
				Query:      "",
				OrderField: "test",
				OrderBy:    -1,
			},
			Result: nil,
			IsError: true,
		},
		TestCase{
			Request: &SearchRequest{
				Limit:      1,
				Offset:     3,
				Query:      "",
				OrderField: "Name",
				OrderBy:    -3,
			},
			Result: nil,
			IsError: true,
		},
	}

	ts := httptest.NewServer(http.HandlerFunc(SearchServer))

	for caseNum, item := range cases {
		c := &SearchClient{
			AccessToken: ts.URL,
			URL:         ts.URL,
		}
		result, err := c.FindUsers(*item.Request)

		if err != nil && !item.IsError {
			t.Errorf("[%d] unexpected error: %#v", caseNum, err)
		}
		if err == nil && item.IsError {
			t.Errorf("[%d] expected error, got nil", caseNum)
		}
		if !reflect.DeepEqual(item.Result, result) {
			t.Errorf("[%d] wrong result, expected %#v\n\n, got %#v", caseNum, item.Result, result)
		}
	}
	ts.Close()
}

func TestTimeOut(t *testing.T) {

	cases := []TestCase{
		TestCase{
			Request: &SearchRequest{
				Limit:      1,
				Offset:     1,
				Query:      "",
				OrderField: "Name",
				OrderBy:    -1,
			},
			Result: nil,
			IsError: true,
		},
	}

	ts := httptest.NewServer(http.HandlerFunc(SearchServer))

	for caseNum, item := range cases {
		c := &SearchClient{
			AccessToken: "sleep",
			URL:         ts.URL,
		}
		result, err := c.FindUsers(*item.Request)
		if err != nil && !item.IsError {
			t.Errorf("[%d] unexpected error: %#v", caseNum, err)
		}
		if err == nil && item.IsError {
			t.Errorf("[%d] expected error, got nil", caseNum)
		}
		if !reflect.DeepEqual(item.Result, result) {
			
			t.Errorf("[%d] wrong result, expected %#v\n\n, got %#v", caseNum, item.Result, result)
		}
	}
	ts.Close()
}

func TestBadJson(t *testing.T) {

	cases := []TestCase{
		TestCase{
			Request: &SearchRequest{
				Limit:      5,
				Offset:     3,
				Query:      "xml",
				OrderField: "Name",
				OrderBy:    -1,
			},
			Result: nil,
			IsError: true,
		},
	}

	ts := httptest.NewServer(http.HandlerFunc(SearchServer))

	for caseNum, item := range cases {
		c := &SearchClient{
			AccessToken: ts.URL,
			URL:         ts.URL,
		}
		result, err := c.FindUsers(*item.Request)
		if err != nil && !item.IsError {
			t.Errorf("[%d] unexpected error: %#v", caseNum, err)
		}
		if err == nil && item.IsError {
			t.Errorf("[%d] expected error, got nil", caseNum)
		}
		if !reflect.DeepEqual(item.Result, result) {
			
			t.Errorf("[%d] wrong result, expected %#v\n\n, got %#v", caseNum, item.Result, result)
		}
	}
	ts.Close()
}

func TestNextPage(t *testing.T) {

	xmlFile, err := os.Open("dataset.xml")
	if err != nil {
		fmt.Println(err)
	}
	defer xmlFile.Close()
	byteValue, _ := ioutil.ReadAll(xmlFile)
	ClientsBase := ReadXmlBase(byteValue)
	cases := []TestCase{
		TestCase{
			Request: &SearchRequest{
				Limit:      23,
				Offset:     0,
				Query:      "",
				OrderField: "ID",
				OrderBy:    -1,
			},
			Result: &SearchResponse{
				Users: ClientsBase[:23],
				NextPage: true,
			},
			IsError: false,
		},
	}
	ts := httptest.NewServer(http.HandlerFunc(SearchServer))

	for caseNum, item := range cases {
		c := &SearchClient{
			AccessToken: ts.URL,
			URL:         ts.URL,
		}
		result, err := c.FindUsers(*item.Request)

		if err != nil && !item.IsError {
			t.Errorf("[%d] unexpected error: %#v", caseNum, err)
		}
		if err == nil && item.IsError {
			t.Errorf("[%d] expected error, got nil", caseNum)
		}
		if !reflect.DeepEqual(item.Result, result) {
			t.Errorf("[%d] wrong result, expected %#v\n\n, got %#v", caseNum, item.Result, result)
		}
	}
	ts.Close()
}


func TestRequestParam(t *testing.T) {

	ts := httptest.NewServer(http.HandlerFunc(SearchServer))
	cases := []SearchClient{
		SearchClient{
			AccessToken: ts.URL,
			URL:         "",
		},
		SearchClient{
			AccessToken: "",
			URL:         ts.URL,
		},
		SearchClient{
			AccessToken: "root",
			URL:         ts.URL,
		},
	}
    
	for caseNum, item := range cases {
		ReqParams := &TestCase{
			Request: &SearchRequest{
				Limit:      25,
				Offset:     0,
				Query:      "",
				OrderField: "ID",
				OrderBy:    -1,
			},
			Result:  nil,
			IsError: true,
		}
		result, err := item.FindUsers(*ReqParams.Request)
		if err != nil && !ReqParams.IsError {
			t.Errorf("[%d] unexpected error: %#v", caseNum, err)
		}
		if err == nil && ReqParams.IsError {
			t.Errorf("[%d] expected error, got nil", caseNum)
		}
		if !reflect.DeepEqual(ReqParams.Result, result) {
			t.Errorf("[%d] wrong result, expected %#v\n\n, got %#v", caseNum, ReqParams.Result, result)
		}
	}
	ts.Close()
}

